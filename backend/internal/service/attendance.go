package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 考勤业务错误
var (
	ErrAttNotFound     = errors.New("考勤记录不存在")
	ErrAttEmpNoUser    = errors.New("当前账号未关联员工档案")
	ErrAttEmpNotFound  = errors.New("员工不存在")
	ErrAttDateBad      = errors.New("日期格式错误，应为 YYYY-MM-DD")
	ErrAttTimeBad      = errors.New("时间格式错误，应为 YYYY-MM-DD HH:MM:SS")
	ErrAttTodayChecked = errors.New("今日已打卡")
	ErrAttNoCheckIn    = errors.New("今日无上班打卡记录")
	ErrAttAlreadyOut   = errors.New("今日已打卡下班")
	ErrAttHROnly       = errors.New("仅人事或管理员可手工补录考勤")
	ErrAttDupRecord    = errors.New("该员工当日已有考勤记录")
	ErrAttStatusBad    = errors.New("考勤状态不合法")
	ErrAttTimeOrderBad = errors.New("下班时间必须晚于上班时间")
)

// 考勤状态集合
var validAttendanceStatus = map[string]bool{
	"normal": true, "late": true, "early_leave": true, "absent": true,
}

// ManualAttendanceRequest HR 手工补录请求
type ManualAttendanceRequest struct {
	EmployeeID uint    `json:"employee_id"`
	WorkDate   string  `json:"work_date"`
	CheckIn    *string `json:"check_in"`  // YYYY-MM-DD HH:MM:SS
	CheckOut   *string `json:"check_out"` // YYYY-MM-DD HH:MM:SS
	Status     string  `json:"status"`    // normal/late/early_leave/absent
	Remark     string  `json:"remark"`
}

// UpdateAttendanceRequest 修改考勤记录请求（指针字段区分「未传」与「置空」）
type UpdateAttendanceRequest struct {
	EmployeeID *uint   `json:"employee_id"`
	WorkDate   *string `json:"work_date"`
	CheckIn    *string `json:"check_in"`  // 空串表示清空
	CheckOut   *string `json:"check_out"` // 空串表示清空
	Status     *string `json:"status"`
	Remark     *string `json:"remark"`
}

// AttendanceService 考勤业务服务
type AttendanceService struct {
	attRepo *repository.AttendanceRepo
	empRepo *repository.EmployeeRepo
}

// NewAttendanceService 创建考勤服务
func NewAttendanceService(attRepo *repository.AttendanceRepo, empRepo *repository.EmployeeRepo) *AttendanceService {
	return &AttendanceService{attRepo: attRepo, empRepo: empRepo}
}

// List 考勤分页列表，支持 employee_id / work_date / month / status 过滤
func (s *AttendanceService) List(employeeID uint, workDate, month, status string, page, pageSize int) ([]repository.AttendanceItem, int64, error) {
	if workDate != "" {
		if _, err := time.Parse("2006-01-02", workDate); err != nil {
			return nil, 0, ErrAttDateBad
		}
	}
	return s.attRepo.List(employeeID, workDate, month, status, page, pageSize)
}

// CheckIn 打卡上班：按当前登录用户关联的员工，今日已打卡则报错
func (s *AttendanceService) CheckIn(userID uint) (*models.Attendance, error) {
	empID, err := s.requireEmployeeID(userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	today := now.Format("2006-01-02")

	// 查找今日记录，不存在则新建
	att, err := s.attRepo.FindByEmployeeAndDate(empID, today)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		att = &models.Attendance{
			EmployeeID: empID,
			WorkDate:   dateOnly(now),
			CheckIn:    &now,
			Status:     computeAttendanceStatus(&now, nil),
		}
		if err := s.attRepo.Create(att); err != nil {
			return nil, err
		}
		return att, nil
	}
	if err != nil {
		return nil, err
	}
	// 已有记录：今日已打卡
	if att.CheckIn != nil {
		return nil, ErrAttTodayChecked
	}
	// 记录存在但无 CheckIn（如手工补录的 absent 记录），补上打卡时间
	fields := map[string]interface{}{"check_in": now, "status": computeAttendanceStatus(&now, att.CheckOut)}
	if err := s.attRepo.Update(att.ID, fields); err != nil {
		return nil, err
	}
	att.CheckIn = &now
	att.Status = fields["status"].(string)
	return att, nil
}

// CheckOut 打卡下班：今日无 CheckIn 报错，已有 CheckOut 报错
func (s *AttendanceService) CheckOut(userID uint) (*models.Attendance, error) {
	empID, err := s.requireEmployeeID(userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	today := now.Format("2006-01-02")

	att, err := s.attRepo.FindByEmployeeAndDate(empID, today)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAttNoCheckIn
		}
		return nil, err
	}
	if att.CheckIn == nil {
		return nil, ErrAttNoCheckIn
	}
	if att.CheckOut != nil {
		return nil, ErrAttAlreadyOut
	}
	status := computeAttendanceStatus(att.CheckIn, &now)
	fields := map[string]interface{}{"check_out": now, "status": status}
	if err := s.attRepo.Update(att.ID, fields); err != nil {
		return nil, err
	}
	att.CheckOut = &now
	att.Status = status
	return att, nil
}

// Manual HR 手工补录考勤（仅人事/管理员）
func (s *AttendanceService) Manual(roleCode string, req ManualAttendanceRequest) (*models.Attendance, error) {
	if roleCode != "admin" && roleCode != "hr" {
		return nil, ErrAttHROnly
	}
	if req.EmployeeID == 0 {
		return nil, ErrAttEmpNotFound
	}
	if _, err := s.empRepo.FindByID(req.EmployeeID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAttEmpNotFound
		}
		return nil, err
	}
	workDate, err := time.Parse("2006-01-02", req.WorkDate)
	if err != nil {
		return nil, ErrAttDateBad
	}
	checkIn, err := parseAttendanceTime(req.CheckIn)
	if err != nil {
		return nil, err
	}
	checkOut, err := parseAttendanceTime(req.CheckOut)
	if err != nil {
		return nil, err
	}
	if checkIn != nil && checkOut != nil && !checkOut.After(*checkIn) {
		return nil, ErrAttTimeOrderBad
	}
	status := req.Status
	if status == "" {
		status = computeAttendanceStatus(checkIn, checkOut)
	}
	if !validAttendanceStatus[status] {
		return nil, ErrAttStatusBad
	}

	// 唯一性：同一员工同一工作日只能有一条记录
	if _, err := s.attRepo.FindByEmployeeAndDate(req.EmployeeID, req.WorkDate); err == nil {
		return nil, ErrAttDupRecord
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	att := &models.Attendance{
		EmployeeID: req.EmployeeID,
		WorkDate:   workDate,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
		Status:     status,
		Remark:     req.Remark,
	}
	if err := s.attRepo.Create(att); err != nil {
		return nil, err
	}
	return att, nil
}

// Update 修改考勤记录
func (s *AttendanceService) Update(id uint, req UpdateAttendanceRequest) error {
	att, err := s.attRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAttNotFound
		}
		return err
	}
	fields := map[string]interface{}{}
	targetEmpID := att.EmployeeID
	targetDate := att.WorkDate.Format("2006-01-02")

	if req.EmployeeID != nil {
		if _, err := s.empRepo.FindByID(*req.EmployeeID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAttEmpNotFound
			}
			return err
		}
		fields["employee_id"] = *req.EmployeeID
		targetEmpID = *req.EmployeeID
	}
	if req.WorkDate != nil {
		t, err := time.Parse("2006-01-02", *req.WorkDate)
		if err != nil {
			return ErrAttDateBad
		}
		fields["work_date"] = t
		targetDate = t.Format("2006-01-02")
	}
	if req.CheckIn != nil {
		checkIn, err := parseAttendanceTime(req.CheckIn)
		if err != nil {
			return err
		}
		fields["check_in"] = checkIn
	}
	if req.CheckOut != nil {
		checkOut, err := parseAttendanceTime(req.CheckOut)
		if err != nil {
			return err
		}
		fields["check_out"] = checkOut
	}
	if req.Status != nil {
		if !validAttendanceStatus[*req.Status] {
			return ErrAttStatusBad
		}
		fields["status"] = *req.Status
	}
	if req.Remark != nil {
		fields["remark"] = *req.Remark
	}

	// 唯一性：若修改了员工或日期，检查目标组合是否已存在其他记录
	if req.EmployeeID != nil || req.WorkDate != nil {
		if existing, err := s.attRepo.FindByEmployeeAndDate(targetEmpID, targetDate); err == nil {
			if existing.ID != id {
				return ErrAttDupRecord
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	// 未显式传 status 时，若打卡时间发生变化则自动重算
	if req.Status == nil {
		if _, hasIn := fields["check_in"]; hasIn {
			var checkIn *time.Time
			if v, ok := fields["check_in"].(*time.Time); ok {
				checkIn = v
			}
			var checkOut *time.Time
			if v, ok := fields["check_out"].(*time.Time); ok {
				checkOut = v
			} else if att.CheckOut != nil && req.CheckOut == nil {
				checkOut = att.CheckOut
			}
			fields["status"] = computeAttendanceStatus(checkIn, checkOut)
		} else if _, hasOut := fields["check_out"]; hasOut {
			var checkOut *time.Time
			if v, ok := fields["check_out"].(*time.Time); ok {
				checkOut = v
			}
			fields["status"] = computeAttendanceStatus(att.CheckIn, checkOut)
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return s.attRepo.Update(id, fields)
}

// Delete 删除考勤记录（软删除）
func (s *AttendanceService) Delete(id uint) error {
	if _, err := s.attRepo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAttNotFound
		}
		return err
	}
	return s.attRepo.Delete(id)
}

// requireEmployeeID 根据登录用户解析关联的员工 ID
func (s *AttendanceService) requireEmployeeID(userID uint) (uint, error) {
	empID, err := s.empRepo.FindEmployeeIDByUserID(userID)
	if err != nil {
		return 0, err
	}
	if empID == nil {
		return 0, ErrAttEmpNoUser
	}
	return *empID, nil
}

// dateOnly 取日期部分（当天零点）
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// computeAttendanceStatus 按打卡时间计算考勤状态：>9:00 late；下班 <18:00 early_leave；无打卡 absent
// 优先级：late > early_leave > normal
func computeAttendanceStatus(checkIn, checkOut *time.Time) string {
	if checkIn == nil {
		return "absent"
	}
	nine := time.Date(checkIn.Year(), checkIn.Month(), checkIn.Day(), 9, 0, 0, 0, checkIn.Location())
	if checkIn.After(nine) {
		return "late"
	}
	if checkOut != nil {
		eighteen := time.Date(checkOut.Year(), checkOut.Month(), checkOut.Day(), 18, 0, 0, 0, checkOut.Location())
		if checkOut.Before(eighteen) {
			return "early_leave"
		}
	}
	return "normal"
}

// parseAttendanceTime 解析打卡时间字符串（支持精确到秒/分）；空串表示清空（返回 nil）
func parseAttendanceTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, *s, time.Local); err == nil {
			return &t, nil
		}
	}
	return nil, ErrAttTimeBad
}
