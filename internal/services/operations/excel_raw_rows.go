package operations

// Phase 44 R3 / Plan 44-02 Task 4 — Excel raw 行读取 helper (供 ImportFromExcel 后处理)
//
// 设计:excel_service.ImportData 不返回 raw Excel 行(只返回 AffectedKeys),
// 资产例外规则导入需要 raw 行做后处理(scope_name→scope_id + TEXT[] 转换)。
// 本 helper 二次打开 Excel,按 SheetName + header 解析,返回 map[name]row。

import (
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ReadRawRowsByName 打开 Excel 指定 sheet,按 header 解析,返回 map[name]row 字段
//
// 行为:
//   - 第 1 行是 header (列名匹配 ExcelConfig.Columns[].Field)
//   - 第 2 行起是数据
//   - 用 name 列(必须存在)做 key,同名后者覆盖前者
//   - 字段值原样保留 CSV 字符串(后续由调用方 ParseCSVToTextArray 转 TEXT[])
//
// 用于 ImportFromExcel 后处理 — 读 scopeName/conflictTypes/exceptionActions 原始值。
func ReadRawRowsByName(file *multipart.FileHeader, sheetName string) (map[string]map[string]interface{}, error) {
	if file == nil {
		return nil, fmt.Errorf("文件不能为空")
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	const (
		unzipSizeLimit    = 200 * 1024 * 1024
		unzipXMLSizeLimit = 100 * 1024 * 1024
	)
	f, err := excelize.OpenReader(src, excelize.Options{
		UnzipSizeLimit:    unzipSizeLimit,
		UnzipXMLSizeLimit: unzipXMLSizeLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("解析 Excel 失败: %w", err)
	}
	defer f.Close()

	// 优先用配置 sheet,找不到则回退第一个
	rows, err := f.GetRows(sheetName)
	if err != nil {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("Excel 无 sheet")
		}
		rows, err = f.GetRows(sheets[0])
		if err != nil {
			return nil, fmt.Errorf("读取 sheet 失败: %w", err)
		}
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel 数据为空(需 header + 至少 1 行数据)")
	}

	// 第 1 行是 header (normalizeHeader 同 excel_service 内实现, 简化版)
	header := rows[0]
	headerNormalized := make([]string, len(header))
	for i, h := range header {
		headerNormalized[i] = normalizeHeaderTrim(h)
	}

	// 找 name 列索引
	nameIdx := -1
	for i, h := range headerNormalized {
		if h == "name" || h == "规则名称" {
			nameIdx = i
			break
		}
	}
	if nameIdx < 0 {
		return nil, fmt.Errorf("Excel 缺少 name 列")
	}

	// 第 2 行起按 header 解析
	result := make(map[string]map[string]interface{}, len(rows)-1)
	for _, row := range rows[1:] {
		if len(row) == 0 {
			continue
		}
		// 取 name (按 nameIdx)
		name := ""
		if nameIdx < len(row) {
			name = strings.TrimSpace(row[nameIdx])
		}
		if name == "" {
			continue
		}

		// 解析整行 → map[headerField]value
		rowMap := make(map[string]interface{}, len(headerNormalized))
		for i, val := range row {
			if i >= len(headerNormalized) {
				break
			}
			rowMap[headerNormalized[i]] = val
		}
		result[name] = rowMap
	}

	return result, nil
}

// normalizeHeaderTrim 简化版 header 规范化(只去首尾空格 + 转小写用于匹配)
//
// 注:excel_service 内部有更完整的 normalizeHeader 处理 BOM 等,
// 此 helper 仅做 TrimSpace(对外 header 已规范化的 Excel 模板场景足够)。
func normalizeHeaderTrim(s string) string {
	return strings.TrimSpace(s)
}
