package operations

import (
	"context"
	"fmt"
	"mime/multipart"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ExcelService Excel导入导出服务
type ExcelService struct {
	db                *gorm.DB
	pwdManager        *security.PasswordManager
	referenceResolver ReferenceResolver
	cacheInvalidator  *CacheInvalidator
	cache             system.CacheProvider
	geocoding         *GeocodingService
}

// NewExcelService 创建Excel服务
func NewExcelService(db *gorm.DB, pwdManager *security.PasswordManager, cache system.CacheProvider, geocoding *GeocodingService) *ExcelService {
	return &ExcelService{
		db:                db,
		pwdManager:        pwdManager,
		referenceResolver: NewReferenceResolver(db),
		cacheInvalidator:  NewCacheInvalidator(cache),
		cache:             cache,
		geocoding:         geocoding,
	}
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

	// 删除默认的 Sheet1
	if deleteErr := f.DeleteSheet("Sheet1"); deleteErr != nil {
		// 忽略删除失败（比如 Sheet1 不存在的情况）
		logger.Debugf("删除默认Sheet1失败: %v", deleteErr)
	}

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
	// 优先使用选项的第一个Label（用于枚举类型，如状态）
	if col.Options != nil {
		for _, v := range col.Options {
			return v // 返回Label（如"正常"、"停用"），而不是Value
		}
	}

	// 根据字段类型返回默认示例
	switch col.Field {
	case "name":
		return "示例A"
	case "deptName":
		return "研发部"
	case "buildingName":
		return "示例楼宇A"
	case "floorName":
		return "示例楼层1F"
	case "workstationName", "roomName":
		return "示例工位001"
	case "code", "buildingCode", "workstationCode", "deviceCode":
		return "B001"
	case "deptCode", "parentDeptCode":
		return "WTF"
	case "departmentCode":
		return "02R"
	case "departmentName":
		return "个客中心代理业务部车商业务销售二部"
	case "departmentGroupCode":
		return "420100"
	case "departmentGroupName":
		return "武汉中心支公司"
	case "cityCode":
		return "110000"
	case "cityName":
		return "北京市"
	case "address":
		return "朝阳区某某街道123号"
	case "longitude":
		return "116.407526"
	case "latitude":
		return "39.90403"
	case "orgName", "orgCode":
		return "总部"
	case "floorNo":
		return "1"
	case "level":
		return "具体楼宇"
	case "leader":
		return "张三"
	case "phone":
		return "13800138000"
	case "email":
		return "zhangsan@example.com"
	case "orderNum":
		return "1"
	// 机房相关
	case "buildingId":
		return "（请填写所属楼宇名称）"
	case "floorId":
		return "（请填写所在楼层名称）"
	case "roomId":
		return "（请填写所在机房名称）"
	// 机房设备相关
	case "vendor":
		return "戴尔"
	case "model":
		return "PowerEdge R740"
	case "serialNumber":
		return "CN123456789"
	case "positionU":
		return "10"
	case "positionDesc":
		return "第1列第5行"
	case "assetNumber":
		return "ZC2024001"
	// 专线相关
	case "workstationType", "infoPointType", "deviceType", "lineType":
		return "固定工位" // 第一选项的Label
	case "bandwidth":
		return "100M"
	case "isp":
		return "中国电信"
	case "sourceDeviceName", "destDeviceName":
		return "核心交换机A"
	case "sourcePort", "destPort":
		return "Gi0/1"
	case "sourceIpAddress", "destIpAddress":
		return "192.168.1.1"
	case "sourceSubnetMask", "destSubnetMask":
		return "255.255.255.0"
	case "vlan":
		return "100"
	case "carrierContactName":
		return "张三"
	case "carrierContactPhone":
		return "13800138000"
	case "contractNo":
		return "CT2024001"
	case "powerConsumption":
		return "500"
	case "monthlyFee":
		return "1000"
	case "purchaseDate", "warrantyDate":
		return "2024-01-01"
	case "status":
		return "正常"
	case "remark":
		return "备注信息"
	default:
		// 非必填字段可以不显示示例，必填字段显示通用示例
		if col.Required {
			return "示例"
		}
		return ""
	}
}

// ImportData 导入Excel数据（增强版：支持引用解析和批量Upsert）
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

	// 阶段1：收集所有行数据和引用解析请求
	var validRecords []map[string]interface{}
	var allRefRequests []ReferenceRequest

	for rowNum := config.StartRow; rowNum <= len(rows); rowNum++ {
		if rowNum-1 >= len(rows) {
			break
		}
		row := rows[rowNum-1]

		// 跳过空行
		if isEmptyRow(row) {
			continue
		}

		// 验证并解析数据（不立即保存）
		data, validationErrors := s.validateAndParseRow(ctx, row, rowNum, config, userID)
		if len(validationErrors) > 0 {
			result.Failed++
			result.Errors = append(result.Errors, validationErrors...)
			continue
		}

		// 收集引用解析请求
		rowRefs := s.extractReferenceRequests(data, config)
		if len(rowRefs) > 0 {
			allRefRequests = append(allRefRequests, rowRefs...)
		}

		validRecords = append(validRecords, data)
	}

	// 阶段2：批量解析所有引用（性能优化）
	if len(allRefRequests) > 0 {
		refResults, err := s.referenceResolver.ResolveBatch(ctx, allRefRequests)
		if err != nil {
			// 引用解析失败，记录错误但继续处理
			logger.Warnf("批量解析引用失败: %v", err)
		}

		// 应用引用解析结果到所有记录
		for _, data := range validRecords {
			s.applyReferenceResults(data, refResults, config)
		}
	}

	// 阶段2.5：批量解析地址获取经纬度（仅针对楼宇类型）
	if entityType == "building" && s.geocoding != nil {
		s.batchGeocodeBuildings(ctx, validRecords)
	}

	// 阶段2.55：预处理部门记录，移除重复的科室编码（针对部门类型）
	if entityType == "department" {
		// 对于科室编码和部门编码相同的记录，移除 deptCode 字段
		// 这样这些记录不会被当作科室创建，只创建部门
		for i := range validRecords {
			deptCode, _ := validRecords[i]["departmentCode"].(string)
			sectionCode, _ := validRecords[i]["deptCode"].(string)
			if deptCode != "" && sectionCode == deptCode {
				// 科室编码与部门编码相同，移除科室编码字段
				delete(validRecords[i], "deptCode")
			}
		}
	}

	// 阶段2.6：处理三层部门结构（仅针对部门类型）
	if entityType == "department" {
		validRecords, result = s.processThreeLevelDepartments(ctx, validRecords, result, config)
	}

	// 阶段3：批量Upsert到数据库
	upserter := NewBatchUpsert(s.db, config)
	if len(validRecords) > 0 {
		// 将记录转换为适合Upsert的格式（处理字段名映射）
		records := s.prepareRecordsForUpsert(validRecords, config)

		inserted, updated, err := upserter.Upsert(ctx, records)
		if err != nil {
			return nil, fmt.Errorf("批量保存数据失败: %w", err)
		}

		result.Success = inserted + updated

		// 阶段4：清理缓存
		if result.Success > 0 && len(config.CachePatterns) > 0 {
			if cacheErr := s.cacheInvalidator.InvalidateByEntityType(ctx, entityType, config.CachePatterns); cacheErr != nil {
				logger.Warnf("[%s] 清理缓存失败: %v", entityType, cacheErr)
			}
		}
	}

	return result, nil
}

// extractReferenceRequests 提取引用解析请求
func (s *ExcelService) extractReferenceRequests(
	data map[string]interface{},
	config ExcelConfig,
) []ReferenceRequest {
	var requests []ReferenceRequest

	for _, col := range config.Columns {
		if col.Reference != "" {
			if value, ok := data[col.Field].(string); ok && value != "" {
				requests = append(requests, ReferenceRequest{
					Reference: col.Reference,
					Value:     value,
				})
			}
		}
	}

	return requests
}

// applyReferenceResults 应用引用解析结果
// 将用户填写的名称/编码替换为数据库中的ID
func (s *ExcelService) applyReferenceResults(
	data map[string]interface{},
	refResults map[string]string,
	config ExcelConfig,
) {
	resolver := &referenceResolverImpl{}

	for _, col := range config.Columns {
		if col.Reference != "" {
			if value, ok := data[col.Field].(string); ok && value != "" {
				key := resolver.makeKey(col.Reference, value)
				if id, exists := refResults[key]; exists {
					// 将名称/编码替换为ID，并映射到正确的字段
					targetField := s.getTargetFieldForReference(col.Field)
					data[targetField] = id
					// 删除原始的名称字段（避免数据冗余）
					delete(data, col.Field)
				} else {
					logger.Debugf("引用解析失败: %s=%s", col.Reference, value)
				}
			}
		}
	}
}

// getTargetFieldForReference 根据引用配置获取目标字段名
// 例如：orgName -> orgId, buildingName -> buildingId
func (s *ExcelService) getTargetFieldForReference(currentField string) string {
	// 如果字段名以Name结尾，替换为Id
	if strings.HasSuffix(currentField, "Name") {
		return strings.TrimSuffix(currentField, "Name") + "Id"
	}
	// 如果字段名以Code结尾，同样处理
	if strings.HasSuffix(currentField, "Code") {
		return strings.TrimSuffix(currentField, "Code") + "Id"
	}
	// 默认返回原字段名
	return currentField
}

// prepareRecordsForUpsert 准备用于批量Upsert的记录
// 处理字段名映射和特殊字段
func (s *ExcelService) prepareRecordsForUpsert(
	records []map[string]interface{},
	config ExcelConfig,
) []map[string]interface{} {
	prepared := make([]map[string]interface{}, 0, len(records))

	for _, record := range records {
		preparedRecord := make(map[string]interface{})

		for _, col := range config.Columns {
			// 跳过没有DBField的字段（这些是临时处理字段，不应写入数据库）
			if col.DBField == "" {
				continue
			}

			if value, exists := record[col.Field]; exists {
				// 映射字段名（处理数据库列名和JSON标签的差异）
				dbFieldName := s.getDBFieldName(col)
				preparedRecord[dbFieldName] = value
			}
		}

		// 保留动态添加的特殊字段（parent_id, ancestors）
		// 这些字段在三级部门结构处理中被设置，必须保留
		if parentID, exists := record["parent_id"]; exists && parentID != nil {
			preparedRecord["parent_id"] = parentID
		}
		if ancestors, exists := record["ancestors"]; exists && ancestors != nil {
			preparedRecord["ancestors"] = ancestors
		}

		// 添加公共字段
		preparedRecord["id"] = generateUUID() // 生成UUID
		preparedRecord["updated_at"] = time.Now()
		preparedRecord["created_at"] = time.Now()

		prepared = append(prepared, preparedRecord)
	}

	return prepared
}

// generateUUID 生成UUID
func generateUUID() string {
	return uuid.New().String()
}

// getDBFieldName 获取数据库字段名（配置驱动）
// 优先使用配置的DBField，如果没有配置则使用字段名本身
func (s *ExcelService) getDBFieldName(col ExcelColumn) string {
	// 如果配置了DBField，优先使用配置
	if col.DBField != "" {
		return col.DBField
	}
	// 否则使用字段名本身
	return col.Field
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

		// 验证格式（Pattern）
		if col.Pattern != "" {
			matched, regexErr := regexp.MatchString(col.Pattern, value)
			if regexErr != nil {
				errors = append(errors, ImportError{
					Row:   rowNum,
					Field: col.Header,
					Value: value,
					Error: fmt.Sprintf("格式验证正则表达式错误: %v", regexErr),
				})
				continue
			}
			if !matched {
				errors = append(errors, ImportError{
					Row:   rowNum,
					Field: col.Header,
					Value: value,
					Error: "格式不正确",
				})
				continue
			}
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
	case "totalArea", "area", "monthlyFee":
		return strconv.ParseFloat(value, 64)
	}

	return value, nil
}

// validateUniqueness 验证唯一性
func (s *ExcelService) validateUniqueness(ctx context.Context, config ExcelConfig, data map[string]interface{}, rowNum int) []ImportError {
	errors := make([]ImportError, 0)

	for _, col := range config.Columns {
		if col.Unique {
			// 获取数据库字段名（重要：必须使用DBField而不是Field）
			dbField := s.getDBFieldName(col)
			code, ok := data[col.Field]
			if !ok {
				continue
			}

			var count int64
			err := s.db.WithContext(ctx).Table(config.TableName).
				Where(dbField+" = ? AND deleted_at IS NULL", code).
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

	// 删除默认的 Sheet1
	if deleteErr := f.DeleteSheet("Sheet1"); deleteErr != nil {
		logger.Debugf("删除默认Sheet1失败: %v", deleteErr)
	}

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

// geocodingTask 地理编码任务
type geocodingTask struct {
	address string
	index   int
}

// batchGeocodeBuildings 批量解析楼宇地址获取经纬度
// 仅解析有地址但无经纬度的记录
func (s *ExcelService) batchGeocodeBuildings(ctx context.Context, records []map[string]interface{}) {
	// 收集需要解析的任务
	tasks := s.collectGeocodingTasks(records)
	if len(tasks) == 0 {
		return
	}

	logger.Infof("开始批量解析 %d 个楼宇地址", len(tasks))

	// 并发执行地理编码
	successCount, failCount := s.executeGeocodingTasks(ctx, tasks, records)

	logger.Infof("批量地址解析完成: 成功 %d, 失败 %d", successCount, failCount)
}

// collectGeocodingTasks 收集需要地理编码的任务
func (s *ExcelService) collectGeocodingTasks(records []map[string]interface{}) []geocodingTask {
	var tasks []geocodingTask

	for i, record := range records {
		address, hasAddress := record["address"].(string)
		if !hasAddress || address == "" {
			continue
		}

		// 跳过已有经纬度的记录
		if s.hasCoordinates(record) {
			continue
		}

		tasks = append(tasks, geocodingTask{
			address: address,
			index:   i,
		})
	}

	return tasks
}

// hasCoordinates 检查记录是否已有经纬度
func (s *ExcelService) hasCoordinates(record map[string]interface{}) bool {
	_, hasLng := record["longitude"]
	_, hasLat := record["latitude"]
	return hasLng && hasLat
}

// executeGeocodingTasks 执行地理编码任务
func (s *ExcelService) executeGeocodingTasks(
	ctx context.Context,
	tasks []geocodingTask,
	records []map[string]interface{},
) (successCount, failCount int) {
	const maxConcurrency = 5
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, task := range tasks {
		sem <- struct{}{}
		wg.Add(1)

		go func(t geocodingTask) {
			defer wg.Done()
			defer func() { <-sem }()

			lng, lat, err := s.geocoding.Geocode(ctx, t.address)
			if err != nil {
				logger.Warnf("地址解析失败 [%d]: %s, 错误: %v", t.index, t.address, err)
				mu.Lock()
				failCount++
				mu.Unlock()
				return
			}

			mu.Lock()
			records[t.index]["longitude"] = lng
			records[t.index]["latitude"] = lat
			successCount++
			mu.Unlock()

			logger.Debugf("地址解析成功 [%d]: %s -> (%f, %f)", t.index, t.address, lng, lat)
		}(task)
	}

	wg.Wait()
	return successCount, failCount
}

// processThreeLevelDepartments 处理三层部门结构（部门组 -> 部门 -> 科室）
// 每条记录代表一个科室，需要确保其上级部门和部门组存在
// 返回过滤后的有效记录和更新后的结果对象
func (s *ExcelService) processThreeLevelDepartments(
	ctx context.Context,
	records []map[string]interface{},
	result *ImportResult,
	config ExcelConfig,
) ([]map[string]interface{}, *ImportResult) {
	// 收集所有唯一的部门组和部门
	type DeptGroup struct {
		Code string
		Name string
	}
	type Dept struct {
		Code      string
		Name      string
		GroupCode string
		GroupName string
	}

	groupMap := make(map[string]DeptGroup)  // groupCode -> DeptGroup
	deptMap := make(map[string]Dept)        // deptCode -> Dept
	sectionCodeSet := make(map[string]bool) // 收集所有科室编码
	validRecords := make([]map[string]interface{}, 0, len(records))
	invalidRowNumbers := make(map[int]bool) // 记录无效行号

	// 第一遍：收集所有部门组和部门信息，检测编码冲突
	for i, record := range records {
		rowNum := i + config.StartRow // 计算实际行号（从2开始，因为第1行是表头）
		groupCode, _ := record["departmentGroupCode"].(string)
		groupName, _ := record["departmentGroupName"].(string)
		deptCode, _ := record["departmentCode"].(string)
		deptName, _ := record["departmentName"].(string)
		sectionCode, _ := record["deptCode"].(string) // 科室编码

		if groupCode != "" && groupName != "" {
			groupMap[groupCode] = DeptGroup{Code: groupCode, Name: groupName}
		}

		// 收集部门信息
		if deptCode != "" && deptName != "" {
			deptMap[deptCode] = Dept{
				Code:      deptCode,
				Name:      deptName,
				GroupCode: groupCode,
				GroupName: groupName,
			}
		}

		// 如果没有科室编码（已在预处理阶段移除），跳过该行（部门已收集到 deptMap）
		if sectionCode == "" {
			continue
		}

		// 检测科室编码是否与部门组编码冲突
		if sectionCode != "" && groupCode != "" && sectionCode == groupCode {
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowNum,
				Field: "deptCode",
				Value: sectionCode,
				Error: fmt.Sprintf("科室编码(%s)不能与部门组编码(%s)相同", sectionCode, groupCode),
			})
			invalidRowNumbers[i] = true
			continue // 跳过该行
		}

		// 收集科室编码，用于检测科室间的重复
		if sectionCode != "" {
			if sectionCodeSet[sectionCode] {
				result.Failed++
				result.Errors = append(result.Errors, ImportError{
					Row:   rowNum,
					Field: "deptCode",
					Value: sectionCode,
					Error: "科室编码重复，请检查Excel数据",
				})
				invalidRowNumbers[i] = true
				continue // 跳过该行
			}
			sectionCodeSet[sectionCode] = true
		}

		// 如果该行没有错误，添加到有效记录列表
		if !invalidRowNumbers[i] {
			validRecords = append(validRecords, record)
		}
	}

	// 第二遍：确保部门组存在
	for _, group := range groupMap {
		if err := s.ensureDeptGroupExists(ctx, group.Code, group.Name); err != nil {
			// 部门组创建失败不应该阻止整个导入，只记录错误
			logger.Warnf("创建部门组失败 [%s]: %v", group.Code, err)
		}
	}

	// 第三遍：确保部门存在，并设置正确的parent_id
	for _, dept := range deptMap {
		// 获取部门组的ID
		groupID, err := s.findDeptIDByCode(ctx, dept.GroupCode)
		if err != nil {
			logger.Warnf("查找部门组失败 [%s]: %v", dept.GroupCode, err)
			continue // 跳过该部门
		}

		if err := s.ensureDeptExists(ctx, dept.Code, dept.Name, groupID); err != nil {
			logger.Warnf("创建部门失败 [%s]: %v", dept.Code, err)
		}
	}

	// 第四遍：为每条有效记录（科室）设置parent_id和ancestors
	for i, record := range validRecords {
		deptCode, _ := record["departmentCode"].(string)
		if deptCode == "" {
			continue
		}

		// 获取部门的ID和ancestors
		var parent struct {
			ID        string `gorm:"column:id"`
			Ancestors string `gorm:"column:ancestors"`
		}
		err := s.db.WithContext(ctx).
			Table("sys_dept").
			Select("id, ancestors").
			Where("dept_code = ? AND deleted_at IS NULL", deptCode).
			First(&parent).Error
		if err != nil {
			// 计算实际行号
			rowNum := i + config.StartRow
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   rowNum,
				Field: "departmentCode",
				Value: deptCode,
				Error: fmt.Sprintf("找不到上级部门: %s", deptCode),
			})
			// 标记为删除，稍后统一清理
			record["_skip"] = true
			continue
		}

		// 设置科室的parent_id
		record["parent_id"] = parent.ID

		// 设置科室的ancestors
		ancestors := parent.Ancestors
		if ancestors != "" {
			ancestors += ","
		}
		ancestors += parent.ID
		record["ancestors"] = ancestors

		// 删除中间字段（避免写入数据库）
		// 需要同时删除原始字段名和数据库字段名
		delete(record, "departmentCode")      // 原始字段名
		delete(record, "department_code")     // 数据库字段名
		delete(record, "departmentName")      // 原始字段名
		delete(record, "department_name")     // 数据库字段名
		delete(record, "departmentGroupCode") // 原始字段名
		delete(record, "group_code")          // 数据库字段名
		delete(record, "departmentGroupName") // 原始字段名
		delete(record, "group_name")          // 数据库字段名
	}

	// 清理被标记跳过的记录
	finalRecords := make([]map[string]interface{}, 0, len(validRecords))
	for _, record := range validRecords {
		if skip, exists := record["_skip"]; !exists || !skip.(bool) {
			delete(record, "_skip")
			finalRecords = append(finalRecords, record)
		}
	}

	return finalRecords, result
}

// findDeptIDByCode 根据部门编码查找部门ID
func (s *ExcelService) findDeptIDByCode(ctx context.Context, deptCode string) (string, error) {
	var result struct {
		ID string `gorm:"column:id"`
	}

	err := s.db.WithContext(ctx).
		Table("sys_dept").
		Select("id").
		Where("dept_code = ? AND deleted_at IS NULL", deptCode).
		First(&result).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("部门不存在: %s", deptCode)
		}
		return "", err
	}

	return result.ID, nil
}

// ensureDeptGroupExists 确保部门组存在，不存在则创建
func (s *ExcelService) ensureDeptGroupExists(ctx context.Context, code, name string) error {
	// 先查找是否存在（使用数据库字段名）
	var count int64
	err := s.db.WithContext(ctx).
		Table("sys_dept").
		Where("dept_code = ? AND deleted_at IS NULL", code).
		Count(&count).Error
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // 已存在
	}

	// 不存在则创建（parent_id为空，表示顶级部门）
	now := time.Now()
	dept := map[string]interface{}{
		"id":         generateUUID(),
		"dept_name":  name,
		"dept_code":  code,
		"parent_id":  nil, // 部门组是顶级部门
		"order_num":  0,
		"status":     0, // 正常
		"ancestors":  "",
		"created_at": now,
		"updated_at": now,
	}

	return s.db.WithContext(ctx).Table("sys_dept").Create(dept).Error
}

// ensureDeptExists 确保部门存在，不存在则创建
func (s *ExcelService) ensureDeptExists(ctx context.Context, code, name, parentID string) error {
	// 先查找是否存在（使用数据库字段名）
	var count int64
	err := s.db.WithContext(ctx).
		Table("sys_dept").
		Where("dept_code = ? AND deleted_at IS NULL", code).
		Count(&count).Error
	if err != nil {
		return err
	}

	if count > 0 {
		return nil // 已存在
	}

	// 不存在则创建
	// 获取parent的ancestors
	var parent struct {
		Ancestors string `gorm:"column:ancestors"`
	}
	err = s.db.WithContext(ctx).
		Table("sys_dept").
		Select("ancestors").
		Where("id = ? AND deleted_at IS NULL", parentID).
		First(&parent).Error
	if err != nil {
		return fmt.Errorf("查询上级部门失败: %w", err)
	}

	// 构建ancestors
	ancestors := parent.Ancestors
	if ancestors != "" {
		ancestors += ","
	}
	ancestors += parentID

	now := time.Now()
	dept := map[string]interface{}{
		"id":         generateUUID(),
		"dept_name":  name,
		"dept_code":  code,
		"parent_id":  parentID,
		"ancestors":  ancestors,
		"order_num":  0,
		"status":     0, // 正常
		"created_at": now,
		"updated_at": now,
	}

	return s.db.WithContext(ctx).Table("sys_dept").Create(dept).Error
}
