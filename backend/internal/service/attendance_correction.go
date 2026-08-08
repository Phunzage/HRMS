package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 补卡申请业务错误
var (
	ErrCorrNotFound       = errors.New("补卡申请不存在")
	ErrCorrEmpNoUser      = errors.New("当前账号未关联员工档案")
	ErrCorrTypeBad        = errors.New("补卡类型不合法，应为 check_in 或 check_out")
	ErrCorrDateBad        = errors.New("日期格式错误，应为 YYYY-MM-DD")
	ErrCorrDateTooOld     = errors.New("补卡日期不得早于 30 天前")
	ErrCorrDateFuture     = errors.New("补卡日期不得晚于今天")
	ErrCorrTimeBad        = errors.New("补卡时间格式错误，应为 HH:MM")
	ErrCorrReasonRequired = errors.New("补卡原因不能为空")
	ErrCorrNotOwner       = errors.New("只能操作自己的补卡申请")
	ErrCorrNotPending     = repository.ErrCorrectionNotPending
)

// 补卡类型中文名
var correctionTypeNames = map[string]string{
	"check_in":  "上班打卡",
	"check_out": "下班打卡",
}

// SubmitCorrectionRequest 提交补卡申请请求
type SubmitCorrectionRequest struct {
	WorkDate       string `json:"work_date"`        // YYYY-MM-DD
	CorrectionType string `json:"correction_type"`  // check_in/check_out
	CorrectionTime string `json:"correction_time"`  // HH:MM
	Reason         string `json:"reason"`
}

// AttendanceCorrectionService 补卡申请业务服务
type AttendanceCorrectionService struct {
	db       *gorm.DB // 数据范围解析与事务用
	corrRepo *repository.AttendanceCorrectionRepo
	empRepo  *repository.EmployeeRepo
}

// NewAttendanceCorrectionService 创建补卡申请服务
func NewAttendanceCorrectionService(
	db *gorm.DB,
	corrRepo *repository.AttendanceCorrectionRepo,
	empRepo *repository.EmployeeRepo,
) *AttendanceCorrectionService {
	return &AttendanceCorrectionService{db: db, corrRepo: corrRepo, empRepo: empRepo}
}

// Create 提交补卡申请：校验日期范围与类型，创建补卡单并启动审批流程（部门经理 → HR 终审）
func (s *AttendanceCorrectionService) Create(userID uint, req SubmitCorrectionRequest) (*models.AttendanceCorrection, error) {
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return nil, err
	}
	if req.CorrectionType != "check_in" && req.CorrectionType != "check_out" {
		return nil, ErrCorrTypeBad
	}
	if req.Reason == "" {
		return nil, ErrCorrReasonRequired
	}
	workDate, err := time.ParseInLocation("2006-01-02", req.WorkDate, time.Local)
	if err != nil {
		return nil, ErrCorrDateBad
	}
	if _, err := time.Parse("15:04", req.CorrectionTime); err != nil {
		return nil, ErrCorrTimeBad
	}
	// 校验日期范围：不得晚于今天、不得早于 30 天前
	today := dateOnly(time.Now())
	if workDate.After(today) {
		return nil, ErrCorrDateFuture
	}
	if workDate.Before(today.AddDate(0, 0, -30)) {
		return nil, ErrCorrDateTooOld
	}

	cor := &models.AttendanceCorrection{
		EmployeeID:     empID,
		WorkDate:       workDate,
		Reason:         req.Reason,
		CorrectionType: req.CorrectionType,
		CorrectionTime: req.CorrectionTime,
	}

	// 兼容字段：记录部门经理（员工ID），仅作列表展示用，审批依据以审批引擎为准
	managerID, err := s.corrRepo.FindManagerEmployeeID(empID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if managerID != nil && *managerID > 0 {
		cor.ApproverID = managerID
	}

	// 事务：创建补卡单 + 启动审批流程（审批实例 + 步骤1经理任务 + 通知经理）
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		corrRepo := repository.NewAttendanceCorrectionRepo(tx)
		if err := corrRepo.Create(cor); err != nil {
			return err
		}
		_, err := StartApproval(tx, "attendance_correction", cor.ID, userID)
		return err
	}); err != nil {
		return nil, err
	}
	return cor, nil
}

// List 补卡申请分页列表，按角色范围过滤（admin/hr 全部；manager 本部门+待自己审批；employee 仅自己）
func (s *AttendanceCorrectionService) List(roleCode string, userID uint, status string, employeeID uint, page, pageSize int) ([]repository.CorrectionItem, int64, error) {
	scope, err := s.buildScope(roleCode, userID)
	if err != nil {
		return nil, 0, err
	}
	return s.corrRepo.List(scope, status, employeeID, page, pageSize)
}

// Mine 当前用户自己的补卡记录（登录即可）
func (s *AttendanceCorrectionService) Mine(userID uint, page, pageSize int) ([]repository.CorrectionItem, int64, error) {
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return nil, 0, err
	}
	return s.corrRepo.ListByEmployee(empID, "", page, pageSize)
}

// ApplyCorrectionApproved 补卡终审通过回调（事务内调用）：状态→approved + 更新对应考勤打卡时间并重算状态 + 通知申请人
func (s *AttendanceCorrectionService) ApplyCorrectionApproved(tx *gorm.DB, correctionID uint) error {
	corrRepo := repository.NewAttendanceCorrectionRepo(tx)
	var cor models.AttendanceCorrection
	if err := tx.First(&cor, correctionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCorrNotFound
		}
		return err
	}
	if cor.Status != "pending" {
		return ErrCorrNotPending
	}
	patch, err := s.buildAttendancePatch(tx, &cor)
	if err != nil {
		return err
	}
	if err := corrRepo.Approve(correctionID, patch); err != nil {
		return err
	}
	// 通知申请人审批通过
	notifyCorrection(tx, &cor, "approve")
	return nil
}

// ApplyCorrectionRejected 补卡驳回回调（事务内调用）：状态→rejected + 通知申请人（含驳回原因）
func (s *AttendanceCorrectionService) ApplyCorrectionRejected(tx *gorm.DB, correctionID uint, reason string) error {
	corrRepo := repository.NewAttendanceCorrectionRepo(tx)
	var cor models.AttendanceCorrection
	if err := tx.First(&cor, correctionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCorrNotFound
		}
		return err
	}
	if cor.Status != "pending" {
		return ErrCorrNotPending
	}
	if err := corrRepo.Reject(correctionID, reason); err != nil {
		return err
	}
	cor.Status = "rejected"
	cor.RejectReason = reason
	// 通知申请人驳回原因
	notifyCorrection(tx, &cor, "reject")
	return nil
}

// ApplyApproved 实现审批终审回调接口：委托给补卡终审通过回调
func (s *AttendanceCorrectionService) ApplyApproved(tx *gorm.DB, bizID uint) error {
	return s.ApplyCorrectionApproved(tx, bizID)
}

// ApplyRejected 实现审批终审回调接口：委托给补卡驳回回调
func (s *AttendanceCorrectionService) ApplyRejected(tx *gorm.DB, bizID uint, reason string) error {
	return s.ApplyCorrectionRejected(tx, bizID, reason)
}

// Cancel 申请人取消自己的 pending 补卡申请：同事务取消补卡单 + 取消审批实例
func (s *AttendanceCorrectionService) Cancel(userID uint, id uint) error {
	cor, err := s.corrRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCorrNotFound
		}
		return err
	}
	if cor.Status != "pending" {
		return ErrCorrNotPending
	}
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return err
	}
	if cor.EmployeeID != empID {
		return ErrCorrNotOwner
	}
	// 事务：取消补卡单 + 取消审批实例（保持状态一致）
	return s.db.Transaction(func(tx *gorm.DB) error {
		corrRepo := repository.NewAttendanceCorrectionRepo(tx)
		if err := corrRepo.Cancel(id); err != nil {
			return err
		}
		inst, err := GetInstanceByBiz(tx, "attendance_correction", id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil // 无审批实例（历史数据）则仅取消业务单
			}
			return err
		}
		return CancelApproval(tx, inst.ID, userID)
	})
}

// buildScope 按角色构建列表访问范围
func (s *AttendanceCorrectionService) buildScope(roleCode string, userID uint) (repository.CorrectionScope, error) {
	switch roleCode {
	case "admin", "hr":
		return repository.CorrectionScope{}, nil
	case "manager":
		empID, err := s.requireEmployee(userID)
		if err != nil {
			return repository.CorrectionScope{Empty: true}, nil
		}
		emp, err := s.empRepo.FindByID(empID)
		if err != nil {
			return repository.CorrectionScope{Empty: true}, nil
		}
		return repository.CorrectionScope{DepartmentID: emp.DepartmentID, ApproverID: empID}, nil
	default: // employee
		empID, err := s.requireEmployee(userID)
		if err != nil {
			return repository.CorrectionScope{Empty: true}, nil
		}
		return repository.CorrectionScope{EmployeeID: empID}, nil
	}
}

// buildAttendancePatch 根据补卡申请计算目标考勤打卡时间与重算状态（供审批通过回调使用）
func (s *AttendanceCorrectionService) buildAttendancePatch(tx *gorm.DB, cor *models.AttendanceCorrection) (repository.CorrectionAttendance, error) {
	t, err := time.ParseInLocation("15:04", cor.CorrectionTime, time.Local)
	if err != nil {
		return repository.CorrectionAttendance{}, ErrCorrTimeBad
	}
	// 用 time.Local 显式指定时区：cor.WorkDate 从 PostgreSQL date 列读出时 Location 为 UTC，
	// 若沿用其 Location 会把 HH:MM 当作 UTC 时间写入，导致打卡时间偏移 8 小时
	correctionTime := time.Date(cor.WorkDate.Year(), cor.WorkDate.Month(), cor.WorkDate.Day(),
		t.Hour(), t.Minute(), 0, 0, time.Local)

	workDate := cor.WorkDate.Format("2006-01-02")
	attRepo := repository.NewAttendanceRepo(tx)
	att, err := attRepo.FindByEmployeeAndDate(cor.EmployeeID, workDate)
	var checkIn, checkOut *time.Time
	if err == nil {
		checkIn = att.CheckIn
		checkOut = att.CheckOut
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.CorrectionAttendance{}, err
	}
	if cor.CorrectionType == "check_in" {
		checkIn = &correctionTime
	} else {
		checkOut = &correctionTime
	}
	return repository.CorrectionAttendance{
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Status:   computeAttendanceStatus(checkIn, checkOut),
	}, nil
}

// notifyCorrection 补卡通知链路：审批通过/驳回→通知申请人（事务内调用，通知随业务同事务提交）
func notifyCorrection(tx *gorm.DB, cor *models.AttendanceCorrection, action string) {
	dateStr := cor.WorkDate.Format("2006-01-02")
	notif := NewNotificationService(repository.NewNotificationRepo(tx), repository.NewEmployeeRepo(tx))
	switch action {
	case "approve":
		notif.SendToEmployee(cor.EmployeeID, NotifTypeAttCorrection,
			"补卡审批通过",
			fmt.Sprintf("%s（%s %s）已通过", correctionTypeNames[cor.CorrectionType], dateStr, cor.CorrectionTime),
			"/attendance")
	case "reject":
		notif.SendToEmployee(cor.EmployeeID, NotifTypeAttCorrection,
			"补卡审批驳回",
			fmt.Sprintf("原因：%s", cor.RejectReason),
			"/attendance")
	}
}

// requireEmployee 根据登录用户解析关联的员工 ID
func (s *AttendanceCorrectionService) requireEmployee(userID uint) (uint, error) {
	empID, err := s.empRepo.FindEmployeeIDByUserID(userID)
	if err != nil {
		return 0, err
	}
	if empID == nil {
		return 0, ErrCorrEmpNoUser
	}
	return *empID, nil
}
