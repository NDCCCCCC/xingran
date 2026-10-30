package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ExcelExportService interface {
	ExportData(ctx context.Context, entityType string, params map[string]interface{}) (*excelize.File, error)
}

type excelExportServiceImpl struct {
	db                  *gorm.DB
	queryBuilderFactory QueryBuilderFactory
}

func NewExcelExportService(db *gorm.DB) ExcelExportService {
	return &excelExportServiceImpl{
		db:                  db,
		queryBuilderFactory: NewQueryBuilderFactory(db),
	}
}

// ExportData 导出数据到Excel
func (s *excelExportServiceImpl) ExportData(
	ctx context.Context,
	entityType string,
	params map[string]interface{},
) (*excelize.File, error) {
	config, exists := GetExportConfig(entityType)
	if !exists {
		return nil, fmt.Errorf("不支持的实体类型: %s", entityType)
	}

	logger.Debugf("使用新导出配置导出: %s", entityType)

	f := excelize.NewFile()
	sheetName := config.SheetName
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	if deleteErr := f.DeleteSheet("Sheet1"); deleteErr != nil {
		logger.Debugf("删除默认Sheet1失败: %v", deleteErr)
	}

	headerStyle, err := createHeaderStyle(f)
	if err != nil {
		return nil, fmt.Errorf("创建表头样式失败: %w", err)
	}

	s.writeExportHeaders(f, sheetName, config.Columns, headerStyle)
	s.setExportColumnWidths(f, sheetName, config.Columns)

	data, err := s.queryExportData(ctx, config, params)
	if err != nil {
		return nil, fmt.Errorf("查询数据失败: %w", err)
	}

	data, err = s.resolveAssociations(ctx, config, data)
	if err != nil {
		return nil, fmt.Errorf("解析关联数据失败: %w", err)
	}

	s.writeExportDataRows(f, sheetName, config.Columns, data)

	_ = f.SetPanes(sheetName, &excelize.Panes{
		Freeze: true,
		XSplit: 1,
		YSplit: 1,
	})

	return f, nil
}

func (s *excelExportServiceImpl) queryExportData(
	ctx context.Context,
	config ExcelExportConfig,
	params map[string]interface{},
) ([]map[string]interface{}, error) {
	queryBuilder, exists := s.queryBuilderFactory.GetQueryBuilder(config.QueryBuilder)
	if !exists {
		return nil, fmt.Errorf("查询构建器不存在: %s", config.QueryBuilder)
	}

	query := queryBuilder.BuildQuery(ctx, s.db, config, params)

	var data []map[string]interface{}
	if err := query.Find(&data).Error; err != nil {
		return nil, err
	}

	return data, nil
}

func (s *excelExportServiceImpl) resolveAssociations(
	ctx context.Context,
	config ExcelExportConfig,
	data []map[string]interface{},
) ([]map[string]interface{}, error) {
	if len(data) == 0 {
		return data, nil
	}

	joins := make(map[string]*JoinConfig)
	for _, col := range config.Columns {
		if col.Join != nil {
			joinKey := col.Join.Table + ":" + col.Join.LeftField
			joins[joinKey] = col.Join
		}
	}

	if len(joins) == 0 {
		return data, nil
	}

	for _, join := range joins {
		ids := make([]string, 0, len(data))
		idSet := make(map[string]bool)
		for _, row := range data {
			if idValue, ok := row[join.LeftField]; ok && idValue != nil {
				if idStr, ok := idValue.(string); ok && idStr != "" {
					if !idSet[idStr] {
						idSet[idStr] = true
						ids = append(ids, idStr)
					}
				}
			}
		}

		if len(ids) == 0 {
			continue
		}

		results := make(map[string]string)
		// RightCast 字段(可选):RightField 类型转换,如 "text" → "id::text IN (?)"
		// 用于 LeftField 是 varchar 存 UUID 字符串但 RightField 是 uuid 类型的场景
		// (例: ops_info_points.workstation_id varchar, sys_workstation.id uuid)
		rightFieldExpr := join.RightField
		if join.RightCast != "" {
			rightFieldExpr = join.RightField + "::" + join.RightCast
		}
		// SkipSoftDelete 字段(可选):某些 join 目标表为硬删除表(无 deleted_at 列,如 sys_device_port_status),
		// 加 `deleted_at IS NULL` 会触发 PG SQLSTATE 42703 → 整个 join 失败 → 关联列整列空,
		// 错误被 logger.Warnf 后 continue 时静默吞掉。此字段为 true 时跳过该过滤。
		queryBuilder := s.db.WithContext(ctx).
			Table(join.Table).
			Select(join.RightField+", "+join.SelectField).
			Where(rightFieldExpr+" IN ?", ids)
		if !join.SkipSoftDelete {
			queryBuilder = queryBuilder.Where("deleted_at IS NULL")
		}
		rows, err := queryBuilder.Rows()

		if err != nil {
			logger.Warnf("批量查询关联数据失败: %v", err)
			continue
		}

		for rows.Next() {
			var id, value string
			if err := rows.Scan(&id, &value); err != nil {
				continue
			}
			results[id] = value
		}
		rows.Close()

		for _, row := range data {
			if idValue, ok := row[join.LeftField]; ok {
				if idStr, ok := idValue.(string); ok {
					if value, exists := results[idStr]; exists {
						row[join.As] = value
					}
				}
			}
		}
	}

	return data, nil
}

func (s *excelExportServiceImpl) writeExportHeaders(
	f *excelize.File,
	sheetName string,
	columns []ExcelExportField,
	headerStyle int,
) {
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheetName, cell, col.Header)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
}

func (s *excelExportServiceImpl) setExportColumnWidths(
	f *excelize.File,
	sheetName string,
	columns []ExcelExportField,
) {
	for i, col := range columns {
		// P2 fix: 用 excelize.ColumnNumberToName 替代自实现 coordinatesToColumnString
		colName, _ := excelize.ColumnNumberToName(i + 1)
		width := calculateColumnWidth(col.Header)
		_ = f.SetColWidth(sheetName, colName, colName, width)
	}
}

func (s *excelExportServiceImpl) writeExportDataRows(
	f *excelize.File,
	sheetName string,
	columns []ExcelExportField,
	data []map[string]interface{},
) {
	for i, row := range data {
		for j, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			value := s.formatExportCellValue(col, row)
			_ = f.SetCellValue(sheetName, cell, value)
		}
	}
}

func (s *excelExportServiceImpl) formatExportCellValue(
	col ExcelExportField,
	row map[string]interface{},
) string {
	var value interface{}

	if col.Join != nil {
		value = row[col.Join.As]
	} else if col.DBField != "" {
		value = row[col.DBField]
	} else {
		value = row[col.Field]
	}

	// GORM 在 Find(&[]map[string]interface{}) 路径下,对**子查询别名列**(例: 本次新增
	// 的 parent_path,由相关子查询生成)会把值包装为 *interface{}; 对普通表列(dept_name
	// / id 等)直接置为 string/int64. 这里统一解包,避免 fmt.Sprintf("%v", *interface{})
	// 把指针地址当字符串写到 xlsx.
	value = unwrapInterface(value)

	if value == nil {
		return ""
	}

	if col.Options != nil {
		if str, ok := col.Options[value]; ok {
			return str
		}
		if i, ok := value.(int64); ok {
			if str, ok := col.Options[int(i)]; ok {
				return str
			}
		}
		if f, ok := value.(float64); ok {
			if str, ok := col.Options[int(f)]; ok {
				return str
			}
		}
	}

	if col.Field == "createdAt" || col.Field == "updatedAt" {
		if t, ok := value.(time.Time); ok {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	return fmt.Sprintf("%v", value)
}

// unwrapInterface 递归解包 GORM 在 Find(&mapSlice) 下产生的指针包装.
//
// 递归深度 ≤ 2 (GORM 最多包装一次; 我们也只额外解一层防极端情况).
// 不解包非 interface{} 指针(如 *string),那是用户代码的真实数据.
func unwrapInterface(v interface{}) interface{} {
	for i := 0; i < 2; i++ {
		if v == nil {
			return nil
		}
		if ptr, ok := v.(*interface{}); ok {
			if ptr == nil {
				return nil
			}
			v = *ptr
			continue
		}
		return v
	}
	return v
}

func calculateColumnWidth(text string) float64 {
	width := 0
	for _, r := range text {
		if r < 128 {
			width++
		} else {
			width += 2
		}
	}
	if width < 10 {
		width = 10
	}
	if width > 50 {
		width = 50
	}
	return float64(width)
}
