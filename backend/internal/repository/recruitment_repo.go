package repository

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// JobItem 招聘职位列表项（附带部门、岗位名称）
type JobItem struct {
	models.JobOpening
	DepartmentName string `json:"department_name"`
	PositionName   string `json:"position_name"`
}

// CandidateItem 候选人列表项（附带应聘职位名称）
type CandidateItem struct {
	models.Candidate
	JobTitle string `json:"job_title"`
}

// RecruitmentRepo 招聘数据仓库
type RecruitmentRepo struct {
	db *gorm.DB
}

// NewRecruitmentRepo 创建招聘仓库
func NewRecruitmentRepo(db *gorm.DB) *RecruitmentRepo {
	return &RecruitmentRepo{db: db}
}

// JobList 招聘职位分页列表，支持 status / keyword 过滤，deptIDs 为数据范围部门 ID（nil 不过滤）
func (r *RecruitmentRepo) JobList(status, keyword string, deptIDs []uint, page, pageSize int) ([]JobItem, int64, error) {
	q := r.db.Model(&models.JobOpening{}).
		Joins("LEFT JOIN departments d ON d.id = job_openings.department_id").
		Joins("LEFT JOIN positions p ON p.id = job_openings.position_id")
	if status != "" {
		q = q.Where("job_openings.status = ?", status)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		q = q.Where("job_openings.title LIKE ? OR d.name LIKE ?", kw, kw)
	}
	if len(deptIDs) > 0 {
		q = q.Where("job_openings.department_id IN ?", deptIDs)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]JobItem, 0)
	err := q.Select("job_openings.*, d.name AS department_name, p.name AS position_name").
		Order("job_openings.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// JobFindByID 根据 ID 查询招聘职位
func (r *RecruitmentRepo) JobFindByID(id uint) (*models.JobOpening, error) {
	var job models.JobOpening
	if err := r.db.First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// JobCreate 创建招聘职位
func (r *RecruitmentRepo) JobCreate(job *models.JobOpening) error {
	return r.db.Create(job).Error
}

// JobUpdate 按字段更新招聘职位
func (r *RecruitmentRepo) JobUpdate(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.JobOpening{}).Where("id = ?", id).Updates(fields).Error
}

// JobDelete 软删除招聘职位
func (r *RecruitmentRepo) JobDelete(id uint) error {
	return r.db.Delete(&models.JobOpening{}, id).Error
}

// CandidateList 候选人分页列表，支持 job_id / status 过滤
func (r *RecruitmentRepo) CandidateList(jobID uint, status string, page, pageSize int) ([]CandidateItem, int64, error) {
	q := r.db.Model(&models.Candidate{}).
		Joins("LEFT JOIN job_openings j ON j.id = candidates.job_opening_id")
	if jobID > 0 {
		q = q.Where("candidates.job_opening_id = ?", jobID)
	}
	if status != "" {
		q = q.Where("candidates.status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]CandidateItem, 0)
	err := q.Select("candidates.*, j.title AS job_title").
		Order("candidates.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CandidateFindByID 根据 ID 查询候选人
func (r *RecruitmentRepo) CandidateFindByID(id uint) (*models.Candidate, error) {
	var c models.Candidate
	if err := r.db.First(&c, id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// CandidateCreate 添加候选人
func (r *RecruitmentRepo) CandidateCreate(c *models.Candidate) error {
	return r.db.Create(c).Error
}

// CandidateUpdate 按字段更新候选人
func (r *RecruitmentRepo) CandidateUpdate(id uint, fields map[string]interface{}) error {
	return r.db.Model(&models.Candidate{}).Where("id = ?", id).Updates(fields).Error
}

// CandidateDelete 软删除候选人
func (r *RecruitmentRepo) CandidateDelete(id uint) error {
	return r.db.Delete(&models.Candidate{}, id).Error
}
