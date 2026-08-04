package service

import (
	"errors"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 招聘业务错误
var (
	ErrJobTitleEmpty          = errors.New("招聘职位名称不能为空")
	ErrJobNotFound            = errors.New("招聘职位不存在")
	ErrJobDeptMissing         = errors.New("需求部门不存在")
	ErrJobPosMissing          = errors.New("对应岗位不存在")
	ErrJobStatusInvalid       = errors.New("招聘职位状态不合法")
	ErrCandidateNameEmpty     = errors.New("候选人姓名不能为空")
	ErrCandidateNotFound      = errors.New("候选人不存在")
	ErrCandidateJobMissing    = errors.New("应聘职位不存在")
	ErrCandidateStatusInvalid = errors.New("候选人状态不合法")
)

// 招聘职位状态集合
var validJobStatus = map[string]bool{"open": true, "closed": true}

// 候选人状态集合
var validCandidateStatus = map[string]bool{
	"applied": true, "interview": true, "offer": true, "hired": true, "rejected": true,
}

// CreateJobRequest 创建招聘职位请求
type CreateJobRequest struct {
	Title        string `json:"title"`
	DepartmentID uint   `json:"department_id"`
	PositionID   uint   `json:"position_id"`
	Headcount    int    `json:"headcount"`
	Requirement  string `json:"requirement"`
	Status       string `json:"status"`
}

// UpdateJobRequest 更新招聘职位请求（指针字段区分「未传」与「置空」）
type UpdateJobRequest struct {
	Title        *string `json:"title"`
	DepartmentID *uint   `json:"department_id"`
	PositionID   *uint   `json:"position_id"`
	Headcount    *int    `json:"headcount"`
	Requirement  *string `json:"requirement"`
	Status       *string `json:"status"`
}

// CreateCandidateRequest 添加候选人请求
type CreateCandidateRequest struct {
	JobOpeningID uint   `json:"job_opening_id"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Remark       string `json:"remark"`
}

// UpdateCandidateRequest 更新候选人请求（指针字段区分「未传」与「置空」）
type UpdateCandidateRequest struct {
	JobOpeningID *uint   `json:"job_opening_id"`
	Name         *string `json:"name"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	Status       *string `json:"status"`
	Remark       *string `json:"remark"`
}

// RecruitmentService 招聘业务服务
type RecruitmentService struct {
	recruitRepo *repository.RecruitmentRepo
	deptRepo    *repository.DepartmentRepo
	posRepo     *repository.PositionRepo
}

// NewRecruitmentService 创建招聘服务
func NewRecruitmentService(recruitRepo *repository.RecruitmentRepo, deptRepo *repository.DepartmentRepo, posRepo *repository.PositionRepo) *RecruitmentService {
	return &RecruitmentService{recruitRepo: recruitRepo, deptRepo: deptRepo, posRepo: posRepo}
}

// JobList 招聘职位分页列表
func (s *RecruitmentService) JobList(status, keyword string, page, pageSize int) ([]repository.JobItem, int64, error) {
	return s.recruitRepo.JobList(status, keyword, page, pageSize)
}

// JobCreate 创建招聘职位：校验名称非空、部门与岗位存在、状态合法
func (s *RecruitmentService) JobCreate(req CreateJobRequest) (*models.JobOpening, error) {
	if req.Title == "" {
		return nil, ErrJobTitleEmpty
	}
	ok, err := s.deptRepo.Exists(req.DepartmentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrJobDeptMissing
	}
	ok, err = s.posRepo.Exists(req.PositionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrJobPosMissing
	}
	status := req.Status
	if status == "" {
		status = "open"
	}
	if !validJobStatus[status] {
		return nil, ErrJobStatusInvalid
	}
	job := &models.JobOpening{
		Title:        req.Title,
		DepartmentID: req.DepartmentID,
		PositionID:   req.PositionID,
		Headcount:    req.Headcount,
		Requirement:  req.Requirement,
		Status:       status,
	}
	if err := s.recruitRepo.JobCreate(job); err != nil {
		return nil, err
	}
	return job, nil
}

// JobUpdate 更新招聘职位：校验存在性与关联字段，返回更新后的记录
func (s *RecruitmentService) JobUpdate(id uint, req UpdateJobRequest) (*models.JobOpening, error) {
	job, err := s.recruitRepo.JobFindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	fields := map[string]interface{}{}
	if req.Title != nil {
		if *req.Title == "" {
			return nil, ErrJobTitleEmpty
		}
		fields["title"] = *req.Title
	}
	if req.DepartmentID != nil {
		ok, err := s.deptRepo.Exists(*req.DepartmentID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrJobDeptMissing
		}
		fields["department_id"] = *req.DepartmentID
	}
	if req.PositionID != nil {
		ok, err := s.posRepo.Exists(*req.PositionID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrJobPosMissing
		}
		fields["position_id"] = *req.PositionID
	}
	if req.Headcount != nil {
		fields["headcount"] = *req.Headcount
	}
	if req.Requirement != nil {
		fields["requirement"] = *req.Requirement
	}
	if req.Status != nil {
		if !validJobStatus[*req.Status] {
			return nil, ErrJobStatusInvalid
		}
		fields["status"] = *req.Status
	}
	if len(fields) == 0 {
		return job, nil
	}
	if err := s.recruitRepo.JobUpdate(id, fields); err != nil {
		return nil, err
	}
	return s.recruitRepo.JobFindByID(id)
}

// JobDelete 删除招聘职位（软删除）
func (s *RecruitmentService) JobDelete(id uint) error {
	if _, err := s.recruitRepo.JobFindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrJobNotFound
		}
		return err
	}
	return s.recruitRepo.JobDelete(id)
}

// CandidateList 候选人分页列表
func (s *RecruitmentService) CandidateList(jobID uint, status string, page, pageSize int) ([]repository.CandidateItem, int64, error) {
	return s.recruitRepo.CandidateList(jobID, status, page, pageSize)
}

// CandidateCreate 添加候选人：校验应聘职位存在、姓名非空，状态默认 applied
func (s *RecruitmentService) CandidateCreate(req CreateCandidateRequest) (*models.Candidate, error) {
	if req.Name == "" {
		return nil, ErrCandidateNameEmpty
	}
	if _, err := s.recruitRepo.JobFindByID(req.JobOpeningID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCandidateJobMissing
		}
		return nil, err
	}
	candidate := &models.Candidate{
		JobOpeningID: req.JobOpeningID,
		Name:         req.Name,
		Phone:        req.Phone,
		Email:        req.Email,
		Status:       "applied",
		Remark:       req.Remark,
	}
	if err := s.recruitRepo.CandidateCreate(candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}

// CandidateUpdate 更新候选人：校验存在性与状态合法，返回更新后的记录
func (s *RecruitmentService) CandidateUpdate(id uint, req UpdateCandidateRequest) (*models.Candidate, error) {
	candidate, err := s.recruitRepo.CandidateFindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCandidateNotFound
		}
		return nil, err
	}
	fields := map[string]interface{}{}
	if req.JobOpeningID != nil {
		if _, err := s.recruitRepo.JobFindByID(*req.JobOpeningID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrCandidateJobMissing
			}
			return nil, err
		}
		fields["job_opening_id"] = *req.JobOpeningID
	}
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ErrCandidateNameEmpty
		}
		fields["name"] = *req.Name
	}
	if req.Phone != nil {
		fields["phone"] = *req.Phone
	}
	if req.Email != nil {
		fields["email"] = *req.Email
	}
	if req.Status != nil {
		if !validCandidateStatus[*req.Status] {
			return nil, ErrCandidateStatusInvalid
		}
		fields["status"] = *req.Status
	}
	if req.Remark != nil {
		fields["remark"] = *req.Remark
	}
	if len(fields) == 0 {
		return candidate, nil
	}
	if err := s.recruitRepo.CandidateUpdate(id, fields); err != nil {
		return nil, err
	}
	return s.recruitRepo.CandidateFindByID(id)
}

// CandidateDelete 删除候选人（软删除）
func (s *RecruitmentService) CandidateDelete(id uint) error {
	if _, err := s.recruitRepo.CandidateFindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCandidateNotFound
		}
		return err
	}
	return s.recruitRepo.CandidateDelete(id)
}
