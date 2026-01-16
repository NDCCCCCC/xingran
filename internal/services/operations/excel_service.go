package operations

import (
	"context"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ExcelService Excel导入导出服务
type ExcelService struct {
	db         *gorm.DB
	pwdManager *security.PasswordManager
}

// NewExcelService 创建Excel服务
func NewExcelService(db *gorm.DB, pwdManager *security.PasswordManager) *ExcelService {
	return &ExcelService{db: db, pwdManager: pwdManager}
}

// ImportError 导入错误
type ImportError struct {
	Row   int    `json:"row"`
	Field string `json:"field"`
	Value string `json:"value"`
	Error string `json:"error"`
}

// ImportResult 导入结果
type ImportResult struct {
	Success int           `json:"success"`
	Failed  int           `json:"failed"`
	Errors  []ImportError `json:"errors"`
}

// GenerateTemplate 生成Excel模板
func (s *ExcelService) GenerateTemplate(entityType string) (*excelize.File, error) {
	config, exists := GetExcelConfig(entityType)
	if !exists {
		return nil, fmt.Errorf("不支持的实体类型: %s", entityType)
	}

	f := excelize.NewFile()
	sheetName := config.SheetName
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	headerStyle, err := createHeaderStyle(f)
	if err != nil {
		return nil, fmt.Errorf("创建表头样式失败: %w", err)
	}

	s.writeHeaders(f, sheetName, config.Columns, headerStyle)
	s.writeExampleData(f, sheetName, config.Columns)
	s.setColumnWidths(f, sheetName, config.Columns)

	f.SetPanes(sheetName, &excelize.Panes{
		Freeze: true,
		XSplit: 1,
		YSplit: 1,
	})

	return f, nil
}

// getExampleValue 获取示例值
func (s *ExcelService) getExampleValue(col ExcelColumn) string {
	if col.Required {
		if col.Options != nil {
			// 返回第一个选项
			for k := range col.Options {
				if str, ok := k.(string); ok {
					return str
				} else if num, ok := k.(int); ok {
					return strconv.Itoa(num)
				}
			}
		}
		switch col.Field {
		case "name", "code", "workstationCode", "deviceCode", "lineCode":
			return "示例001"
		case "status":
			return "0"
		default:
			return "示例"
		}
	}
	return ""
}

// ImportData 导入Excel数据
func (s *ExcelService) ImportData(ctx context.Context, entityType string, file *multipart.FileHeader, userID string) (*ImportResult, error) {
	config, exists := GetExcelConfig(entityType)
	if !exists {
		return nil, fmt.Errorf("不支持的实体类型: %s", entityType)
	}

	// 打开Excel文件
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	f, err := excelize.OpenReader(src)
	if err != nil {
		return nil, fmt.Errorf("解析Excel文件失败: %w", err)
	}
	defer f.Close()

	// 获取所有行数据
	rows, err := f.GetRows(config.SheetName)
	if err != nil {
		return nil, fmt.Errorf("读取数据失败: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel文件没有数据")
	}

	result := &ImportResult{
		Errors: make([]ImportError, 0),
	}

	// 解析每一行数据
	for rowNum := config.StartRow; rowNum <= len(rows); rowNum++ {
		if rowNum-1 >= len(rows) {
			break
		}
		row := rows[rowNum-1]

		// 跳过空行
		if isEmptyRow(row) {
			continue
		}

		// 验证并解析数据
		data, validationErrors := s.validateAndParseRow(ctx, row, rowNum, config, userID)
		if len(validationErrors) > 0 {
			result.Failed++
			result.Errors = append(result.Errors, validationErrors...)
			continue
		}

		// 保存到数据库
		if err := s.saveData(ctx, entityType, data); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowNum,
				Field: "all",
				Error: err.Error(),
			})
			continue
		}

		result.Success++
	}

	return result, nil
}

// isEmptyRow 检查是否为空行
func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// validateAndParseRow 验证并解析行数据
func (s *ExcelService) validateAndParseRow(ctx context.Context, row []string, rowNum int, config ExcelConfig, _ string) (map[string]interface{}, []ImportError) {
	data := make(map[string]interface{})
	errors := make([]ImportError, 0)

	for i, col := range config.Columns {
		if i >= len(row) {
			if col.Required {
				errors = append(errors, ImportError{
					Row:   rowNum,
					Field: col.Header,
					Error: "必填字段不能为空",
				})
			}
			continue
		}

		value := strings.TrimSpace(row[i])
		if value == "" {
			if col.Required {
				errors = append(errors, ImportError{
					Row:   rowNum,
					Field: col.Header,
					Error: "必填字段不能为空",
				})
			}
			continue
		}

		// 验证长度
		if col.MaxLength > 0 && len(value) > col.MaxLength {
			errors = append(errors, ImportError{
				Row:   rowNum,
				Field: col.Header,
				Value: value,
				Error: fmt.Sprintf("长度超过最大限制 %d", col.MaxLength),
			})
			continue
		}

		// 解析字段值
		parsedValue, err := s.parseFieldValue(col, value)
		if err != nil {
			errors = append(errors, ImportError{
				Row:   rowNum,
				Field: col.Header,
				Value: value,
				Error: err.Error(),
			})
			continue
		}

		data[col.Field] = parsedValue
	}

	// 验证唯一性（如果在配置中指定）
	if len(errors) == 0 {
		uniqueErrors := s.validateUniqueness(ctx, config, data, rowNum)
		errors = append(errors, uniqueErrors...)
	}

	return data, errors
}

// parseFieldValue 解析字段值
func (s *ExcelService) parseFieldValue(col ExcelColumn, value string) (interface{}, error) {
	if col.Options != nil {
		// 枚举值映射
		for k, v := range col.Options {
			if v == value {
				return k, nil
			}
		}
		// 尝试直接使用值
		return value, nil
	}

	// 数值类型
	switch col.Field {
	case "totalFloors", "capacity", "rackCount", "positionU", "heightU", "powerConsumption":
		return strconv.Atoi(value)
	case "totalArea", "area", "bandwidth", "monthlyFee":
		return strconv.ParseFloat(value, 64)
	}

	return value, nil
}

// validateUniqueness 验证唯一性
func (s *ExcelService) validateUniqueness(ctx context.Context, config ExcelConfig, data map[string]interface{}, rowNum int) []ImportError {
	errors := make([]ImportError, 0)

	for _, col := range config.Columns {
		if col.Unique {
			codeField := col.Field
			code, ok := data[codeField]
			if !ok {
				continue
			}

			var count int64
			err := s.db.WithContext(ctx).Table(config.TableName).
				Where(codeField+" = ? AND deleted_at IS NULL", code).
				Count(&count).Error
			if err != nil {
				continue
			}

			if count > 0 {
				errors = append(errors, ImportError{
					Row:   rowNum,
					Field: col.Header,
					Value: fmt.Sprintf("%v", code),
					Error: "该值已存在",
				})
			}
		}
	}

	return errors
}

// saveData 保存数据到数据库
func (s *ExcelService) saveData(ctx context.Context, entityType string, data map[string]interface{}) error {
	switch entityType {
	case "user":
		return s.saveUser(ctx, data)
	// 其他实体类型的处理可以在这里添加
	// case "building":
	//     return s.saveBuilding(ctx, data)
	default:
		return fmt.Errorf("不支持的实体类型: %s", entityType)
	}
}

// saveUser 保存用户数据
func (s *ExcelService) saveUser(ctx context.Context, data map[string]interface{}) error {
	// 提取用户名
	username, ok := data["username"].(string)
	if !ok || username == "" {
		return fmt.Errorf("用户名不能为空")
	}

	// 检查用户名是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Table("sys_user").Where("username = ?", username).Count(&count).Error; err != nil {
		return fmt.Errorf("检查用户名失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("用户名 %s 已存在", username)
	}

	// 创建用户（使用事务）
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 使用默认密码 "123456" 进行哈希（使用SM3）
		hashedPassword, err := s.pwdManager.HashPassword("123456")
		if err != nil {
			return fmt.Errorf("密码加密失败: %w", err)
		}

		// 提取字段值
		nickname, _ := data["nickname"].(string)
		employeeNo, _ := data["employeeNo"].(string)
		email, _ := data["email"].(string)
		phone, _ := data["phone"].(string)
		remark, _ := data["remark"].(string)

		// 处理性别（默认为保密）
		gender := models.GenderSecret
		if g, ok := data["gender"].(int); ok {
			gender = models.Gender(g)
		}

		// 处理状态（默认为正常）
		status := models.UserStatusEnabled
		if st, ok := data["status"].(int); ok {
			status = models.UserStatus(st)
		}

		// 处理部门编码
		var deptID *string
		if deptCode, ok := data["deptCode"].(string); ok && deptCode != "" {
			var dept models.Department
			if err := tx.Where("dept_code = ?", deptCode).First(&dept).Error; err != nil {
				return fmt.Errorf("部门编码 %s 不存在", deptCode)
			}
			deptID = &dept.ID
		}

		// 创建用户记录
		user := &models.User{
			Username:   username,
			Password:   hashedPassword,
			Nickname:   &nickname,
			EmployeeNo: &employeeNo,
			Email:      &email,
			Phone:      &phone,
			Gender:     gender,
			Status:     status,
			DeptID:     deptID,
			Remark:     remark,
		}
		if err := tx.Create(user).Error; err != nil {
			return fmt.Errorf("创建用户失败: %w", err)
		}

		return nil
	})
}

// ExportData 导出数据到Excel
func (s *ExcelService) ExportData(ctx context.Context, entityType string, params map[string]interface{}) (*excelize.File, error) {
	config, exists := GetExcelConfig(entityType)
	if !exists {
		return nil, fmt.Errorf("不支持的实体类型: %s", entityType)
	}

	f := excelize.NewFile()
	sheetName := config.SheetName
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, fmt.Errorf("创建工作表失败: %w", err)
	}
	f.SetActiveSheet(index)

	headerStyle, err := createHeaderStyle(f)
	if err != nil {
		return nil, fmt.Errorf("创建表头样式失败: %w", err)
	}

	s.writeHeaders(f, sheetName, config.Columns, headerStyle)
	s.setColumnWidths(f, sheetName, config.Columns)

	data, err := s.queryData(ctx, config, params)
	if err != nil {
		return nil, fmt.Errorf("查询数据失败: %w", err)
	}

	s.writeDataRows(f, sheetName, config.Columns, data)

	f.SetPanes(sheetName, &excelize.Panes{
		Freeze: true,
		XSplit: 1,
		YSplit: 1,
	})

	return f, nil
}

// queryData 查询数据
func (s *ExcelService) queryData(_ context.Context, config ExcelConfig, params map[string]interface{}) ([]map[string]interface{}, error) {
	// 构建查询
	query := s.db.Table(config.TableName).Where("deleted_at IS NULL")

	// 应用过滤条件
	if name, ok := params["name"]; ok && name != "" {
		query = query.Where("name LIKE ?", "%"+name.(string)+"%")
	}
	if code, ok := params["code"]; ok && code != "" {
		query = query.Where("code LIKE ?", "%"+code.(string)+"%")
	}
	if status, ok := params["status"]; ok && status != nil {
		query = query.Where("status = ?", status)
	}

	// 执行查询
	var data []map[string]interface{}
	if err := query.Find(&data).Error; err != nil {
		return nil, err
	}

	return data, nil
}

// formatCellValue 格式化单元格值
func (s *ExcelService) formatCellValue(col ExcelColumn, row map[string]interface{}) string {
	value := row[col.Field]
	if value == nil {
		return ""
	}

	// 枚举值转换
	if col.Options != nil {
		if str, ok := col.Options[value]; ok {
			return str
		}
	}

	// 日期格式
	if col.Field == "createdAt" || col.Field == "updatedAt" {
		if t, ok := value.(time.Time); ok {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	return fmt.Sprintf("%v", value)
}

// createHeaderStyle 创建表头样式
func createHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
		},
	})
}

// writeHeaders 写入表头
func (s *ExcelService) writeHeaders(f *excelize.File, sheetName string, columns []ExcelColumn, headerStyle int) {
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, col.Header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
}

// writeExampleData 写入示例数据
func (s *ExcelService) writeExampleData(f *excelize.File, sheetName string, columns []ExcelColumn) {
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		exampleValue := s.getExampleValue(col)
		f.SetCellValue(sheetName, cell, exampleValue)
	}
}

// setColumnWidths 设置列宽
func (s *ExcelService) setColumnWidths(f *excelize.File, sheetName string, columns []ExcelColumn) {
	for i, col := range columns {
		colWidth := 15.0
		if col.Field == "name" || col.Field == "code" || col.Field == "description" || col.Field == "remark" {
			colWidth = 20.0
		}
		startCol, _ := excelize.CoordinatesToCellName(i+1, 1)
		endCol, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetColWidth(sheetName, startCol, endCol, colWidth)
	}
}

// writeDataRows 写入数据行
func (s *ExcelService) writeDataRows(f *excelize.File, sheetName string, columns []ExcelColumn, data []map[string]interface{}) {
	for i, row := range data {
		for j, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			value := s.formatCellValue(col, row)
			f.SetCellValue(sheetName, cell, value)
		}
	}
}
