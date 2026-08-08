package service

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// newCorrectionSvc 构造绑定测试事务的补卡服务（审批引擎与补卡共用同一事务）
func newCorrectionSvc(tx *gorm.DB) *AttendanceCorrectionService {
	return NewAttendanceCorrectionService(tx, repository.NewAttendanceCorrectionRepo(tx), repository.NewEmployeeRepo(tx))
}

// approveCorrectionTask 模拟 handler 审批链路：事务内审批任务，终审通过时调用业务回调
func approveCorrectionTask(t *testing.T, tx *gorm.DB, svc *AttendanceCorrectionService, taskID uint, approverUserID uint) *models.ApprovalInstance {
	t.Helper()
	inst, err := ApproveTask(tx, taskID, approverUserID, "同意")
	if err != nil {
		t.Fatalf("审批通过失败: %v", err)
	}
	if inst.Status == "approved" {
		if err := svc.ApplyApproved(tx, inst.BizID); err != nil {
			t.Fatalf("终审通过回调失败: %v", err)
		}
	}
	return inst
}

// corrDate 生成补卡日期字符串（今天或昨天，满足 30 天内且不晚于今天）
func corrDate() string {
	return time.Now().AddDate(0, 0, -1).Format("2006-01-02")
}

// TestCorrectionCreateStartsApproval 提交补卡创建审批实例且步骤1指派给部门经理，业务单保持 pending
func TestCorrectionCreateStartsApproval(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newCorrectionSvc(tx)

	cor, err := svc.Create(fx.EmpUserID, SubmitCorrectionRequest{
		WorkDate: corrDate(), CorrectionType: "check_in", CorrectionTime: "09:00", Reason: "忘打卡",
	})
	if err != nil {
		t.Fatalf("提交补卡失败: %v", err)
	}
	if cor.Status != "pending" {
		t.Errorf("期望 pending，实际 %s", cor.Status)
	}
	inst, err := GetInstanceByBiz(tx, "attendance_correction", cor.ID)
	if err != nil {
		t.Fatalf("审批实例应已创建: %v", err)
	}
	task := stepTask(t, tx, inst, 1)
	if task.ApproverID == nil || *task.ApproverID != fx.ManagerUserID {
		t.Errorf("步骤1审批人应为部门经理，实际 %v", task.ApproverID)
	}
}

// TestCorrectionApproveAppliesAttendance 两级审批通过后考勤记录写入补卡打卡时间
func TestCorrectionApproveAppliesAttendance(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newCorrectionSvc(tx)

	workDate := corrDate()
	cor, err := svc.Create(fx.EmpUserID, SubmitCorrectionRequest{
		WorkDate: workDate, CorrectionType: "check_in", CorrectionTime: "09:00", Reason: "忘打卡",
	})
	if err != nil {
		t.Fatalf("提交补卡失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "attendance_correction", cor.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	approveCorrectionTask(t, tx, svc, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID) // 经理审批
	approveCorrectionTask(t, tx, svc, stepTask(t, tx, inst, 2).ID, fx.HRUserID)      // HR 终审

	att, err := repository.NewAttendanceRepo(tx).FindByEmployeeAndDate(fx.Emp.ID, workDate)
	if err != nil {
		t.Fatalf("查询考勤失败: %v", err)
	}
	if att.CheckIn == nil {
		t.Fatal("补卡通过后应有上班打卡时间")
	}
	if got := att.CheckIn.Format("15:04"); got != "09:00" {
		t.Errorf("期望打卡时间 09:00，实际 %s", got)
	}
	if att.Status != "normal" {
		t.Errorf("09:00 打卡应 normal，实际 %s", att.Status)
	}
	// 业务单状态置 approved
	c, err := repository.NewAttendanceCorrectionRepo(tx).FindByID(cor.ID)
	if err != nil {
		t.Fatalf("查询补卡单失败: %v", err)
	}
	if c.Status != "approved" {
		t.Errorf("期望补卡单 approved，实际 %s", c.Status)
	}
}

// TestCorrectionReject 经理驳回后补卡单置 rejected 且不写考勤
func TestCorrectionReject(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newCorrectionSvc(tx)

	workDate := corrDate()
	cor, err := svc.Create(fx.EmpUserID, SubmitCorrectionRequest{
		WorkDate: workDate, CorrectionType: "check_out", CorrectionTime: "18:30", Reason: "忘打卡",
	})
	if err != nil {
		t.Fatalf("提交补卡失败: %v", err)
	}
	inst, err := GetInstanceByBiz(tx, "attendance_correction", cor.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	if err := RejectTask(tx, stepTask(t, tx, inst, 1).ID, fx.ManagerUserID, "记录不符"); err != nil {
		t.Fatalf("驳回失败: %v", err)
	}
	if err := svc.ApplyRejected(tx, inst.BizID, "记录不符"); err != nil {
		t.Fatalf("驳回回调失败: %v", err)
	}

	c, err := repository.NewAttendanceCorrectionRepo(tx).FindByID(cor.ID)
	if err != nil {
		t.Fatalf("查询补卡单失败: %v", err)
	}
	if c.Status != "rejected" {
		t.Errorf("期望状态 rejected，实际 %s", c.Status)
	}
	if c.RejectReason != "记录不符" {
		t.Errorf("期望驳回原因记录，实际 %s", c.RejectReason)
	}
	// 驳回不应写考勤
	if _, err := repository.NewAttendanceRepo(tx).FindByEmployeeAndDate(fx.Emp.ID, workDate); err == nil {
		t.Errorf("驳回后不应创建考勤记录")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("查询考勤失败: %v", err)
	}
}

// TestCorrectionCancel 申请人取消：补卡单 cancelled 且审批实例同步 cancelled
func TestCorrectionCancel(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	fx := setupLeaveFixture(t, tx)
	svc := newCorrectionSvc(tx)

	cor, err := svc.Create(fx.EmpUserID, SubmitCorrectionRequest{
		WorkDate: corrDate(), CorrectionType: "check_in", CorrectionTime: "09:00", Reason: "记错日期",
	})
	if err != nil {
		t.Fatalf("提交补卡失败: %v", err)
	}
	if err := svc.Cancel(fx.EmpUserID, cor.ID); err != nil {
		t.Fatalf("取消补卡失败: %v", err)
	}
	c, err := repository.NewAttendanceCorrectionRepo(tx).FindByID(cor.ID)
	if err != nil {
		t.Fatalf("查询补卡单失败: %v", err)
	}
	if c.Status != "cancelled" {
		t.Errorf("期望补卡单 cancelled，实际 %s", c.Status)
	}
	// 实例同步取消
	got, err := GetInstanceByBiz(tx, "attendance_correction", cor.ID)
	if err != nil {
		t.Fatalf("查询审批实例失败: %v", err)
	}
	if got.Status != "cancelled" {
		t.Errorf("期望审批实例 cancelled，实际 %s", got.Status)
	}
}
