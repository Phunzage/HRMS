package service

import (
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// ResolveScope 解析当前登录用户的数据范围：返回可访问的部门 ID 列表
// admin / manager / employee 角色 → 返回 nil（不过滤，保持全量或既有角色逻辑）
// hr 角色 → 取用户绑定的 DeptScopeID：为空返回 nil（全量）；非空返回该部门及其所有子孙部门 ID
// 列表过滤的 service 层方法在开头调用本函数，将结果作为可选 deptIDs 参数传给 repository（nil 不过滤）
func ResolveScope(db *gorm.DB, userID uint, roleCode string) ([]uint, error) {
	if roleCode != "hr" {
		return nil, nil
	}
	var user models.User
	if err := db.Select("dept_scope_id").First(&user, userID).Error; err != nil {
		return nil, err
	}
	if user.DeptScopeID == nil {
		return nil, nil
	}
	return CollectDeptSubtree(db, *user.DeptScopeID)
}

// CollectDeptSubtree 收集部门及其所有子孙部门的 ID（递归遍历 departments.parent_id）
func CollectDeptSubtree(db *gorm.DB, rootID uint) ([]uint, error) {
	// 一次性取全部未删除部门，内存中构建子节点映射后递归，避免逐层查库
	var depts []models.Department
	if err := db.Select("id", "parent_id").Find(&depts).Error; err != nil {
		return nil, err
	}
	children := make(map[uint][]uint, len(depts))
	for _, d := range depts {
		children[d.ParentID] = append(children[d.ParentID], d.ID)
	}
	ids := make([]uint, 0)
	var walk func(id uint)
	walk = func(id uint) {
		ids = append(ids, id)
		for _, c := range children[id] {
			walk(c)
		}
	}
	walk(rootID)
	return ids, nil
}
