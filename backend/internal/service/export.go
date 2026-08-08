package service

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// WriteExcel 生成 Excel 文件二进制内容：
// - 表头加粗、冻结首行、列宽自适应（中文按 2 个字符宽度估算）
// - filename 仅用于错误上下文，实际下载文件名由 handler 在 Content-Disposition 中指定
func WriteExcel(filename, sheetName string, headers []string, rows [][]interface{}) ([]byte, error) {
	f := excelize.NewFile()
	// 重命名默认工作表为目标表名
	f.SetSheetName("Sheet1", sheetName)

	// 写表头
	headerRow := make([]interface{}, len(headers))
	for i, h := range headers {
		headerRow[i] = h
	}
	if err := f.SetSheetRow(sheetName, "A1", &headerRow); err != nil {
		return nil, fmt.Errorf("导出 %s 写入表头失败: %w", filename, err)
	}

	// 表头加粗样式
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	if err != nil {
		return nil, fmt.Errorf("导出 %s 创建表头样式失败: %w", filename, err)
	}
	lastCol, err := excelize.ColumnNumberToName(len(headers))
	if err != nil {
		return nil, fmt.Errorf("导出 %s 计算列号失败: %w", filename, err)
	}
	if err := f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%s1", lastCol), headerStyle); err != nil {
		return nil, fmt.Errorf("导出 %s 应用表头样式失败: %w", filename, err)
	}

	// 写数据行
	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+2)
		if err != nil {
			return nil, fmt.Errorf("导出 %s 计算单元格失败: %w", filename, err)
		}
		if err := f.SetSheetRow(sheetName, cell, &row); err != nil {
			return nil, fmt.Errorf("导出 %s 写入数据行失败: %w", filename, err)
		}
	}

	// 列宽自适应：每列取表头与数据中最大宽度（中文按 2 计），加 2 字符内边距
	for i, h := range headers {
		maxW := displayWidth(h)
		for _, row := range rows {
			if i < len(row) {
				if w := displayWidth(row[i]); w > maxW {
					maxW = w
				}
			}
		}
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return nil, fmt.Errorf("导出 %s 计算列号失败: %w", filename, err)
		}
		width := maxW + 2
		if width < 8 {
			width = 8
		}
		if err := f.SetColWidth(sheetName, col, col, width); err != nil {
			return nil, fmt.Errorf("导出 %s 设置列宽失败: %w", filename, err)
		}
	}

	// 冻结首行
	if err := f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return nil, fmt.Errorf("导出 %s 冻结首行失败: %w", filename, err)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("导出 %s 生成文件失败: %w", filename, err)
	}
	return buf.Bytes(), nil
}

// displayWidth 估算单元格文本显示宽度：中文全角按 2 计，其余按 1 计
func displayWidth(v interface{}) float64 {
	s := fmt.Sprint(v)
	w := 0.0
	for _, r := range s {
		if r > 127 {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}
