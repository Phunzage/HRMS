package service

import (
	"errors"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 岗位业务错误
var (
	ErrPosNotFound     = errors.New("岗位不存在")
	ErrPosNameEmpty    = errors.New("岗位名称不能为空")
	ErrPosCodeEmpty    = errors.New("岗位编码不能为空")
	ErrPosDeptNotFound = errors.New("所属部门不存在")
	ErrPosCodeExists   = errors.New("岗位编码已存在")
	ErrPosInUse        = errors.New("该岗位下存在员工，无法删除")
)

// CreatePositionRequest 创建岗位请求
type CreatePositionRequest struct {
	Name         string `json:"name"`
	Code         string `json:"code"`
	DepartmentID uint   `json:"department_id"`
	Level        int    `json:"level"`
	Remark       string `json:"remark"`
}

// UpdatePositionRequest 更新岗位请求（指针字段区分「未传」与「置空」）
type UpdatePositionRequest struct {
	Name         *string `json:"name"`
	Code         *string `json:"code"`
	DepartmentID *uint   `json:"department_id"`
	Level        *int    `json:"level"`
	Remark       *string `json:"remark"`
}

// PositionService 岗位业务服务
type PositionService struct {
	repo *repository.PositionRepo
}

// NewPositionService 创建岗位服务
func NewPositionService(repo *repository.PositionRepo) *PositionService {
	return &PositionService{repo: repo}
}

// List 岗位分页列表，支持 keyword / department_id 过滤
func (s *PositionService) List(keyword string, departmentID uint, page, pageSize int) ([]repository.PositionItem, int64, error) {
	return s.repo.List(keyword, departmentID, page, pageSize)
}

// GetByID 查询岗位，供处理器拼接审计描述等场景使用
func (s *PositionService) GetByID(id uint) (*models.Position, error) {
	pos, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPosNotFound
		}
		return nil, err
	}
	return pos, nil
}

// Create 创建岗位：校验名称、编码、部门存在、编码唯一
func (s *PositionService) Create(req CreatePositionRequest) (*models.Position, error) {
	if req.Name == "" {
		return nil, ErrPosNameEmpty
	}
	if req.Code == "" {
		return nil, ErrPosCodeEmpty
	}
	ok, err := s.repo.DeptExists(req.DepartmentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrPosDeptNotFound
	}
	exists, err := s.repo.CodeExists(req.Code, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrPosCodeExists
	}
	pos := &models.Position{
		Name:         req.Name,
		Code:         req.Code,
		DepartmentID: req.DepartmentID,
		Level:        req.Level,
		Remark:       req.Remark,
	}
	if err := s.repo.Create(pos); err != nil {
		return nil, err
	}
	return pos, nil
}

// Update 更新岗位：校验存在性、编码唯一、部门存在
func (s *PositionService) Update(id uint, req UpdatePositionRequest) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPosNotFound
		}
		return err
	}
	fields := map[string]interface{}{}
	if req.Name != nil && *req.Name != "" {
		fields["name"] = *req.Name
	}
	if req.Code != nil && *req.Code != "" {
		exists, err := s.repo.CodeExists(*req.Code, id)
		if err != nil {
			return err
		}
		if exists {
			return ErrPosCodeExists
		}
		fields["code"] = *req.Code
	}
	if req.DepartmentID != nil {
		ok, err := s.repo.DeptExists(*req.DepartmentID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrPosDeptNotFound
		}
		fields["department_id"] = *req.DepartmentID
	}
	if req.Level != nil {
		fields["level"] = *req.Level
	}
	if req.Remark != nil {
		fields["remark"] = *req.Remark
	}
	if len(fields) == 0 {
		return nil
	}
	return s.repo.Update(id, fields)
}

// Delete 删除岗位：岗位被员工占用时拒绝
func (s *PositionService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPosNotFound
		}
		return err
	}
	count, err := s.repo.CountByEmployee(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrPosInUse
	}
	return s.repo.Delete(id)
}
