package service

import (
	"sync"
	"testing"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// TestGenerateEmpNo 工号生成：库中最大 EMP0005 → 生成 EMP0006
func TestGenerateEmpNo(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	// 事务内清空员工表，保证最大工号确定性（回滚后不影响已提交数据）
	if err := tx.Unscoped().Where("1 = 1").Delete(&models.Employee{}).Error; err != nil {
		t.Fatalf("清理员工表失败: %v", err)
	}
	dept, pos, err := createTestDeptPos(tx)
	if err != nil {
		t.Fatalf("构造部门/岗位失败: %v", err)
	}
	// 直接插入最大工号 EMP0005
	if _, err := createTestEmployee(tx, "EMP0005", "占位员工", dept.ID, pos.ID); err != nil {
		t.Fatalf("插入占位员工失败: %v", err)
	}

	svc := NewEmployeeService(tx, repository.NewEmployeeRepo(tx), repository.NewDepartmentRepo(tx), repository.NewPositionRepo(tx))
	res, err := svc.Create(CreateEmployeeRequest{Name: "新员工", DepartmentID: dept.ID, PositionID: pos.ID})
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}
	if res.Employee.EmpNo != "EMP0006" {
		t.Errorf("期望工号 EMP0006，实际 %s", res.Employee.EmpNo)
	}
}

// TestEmpNoConcurrentUnique 并发创建员工：10 个全部成功且工号不重复（pg_advisory_xact_lock 串行化）
func TestEmpNoConcurrentUnique(t *testing.T) {
	db := mustTestDB(t)
	// 共享部门/岗位（seedTestBasics 已提交），并发创建各自开独立事务提交
	var dept models.Department
	if err := db.Where("name = ?", "测试技术部").First(&dept).Error; err != nil {
		t.Fatalf("查询共享测试部门失败: %v", err)
	}
	var pos models.Position
	if err := db.Where("code = ?", "test_engineer").First(&pos).Error; err != nil {
		t.Fatalf("查询共享测试岗位失败: %v", err)
	}

	svc := NewEmployeeService(db, repository.NewEmployeeRepo(db), repository.NewDepartmentRepo(db), repository.NewPositionRepo(db))

	const n = 10
	var wg sync.WaitGroup
	empNos := make([]string, n)
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := svc.Create(CreateEmployeeRequest{
				Name:         uniqueStr("并发员工"),
				DepartmentID: dept.ID,
				PositionID:   pos.ID,
			})
			if err != nil {
				errs[i] = err
				return
			}
			empNos[i] = res.Employee.EmpNo
		}(i)
	}
	wg.Wait()

	for i := range n {
		if errs[i] != nil {
			t.Fatalf("第 %d 个并发创建失败: %v", i, errs[i])
		}
	}
	seen := make(map[string]bool, n)
	for _, no := range empNos {
		if no == "" {
			t.Fatal("存在空工号")
		}
		if seen[no] {
			t.Errorf("工号重复: %s", no)
		}
		seen[no] = true
	}
}
