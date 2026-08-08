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

// Recruitment 招聘处理器
type Recruitment struct {
	service *service.RecruitmentService
	db      *gorm.DB
}

// NewRecruitment 创建招聘处理器
func NewRecruitment(svc *service.RecruitmentService, db *gorm.DB) *Recruitment {
	return &Recruitment{service: svc, db: db}
}

// JobList 招聘职位分页列表
func (h *Recruitment) JobList(c *gin.Context) {
	status := c.Query("status")
	keyword := c.Query("keyword")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.JobList(status, keyword, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取招聘职位列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// JobCreate 创建招聘职位
func (h *Recruitment) JobCreate(c *gin.Context) {
	var req service.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	job, err := h.service.JobCreate(req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "recruitment", "create", fmt.Sprintf("job:%d", job.ID), fmt.Sprintf("创建招聘职位 %s", job.Title))
	response.OK(c, job)
}

// JobUpdate 更新招聘职位
func (h *Recruitment) JobUpdate(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req service.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	job, err := h.service.JobUpdate(id, req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "recruitment", "update", fmt.Sprintf("job:%d", job.ID), fmt.Sprintf("更新招聘职位 %s", job.Title))
	response.OK(c, job)
}

// JobDelete 删除招聘职位
func (h *Recruitment) JobDelete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.JobDelete(id); err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "recruitment", "delete", fmt.Sprintf("job:%d", id), fmt.Sprintf("删除招聘职位(ID:%d)", id))
	response.OK(c, nil)
}

// CandidateList 候选人分页列表
func (h *Recruitment) CandidateList(c *gin.Context) {
	jobID := parseUintQuery(c, "job_id")
	status := c.Query("status")
	page, pageSize := response.Pagination(c)
	items, total, err := h.service.CandidateList(jobID, status, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, 500, "获取候选人列表失败")
		return
	}
	response.PageOK(c, items, total, page, pageSize)
}

// CandidateCreate 添加候选人
func (h *Recruitment) CandidateCreate(c *gin.Context) {
	var req service.CreateCandidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	candidate, err := h.service.CandidateCreate(req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "recruitment", "create", fmt.Sprintf("candidate:%d", candidate.ID), fmt.Sprintf("添加候选人 %s", candidate.Name))
	response.OK(c, candidate)
}

// CandidateUpdate 更新候选人（含状态流转）
func (h *Recruitment) CandidateUpdate(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	var req service.UpdateCandidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	candidate, err := h.service.CandidateUpdate(id, req)
	if err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "recruitment", "update", fmt.Sprintf("candidate:%d", candidate.ID), fmt.Sprintf("更新候选人 %s", candidate.Name))
	response.OK(c, candidate)
}

// CandidateDelete 删除候选人
func (h *Recruitment) CandidateDelete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if err := h.service.CandidateDelete(id); err != nil {
		h.handleErr(c, err)
		return
	}
	middleware.Record(h.db, c, "recruitment", "delete", fmt.Sprintf("candidate:%d", id), fmt.Sprintf("删除候选人(ID:%d)", id))
	response.OK(c, nil)
}

// handleErr 招聘模块错误映射
func (h *Recruitment) handleErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrJobTitleEmpty),
		errors.Is(err, service.ErrJobDeptMissing),
		errors.Is(err, service.ErrJobPosMissing),
		errors.Is(err, service.ErrJobStatusInvalid),
		errors.Is(err, service.ErrCandidateNameEmpty),
		errors.Is(err, service.ErrCandidateJobMissing),
		errors.Is(err, service.ErrCandidateStatusInvalid):
		response.Error(c, http.StatusBadRequest, 400, err.Error())
	case errors.Is(err, service.ErrJobNotFound),
		errors.Is(err, service.ErrCandidateNotFound):
		response.Error(c, http.StatusNotFound, 404, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, 500, "操作失败")
	}
}
