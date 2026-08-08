package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"hrms/backend/internal/models"
	"hrms/backend/internal/repository"
)

// createSalaryEmployee 创建一条仅用于薪酬测试的员工记录
func createSalaryEmployee(tx *gorm.DB) (*models.Employee, error) {
	emp := &models.Employee{
		EmpNo:  uniqueStr("EMP"),
		Name:   uniqueStr("薪资员工"),
		Gender: 1,
		Status: 1,
	}
	if err := tx.Create(emp).Error; err != nil {
		return nil, err
	}
	return emp, nil
}

// newSalarySvc 构造薪酬服务（基于事务内的仓库）
func newSalarySvc(tx *gorm.DB) *SalaryService {
	return NewSalaryService(tx, repository.NewSalaryRepo(tx), repository.NewEmployeeRepo(tx))
}

// TestNetSalaryCalculation 实发工资计算：base + allowance + bonus - deduction
func TestNetSalaryCalculation(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	emp, err := createSalaryEmployee(tx)
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}

	svc := newSalarySvc(tx)
	rec, err := svc.Create(CreateSalaryRequest{
		EmployeeID: emp.ID, Month: "2026-08",
		BaseSalary: 10000, Allowance: 1500, Bonus: 2000, Deduction: 500,
	})
	if err != nil {
		t.Fatalf("创建工资单失败: %v", err)
	}
	if rec.NetSalary != 13000 {
		t.Errorf("期望实发 13000，实际 %v", rec.NetSalary)
	}
}

// TestSalaryDuplicateMonth 同员工同月创建第二张工资单应返回错误
func TestSalaryDuplicateMonth(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	emp, err := createSalaryEmployee(tx)
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}

	svc := newSalarySvc(tx)
	if _, err := svc.Create(CreateSalaryRequest{EmployeeID: emp.ID, Month: "2026-08", BaseSalary: 8000}); err != nil {
		t.Fatalf("首次创建应成功: %v", err)
	}
	if _, err := svc.Create(CreateSalaryRequest{EmployeeID: emp.ID, Month: "2026-08", BaseSalary: 9000}); !errors.Is(err, ErrSalaryAlreadyExists) {
		t.Fatalf("同月重复创建应返回 ErrSalaryAlreadyExists，实际: %v", err)
	}
}

// TestSalaryStatusFlow 状态流转：draft 可更新；confirm 后可 pay；paid 再 confirm 报错；draft 直接 pay 报错
func TestSalaryStatusFlow(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	emp, err := createSalaryEmployee(tx)
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}

	svc := newSalarySvc(tx)
	rec, err := svc.Create(CreateSalaryRequest{EmployeeID: emp.ID, Month: "2026-09", BaseSalary: 8000})
	if err != nil {
		t.Fatalf("创建工资单失败: %v", err)
	}

	// draft 可更新
	base := 9000.0
	updated, err := svc.Update(rec.ID, UpdateSalaryRequest{BaseSalary: &base})
	if err != nil {
		t.Fatalf("draft 状态更新应成功: %v", err)
	}
	if updated.NetSalary != 9000 {
		t.Errorf("更新后期望实发 9000，实际 %v", updated.NetSalary)
	}

	// confirm 后可 pay
	if err := svc.Confirm(rec.ID); err != nil {
		t.Fatalf("confirm 应成功: %v", err)
	}
	if err := svc.Pay(rec.ID); err != nil {
		t.Fatalf("pay 应成功: %v", err)
	}

	// 已 pay 再 confirm → 错误
	if err := svc.Confirm(rec.ID); !errors.Is(err, ErrSalaryNotDraft) {
		t.Fatalf("paid 再 confirm 应返回 ErrSalaryNotDraft，实际: %v", err)
	}

	// draft 直接 pay → 错误
	rec2, err := svc.Create(CreateSalaryRequest{EmployeeID: emp.ID, Month: "2026-10", BaseSalary: 7000})
	if err != nil {
		t.Fatalf("创建第二张工资单失败: %v", err)
	}
	if err := svc.Pay(rec2.ID); !errors.Is(err, ErrSalaryNotConfirmed) {
		t.Fatalf("draft 直接 pay 应返回 ErrSalaryNotConfirmed，实际: %v", err)
	}
}

// TestSalaryUpdateRecalculatesNet 修改基本工资后实发自动重算
func TestSalaryUpdateRecalculatesNet(t *testing.T) {
	db := mustTestDB(t)
	tx := db.Begin()
	defer tx.Rollback()
	emp, err := createSalaryEmployee(tx)
	if err != nil {
		t.Fatalf("创建员工失败: %v", err)
	}

	svc := newSalarySvc(tx)
	rec, err := svc.Create(CreateSalaryRequest{EmployeeID: emp.ID, Month: "2026-11", BaseSalary: 10000})
	if err != nil {
		t.Fatalf("创建工资单失败: %v", err)
	}
	if rec.NetSalary != 10000 {
		t.Errorf("初始实发期望 10000，实际 %v", rec.NetSalary)
	}

	base := 12000.0
	updated, err := svc.Update(rec.ID, UpdateSalaryRequest{BaseSalary: &base})
	if err != nil {
		t.Fatalf("更新工资单失败: %v", err)
	}
	if updated.NetSalary != 12000 {
		t.Errorf("更新后实发期望 12000，实际 %v", updated.NetSalary)
	}
}
