package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"hrms/backend/internal/middleware"
	"hrms/backend/internal/response"
	"hrms/backend/internal/service"
)

// Attendance 考勤处理器
type Attendance struct {
	service *service.AttendanceService
	db      *gorm.DB
}

// NewAttendance 创建考勤处理器
func NewAttendance(svc *service.AttendanceService, db *gorm.DB) *Attendance {
	return &Attendance{service: svc, db: db}
}

// List 考勤分页列表
func (h *Attendance) List(c *gin.Context) {
	employeeID := parseUintQuery(c, "employee_id")
	workDate := c.Query("work_date")
	month := c.Query("month")
	status := c.Query("status")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.List(employeeID, workDate, month, status, page, pageSize)
	if err != nil {
		if errors.Is(err, service.ErrAttDateBad) {
			response.Error(c, http.StatusBadRequest, 400, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, 500, "获取考勤列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// CheckIn 打卡上班（员工自助）
func (h *Attendance) CheckIn(c *gin.Context) {
	att, err := h.service.CheckIn(middleware.GetUserID(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAttEmpNoUser):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		case errors.Is(err, service.ErrAttTodayChecked):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "打卡失败")
		}
		return
	}
	response.OK(c, att)
}

// CheckOut 打卡下班（员工自助）
func (h *Attendance) CheckOut(c *gin.Context) {
	att, err := h.service.CheckOut(middleware.GetUserID(c))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAttEmpNoUser):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		case errors.Is(err, service.ErrAttNoCheckIn):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		case errors.Is(err, service.ErrAttAlreadyOut):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "打卡失败")
		}
		return
	}
	response.OK(c, att)
}

// Manual HR 手工补录
func (h *Attendance) Manual(c *gin.Context) {
	var req service.ManualAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	att, err := h.service.Manual(middleware.GetRoleCode(c), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAttHROnly):
			response.Error(c, http.StatusForbidden, 403, err.Error())
		case errors.Is(err, service.ErrAttEmpNotFound),
			errors.Is(err, service.ErrAttDateBad),
			errors.Is(err, service.ErrAttTimeBad),
			errors.Is(err, service.ErrAttTimeOrderBad),
			errors.Is(err, service.ErrAttStatusBad),
			errors.Is(err, service.ErrAttDupRecord):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "补录考勤失败")
		}
		return
	}
	middleware.Record(h.db, c, "attendance", "create", fmt.Sprintf("attendance:%d", att.ID), fmt.Sprintf("手工补录考勤 员工ID:%d 日期:%s", att.EmployeeID, att.WorkDate.Format("2006-01-02")))
	response.OK(c, att)
}

// Update 修改考勤记录
func (h *Attendance) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req service.UpdateAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Update(id, req); err != nil {
		switch {
		case errors.Is(err, service.ErrAttNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		case errors.Is(err, service.ErrAttEmpNotFound),
			errors.Is(err, service.ErrAttDateBad),
			errors.Is(err, service.ErrAttTimeBad),
			errors.Is(err, service.ErrAttStatusBad),
			errors.Is(err, service.ErrAttDupRecord):
			response.Error(c, http.StatusBadRequest, 400, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "更新考勤记录失败")
		}
		return
	}
	middleware.Record(h.db, c, "attendance", "update", fmt.Sprintf("attendance:%d", id), fmt.Sprintf("修改考勤记录(ID:%d)", id))
	response.OK(c, nil)
}

// Delete 删除考勤记录
func (h *Attendance) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.Delete(id); err != nil {
		switch {
		case errors.Is(err, service.ErrAttNotFound):
			response.Error(c, http.StatusNotFound, 404, err.Error())
		default:
			response.Error(c, http.StatusInternalServerError, 500, "删除考勤记录失败")
		}
		return
	}
	middleware.Record(h.db, c, "attendance", "delete", fmt.Sprintf("attendance:%d", id), fmt.Sprintf("删除考勤记录(ID:%d)", id))
	response.OK(c, nil)
}
