package models

// RolePermission 角色-权限关联模型（复合主键保证唯一）
type RolePermission struct {
	RoleID       uint `gorm:"primaryKey" json:"role_id"`
	PermissionID uint `gorm:"primaryKey" json:"permission_id"`
}

// TableName 指定表名
func (RolePermission) TableName() string {
	return "role_permissions"
}
