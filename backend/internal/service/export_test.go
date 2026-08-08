package service

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestWriteExcel Excel 导出：生成的字节可被 excelize 重新打开，sheet 名/表头/行数正确
func TestWriteExcel(t *testing.T) {
	headers := []string{"月份", "工号", "姓名"}
	rows := [][]interface{}{
		{"2026-07", "EMP0001", "张三"},
		{"2026-08", "EMP0002", "李四"},
	}

	data, err := WriteExcel("salary.xlsx", "工资表", headers, rows)
	if err != nil {
		t.Fatalf("生成 Excel 失败: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("重新打开 Excel 失败: %v", err)
	}
	defer f.Close()

	// sheet 名
	sheet := f.GetSheetName(0)
	if sheet != "工资表" {
		t.Errorf("期望 sheet 名 工资表，实际 %s", sheet)
	}

	// 表头
	h1, _ := f.GetCellValue(sheet, "A1")
	h2, _ := f.GetCellValue(sheet, "B1")
	h3, _ := f.GetCellValue(sheet, "C1")
	if h1 != "月份" || h2 != "工号" || h3 != "姓名" {
		t.Errorf("表头不正确: %s | %s | %s", h1, h2, h3)
	}

	// 行数（表头 + 2 数据行）
	got, err := f.GetRows(sheet)
	if err != nil {
		t.Fatalf("读取行失败: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("期望 3 行（表头+2 数据），实际 %d", len(got))
	}
}
