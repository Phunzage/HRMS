package service

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 请假业务错误
var (
	ErrLeaveNotFound       = errors.New("请假申请不存在")
	ErrLeaveEmpNoUser      = errors.New("当前账号未关联员工档案")
	ErrLeaveEmpNotFound    = errors.New("员工不存在")
	ErrLeaveTypeInvalid    = errors.New("请假类型不合法")
	ErrLeaveDateBad        = errors.New("日期格式错误，应为 YYYY-MM-DD")
	ErrLeaveDateOrder      = errors.New("结束日期不能早于开始日期")
	ErrLeaveReasonRequired = errors.New("请假事由不能为空")
	ErrLeaveNotOwner       = errors.New("只能操作自己的请假申请")
	ErrLeaveNotApprover    = errors.New("无权限审批该请假申请")
	ErrLeaveNotPending     = repository.ErrLeaveNotPending
)

// 假期类型默认年度额度（单位：天）
var leaveQuota = map[string]float64{
	"annual":    10,
	"sick":      5,
	"personal":  3,
	"maternity": 98,
	"marriage":  3,
	"other":     5,
}

// 假期类型中文名（用于错误提示与列表展示）
var leaveTypeNames = map[string]string{
	"annual":    "年假",
	"sick":      "病假",
	"personal":  "事假",
	"maternity": "产假",
	"marriage":  "婚假",
	"other":     "其他",
}

// leaveTypeOrder 余额列表固定展示顺序
var leaveTypeOrder = []string{"annual", "sick", "personal", "maternity", "marriage", "other"}

// BalanceInsufficientError 假期余额不足错误（含具体剩余与本次所需天数）
type BalanceInsufficientError struct {
	TypeName  string
	Remaining float64
	Need      float64
}

func (e *BalanceInsufficientError) Error() string {
	return fmt.Sprintf("%s余额不足：剩余 %v 天，本次需 %v 天",
		e.TypeName, formatDays(e.Remaining), formatDays(e.Need))
}

// SubmitLeaveRequest 提交请假申请请求
type SubmitLeaveRequest struct {
	LeaveType string `json:"leave_type"`
	StartDate string `json:"start_date"` // YYYY-MM-DD
	EndDate   string `json:"end_date"`   // YYYY-MM-DD
	Reason    string `json:"reason"`
}

// BalanceItem 假期余额项
type BalanceItem struct {
	LeaveType string  `json:"leave_type"`
	Total     float64 `json:"total"`
	Used      float64 `json:"used"`
	Remaining float64 `json:"remaining"`
}

// MineResult 我的请假列表（含余额信息）
type MineResult struct {
	List     []repository.LeaveItem `json:"list"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Balance  []BalanceItem          `json:"balance"`
}

// LeaveService 请假业务服务
type LeaveService struct {
	db        *gorm.DB // 数据范围解析用（ResolveScope）
	leaveRepo *repository.LeaveRepo
	empRepo   *repository.EmployeeRepo
	notif     *NotificationService
}

// NewLeaveService 创建请假服务
func NewLeaveService(db *gorm.DB, leaveRepo *repository.LeaveRepo, empRepo *repository.EmployeeRepo, notif *NotificationService) *LeaveService {
	return &LeaveService{db: db, leaveRepo: leaveRepo, empRepo: empRepo, notif: notif}
}

// Create 提交请假申请：校验日期与余额，自动匹配部门负责人作为审批人
func (s *LeaveService) Create(userID uint, req SubmitLeaveRequest) (*models.LeaveRequest, error) {
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return nil, err
	}
	if _, ok := leaveQuota[req.LeaveType]; !ok {
		return nil, ErrLeaveTypeInvalid
	}
	if req.Reason == "" {
		return nil, ErrLeaveReasonRequired
	}
	start, err := time.ParseInLocation("2006-01-02", req.StartDate, time.Local)
	if err != nil {
		return nil, ErrLeaveDateBad
	}
	end, err := time.ParseInLocation("2006-01-02", req.EndDate, time.Local)
	if err != nil {
		return nil, ErrLeaveDateBad
	}
	if end.Before(start) {
		return nil, ErrLeaveDateOrder
	}
	// 按自然日计算天数（end - start + 1）
	days := float64(int(end.Sub(start).Hours()/24) + 1)

	// 校验假期余额（当年）
	year := time.Now().Year()
	total, used, err := s.leaveBalance(empID, req.LeaveType, year)
	if err != nil {
		return nil, err
	}
	if used+days > total {
		return nil, &BalanceInsufficientError{
			TypeName:  leaveTypeNames[req.LeaveType],
			Remaining: total - used,
			Need:      days,
		}
	}

	leave := &models.LeaveRequest{
		EmployeeID: empID,
		LeaveType:  req.LeaveType,
		StartDate:  start,
		EndDate:    end,
		Days:       days,
		Reason:     req.Reason,
	}

	// 自动匹配审批人：员工所属部门负责人；找不到则直接审批通过（同时扣减余额）
	managerID, err := s.leaveRepo.FindManagerEmployeeID(empID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if managerID != nil && *managerID > 0 {
		leave.Status = "pending"
		leave.ApproverID = managerID
		if err := s.leaveRepo.Create(leave); err != nil {
			return nil, err
		}
		// 通知审批人待审批
		s.notifyApprover(leave)
		return leave, nil
	}
	// 无审批人：直接通过并扣减余额（原子事务）
	leave.Status = "approved"
	now := time.Now()
	leave.ApprovedAt = &now
	if err := s.leaveRepo.CreateAutoApproved(leave, year, leaveQuota[req.LeaveType]); err != nil {
		return nil, err
	}
	return leave, nil
}

// List 请假分页列表，按角色范围过滤（admin/hr 全部；manager 本部门+待自己审批；employee 仅自己）
func (s *LeaveService) List(roleCode string, userID uint, status string, employeeID uint, page, pageSize int) ([]repository.LeaveItem, int64, error) {
	scope, err := s.buildScope(roleCode, userID)
	if err != nil {
		return nil, 0, err
	}
	return s.leaveRepo.List(scope, status, employeeID, page, pageSize)
}

// Mine 当前用户自己的请假列表（含余额信息）
func (s *LeaveService) Mine(userID uint, page, pageSize int) (*MineResult, error) {
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return nil, err
	}
	items, total, err := s.leaveRepo.ListByEmployee(empID, "", page, pageSize)
	if err != nil {
		return nil, err
	}
	balance, err := s.Balance(userID)
	if err != nil {
		return nil, err
	}
	return &MineResult{List: items, Total: total, Page: page, PageSize: pageSize, Balance: balance}, nil
}

// Balance 当前用户各类型假期余额
func (s *LeaveService) Balance(userID uint) ([]BalanceItem, error) {
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return nil, err
	}
	year := time.Now().Year()
	list, err := s.leaveRepo.GetBalanceList(empID, year)
	if err != nil {
		return nil, err
	}
	balMap := make(map[string]models.LeaveBalance, len(list))
	for _, b := range list {
		balMap[b.LeaveType] = b
	}
	items := make([]BalanceItem, 0, len(leaveTypeOrder))
	for _, lt := range leaveTypeOrder {
		total := leaveQuota[lt]
		used := 0.0
		if b, ok := balMap[lt]; ok {
			total = b.TotalDays
			used = b.UsedDays
		}
		items = append(items, BalanceItem{
			LeaveType: lt,
			Total:     total,
			Used:      used,
			Remaining: total - used,
		})
	}
	return items, nil
}

// Approve 审批通过（manager/admin/hr）：更新状态 + 扣减余额（事务原子）
func (s *LeaveService) Approve(roleCode string, userID uint, id uint) error {
	leave, err := s.leaveRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeaveNotFound
		}
		return err
	}
	if err := s.checkApprover(roleCode, userID, leave); err != nil {
		return err
	}
	if err := s.leaveRepo.ApproveAndDeduct(id, time.Now().Year(), leaveQuota[leave.LeaveType]); err != nil {
		return err
	}
	// 通知申请人审批通过
	s.notifyLeaveResult(leave, true, "")
	return nil
}

// Reject 驳回申请
func (s *LeaveService) Reject(roleCode string, userID uint, id uint, reason string) error {
	leave, err := s.leaveRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeaveNotFound
		}
		return err
	}
	if err := s.checkApprover(roleCode, userID, leave); err != nil {
		return err
	}
	if err := s.leaveRepo.Reject(id, reason); err != nil {
		return err
	}
	// 通知申请人审批驳回
	s.notifyLeaveResult(leave, false, reason)
	return nil
}

// Cancel 申请人取消自己的 pending 申请
func (s *LeaveService) Cancel(userID uint, id uint) error {
	leave, err := s.leaveRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeaveNotFound
		}
		return err
	}
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return err
	}
	if leave.EmployeeID != empID {
		return ErrLeaveNotOwner
	}
	return s.leaveRepo.Cancel(id)
}

// notifyApprover 提交申请后通知审批人（审批人无账号则静默跳过）
func (s *LeaveService) notifyApprover(leave *models.LeaveRequest) {
	if leave.ApproverID == nil || *leave.ApproverID == 0 {
		return
	}
	emp, err := s.empRepo.FindByID(leave.EmployeeID)
	if err != nil {
		return
	}
	content := fmt.Sprintf("%s 提交了 %s %s 天（%s~%s）",
		emp.Name, leaveTypeNames[leave.LeaveType], formatDays(leave.Days),
		leave.StartDate.Format("2006-01-02"), leave.EndDate.Format("2006-01-02"))
	s.notif.SendToEmployee(*leave.ApproverID, NotifTypeLeaveApproval, "新的请假申请待审批", content, "/leaves")
}

// notifyLeaveResult 审批结果通知申请人
func (s *LeaveService) notifyLeaveResult(leave *models.LeaveRequest, approved bool, rejectReason string) {
	var title, content string
	if approved {
		title = "请假审批通过"
		content = fmt.Sprintf("%s %s 天（%s~%s）已通过",
			leaveTypeNames[leave.LeaveType], formatDays(leave.Days),
			leave.StartDate.Format("2006-01-02"), leave.EndDate.Format("2006-01-02"))
	} else {
		title = "请假审批驳回"
		content = fmt.Sprintf("原因：%s", rejectReason)
	}
	s.notif.SendToEmployee(leave.EmployeeID, NotifTypeLeaveResult, title, content, "/leaves")
}

// checkApprover 审批权限校验：admin/hr 可审批；manager 必须是该申请审批人
func (s *LeaveService) checkApprover(roleCode string, userID uint, leave *models.LeaveRequest) error {
	if roleCode == "admin" || roleCode == "hr" {
		return nil
	}
	if roleCode != "manager" {
		return ErrLeaveNotApprover
	}
	empID, err := s.requireEmployee(userID)
	if err != nil {
		return err
	}
	if leave.ApproverID == nil || *leave.ApproverID != empID {
		return ErrLeaveNotApprover
	}
	return nil
}

// buildScope 按角色构建列表访问范围
func (s *LeaveService) buildScope(roleCode string, userID uint) (repository.LeaveScope, error) {
	switch roleCode {
	case "admin":
		return repository.LeaveScope{}, nil
	case "hr":
		// HR 数据范围：绑定负责部门则仅可见该部门及子部门，未绑定则全量
		scopeDeptIDs, err := ResolveScope(s.db, userID, roleCode)
		if err != nil {
			return repository.LeaveScope{}, err
		}
		return repository.LeaveScope{DeptIDs: scopeDeptIDs}, nil
	case "manager":
		empID, err := s.requireEmployee(userID)
		if err != nil {
			return repository.LeaveScope{Empty: true}, nil
		}
		emp, err := s.empRepo.FindByID(empID)
		if err != nil {
			return repository.LeaveScope{Empty: true}, nil
		}
		return repository.LeaveScope{DepartmentID: emp.DepartmentID, ApproverID: empID}, nil
	default: // employee
		empID, err := s.requireEmployee(userID)
		if err != nil {
			return repository.LeaveScope{Empty: true}, nil
		}
		return repository.LeaveScope{EmployeeID: empID}, nil
	}
}

// requireEmployee 根据登录用户解析关联的员工 ID
func (s *LeaveService) requireEmployee(userID uint) (uint, error) {
	empID, err := s.empRepo.FindEmployeeIDByUserID(userID)
	if err != nil {
		return 0, err
	}
	if empID == nil {
		return 0, ErrLeaveEmpNoUser
	}
	return *empID, nil
}

// leaveBalance 查询某员工某类型当年余额：total 优先取余额表配置，否则用默认额度
func (s *LeaveService) leaveBalance(employeeID uint, leaveType string, year int) (total, used float64, err error) {
	total = leaveQuota[leaveType]
	bal, err := s.leaveRepo.GetBalance(employeeID, leaveType, year)
	if err != nil {
		return 0, 0, err
	}
	if bal != nil {
		total = bal.TotalDays
		used = bal.UsedDays
	}
	return total, used, nil
}

// formatDays 天数格式化：整数不带小数，非整数保留一位小数
func formatDays(d float64) string {
	if d == float64(int(d)) {
		return fmt.Sprintf("%d", int(d))
	}
	return fmt.Sprintf("%.1f", d)
}
