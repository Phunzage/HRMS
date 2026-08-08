package service

import (
	"errors"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// 部门业务错误
var (
	ErrDeptNotFound      = errors.New("部门不存在")
	ErrDeptNameEmpty     = errors.New("部门名称不能为空")
	ErrDeptParentMissing = errors.New("上级部门不存在")
	ErrDeptCycle         = errors.New("上级部门不能是自己或其下级部门")
	ErrDeptHasChildren   = errors.New("该部门存在子部门，无法删除")
	ErrDeptHasEmployees  = errors.New("该部门下存在员工，无法删除")
)

// CreateDepartmentRequest 创建部门请求
type CreateDepartmentRequest struct {
	Name      string `json:"name"`
	ParentID  uint   `json:"parent_id"`
	ManagerID *uint  `json:"manager_id"`
	SortOrder int    `json:"sort_order"`
	Remark    string `json:"remark"`
}

// UpdateDepartmentRequest 更新部门请求（指针字段区分「未传」与「置空」）
type UpdateDepartmentRequest struct {
	Name      *string `json:"name"`
	ParentID  *uint   `json:"parent_id"`
	ManagerID *uint   `json:"manager_id"`
	SortOrder *int    `json:"sort_order"`
	Remark    *string `json:"remark"`
}

// DepartmentNode 部门树节点
type DepartmentNode struct {
	models.Department
	Children []*DepartmentNode `json:"children"`
}

// DepartmentService 部门业务服务
type DepartmentService struct {
	repo *repository.DepartmentRepo
}

// NewDepartmentService 创建部门服务
func NewDepartmentService(repo *repository.DepartmentRepo) *DepartmentService {
	return &DepartmentService{repo: repo}
}

// GetByID 查询部门，供处理器拼接审计描述等场景使用
func (s *DepartmentService) GetByID(id uint) (*models.Department, error) {
	dept, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeptNotFound
		}
		return nil, err
	}
	return dept, nil
}

// ListTree 返回部门树形列表（一次性返回全部，前端组树）
func (s *DepartmentService) ListTree() ([]*DepartmentNode, error) {
	depts, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	return buildDeptTree(depts, 0), nil
}

// buildDeptTree 递归构建部门树
func buildDeptTree(depts []models.Department, parentID uint) []*DepartmentNode {
	nodes := make([]*DepartmentNode, 0)
	for _, d := range depts {
		if d.ParentID == parentID {
			node := &DepartmentNode{Department: d}
			node.Children = buildDeptTree(depts, d.ID)
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// Create 创建部门：校验名称非空、上级部门存在
func (s *DepartmentService) Create(req CreateDepartmentRequest) (*models.Department, error) {
	if req.Name == "" {
		return nil, ErrDeptNameEmpty
	}
	if req.ParentID > 0 {
		ok, err := s.repo.Exists(req.ParentID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrDeptParentMissing
		}
	}
	dept := &models.Department{
		Name:      req.Name,
		ParentID:  req.ParentID,
		ManagerID: req.ManagerID,
		SortOrder: req.SortOrder,
		Remark:    req.Remark,
	}
	if err := s.repo.Create(dept); err != nil {
		return nil, err
	}
	return dept, nil
}

// Update 更新部门：校验存在性、上级部门存在性与环引用
func (s *DepartmentService) Update(id uint, req UpdateDepartmentRequest) error {
	dept, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDeptNotFound
		}
		return err
	}
	fields := map[string]interface{}{}
	if req.Name != nil && *req.Name != "" {
		fields["name"] = *req.Name
	}
	if req.ParentID != nil {
		if *req.ParentID == id {
			return ErrDeptCycle
		}
		if *req.ParentID > 0 {
			ok, err := s.repo.Exists(*req.ParentID)
			if err != nil {
				return err
			}
			if !ok {
				return ErrDeptParentMissing
			}
			// 防止把部门挂到自己的下级，形成环
			descendant, err := s.isDescendant(dept, *req.ParentID)
			if err != nil {
				return err
			}
			if descendant {
				return ErrDeptCycle
			}
		}
		fields["parent_id"] = *req.ParentID
	}
	if req.ManagerID != nil {
		if *req.ManagerID == 0 {
			fields["manager_id"] = nil
		} else {
			fields["manager_id"] = *req.ManagerID
		}
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}
	if req.Remark != nil {
		fields["remark"] = *req.Remark
	}
	if len(fields) == 0 {
		return nil
	}
	return s.repo.Update(id, fields)
}

// isDescendant 判断 candidateID 是否为 dept 的下级部门（沿父链向上查找）
func (s *DepartmentService) isDescendant(dept *models.Department, candidateID uint) (bool, error) {
	visited := make(map[uint]bool)
	cur := candidateID
	for cur != 0 {
		if cur == dept.ID {
			return true, nil
		}
		if visited[cur] {
			return false, nil
		}
		visited[cur] = true
		d, err := s.repo.FindByID(cur)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return false, err
		}
		cur = d.ParentID
	}
	return false, nil
}

// Delete 删除部门：有子部门或员工时拒绝
func (s *DepartmentService) Delete(id uint) error {
	if _, err := s.repo.FindByID(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDeptNotFound
		}
		return err
	}
	hasChildren, err := s.repo.HasChildren(id)
	if err != nil {
		return err
	}
	if hasChildren {
		return ErrDeptHasChildren
	}
	hasEmployees, err := s.repo.HasEmployees(id)
	if err != nil {
		return err
	}
	if hasEmployees {
		return ErrDeptHasEmployees
	}
	return s.repo.Delete(id)
}
