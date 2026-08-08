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
	ErrCorrNotApprover    = errors.New("无权限审批该补卡申请")
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
	corrRepo *repository.AttendanceCorrectionRepo
	attRepo  *repository.AttendanceRepo
	empRepo  *repository.EmployeeRepo
	notif    *NotificationService
}

// NewAttendanceCorrectionService 创建补卡申请服务
func NewAttendanceCorrectionService(
	corrRepo *repository.AttendanceCorrectionRepo,
	attRepo *repository.AttendanceRepo,
	empRepo *repository.EmployeeRepo,
	notif *NotificationService,
) *AttendanceCorrectionService {
	return &AttendanceCorrectionService{corrRepo: corrRepo, attRepo: attRepo, empRepo: empRepo, notif: notif}
}

// Create 提交补卡申请：校验日期范围与类型，自动匹配部门经理作为审批人，无经理则直接通过并应用考勤修正
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

	// 自动匹配审批人：员工所属部门经理；找不到则直接审批通过（同时应用考勤修正）
	managerID, err := s.corrRepo.FindManagerEmployeeID(empID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if managerID != nil && *managerID > 0 {
		cor.Status = "pending"
		cor.ApproverID = managerID
		if err := s.corrRepo.Create(cor); err != nil {
			return nil, err
		}
		s.notifyCorrection(cor, "submit")
		return cor, nil
	}
	// 无经理：直接审批通过并应用考勤修正（原子）
	cor.Status = "approved"
	now := time.Now()
	cor.ApprovedAt = &now
	patch, err := s.buildAttendancePatch(cor)
	if err != nil {
		return nil, err
	}
	if err := s.corrRepo.CreateAutoApproved(cor, patch); err != nil {
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

// Approve 审批通过：校验权限 + 状态，同事务更新考勤并置 approved，通知申请人
func (s *AttendanceCorrectionService) Approve(roleCode string, userID uint, id uint) error {
	cor, err := s.corrRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCorrNotFound
		}
		return err
	}
	if err := s.checkApprover(roleCode, userID, cor); err != nil {
		return err
	}
	patch, err := s.buildAttendancePatch(cor)
	if err != nil {
		return err
	}
	if err := s.corrRepo.Approve(id, patch); err != nil {
		return err
	}
	s.notifyCorrection(cor, "approve")
	return nil
}

// Reject 驳回补卡申请：校验权限，置 rejected，通知申请人
func (s *AttendanceCorrectionService) Reject(roleCode string, userID uint, id uint, reason string) error {
	cor, err := s.corrRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCorrNotFound
		}
		return err
	}
	if err := s.checkApprover(roleCode, userID, cor); err != nil {
		return err
	}
	if err := s.corrRepo.Reject(id, reason); err != nil {
		return err
	}
	cor.Status = "rejected"
	cor.RejectReason = reason
	s.notifyCorrection(cor, "reject")
	return nil
}

// Cancel 申请人取消自己的 pending 补卡申请（登录即可）
func (s *AttendanceCorrectionService) Cancel(userID uint, id uint) error {
	cor, err := s.corrRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCorrNotFound
		}
		return err
	}
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return err
	}
	if cor.EmployeeID != empID {
		return ErrCorrNotOwner
	}
	return s.corrRepo.Cancel(id)
}

// checkApprover 审批权限校验：admin/hr 可审批；manager 必须是该申请审批人
func (s *AttendanceCorrectionService) checkApprover(roleCode string, userID uint, cor *models.AttendanceCorrection) error {
	if roleCode == "admin" || roleCode == "hr" {
		return nil
	}
	if roleCode != "manager" {
		return ErrCorrNotApprover
	}
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return err
	}
	if cor.ApproverID == nil || *cor.ApproverID != empID {
		return ErrCorrNotApprover
	}
	return nil
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

// buildAttendancePatch 根据补卡申请计算目标考勤打卡时间与重算状态（供审批通过/自动通过使用）
func (s *AttendanceCorrectionService) buildAttendancePatch(cor *models.AttendanceCorrection) (repository.CorrectionAttendance, error) {
	t, err := time.ParseInLocation("15:04", cor.CorrectionTime, time.Local)
	if err != nil {
		return repository.CorrectionAttendance{}, ErrCorrTimeBad
	}
	// 用 time.Local 显式指定时区：cor.WorkDate 从 PostgreSQL date 列读出时 Location 为 UTC，
	// 若沿用其 Location 会把 HH:MM 当作 UTC 时间写入，导致打卡时间偏移 8 小时
	correctionTime := time.Date(cor.WorkDate.Year(), cor.WorkDate.Month(), cor.WorkDate.Day(),
		t.Hour(), t.Minute(), 0, 0, time.Local)

	workDate := cor.WorkDate.Format("2006-01-02")
	att, err := s.attRepo.FindByEmployeeAndDate(cor.EmployeeID, workDate)
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

// notifyCorrection 补卡通知链路：提交→通知审批人；审批通过/驳回→通知申请人
func (s *AttendanceCorrectionService) notifyCorrection(cor *models.AttendanceCorrection, action string) {
	dateStr := cor.WorkDate.Format("2006-01-02")
	switch action {
	case "submit":
		if cor.ApproverID == nil || *cor.ApproverID == 0 {
			return
		}
		emp, err := s.empRepo.FindByID(cor.EmployeeID)
		if err != nil {
			return
		}
		s.notif.SendToEmployee(*cor.ApproverID, NotifTypeAttCorrection,
			"新的补卡申请待审批",
			fmt.Sprintf("%s 提交了%s补卡申请（%s %s）", emp.Name, correctionTypeNames[cor.CorrectionType], dateStr, cor.CorrectionTime),
			"/attendance")
	case "approve":
		s.notif.SendToEmployee(cor.EmployeeID, NotifTypeAttCorrection,
			"补卡审批通过",
			fmt.Sprintf("%s（%s %s）已通过", correctionTypeNames[cor.CorrectionType], dateStr, cor.CorrectionTime),
			"/attendance")
	case "reject":
		s.notif.SendToEmployee(cor.EmployeeID, NotifTypeAttCorrection,
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
