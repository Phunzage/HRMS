package database

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"hrms/backend/internal/models"
)

// Seed 写入种子数据（幂等：已存在则跳过）
func Seed(db *gorm.DB) error {
	if err := seedRoles(db); err != nil {
		return err
	}
	if err := seedPermissions(db); err != nil {
		return err
	}
	if err := seedRolePermissions(db); err != nil {
		return err
	}
	userIDs, err := seedUsers(db)
	if err != nil {
		return err
	}
	deptIDs, err := seedDepartments(db)
	if err != nil {
		return err
	}
	posIDs, err := seedPositions(db, deptIDs)
	if err != nil {
		return err
	}
	if err := seedEmployees(db, userIDs, deptIDs, posIDs); err != nil {
		return err
	}
	return seedApprovalFlows(db)
}

// seedApprovalFlows 写入审批流程定义（固定两级：部门经理 → HR 终审），幂等
func seedApprovalFlows(db *gorm.DB) error {
	flows := []struct {
		code  string
		name  string
		steps []models.ApprovalStep
	}{
		{
			code: "leave",
			name: "请假审批",
			steps: []models.ApprovalStep{
				{StepOrder: 1, ApproverType: "manager", Name: "部门经理审批"},
				{StepOrder: 2, ApproverType: "hr", Name: "HR终审"},
			},
		},
		{
			code: "attendance_correction",
			name: "补卡审批",
			steps: []models.ApprovalStep{
				{StepOrder: 1, ApproverType: "manager", Name: "部门经理审批"},
				{StepOrder: 2, ApproverType: "hr", Name: "HR终审"},
			},
		},
	}
	for _, f := range flows {
		var existing models.ApprovalFlow
		err := db.Where("code = ?", f.code).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			flow := models.ApprovalFlow{Code: f.code, Name: f.name, Status: "active"}
			if err := db.Create(&flow).Error; err != nil {
				return fmt.Errorf("创建审批流程 %s 失败: %w", f.code, err)
			}
			for _, st := range f.steps {
				st.FlowID = flow.ID
				if err := db.Create(&st).Error; err != nil {
					return fmt.Errorf("创建审批流程 %s 步骤 %d 失败: %w", f.code, st.StepOrder, err)
				}
			}
			slog.Info("种子审批流程已创建", "code", f.code, "name", f.name)
		case err != nil:
			return fmt.Errorf("查询审批流程 %s 失败: %w", f.code, err)
		}
	}
	return nil
}

// seedRoles 写入内置角色
func seedRoles(db *gorm.DB) error {
	roles := []models.Role{
		{Name: "管理员", Code: "admin", Description: "系统管理员，拥有全部权限"},
		{Name: "人事", Code: "hr", Description: "人事专员，负责人事/考勤/薪酬"},
		{Name: "经理", Code: "manager", Description: "部门经理，负责本部门审批"},
		{Name: "员工", Code: "employee", Description: "普通员工，自助服务"},
	}
	for _, role := range roles {
		var existing models.Role
		err := db.Where("code = ?", role.Code).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			if err := db.Create(&role).Error; err != nil {
				return fmt.Errorf("创建角色 %s 失败: %w", role.Code, err)
			}
			slog.Info("种子角色已创建", "code", role.Code, "name", role.Name)
		case err != nil:
			return fmt.Errorf("查询角色 %s 失败: %w", role.Code, err)
		}
	}
	return nil
}

// seedPermissions 写入权限点
func seedPermissions(db *gorm.DB) error {
	permissions := []models.Permission{
		{Name: "员工查看", Code: "employee:read", Module: "employee"},
		{Name: "员工编辑", Code: "employee:write", Module: "employee"},
		{Name: "部门查看", Code: "department:read", Module: "department"},
		{Name: "部门编辑", Code: "department:write", Module: "department"},
		{Name: "岗位查看", Code: "position:read", Module: "position"},
		{Name: "岗位编辑", Code: "position:write", Module: "position"},
		{Name: "考勤查看", Code: "attendance:read", Module: "attendance"},
		{Name: "考勤编辑", Code: "attendance:write", Module: "attendance"},
		{Name: "请假查看", Code: "leave:read", Module: "leave"},
		{Name: "请假编辑", Code: "leave:write", Module: "leave"},
		{Name: "薪资查看", Code: "salary:read", Module: "salary"},
		{Name: "薪资编辑", Code: "salary:write", Module: "salary"},
		{Name: "招聘查看", Code: "recruitment:read", Module: "recruitment"},
		{Name: "招聘编辑", Code: "recruitment:write", Module: "recruitment"},
		{Name: "统计仪表盘查看", Code: "dashboard:read", Module: "dashboard"},
		{Name: "用户查看", Code: "user:read", Module: "user"},
		{Name: "用户编辑", Code: "user:write", Module: "user"},
		{Name: "角色查看", Code: "role:read", Module: "role"},
		{Name: "角色编辑", Code: "role:write", Module: "role"},
		{Name: "审计日志查看", Code: "audit:read", Module: "audit"},
		{Name: "公告查看", Code: "announcement:read", Module: "announcement"},
		{Name: "公告编辑", Code: "announcement:write", Module: "announcement"},
		{Name: "绩效查看", Code: "performance:read", Module: "performance"},
		{Name: "绩效编辑", Code: "performance:write", Module: "performance"},
		{Name: "审批流程查看", Code: "approval_flow:read", Module: "approval"},
		{Name: "审批流程配置", Code: "approval_flow:write", Module: "approval"},
		{Name: "请假审批", Code: "leave:approve", Module: "approval"},
		{Name: "补卡审批", Code: "attendance_correction:approve", Module: "approval"},
	}
	for _, p := range permissions {
		var existing models.Permission
		err := db.Where("code = ?", p.Code).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			if err := db.Create(&p).Error; err != nil {
				return fmt.Errorf("创建权限点 %s 失败: %w", p.Code, err)
			}
			slog.Info("种子权限点已创建", "code", p.Code)
		case err != nil:
			return fmt.Errorf("查询权限点 %s 失败: %w", p.Code, err)
		}
	}
	return nil
}

// seedRolePermissions 写入角色-权限关联（admin 不再直接放行，统一查 role_permissions，保证业务只读生效）
func seedRolePermissions(db *gorm.DB) error {
	rolePerms := map[string][]string{
		// admin = 系统管理员：专注系统管理（用户/角色/审计）+ 审批流程配置，仅保留公告与仪表盘监督；
		// 唯一例外 position:read：审批流程配置「指定岗位」步骤需要岗位列表填充下拉
		"admin": {
			"user:read", "user:write",
			"role:read", "role:write",
			"audit:read",
			"dashboard:read",
			"announcement:read", "announcement:write",
			"approval_flow:read", "approval_flow:write",
			"position:read",
		},
		// hr = 业务管理员：全公司口径，负责人事业务读写（含公告），不再有用户/角色/审计管理权限；
		// 审批终审职能改由权限控制（leave:approve + attendance_correction:approve），不再是角色写死
		"hr": {
			"employee:read", "employee:write",
			"department:read", "department:write",
			"position:read", "position:write",
			"attendance:read", "attendance:write",
			"leave:read", "leave:write",
			"salary:read", "salary:write",
			"recruitment:read", "recruitment:write",
			"performance:read", "performance:write",
			"announcement:read", "announcement:write",
			"dashboard:read",
			"leave:approve", "attendance_correction:approve",
		},
		// manager：部门经理继续一审请假（leave:approve），机制改为权限控制
		"manager": {
			"employee:read",
			// position:read：经理查看本部门岗位、员工管理页岗位筛选需要（工资/招聘仍为 hr/admin 专属）
			"position:read",
			"attendance:read", "attendance:write",
			"leave:read", "leave:write",
			"department:read",
			"announcement:read",
			"performance:read", "performance:write",
			"leave:approve",
		},
		// employee 角色无 employee:read：普通员工的员工档案只能通过 GET /employees/me 访问本人
		// attendance:write 用于员工自助打卡（check-in/check-out）
		"employee": {
			"attendance:read", "attendance:write",
			"leave:read", "leave:write",
			"salary:read",
			"announcement:read",
			"performance:read",
		},
	}
	for roleCode, codes := range rolePerms {
		var role models.Role
		if err := db.Where("code = ?", roleCode).First(&role).Error; err != nil {
			return fmt.Errorf("查询角色 %s 失败: %w", roleCode, err)
		}
		for _, code := range codes {
			var perm models.Permission
			if err := db.Where("code = ?", code).First(&perm).Error; err != nil {
				return fmt.Errorf("查询权限点 %s 失败: %w", code, err)
			}
			var count int64
			db.Model(&models.RolePermission{}).
				Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).
				Count(&count)
			if count == 0 {
				if err := db.Create(&models.RolePermission{RoleID: role.ID, PermissionID: perm.ID}).Error; err != nil {
					return fmt.Errorf("关联角色 %s 权限 %s 失败: %w", roleCode, code, err)
				}
			}
		}
		// 清理：移除该角色下不在配置中的权限关联，保持与种子定义一致（幂等自愈）
		var permIDs []uint
		if err := db.Model(&models.Permission{}).Where("code IN ?", codes).Pluck("id", &permIDs).Error; err != nil {
			return fmt.Errorf("查询角色 %s 权限点失败: %w", roleCode, err)
		}
		if len(permIDs) > 0 {
			if err := db.Where("role_id = ? AND permission_id NOT IN ?", role.ID, permIDs).Delete(&models.RolePermission{}).Error; err != nil {
				return fmt.Errorf("清理角色 %s 权限关联失败: %w", roleCode, err)
			}
		}
	}
	return nil
}

// seedUsers 写入内置账号（bcrypt 哈希存储，幂等）
// 数据范围：hr 角色账号默认不设置 dept_scope_id（NULL = 全公司口径，负责整个公司人力资源）
func seedUsers(db *gorm.DB) (map[string]uint, error) {
	accounts := []struct {
		username string
		password string
		email    string
		fullName string
		roleCode string
	}{
		{username: "admin", password: "admin123", email: "admin@example.com", fullName: "系统管理员", roleCode: "admin"},
		{username: "hr", password: "hr123", email: "hr@example.com", fullName: "人事专员", roleCode: "hr"},
		{username: "zhangwei", password: "123456", email: "zhangwei@example.com", fullName: "张伟", roleCode: "manager"},
		{username: "liuyang", password: "123456", email: "liuyang@example.com", fullName: "刘洋", roleCode: "employee"},
	}
	userIDs := make(map[string]uint, len(accounts))
	for _, acc := range accounts {
		var existing models.User
		err := db.Where("username = ?", acc.username).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			var role models.Role
			if err := db.Where("code = ?", acc.roleCode).First(&role).Error; err != nil {
				return nil, fmt.Errorf("查询角色 %s 失败: %w", acc.roleCode, err)
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(acc.password), bcrypt.DefaultCost)
			if err != nil {
				return nil, fmt.Errorf("生成密码哈希失败: %w", err)
			}
			user := models.User{
				Username:     acc.username,
				PasswordHash: string(hash),
				Email:        acc.email,
				FullName:     acc.fullName,
				RoleID:       role.ID,
				Status:       1,
			}
			if err := db.Create(&user).Error; err != nil {
				return nil, fmt.Errorf("创建账号 %s 失败: %w", acc.username, err)
			}
			userIDs[acc.username] = user.ID
			slog.Info("种子账号已创建", "username", acc.username, "role", acc.roleCode)
		case err != nil:
			return nil, fmt.Errorf("查询账号 %s 失败: %w", acc.username, err)
		default:
			userIDs[acc.username] = existing.ID
		}
	}
	return userIDs, nil
}

// seedDepartments 写入部门（技术部下设前端组/后端组），返回 部门名 → ID
func seedDepartments(db *gorm.DB) (map[string]uint, error) {
	deptIDs := make(map[string]uint, 6)
	// 顶级部门
	topLevel := []models.Department{
		{Name: "技术部", SortOrder: 1, Remark: "负责产品研发"},
		{Name: "人事部", SortOrder: 2, Remark: "负责人力资源管理"},
		{Name: "财务部", SortOrder: 3, Remark: "负责财务管理"},
		{Name: "市场部", SortOrder: 4, Remark: "负责市场推广"},
	}
	for _, d := range topLevel {
		var existing models.Department
		err := db.Where("name = ? AND parent_id = 0", d.Name).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			if err := db.Create(&d).Error; err != nil {
				return nil, fmt.Errorf("创建部门 %s 失败: %w", d.Name, err)
			}
			deptIDs[d.Name] = d.ID
			slog.Info("种子部门已创建", "name", d.Name)
		case err != nil:
			return nil, fmt.Errorf("查询部门 %s 失败: %w", d.Name, err)
		default:
			deptIDs[d.Name] = existing.ID
		}
	}
	// 技术部子部门
	children := []models.Department{
		{Name: "前端组", ParentID: deptIDs["技术部"], SortOrder: 1, Remark: "负责前端开发"},
		{Name: "后端组", ParentID: deptIDs["技术部"], SortOrder: 2, Remark: "负责后端开发"},
	}
	for _, d := range children {
		var existing models.Department
		err := db.Where("name = ? AND parent_id = ?", d.Name, d.ParentID).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			if err := db.Create(&d).Error; err != nil {
				return nil, fmt.Errorf("创建部门 %s 失败: %w", d.Name, err)
			}
			deptIDs[d.Name] = d.ID
			slog.Info("种子部门已创建", "name", d.Name, "parent_id", d.ParentID)
		case err != nil:
			return nil, fmt.Errorf("查询部门 %s 失败: %w", d.Name, err)
		default:
			deptIDs[d.Name] = existing.ID
		}
	}
	return deptIDs, nil
}

// seedPositions 写入岗位，返回 岗位编码 → ID
func seedPositions(db *gorm.DB, deptIDs map[string]uint) (map[string]uint, error) {
	positions := []struct {
		name  string
		code  string
		dept  string
		level int
	}{
		{name: "部门经理", code: "dept_manager", dept: "技术部", level: 3},
		{name: "后端开发工程师", code: "backend_engineer", dept: "后端组", level: 2},
		{name: "前端开发工程师", code: "frontend_engineer", dept: "前端组", level: 2},
		{name: "人事专员", code: "hr_specialist", dept: "人事部", level: 1},
		{name: "财务专员", code: "finance_specialist", dept: "财务部", level: 1},
	}
	posIDs := make(map[string]uint, len(positions))
	for _, p := range positions {
		var existing models.Position
		err := db.Where("code = ?", p.code).First(&existing).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			pos := models.Position{
				Name:         p.name,
				Code:         p.code,
				DepartmentID: deptIDs[p.dept],
				Level:        p.level,
			}
			if err := db.Create(&pos).Error; err != nil {
				return nil, fmt.Errorf("创建岗位 %s 失败: %w", p.code, err)
			}
			posIDs[p.code] = pos.ID
			slog.Info("种子岗位已创建", "code", p.code, "name", p.name)
		case err != nil:
			return nil, fmt.Errorf("查询岗位 %s 失败: %w", p.code, err)
		default:
			posIDs[p.code] = existing.ID
		}
	}
	return posIDs, nil
}

// seedEmployees 写入示例员工，并回写账号与员工互相关联
func seedEmployees(db *gorm.DB, userIDs, deptIDs, posIDs map[string]uint) error {
	employees := []struct {
		name     string
		username string
		dept     string
		pos      string
		gender   int
		phone    string
		email    string
	}{
		{name: "张伟", username: "zhangwei", dept: "技术部", pos: "dept_manager", gender: 1, phone: "13800000001", email: "zhangwei@example.com"},
		{name: "刘洋", username: "liuyang", dept: "后端组", pos: "backend_engineer", gender: 1, phone: "13800000002", email: "liuyang@example.com"},
	}
	var managerEmpID uint // 张伟的员工ID，用于回写部门负责人
	for _, e := range employees {
		userID := userIDs[e.username]
		var emp models.Employee
		err := db.Where("name = ?", e.name).First(&emp).Error
		switch {
		case err == gorm.ErrRecordNotFound:
			empNo, err := nextEmpNo(db)
			if err != nil {
				return err
			}
			emp = models.Employee{
				EmpNo:        empNo,
				UserID:       &userID,
				Name:         e.name,
				Gender:       e.gender,
				Phone:        e.phone,
				Email:        e.email,
				DepartmentID: deptIDs[e.dept],
				PositionID:   posIDs[e.pos],
				Status:       1,
			}
			if err := db.Create(&emp).Error; err != nil {
				return fmt.Errorf("创建员工 %s 失败: %w", e.name, err)
			}
			slog.Info("种子员工已创建", "emp_no", empNo, "name", e.name)
		case err != nil:
			return fmt.Errorf("查询员工 %s 失败: %w", e.name, err)
		}
		// 回写双向关联（幂等）
		if err := db.Model(&models.Employee{}).Where("id = ?", emp.ID).Update("user_id", userID).Error; err != nil {
			return fmt.Errorf("回写员工账号关联失败: %w", err)
		}
		if err := db.Model(&models.User{}).Where("id = ?", userID).Update("employee_id", emp.ID).Error; err != nil {
			return fmt.Errorf("回写账号员工关联失败: %w", err)
		}
		if e.username == "zhangwei" {
			managerEmpID = emp.ID
		}
	}
	// 回写部门负责人：技术部/后端组经理 = 张伟（保证请假审批流能自动找到审批人，幂等）
	if managerEmpID > 0 {
		deptNames := []string{"技术部", "后端组"}
		for _, name := range deptNames {
			if err := db.Model(&models.Department{}).Where("name = ?", name).Update("manager_id", managerEmpID).Error; err != nil {
				return fmt.Errorf("回写部门 %s 负责人失败: %w", name, err)
			}
		}
		slog.Info("部门负责人已设置", "manager_id", managerEmpID)
	}
	return nil
}

// nextEmpNo 生成下一个工号（EMP0001 起，递增）
func nextEmpNo(db *gorm.DB) (string, error) {
	var maxNo string
	if err := db.Unscoped().Model(&models.Employee{}).Order("emp_no DESC").Limit(1).Pluck("emp_no", &maxNo).Error; err != nil {
		return "", err
	}
	if maxNo == "" {
		return "EMP0001", nil
	}
	seq := 0
	if len(maxNo) > 3 {
		if v, err := strconv.Atoi(maxNo[3:]); err == nil {
			seq = v
		}
	}
	return fmt.Sprintf("EMP%04d", seq+1), nil
}
