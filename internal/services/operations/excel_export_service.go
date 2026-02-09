package operations

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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

	f.SetPanes(sheetName, &excelize.Panes{
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
		rows, err := s.db.WithContext(ctx).
			Table(join.Table).
			Select(join.RightField+", "+join.SelectField).
			Where(join.RightField+" IN ?", ids).
			Where("deleted_at IS NULL").
			Rows()

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
		f.SetCellValue(sheetName, cell, col.Header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
}

func (s *excelExportServiceImpl) setExportColumnWidths(
	f *excelize.File,
	sheetName string,
	columns []ExcelExportField,
) {
	for i, col := range columns {
		colName, _ := coordinatesToColumnString(i + 1)
		width := calculateColumnWidth(col.Header)
		f.SetColWidth(sheetName, colName, colName, width)
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
			f.SetCellValue(sheetName, cell, value)
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

func coordinatesToColumnString(colNum int) (string, error) {
	if colNum < 1 {
		return "", fmt.Errorf("invalid column number: %d", colNum)
	}

	colName := ""
	for colNum > 0 {
		colNum--
		colName = string(rune('A'+colNum%26)) + colName
		colNum /= 26
	}
	return colName, nil
}
