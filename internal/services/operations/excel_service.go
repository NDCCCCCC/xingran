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
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ExcelService struct {
	db                *gorm.DB
	pwdManager        *security.PasswordManager
	referenceResolver ReferenceResolver
	cacheInvalidator  *CacheInvalidator
	cache             system.CacheProvider
	geocoding         *GeocodingService
	// deviceService 可选注入,用于 Excel 工位导入 post-import hook 调用 SetPrimaryAndSaveBySerial。
	// nil 时跳过(测试场景或构造时未注入),不阻断主流程。
	deviceService WorkstationDeviceService

	// P1-C6 修复: validateUniqueness N+1 防御 — 按 tableName 缓存每个唯一列
	// 的"已存在值集合",首次调用时单次 IN 查询一次性加载,后续 per-row 调用
	// 仅做内存 set 查询。把 N(rows) × M(columns) 次 Count 降为 M 次 Pluck。
	uniqueValueMu     sync.Mutex
	uniqueValueCache  map[string]map[string]map[string]struct{} // tableName -> colField -> set(value)
	uniqueValueLoaded map[string]bool                           // tableName -> 是否已加载
}

func NewExcelService(db *gorm.DB, pwdManager *security.PasswordManager, cache system.CacheProvider, geocoding *GeocodingService) *ExcelService {
	return &ExcelService{
		db:                db,
		pwdManager:        pwdManager,
		referenceResolver: NewReferenceResolver(db),
		cacheInvalidator:  NewCacheInvalidator(cache),
		cache:             cache,
		geocoding:         geocoding,
		uniqueValueCache:  make(map[string]map[string]map[string]struct{}),
		uniqueValueLoaded: make(map[string]bool),
	}
}

// WithDeviceService 注入设备服务(post-import 主设备同步)。
// 单独 setter 模式避免构造函数参数膨胀,且兼容旧调用。
func (s *ExcelService) WithDeviceService(svc WorkstationDeviceService) *ExcelService {
	s.deviceService = svc
	return s
}

type ImportError struct {
	Row   int    `json:"row"`
	Field string `json:"field"`
	Value string `json:"value"`
	Error string `json:"error"`
}

type ImportResult struct {
	Inserted int           `json:"inserted"`
	Updated  int           `json:"updated"`
	Failed   int           `json:"failed"`
	Errors   []ImportError `json:"errors"`
	// AffectedKeys 收集本次导入真正写入（insert+update）记录的 UpsertKey 值
	// （如用户的 username）。供调用方做导入后处理（如触发 AD 域控同步）。
	// 仅在 ExcelConfig 配置了 UpsertKey 列时填充，其余场景为空。
	AffectedKeys []string `json:"affectedKeys,omitempty"`
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

	if deleteErr := f.DeleteSheet("Sheet1"); deleteErr != nil {
		logger.Debugf("删除默认Sheet1失败: %v", deleteErr)
	}

	headerStyle, err := createHeaderStyle(f)
	if err != nil {
		return nil, fmt.Errorf("创建表头样式失败: %w", err)
	}

	// 写入说明行（如果有）
	s.writeInstructions(f, sheetName, config.Columns, config.Instructions)

	// 计算表头行号（如果有说明行，表头从第2行开始，否则从第1行开始）
	headerRow := 1
	if len(config.Instructions) > 0 {
		headerRow = len(config.Instructions) + 1
	}

	s.writeHeaders(f, sheetName, config.Columns, headerStyle, headerRow)
	s.writeExampleData(f, sheetName, config.Columns, headerRow)
	s.setColumnWidths(f, sheetName, config.Columns)

	// 冻结窗格：如果有说明行，冻结表头行；否则冻结第一行
	freezeRow := headerRow
	if len(config.Instructions) > 0 {
		_ = f.SetPanes(sheetName, &excelize.Panes{
			Freeze: true,
			XSplit: 1,
			YSplit: freezeRow,
		})
	} else {
		_ = f.SetPanes(sheetName, &excelize.Panes{
			Freeze: true,
			XSplit: 1,
			YSplit: 1,
		})
	}

	return f, nil
}

func (s *ExcelService) getExampleValue(col ExcelColumn) string {
	if col.Options != nil {
		for _, v := range col.Options {
			return v
		}
	}

	switch col.Field {
	case "name":
		return "示例A"
	case "deptName":
		return "研发部"
	case "buildingName":
		return "示例楼宇A"
	case "floorName":
		return "示例楼层1F"
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
	case "buildingId":
		return "（请填写所属楼宇名称）"
	case "floorId":
		return "（请填写所在楼层名称）"
	case "roomId":
		return "（请填写所在机房名称）"
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
	case "workstationType", "deviceType":
		return "固定工位" // 第一选项的Label
	case "infoPointType":
		return "网络信息点" // 第一选项的Label
	case "lineType":
		return "internet" // 互联网专线
	case "isp":
		return "telecom" // 电信
	case "sourceDeviceName", "destDeviceName", "deviceName":
		return "核心交换机A"
	case "sourcePort", "destPort", "portName":
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
		if col.Required {
			return "示例"
		}
		return ""
	}
}

func (s *ExcelService) ImportData(ctx context.Context, entityType string, file *multipart.FileHeader, userID string) (*ImportResult, error) {
	config, exists := GetExcelConfig(entityType)
	if !exists {
		return nil, fmt.Errorf("不支持的实体类型: %s", entityType)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer src.Close()

	// P1 fix: 给 excelize.OpenReader 加 UnzipSizeLimit (200 MB) 和
	// UnzipXMLSizeLimit (100 MB) 选项,防御恶意 ZIP bomb / 超大 OOXML 文件
	// 在内存中解压触发 OOM。配合上层 handler 的 maxExcelUploadSize=50MB
	// 形成多层防护。
	const (
		unzipSizeLimit    = 200 * 1024 * 1024 // 解压后总大小上限
		unzipXMLSizeLimit = 100 * 1024 * 1024 // 单个 XML 大小上限
	)
	f, err := excelize.OpenReader(src,
		excelize.Options{
			UnzipSizeLimit:    unzipSizeLimit,
			UnzipXMLSizeLimit: unzipXMLSizeLimit,
		})
	if err != nil {
		return nil, fmt.Errorf("解析Excel文件失败: %w", err)
	}
	defer f.Close()

	// Try to get the configured sheet name first, then fall back to flexible lookup
	rows, err := f.GetRows(config.SheetName)
	sheetName := config.SheetName
	if err != nil {
		// Get all available sheets
		sheets := f.GetSheetList()
		logger.Warnf("Configured sheet '%s' not found, available sheets: %v", config.SheetName, sheets)

		// Try to find a matching sheet (fuzzy matching)
		found := false
		for _, sheet := range sheets {
			// Try exact match first (case-insensitive)
			if strings.EqualFold(sheet, config.SheetName) {
				sheetName = sheet
				found = true
				logger.Infof("Found matching sheet (case-insensitive): '%s'", sheetName)
				break
			}
			// Try partial match (contains)
			if strings.Contains(strings.ToLower(sheet), strings.ToLower(config.SheetName)) ||
				strings.Contains(strings.ToLower(config.SheetName), strings.ToLower(sheet)) {
				sheetName = sheet
				found = true
				logger.Infof("Found matching sheet (partial match): '%s' (configured: '%s')", sheetName, config.SheetName)
				break
			}
		}

		// Fall back to first sheet if no match found
		if !found && len(sheets) > 0 {
			sheetName = sheets[0]
			logger.Warnf("No matching sheet found, using first available sheet: '%s'", sheetName)
		}

		rows, err = f.GetRows(sheetName)
		if err != nil {
			return nil, fmt.Errorf("读取数据失败: 无法找到工作表 '%s' (可用: %v): %w", config.SheetName, sheets, err)
		}
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("Excel文件没有数据")
	}

	result := &ImportResult{
		Errors: make([]ImportError, 0),
	}

	var validRecords []map[string]any
	var allRefRequests []ReferenceRequest

	// 按表头文字匹配列(支持任意列序), 匹配不全则回退到 config.Columns 的位置匹配。
	// 仅当所有 Required 列都被表头匹配到时才启用, 避免误匹配破坏已验证模块(building/user 等)。
	effectiveColumns := config.Columns
	if config.HasHeader && len(config.Instructions) == 0 && config.StartRow >= 2 && len(rows) >= config.StartRow-1 {
		headerCells := rows[config.StartRow-2]
		matched, matchedRequired := resolveColumnsByHeader(headerCells, config)
		requiredTotal := 0
		for _, col := range config.Columns {
			if col.Required {
				requiredTotal++
			}
		}
		if requiredTotal > 0 && matchedRequired == requiredTotal {
			effectiveColumns = matched
			logger.Infof("[%s] 启用按表头列名匹配: %d/%d 个必填列已匹配", entityType, matchedRequired, requiredTotal)
		} else if matchedRequired > 0 {
			logger.Warnf("[%s] 表头匹配不全(%d/%d 必填列), 回退按位置匹配", entityType, matchedRequired, requiredTotal)
		}
	}

	// F-20: 循环条件改为 `rowNum-1 < len(rows)`,使越界保护与循环不变量等价,
	// 删除冗余的运行时 break 检查(原代码 `if rowNum-1 >= len(rows) { break }` 是死代码,
	// 但若误改循环上界为 `<=len(rows)+1` 之类会立刻引发 index out of range panic)。
	for rowNum := config.StartRow; rowNum-1 < len(rows); rowNum++ {
		row := rows[rowNum-1]

		if isEmptyRow(row) {
			continue
		}

		data, validationErrors := s.validateAndParseRow(ctx, row, rowNum, config, effectiveColumns, userID)
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

	if len(allRefRequests) > 0 {
		// 第一阶段：解析独立的引用（无依赖的引用）
		independentRefs, dependentRefs := s.splitReferencesByDependency(allRefRequests, config)

		refResults, err := s.referenceResolver.ResolveBatch(ctx, independentRefs)
		if err != nil {
			logger.Warnf("批量解析独立引用失败: %v", err)
		}

		// 应用独立引用的解析结果
		for _, data := range validRecords {
			s.applyReferenceResults(data, refResults, config)
		}

		// 第二阶段：批量解析依赖引用（按依赖条件分组处理）
		if len(dependentRefs) > 0 {
			// 按依赖条件分组批量解析
			dependentResults := s.resolveDependentReferencesBatch(ctx, validRecords, dependentRefs, config)
			logger.Infof("依赖引用解析完成，解析到 %d 条结果", len(dependentResults))
			// 批量应用依赖引用的解析结果
			for _, data := range validRecords {
				s.applyDependentReferenceResults(data, dependentResults, config)
			}
			// 调试：打印前3条记录的解析状态
			for i := 0; i < len(validRecords) && i < 3; i++ {
				data := validRecords[i]
				logger.Infof("记录 %d: floorName=%v, floor_id=%v, buildingName=%v, building_id=%v",
					i, data["floorName"], data["floor_id"], data["buildingName"], data["building_id"])
			}
		}

		// 验证必填的引用字段是否成功解析
		validRecords = s.validateReferenceFields(validRecords, config, result)
	}

	if entityType == "building" && s.geocoding != nil {
		s.batchGeocodeBuildings(ctx, validRecords)
	}

	if entityType == "department" {
		for i := range validRecords {
			deptCode, _ := validRecords[i]["departmentCode"].(string)
			sectionCode, _ := validRecords[i]["deptCode"].(string)
			if deptCode != "" && sectionCode == deptCode {
				delete(validRecords[i], "deptCode")
			}
		}
	}

	// P2-A8: Wrap department-processing and upsert in a single transaction so
	// partial failures roll back the entire import (depts + records).
	// processThreeLevelDepartments and Upsert share the same tx.
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if entityType == "department" {
			validRecords, result = s.processThreeLevelDepartments(ctx, tx, validRecords, result, config)
		}

		if len(validRecords) > 0 {
			upserter := NewBatchUpsert(tx, config)
			records := s.prepareRecordsForUpsert(validRecords, config)

			// 用户导入特例：sys_user.password/salt 为 NOT NULL，新用户必须设密码。
			// 参考 UserSyncService 的 AD 用户同步策略：默认密码 "123456" + InitFlag=true。
			// 仅对新增用户设值（已存在用户不改密码）。
			if entityType == "user" {
				if err := s.populateNewUserPasswords(ctx, tx, records); err != nil {
					return err
				}
			}

			inserted, updated, err := upserter.Upsert(ctx, records)
			if err != nil {
				return fmt.Errorf("批量保存数据失败: %w", err)
			}

			result.Inserted = inserted
			result.Updated = updated

			// 收集受影响的 UpsertKey 值（如 username），供导入后处理使用
			// （如用户导入后触发 AD 域控同步）。仅在配置了 UpsertKey 列时填充。
			upsertKeyField := ""
			for _, col := range config.Columns {
				if col.UpsertKey {
					upsertKeyField = col.Field
					break
				}
			}
			if upsertKeyField != "" {
				for _, rec := range validRecords {
					if v, ok := rec[upsertKeyField].(string); ok && v != "" {
						result.AffectedKeys = append(result.AffectedKeys, v)
					}
				}
			}

			// 用户导入：为本次导入的所有用户中无角色的分配默认角色（role_key='common' 普通用户）。
			// 覆盖新用户 + 已存在但无角色的用户（幂等：已有角色的不覆盖）；
			// 必须在 Upsert 后（用户已入库可查 id）；失败不阻断导入。
			// 修复：原先只处理 populateNewUserPasswords 识别的新用户，导致重新导入已存在
			// （走 Updated 路径）的无角色用户被遗漏。
			if entityType == "user" {
				importedUsernames := make([]string, 0, len(records))
				for _, r := range records {
					if u, ok := r["username"].(string); ok && u != "" {
						importedUsernames = append(importedUsernames, u)
					}
				}
				if len(importedUsernames) > 0 {
					if err := s.assignDefaultRolesToNewUsers(ctx, tx, importedUsernames); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// P2-A8: Cache invalidation runs post-commit — cache writes are not
	// transactional, so they must run only after the DB transaction succeeds.
	if (result.Inserted+result.Updated) > 0 && len(config.CachePatterns) > 0 {
		if cacheErr := s.cacheInvalidator.InvalidateByEntityType(ctx, entityType, config.CachePatterns); cacheErr != nil {
			logger.Warnf("[%s] 清理缓存失败: %v", entityType, cacheErr)
		}
	}

	// post-import hook: 工位导入若含 deviceSerial 列,对每行触发主设备同步。
	// 失败计入 result.Failed/Errors,但不阻断整体导入流程(内部 hook 调用,无 operlog)。
	if entityType == "workstation" {
		s.postImportWorkstationPrimaryDevice(ctx, validRecords, result)
	}

	return result, nil
}

// postImportWorkstationPrimaryDevice 遍历工位导入记录,对含非空 deviceSerial 的行
// 调用 SetPrimaryAndSaveBySerial 同步主设备。失败计入 result.Failed/Errors,不阻断循环。
//
// 设计要点:
//   - 不写 operlog: 这是 service 层内部 hook,CLAUDE.md operlog 约定仅 HTTP handler 入口必记
//   - 不阻断: 设备同步失败不影响工位主体数据已成功 upsert 的事实
//   - 通过 (floor_id, workstation_name) 反查 workstation_id,因为 validRecords 来自
//     prepareRecordsForUpsert 之前的数据,不含 DB 生成的主键 id
func (s *ExcelService) postImportWorkstationPrimaryDevice(ctx context.Context, validRecords []map[string]any, result *ImportResult) {
	if s.deviceService == nil {
		// 未注入时静默跳过 — 与现有 batchGeocodeBuildings 中 s.geocoding != nil 守卫一致。
		return
	}

	for i, rec := range validRecords {
		serial, _ := rec["deviceSerial"].(string)
		if serial == "" {
			continue
		}

		floorID, _ := rec["floor_id"].(string)
		name, _ := rec["name"].(string)
		if floorID == "" || name == "" {
			logger.Warnf("[WorkstationImport] 跳过主设备同步,缺少 floor_id 或 name - serial: %s", serial)
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   configStartRowOrDefault(result, i),
				Field: "主设备序列号",
				Value: serial,
				Error: "缺少工位定位信息(floor_id 或 workstation_name 为空),无法同步主设备",
			})
			continue
		}

		// 通过 (floor_id, workstation_name) 反查 workstation_id。
		var workstationID string
		if err := s.db.WithContext(ctx).
			Table("sys_workstation").
			Select("id").
			Where("floor_id = ? AND workstation_name = ? AND deleted_at IS NULL", floorID, name).
			Scan(&workstationID).Error; err != nil {
			logger.Warnf("[WorkstationImport] 查找工位ID失败 floorID=%s name=%s: %v", floorID, name, err)
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   configStartRowOrDefault(result, i),
				Field: "主设备序列号",
				Value: serial,
				Error: fmt.Sprintf("查找工位失败: %v", err),
			})
			continue
		}
		if workstationID == "" {
			logger.Warnf("[WorkstationImport] 未找到工位 floorID=%s name=%s (可能 upsert 失败)", floorID, name)
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   configStartRowOrDefault(result, i),
				Field: "主设备序列号",
				Value: serial,
				Error: "工位未找到,跳过主设备同步(请确认工位本体已成功 upsert)",
			})
			continue
		}

		if err := s.deviceService.SetPrimaryAndSaveBySerial(ctx, workstationID, serial, nil); err != nil {
			logger.Warnf("[WorkstationImport] 主设备同步失败 workstationID=%s serial=%s: %v", workstationID, serial, err)
			result.Failed++
			result.Errors = append(result.Errors, ImportError{
				Row:   configStartRowOrDefault(result, i),
				Field: "主设备序列号",
				Value: serial,
				Error: fmt.Sprintf("主设备同步失败: %v", err),
			})
			continue
		}

		logger.Infof("[WorkstationImport] 主设备同步成功 workstationID=%s serial=%s", workstationID, serial)
	}
}

// configStartRowOrDefault 推算 Excel 行号: 配置的 StartRow + 索引 i + 1(表头偏移)。
// postImportWorkstationPrimaryDevice 不持有 ExcelConfig,这里用经验默认 StartRow=2
// + i + 1(行号 1-based)。result 已使用 Inserted/Updated/Failed 计数。
func configStartRowOrDefault(_ *ImportResult, i int) int {
	return 2 + i + 1
}

func (s *ExcelService) extractReferenceRequests(
	data map[string]any,
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

func (s *ExcelService) applyReferenceResults(
	data map[string]any,
	refResults map[string]string,
	config ExcelConfig,
) {
	resolver := &referenceResolverImpl{}

	for _, col := range config.Columns {
		if col.Reference != "" {
			if value, ok := data[col.Field].(string); ok && value != "" {
				key := resolver.makeKey(col.Reference, value)
				if id, exists := refResults[key]; exists {
					// 将名称/编码替换为ID
					// 优先使用配置的 DBField，否则使用自动转换的字段名
					targetField := s.getTargetFieldForReference(col)
					data[targetField] = id

					// InfoPoint特殊处理：保留名称字段用于显示
					// InfoPoint模型有冗余字段（deviceName, portName）用于前端显示
					// 而Service层没有JOIN设备和端口表，所以需要在导入时保存名称
					if config.TableName == "ops_info_points" {
						switch col.Field {
						case "deviceName":
							data["device_name"] = value
						case "portName":
							data["port_name"] = value
						}
						// 保留原始字段供前端使用
					} else {
						// 其他实体：删除原始的名称字段（避免数据冗余）
						delete(data, col.Field)
					}
				} else {
					// 引用解析失败
					logger.Debugf("引用解析失败: %s=%s", col.Reference, value)
					// InfoPoint特殊处理：即使引用解析失败，也保存名称字段
					// 这样用户能看到原始的Excel值，便于排查问题
					if config.TableName == "ops_info_points" {
						switch col.Field {
						case "deviceName":
							data["device_name"] = value
						case "portName":
							data["port_name"] = value
						}
					}
				}
			}
		}
	}
}

// getTargetFieldForReference 获取引用字段解析后的目标字段名
// 如果配置了 DBField，直接使用 DBField
// 否则根据字段名后缀自动转换（如 buildingName -> buildingId）
func (s *ExcelService) getTargetFieldForReference(col ExcelColumn) string {
	// 优先使用配置的 DBField（对于引用字段，DBField 通常指向最终的数据库字段，如 building_id）
	if col.DBField != "" {
		return col.DBField
	}

	// 如果没有配置 DBField，使用自动转换规则
	field := col.Field
	if suffix, ok := strings.CutSuffix(field, "Name"); ok {
		return suffix + "Id"
	}
	if suffix, ok := strings.CutSuffix(field, "Code"); ok {
		return suffix + "Id"
	}
	return field
}

// splitReferencesByDependency 将引用请求分为独立引用和依赖引用
func (s *ExcelService) splitReferencesByDependency(
	refs []ReferenceRequest,
	config ExcelConfig,
) (independent, dependent []ReferenceRequest) {
	for _, ref := range refs {
		isDependent := false
		for _, col := range config.Columns {
			if col.Reference == ref.Reference && col.DependsOn != "" {
				isDependent = true
				break
			}
		}
		if isDependent {
			dependent = append(dependent, ref)
		} else {
			independent = append(independent, ref)
		}
	}
	return independent, dependent
}

// applyDependentReferenceResults 应用依赖引用的解析结果
func (s *ExcelService) applyDependentReferenceResults(
	data map[string]any,
	refResults map[string]string,
	config ExcelConfig,
) {
	resolver := &referenceResolverImpl{}

	for _, col := range config.Columns {
		if col.Reference != "" && col.DependsOn != "" {
			if value, ok := data[col.Field].(string); ok && value != "" {
				key := resolver.makeKey(col.Reference, value)
				if id, exists := refResults[key]; exists {
					// 将名称/编码替换为ID
					targetField := s.getTargetFieldForReference(col)
					data[targetField] = id
					// 删除原始的名称字段（避免数据冗余）
					delete(data, col.Field)
				} else {
					logger.Debugf("依赖引用解析失败: %s=%s (依赖 %s)", col.Reference, value, col.DependsOn)
				}
			}
		}
	}
}

// resolveDependentReferencesBatch 批量解析依赖引用
// 将记录按依赖条件分组，每组条件只执行一次批量查询
// 例如：对于工位导入，按 building_id 分组，每个楼宇只查询一次所有楼层
func (s *ExcelService) resolveDependentReferencesBatch(
	ctx context.Context,
	records []map[string]any,
	dependentRefs []ReferenceRequest,
	config ExcelConfig,
) map[string]string {
	result := make(map[string]string)

	if len(dependentRefs) == 0 || len(records) == 0 {
		return result
	}

	// 找到所有依赖字段的配置（可能有多个字段有依赖关系）
	depCols := make([]ExcelColumn, 0)
	for _, c := range config.Columns {
		if c.DependsOn != "" {
			depCols = append(depCols, c)
		}
	}

	if len(depCols) == 0 {
		return result
	}

	logger.Infof("开始批量解析依赖引用，共有 %d 个依赖字段配置，%d 条记录", len(depCols), len(records))

	// 对每个依赖字段进行处理（如 floorName 依赖 buildingName）
	for depColIdx, depCol := range depCols {
		logger.Infof("处理依赖字段 %d/%d: %s (依赖: %s, 引用: %s)", depColIdx+1, len(depCols), depCol.Field, depCol.DependsOn, depCol.Reference)

		// 按依赖ID值分组记录（如按 building_id 分组）
		groups := s.groupRecordsByDependencyID(records, depCol, config)
		logger.Infof("  按 %s 分组，共 %d 个组", s.getTargetFieldForReference(depCol), len(groups))

		// 为每组（每个楼宇）执行一次批量查询
		for depID, groupRecords := range groups {
			// 提取该组中所有记录的待解析值（该楼宇的所有楼层名称，去重）
			valueToRecords := s.extractDependentValues(groupRecords, depCol, config)
			logger.Infof("  组 %s: 共 %d 条记录，%d 个唯一值", depID, len(groupRecords), len(valueToRecords))

			if len(valueToRecords) == 0 {
				continue
			}

			// 构建查询条件（如 {"building_id": "xxx-uuid"}）
			conditions := map[string]string{
				s.getTargetFieldForReferenceByName(depCol.DependsOn, config): depID,
			}

			// 执行批量查询：一次查询该楼宇的所有楼层
			idMap, err := s.referenceResolver.ResolveBatchWithCondition(ctx, depCol.Reference, valueToRecords, conditions)
			if err != nil {
				logger.Warnf("批量解析依赖引用失败 [%s, 条件: %s=%s]: %v", depCol.Reference, s.getTargetFieldForReferenceByName(depCol.DependsOn, config), depID, err)
				continue
			}
			logger.Infof("  批量查询成功，返回 %d 条结果", len(idMap))

			// 将解析结果应用到该组所有记录
			successCount := 0
			for _, record := range groupRecords {
				if value, exists := record[depCol.Field]; exists && value != nil {
					if valueStr, ok := value.(string); ok && valueStr != "" {
						// 使用批量查询的结果
						if id, found := idMap[valueStr]; found {
							key := (&referenceResolverImpl{}).makeKey(depCol.Reference, valueStr)
							result[key] = id
							// 同时更新记录中的目标字段
							targetField := s.getTargetFieldForReference(depCol)
							record[targetField] = id
							successCount++
						} else {
							logger.Warnf("依赖引用解析失败: %s=%s (条件: %s=%s)", depCol.Reference, valueStr, s.getTargetFieldForReferenceByName(depCol.DependsOn, config), depID)
						}
					}
				}
			}
			logger.Infof("  组 %s: 成功应用 %d 条结果", depID, successCount)
		}
	}

	logger.Infof("依赖引用批量解析完成，总共解析 %d 条结果", len(result))
	return result
}

// groupRecordsByDependencyID 按依赖ID值分组记录
// 返回：{ "building-id-1": [record1, record2, ...], "building-id-2": [...] }
// 对于 floorName (依赖 buildingName)，按 building_id 分组
func (s *ExcelService) groupRecordsByDependencyID(
	records []map[string]any,
	depCol ExcelColumn,
	config ExcelConfig,
) map[string][]map[string]any {
	groups := make(map[string][]map[string]any)

	// 获取被依赖的字段配置（如 floorName 依赖 buildingName，则获取 buildingName 的配置）
	dependsOnCol := s.getColumnByName(depCol.DependsOn, config)
	if dependsOnCol.Field == "" {
		logger.Warnf("找不到被依赖的字段配置: %s", depCol.DependsOn)
		return groups
	}

	// 获取被依赖字段的目标字段（如 buildingName -> building_id）
	depTargetField := s.getTargetFieldForReference(dependsOnCol)

	logger.Infof("按 %s (从 %s 解析) 分组记录", depTargetField, dependsOnCol.Field)

	for _, record := range records {
		if depID, exists := record[depTargetField]; exists && depID != nil {
			if depIDStr, ok := depID.(string); ok && depIDStr != "" {
				groups[depIDStr] = append(groups[depIDStr], record)
			}
		}
	}

	return groups
}

// extractDependentValues 提取一组记录中所有待解析的依赖引用值
// 对于指定的依赖列，提取所有唯一值（如所有楼层名称）
func (s *ExcelService) extractDependentValues(
	records []map[string]any,
	depCol ExcelColumn,
	_ ExcelConfig,
) []string {
	valueSet := make(map[string]bool)

	for _, record := range records {
		if value, ok := record[depCol.Field].(string); ok && value != "" {
			valueSet[value] = true
		}
	}

	values := make([]string, 0, len(valueSet))
	for v := range valueSet {
		values = append(values, v)
	}
	return values
}

// getTargetFieldForReferenceByName 根据字段名获取其解析后的目标字段名
func (s *ExcelService) getTargetFieldForReferenceByName(fieldName string, config ExcelConfig) string {
	for _, col := range config.Columns {
		if col.Field == fieldName {
			return s.getTargetFieldForReference(col)
		}
	}
	return ""
}

// getColumnByName 根据字段名获取列配置
func (s *ExcelService) getColumnByName(fieldName string, config ExcelConfig) ExcelColumn {
	for _, col := range config.Columns {
		if col.Field == fieldName {
			return col
		}
	}
	return ExcelColumn{}
}

func (s *ExcelService) prepareRecordsForUpsert(
	records []map[string]any,
	config ExcelConfig,
) []map[string]any {
	prepared := make([]map[string]any, 0, len(records))

	for _, record := range records {
		preparedRecord := make(map[string]any)

		for _, col := range config.Columns {
			if col.DBField == "" {
				continue
			}

			var value any
			var exists bool

			// 对于引用字段，只使用解析后的值（不使用原始值）
			if col.Reference != "" {
				// 引用字段：优先使用解析后的字段（如 floor_id）
				resolvedField := s.getTargetFieldForReference(col)
				value, exists = record[resolvedField]
				// 如果解析后的字段不存在，说明引用解析失败，不使用原始值
			} else {
				// 非引用字段：直接使用原始值
				value, exists = record[col.Field]
			}

			if exists {
				dbFieldName := s.getDBFieldName(col)
				// 同 DBField 多列时保留先写非空值（first-non-empty-wins）。
				// 解决 P0-4: machineIP + domainIP 同写 machine_ip 时真实 IP 被覆盖
				// (e.g. D9L90K2 的 10.62.101.143 被 PR.intra.cpic.com.cn 覆盖)。
				// 仅影响 config 中存在同 DBField 多列的场景, 其他表无此情况故零回归。
				if existing, already := preparedRecord[dbFieldName]; already {
					if !isEmptyValue(existing) {
						continue
					}
				}
				preparedRecord[dbFieldName] = value
			}
		}
		// InfoPoint特殊处理：包含动态添加的冗余字段
		// InfoPoint模型有冗余字段（device_name, port_name）用于前端显示
		// 这些字段是在applyReferenceResults中动态添加的，不在config.Columns中定义
		if config.TableName == "ops_info_points" {
			if deviceName, exists := record["device_name"]; exists && deviceName != nil {
				preparedRecord["device_name"] = deviceName
			}
			if portName, exists := record["port_name"]; exists && portName != nil {
				preparedRecord["port_name"] = portName
			}
		}

		if parentID, exists := record["parent_id"]; exists && parentID != nil {
			preparedRecord["parent_id"] = parentID
		}
		if ancestors, exists := record["ancestors"]; exists && ancestors != nil {
			preparedRecord["ancestors"] = ancestors
		}

		preparedRecord["id"] = generateUUID()
		preparedRecord["updated_at"] = time.Now()
		preparedRecord["created_at"] = time.Now()
		preparedRecord["deleted_at"] = nil // 显式设置为未删除状态

		// ✨ 联动 user_id ↔ status(占用/空闲),Maintain 状态保留
		// Excel 中的 status 列将被忽略,以 user_id 为准
		// 仅对 sys_workstation 生效(其他表 status 字段语义不同)
		if config.TableName == "sys_workstation" {
			if userID, ok := preparedRecord["user_id"]; ok && userID != nil && fmt.Sprintf("%v", userID) != "" {
				preparedRecord["status"] = int(models.WorkstationStatusOccupied)
			} else {
				preparedRecord["status"] = int(models.WorkstationStatusAvailable)
			}
		}

		prepared = append(prepared, preparedRecord)
	}

	return prepared
}

// populateNewUserPasswords 为新增的 sys_user 记录填充默认密码与初始标志。
// 仅在 entityType=="user" 时由 ImportData 调用。
//
// sys_user.password 为 NOT NULL，Excel 无密码列 → 新用户 INSERT 会违反约束。
// 策略对齐 UserSyncService 的 AD 用户同步（user_sync_service.go:89-100）：
//   - 默认密码 "123456"，经 pwdManager.HashPassword 哈希（salt 编码进 hash）
//   - InitFlag=true（首次登录强制改密）、PwdExpireDays=90
//
// 仅对新增用户（username 在 sys_user 不存在）设值；已存在用户不改密码，
// 交由 partialUpsert 仅更新 Excel 提供的字段。
// populateNewUserPasswords 为新增的 sys_user 记录填充默认密码与初始标志。
//
// sys_user.password 为 NOT NULL，Excel 无密码列 → 新用户 INSERT 会违反约束。
// 策略对齐 UserSyncService 的 AD 用户同步：
//   - 默认密码 "123456"，经 pwdManager.HashPassword 哈希（salt 编码进 hash）
//   - InitFlag=true（首次登录强制改密）、PwdExpireDays=90
//
// 仅对新增用户（username 在 sys_user 不存在）设值；已存在用户不改密码，
// 交由 partialUpsert 仅更新 Excel 提供的字段。
//
// 角色分配（assignDefaultRolesToNewUsers）由 ImportData 对**所有导入用户**触发
// （含已存在但无角色的），不依赖此处的"新用户"判定。
func (s *ExcelService) populateNewUserPasswords(ctx context.Context, tx *gorm.DB, records []map[string]any) error {
	if len(records) == 0 {
		return nil
	}
	// pwdManager 未注入时跳过（测试场景）——生产环境由 NewExcelService 保证非 nil
	if s.pwdManager == nil {
		logger.Warnf("[USER-IMPORT] pwdManager 未注入，跳过新用户默认密码填充")
		return nil
	}

	// 收集所有 username（prepareRecordsForUpsert 已用 DBField="username" 写入）
	usernames := make([]string, 0, len(records))
	for _, r := range records {
		if u, ok := r["username"].(string); ok && u != "" {
			usernames = append(usernames, u)
		}
	}
	if len(usernames) == 0 {
		return nil
	}

	// 查询已存在的 username（Unscoped 含软删除记录，恢复时也跳过改密）
	var existing []string
	if err := tx.WithContext(ctx).Table("sys_user").
		Unscoped().Where("username IN ?", usernames).
		Pluck("username", &existing).Error; err != nil {
		return fmt.Errorf("查询现有用户失败: %w", err)
	}
	existingSet := make(map[string]bool, len(existing))
	for _, u := range existing {
		existingSet[u] = true
	}

	// 所有新用户同一默认密码 "123456"，只需哈希一次。
	// HashPassword 内部 pbkdf2SM3（iterations×SM3），单次约 10-50ms；
	// 若每用户单独哈希，1000+ 新用户将耗时 30+ 秒，把 HTTP 请求卡死。
	// 密码本就相同，共用一份 hash 不降低安全性（弱密码 + InitFlag 强制首登改密）。
	const defaultImportPassword = "123456"
	hashed, err := s.pwdManager.HashPassword(defaultImportPassword)
	if err != nil {
		return fmt.Errorf("生成默认密码失败: %w", err)
	}

	for _, r := range records {
		u, _ := r["username"].(string)
		if u == "" || existingSet[u] {
			continue // 已存在用户不改密码
		}
		r["password"] = hashed
		r["salt"] = "" // Salt 列 NOT NULL；HashPassword 的 salt 已编码进 hash 字符串，独立 Salt 列留空字符串占位（与 UserSyncService 一致）
		r["init_flag"] = true
		r["pwd_expire_days"] = 90
	}
	return nil
}

// assignDefaultRolesToNewUsers 为新导入的用户分配默认角色（role_key='user' 普通用户）。
//
// 业务规则：仅给"尚无任何角色"的用户分配默认角色，已有角色的用户不会被覆盖
// （幂等）。这样导入新用户会拿到默认"普通用户"角色，重新导入已存在但无角色
// 的用户也会补齐，但已分配角色的用户保持原样。
//
// 必须在 Upsert 之后调用（此时新用户已入库，可按 username 查到 id）。
// 批量 INSERT（单次 SQL），适合大批量导入（1000+ 用户）。
//
// 与 createDefaultRole（database.go）保持一致：role_key='user'，角色名'普通用户'。
func (s *ExcelService) assignDefaultRolesToNewUsers(ctx context.Context, tx *gorm.DB, usernames []string) error {
	if len(usernames) == 0 {
		return nil
	}

	// 1. 查普通用户角色（role_key='user'，与 createDefaultRole 一致）。
	// 找不到时仅 WARN 跳过，不阻断导入（环境未配置该角色时静默降级）。
	var defaultRole models.Role
	if err := tx.WithContext(ctx).Where("role_key = ?", "user").First(&defaultRole).Error; err != nil {
		logger.Warnf("[USER-IMPORT] 未找到普通用户角色(role_key=user)，跳过默认角色分配: %v", err)
		return nil
	}

	// 2. 查新用户中尚无角色的（幂等：避免重复插入 + 不覆盖已分配角色）
	type userRow struct {
		ID string
	}
	var users []userRow
	if err := tx.WithContext(ctx).
		Table("sys_user").
		Select("id").
		Where("username IN ?", usernames).
		Where("id NOT IN (SELECT user_id FROM sys_user_role)").
		Find(&users).Error; err != nil {
		return fmt.Errorf("查询无角色的新用户失败: %w", err)
	}
	if len(users) == 0 {
		return nil
	}

	// 3. 批量 INSERT sys_user_role（单次 SQL，性能优）
	valueStrings := make([]string, 0, len(users))
	valueArgs := make([]interface{}, 0, len(users)*2)
	for _, u := range users {
		valueStrings = append(valueStrings, "(?, ?)")
		valueArgs = append(valueArgs, u.ID, defaultRole.ID)
	}
	sqlStr := "INSERT INTO sys_user_role (user_id, role_id) VALUES " + strings.Join(valueStrings, ", ")
	if err := tx.WithContext(ctx).Exec(sqlStr, valueArgs...).Error; err != nil {
		return fmt.Errorf("批量分配默认角色失败: %w", err)
	}
	logger.Infof("[USER-IMPORT] 为 %d 个新用户分配普通用户角色(role_key=user)", len(users))
	return nil
}

func generateUUID() string {
	return uuid.New().String()
}

func (s *ExcelService) getDBFieldName(col ExcelColumn) string {
	if col.DBField != "" {
		return col.DBField
	}
	return col.Field
}

// isEmptyValue 判断 Excel 单元格值是否为空（nil / 空字符串 / 全空白）。
// 用于 first-non-empty-wins 合并逻辑: preparedRecord 中已存在非空值时跳过覆盖。
func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val) == ""
	case *string:
		return val == nil || strings.TrimSpace(*val) == ""
	default:
		return false
	}
}

// validateReferenceFields 验证必填的引用字段是否成功解析
func (s *ExcelService) validateReferenceFields(
	records []map[string]any,
	config ExcelConfig,
	result *ImportResult,
) []map[string]any {
	validated := make([]map[string]any, 0, len(records))

	logger.Infof("开始验证必填引用字段，共 %d 条记录", len(records))

	for i, data := range records {
		valid := true
		for _, col := range config.Columns {
			// 只验证必填的引用字段
			if !col.Required || col.Reference == "" {
				continue
			}

			// 获取目标字段名（如 buildingId）
			targetField := s.getTargetFieldForReference(col)

			// 检查目标字段是否有值
			if val, exists := data[targetField]; !exists || val == nil || val == "" {
				// 必填引用字段解析失败
				logger.Infof("记录 %d 验证失败: %s (原始值: %v, 目标字段: %s=%v)", i, col.Field, data[col.Field], targetField, val)
				result.Failed++
				result.Errors = append(result.Errors, ImportError{
					Row:   config.StartRow + i + 1, // 实际 Excel 行号（i 从 0 开始，StartRow 从 2 开始）
					Field: col.Header,
					Value: fmt.Sprintf("%v", data[col.Field]),
					Error: fmt.Sprintf("引用的 %s 不存在", col.Header),
				})
				valid = false
				break
			}
		}

		if valid {
			validated = append(validated, data)
		}
	}

	logger.Infof("引用字段验证完成，成功 %d 条，失败 %d 条", len(validated), result.Failed)
	return validated
}

func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// headerBracketRegexp 匹配表头中的中/英文括号及内容(如英文标识 SECTION_OFFICE_CODE),
// 用于归一化时去除,使"科室编码(SECTION_OFFICE_CODE)"与裸"科室编码"可匹配。
var headerBracketRegexp = regexp.MustCompile(`[（(][^)）]*[)）]`)

// normalizeHeader 归一化表头文字以便跨格式匹配:
//   - 去除括号及内容(英文标识)
//   - trim 首尾空格、转小写
//   - "代码"统一为"编码"(业务侧常用"部门代码/科室代码", config 用"部门编码/科室编码")
//
// 例: "科室编码(SECTION_OFFICE_CODE)" → "科室编码"; 用户"科室代码" → "科室编码"。
func normalizeHeader(s string) string {
	s = strings.TrimSpace(s)
	s = headerBracketRegexp.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "代码", "编码")
	return s
}

// resolveColumnsByHeader 按 Excel 表头文字匹配 config 列, 返回按 Excel 列序排列的
// effectiveColumns(长度 == 表头列数)。未匹配到的列位置为零值 ExcelColumn{Field:""},
// 解析时跳过。同一字段只匹配第一个出现的列, 避免重复。
// matchedRequired 返回成功匹配的 Required 字段数, 调用方据此判断是否启用按表头匹配。
func resolveColumnsByHeader(headerCells []string, config ExcelConfig) (effective []ExcelColumn, matchedRequired int) {
	normalizedConfig := make(map[string]ExcelColumn, len(config.Columns))
	for _, col := range config.Columns {
		normalizedConfig[normalizeHeader(col.Header)] = col
	}

	effective = make([]ExcelColumn, len(headerCells))
	matchedFields := make(map[string]bool)
	for i, cell := range headerCells {
		if strings.TrimSpace(cell) == "" {
			continue
		}
		col, ok := normalizedConfig[normalizeHeader(cell)]
		if !ok {
			continue
		}
		if matchedFields[col.Field] {
			continue
		}
		effective[i] = col
		matchedFields[col.Field] = true
		if col.Required {
			matchedRequired++
		}
	}
	return effective, matchedRequired
}

func (s *ExcelService) validateAndParseRow(ctx context.Context, row []string, rowNum int, config ExcelConfig, columns []ExcelColumn, _ string) (map[string]any, []ImportError) {
	data := make(map[string]any)
	errors := make([]ImportError, 0)

	for i, col := range columns {
		if col.Field == "" {
			continue // 该 Excel 列无对应字段(表头未匹配或占位列), 跳过
		}
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

		if col.MaxLength > 0 && len(value) > col.MaxLength {
			errors = append(errors, ImportError{
				Row:   rowNum,
				Field: col.Header,
				Value: value,
				Error: fmt.Sprintf("长度超过最大限制 %d", col.MaxLength),
			})
			continue
		}

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

	// 应用默认值：当字段值为空且配置了Default时
	for _, col := range config.Columns {
		if _, exists := data[col.Field]; !exists && col.Default != nil {
			data[col.Field] = col.Default
		}
	}

	if len(errors) == 0 {
		uniqueErrors := s.validateUniqueness(ctx, config, data, rowNum)
		errors = append(errors, uniqueErrors...)
	}

	return data, errors
}

func (s *ExcelService) parseFieldValue(col ExcelColumn, value string) (any, error) {
	if col.Options != nil {
		for k, v := range col.Options {
			if v == value {
				return k, nil
			}
		}
		return value, nil
	}

	switch col.Field {
	case "totalFloors", "capacity", "rackCount", "positionU", "heightU", "powerConsumption":
		return strconv.Atoi(value)
	case "totalArea", "area", "monthlyFee":
		return strconv.ParseFloat(value, 64)
	}

	return value, nil
}

func (s *ExcelService) validateUniqueness(ctx context.Context, config ExcelConfig, data map[string]any, rowNum int) []ImportError {
	errors := make([]ImportError, 0)

	// P1-C6 修复: 从 N(rows)×M(cols) 次 Count 降为 M(cols) 次单 IN/Pluck 查询。
	// 首次访问某 tableName 时,把所有 Unique 列的非空值集合一次性加载到内存缓存,
	// 后续 per-row 调用直接走 set 查找。
	if err := s.ensureUniqueValueCacheLoaded(ctx, config); err != nil {
		// 加载缓存失败时回退到 silent skip,与原实现行为一致(原代码遇到 err 也 continue)
		logger.Warnf("加载唯一值缓存失败,跳过本行唯一性校验 [%s]: %v", config.TableName, err)
		return errors
	}

	s.uniqueValueMu.Lock()
	tableCache, ok := s.uniqueValueCache[config.TableName]
	s.uniqueValueMu.Unlock()
	if !ok {
		return errors
	}

	for _, col := range config.Columns {
		if col.UpsertKey {
			continue
		}

		if col.Unique {
			code, exists := data[col.Field]
			if !exists {
				continue
			}

			key := fmt.Sprintf("%v", code)
			if _, hit := tableCache[col.Field][key]; hit {
				errors = append(errors, ImportError{
					Row:   rowNum,
					Field: col.Header,
					Value: key,
					Error: "该值已存在",
				})
			}
		}
	}

	return errors
}

// ensureUniqueValueCacheLoaded 懒加载 table 级别"已存在值"缓存。
// 对每个 col.Unique=true 且 col.UpsertKey=false 的列,执行一次
// `SELECT col FROM table WHERE col IS NOT NULL AND deleted_at IS NULL` 加载
// 当前已存在的所有值。
func (s *ExcelService) ensureUniqueValueCacheLoaded(ctx context.Context, config ExcelConfig) error {
	s.uniqueValueMu.Lock()
	if s.uniqueValueLoaded[config.TableName] {
		s.uniqueValueMu.Unlock()
		return nil
	}
	s.uniqueValueMu.Unlock()

	tableCache := make(map[string]map[string]struct{})
	for _, col := range config.Columns {
		if col.UpsertKey || !col.Unique {
			continue
		}
		dbField := s.getDBFieldName(col)

		var values []string
		err := s.db.WithContext(ctx).Table(config.TableName).
			Select(dbField).
			Where(dbField+" IS NOT NULL AND deleted_at IS NULL").
			Pluck(dbField, &values).Error
		if err != nil {
			return fmt.Errorf("加载唯一列 %s 现有值失败: %w", dbField, err)
		}

		set := make(map[string]struct{}, len(values))
		for _, v := range values {
			set[v] = struct{}{}
		}
		tableCache[col.Field] = set
	}

	s.uniqueValueMu.Lock()
	s.uniqueValueCache[config.TableName] = tableCache
	s.uniqueValueLoaded[config.TableName] = true
	s.uniqueValueMu.Unlock()

	return nil
}

func (s *ExcelService) ExportData(ctx context.Context, entityType string, params map[string]any) (*excelize.File, error) {
	if _, exists := GetExportConfig(entityType); exists {
		exportService := NewExcelExportService(s.db)
		f, err := exportService.ExportData(ctx, entityType, params)
		if err != nil {
			return nil, err
		}
		// 工位管理 (Phase 35 增强): 追加 3 类设备 sheet
		// - AD设备: 域控 (ComputerName/OS/MAC/Serial/LastLogon)
		// - 资产设备: 资产管理 (DeviceName/Model/Type/IP/ResponsibleUser)
		// - 物理链路设备: MAC→port→infoPoint→workstation (MAC/Port/InfoPoint/LastSeen/Confidence)
		// 失败不阻断主流程, 仅记录 warning。
		if entityType == "workstation" {
			impl, _ := exportService.(*excelExportServiceImpl)
			if impl != nil {
				if appendErr := s.appendWorkstationDeviceSheets(ctx, f, impl, params); appendErr != nil {
					logger.Warnf("[ExcelExport] 工位导出追加设备 sheet 失败: %v", appendErr)
				}
			}
		}
		return f, nil
	}

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

	if deleteErr := f.DeleteSheet("Sheet1"); deleteErr != nil {
		logger.Debugf("删除默认Sheet1失败: %v", deleteErr)
	}

	headerStyle, err := createHeaderStyle(f)
	if err != nil {
		return nil, fmt.Errorf("创建表头样式失败: %w", err)
	}

	s.writeHeaders(f, sheetName, config.Columns, headerStyle, 1)
	s.setColumnWidths(f, sheetName, config.Columns)

	data, err := s.queryData(ctx, config, params)
	if err != nil {
		return nil, fmt.Errorf("查询数据失败: %w", err)
	}

	s.writeDataRows(f, sheetName, config.Columns, data)

	_ = f.SetPanes(sheetName, &excelize.Panes{
		Freeze: true,
		XSplit: 1,
		YSplit: 1,
	})

	return f, nil
}

func (s *ExcelService) queryData(_ context.Context, config ExcelConfig, params map[string]any) ([]map[string]any, error) {
	query := s.db.Table(config.TableName).Where("deleted_at IS NULL")

	nameField := "name"
	codeField := "code"
	if config.TableName == "sys_dept" {
		nameField = "dept_name"
		codeField = "dept_code"
	}

	if name, ok := params["name"]; ok && name != "" {
		query = query.Where(nameField+" LIKE ?", "%"+name.(string)+"%")
	}
	if code, ok := params["code"]; ok && code != "" {
		query = query.Where(codeField+" LIKE ?", "%"+code.(string)+"%")
	}
	if status, ok := params["status"]; ok && status != nil {
		query = query.Where("status = ?", status)
	}

	var data []map[string]any
	if err := query.Find(&data).Error; err != nil {
		return nil, err
	}

	return data, nil
}

func (s *ExcelService) formatCellValue(col ExcelColumn, row map[string]any) string {
	fieldKey := col.Field
	if col.DBField != "" {
		fieldKey = col.DBField
	}

	value := row[fieldKey]
	if value == nil {
		return ""
	}

	if col.Options != nil {
		if str, ok := col.Options[value]; ok {
			return str
		}
	}

	if col.Field == "createdAt" || col.Field == "updatedAt" {
		if t, ok := value.(time.Time); ok {
			return t.Format("2006-01-02 15:04:05")
		}
	}

	return fmt.Sprintf("%v", value)
}

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

func (s *ExcelService) writeHeaders(f *excelize.File, sheetName string, columns []ExcelColumn, headerStyle int, headerRow int) {
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, headerRow)
		_ = f.SetCellValue(sheetName, cell, col.Header)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
}

func (s *ExcelService) writeExampleData(f *excelize.File, sheetName string, columns []ExcelColumn, headerRow int) {
	dataRow := headerRow + 1
	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, dataRow)
		exampleValue := s.getExampleValue(col)
		_ = f.SetCellValue(sheetName, cell, exampleValue)
	}
}

// writeInstructions 写入模板顶部的说明文字
func (s *ExcelService) writeInstructions(f *excelize.File, sheetName string, columns []ExcelColumn, instructions []string) {
	if len(instructions) == 0 {
		return
	}

	// 创建说明行样式
	instructionStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#C00000", Size: 10},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	})
	if err != nil {
		logger.Debugf("创建说明样式失败: %v", err)
		return
	}

	// 设置第一列宽度以容纳说明文字
	_ = f.SetColWidth(sheetName, "A", "A", 60)

	// 写入说明文字（跨所有列合并）
	lastCol, _ := excelize.CoordinatesToCellName(len(columns), 1)

	for i, text := range instructions {
		row := i + 1
		if text == "" {
			continue // 跳过空行
		}

		startCell, _ := excelize.CoordinatesToCellName(1, row)
		// 合并单元格以显示完整说明
		if err := f.MergeCell(sheetName, startCell, lastCol); err != nil {
			logger.Debugf("合并单元格失败: %v", err)
			continue
		}

		_ = f.SetCellValue(sheetName, startCell, text)
		_ = f.SetCellStyle(sheetName, startCell, startCell, instructionStyle)
	}

	// 设置说明行高度
	for i := range instructions {
		_ = f.SetRowHeight(sheetName, i+1, 25)
	}
}

func (s *ExcelService) setColumnWidths(f *excelize.File, sheetName string, columns []ExcelColumn) {
	for i, col := range columns {
		colWidth := 15.0
		if col.Field == "name" || col.Field == "code" || col.Field == "description" || col.Field == "remark" {
			colWidth = 20.0
		}
		startCol, _ := excelize.CoordinatesToCellName(i+1, 1)
		endCol, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetColWidth(sheetName, startCol, endCol, colWidth)
	}
}

func (s *ExcelService) writeDataRows(f *excelize.File, sheetName string, columns []ExcelColumn, data []map[string]any) {
	for i, row := range data {
		for j, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			value := s.formatCellValue(col, row)
			_ = f.SetCellValue(sheetName, cell, value)
		}
	}
}

type geocodingTask struct {
	address string
	index   int
}

func (s *ExcelService) batchGeocodeBuildings(ctx context.Context, records []map[string]any) {
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

func (s *ExcelService) collectGeocodingTasks(records []map[string]any) []geocodingTask {
	var tasks []geocodingTask

	for i, record := range records {
		address, hasAddress := record["address"].(string)
		if !hasAddress || address == "" {
			continue
		}

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

func (s *ExcelService) hasCoordinates(record map[string]any) bool {
	_, hasLng := record["longitude"]
	_, hasLat := record["latitude"]
	return hasLng && hasLat
}

func (s *ExcelService) executeGeocodingTasks(
	ctx context.Context,
	tasks []geocodingTask,
	records []map[string]any,
) (successCount, failCount int) {
	const excelGeocodingMaxConcurrency = 5
	sem := make(chan struct{}, excelGeocodingMaxConcurrency)
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

func (s *ExcelService) processThreeLevelDepartments(
	ctx context.Context,
	db *gorm.DB,
	records []map[string]any,
	result *ImportResult,
	config ExcelConfig,
) ([]map[string]any, *ImportResult) {
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
	validRecords := make([]map[string]any, 0, len(records))
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
		if err := s.ensureDeptGroupExists(ctx, db, group.Code, group.Name); err != nil {
			// 部门组创建失败不应该阻止整个导入，只记录错误
			logger.Warnf("创建部门组失败 [%s]: %v", group.Code, err)
		}
	}

	// 第三遍：确保部门存在，并设置正确的parent_id
	for _, dept := range deptMap {
		// 获取部门组的ID
		groupID, err := s.findDeptIDByCode(ctx, db, dept.GroupCode)
		if err != nil {
			logger.Warnf("查找部门组失败 [%s]: %v", dept.GroupCode, err)
			continue // 跳过该部门
		}

		if err := s.ensureDeptExists(ctx, db, dept.Code, dept.Name, groupID); err != nil {
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
		err := db.WithContext(ctx).
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
	finalRecords := make([]map[string]any, 0, len(validRecords))
	for _, record := range validRecords {
		if skip, exists := record["_skip"]; !exists || !skip.(bool) {
			delete(record, "_skip")
			finalRecords = append(finalRecords, record)
		}
	}

	return finalRecords, result
}

// findDeptIDByCode 根据部门编码查找部门ID
func (s *ExcelService) findDeptIDByCode(ctx context.Context, db *gorm.DB, deptCode string) (string, error) {
	var result struct {
		ID string `gorm:"column:id"`
	}

	err := db.WithContext(ctx).
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
func (s *ExcelService) ensureDeptGroupExists(ctx context.Context, db *gorm.DB, code, name string) error {
	// 先查找是否存在（使用数据库字段名）
	var count int64
	err := db.WithContext(ctx).
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
	dept := map[string]any{
		"id":         generateUUID(),
		"dept_name":  name,
		"dept_code":  code,
		"parent_id":  nil, // 部门组是顶级部门
		"order_num":  0,
		"status":     models.DeptStatusNormal,
		"ancestors":  "",
		"created_at": now,
		"updated_at": now,
	}

	return db.WithContext(ctx).Table("sys_dept").Create(dept).Error
}

// ensureDeptExists 确保部门存在，不存在则创建
func (s *ExcelService) ensureDeptExists(ctx context.Context, db *gorm.DB, code, name, parentID string) error {
	// 先查找是否存在（使用数据库字段名）
	var count int64
	err := db.WithContext(ctx).
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
	err = db.WithContext(ctx).
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
	dept := map[string]any{
		"id":         generateUUID(),
		"dept_name":  name,
		"dept_code":  code,
		"parent_id":  parentID,
		"ancestors":  ancestors,
		"order_num":  0,
		"status":     models.DeptStatusNormal,
		"created_at": now,
		"updated_at": now,
	}

	return db.WithContext(ctx).Table("sys_dept").Create(dept).Error
}

// =========================================================================
// Phase 35: 工位导出增强 — 追加 3 类设备 sheet
// =========================================================================
//
// 工位管理导出时,除 Sheet 1 (工位列表) 外,还追加 3 个设备 sheet:
//   - AD设备:        域控 (ComputerName / OS / MAC / Serial / LastLogon)
//   - 资产设备:      资产管理 (DeviceName / Model / Type / IP / ResponsibleUser)
//   - 物理链路设备:  MAC→port→infoPoint→workstation (MAC / Port / InfoPoint / LastSeen / Confidence)
//
// 设计决策:
//   - 每个工位在 3 个设备 sheet 中占 0~N 行 (基于 GetADDevices / GetAssetDevices / GetPhysicalDevices)
//   - 始终生成 sheet + 表头 (即使无数据,便于用户知道有这个数据维度)
//   - batch enrichment: workstation_name / sys_ad_computer (OS+LastLogon) / ops_asset (IP)
//   - 物理 Port/InfoPoint 从 WorkstationDevice.Description 正则解析,避免新增 JOIN
//     Description 格式: "端口 GE2/6 (信息点 WH-04F-130)" 或附加历史关联提示
//
// 失败不阻断主流程, 调用方 (ExportData) 仅记录 warning。
func (s *ExcelService) appendWorkstationDeviceSheets(
	ctx context.Context,
	f *excelize.File,
	exportService *excelExportServiceImpl,
	params map[string]any,
) error {
	logger.Infof("[ExcelExport] appendWorkstationDeviceSheets 入口: deviceService=%v params=%v",
		s.deviceService != nil, params)

	if s.deviceService == nil {
		return fmt.Errorf("deviceService 未注入, 跳过工位设备 sheet 追加")
	}

	// 1. 提取当前过滤条件下的工位 ID 列表 (复用 exportService 的 queryBuilder)
	workstationIDs, err := s.queryWorkstationIDsForExport(ctx, exportService, params)
	if err != nil {
		return fmt.Errorf("查询工位 ID 失败: %w", err)
	}
	logger.Infof("[ExcelExport] 查询到 %d 个工位 ID", len(workstationIDs))
	if len(workstationIDs) == 0 {
		logger.Infof("[ExcelExport] 当前过滤条件下无工位, 跳过设备 sheet 追加")
		return nil
	}

	// 2. 真批量查询 3 类设备 (Phase 35 优化)
	// 关键性能点: 用真批量方法 (GetADDevicesByWorkstations 等), 把 717 次 SQL 降到 ~3 次
	// 内部使用独立的超时 context (避免 HTTP 请求 ctx 在中途被取消)
	deviceCtx, deviceCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer deviceCancel()

	adMap, adErr := s.deviceService.GetADDevicesByWorkstations(deviceCtx, workstationIDs)
	if adErr != nil {
		logger.Warnf("[ExcelExport] 批量查询 AD 设备失败: %v", adErr)
		adMap = make(map[string][]*models.WorkstationDevice)
	}
	assetMap, assetErr := s.deviceService.GetAssetDevicesByWorkstations(deviceCtx, workstationIDs)
	if assetErr != nil {
		logger.Warnf("[ExcelExport] 批量查询资产设备失败: %v", assetErr)
		assetMap = make(map[string][]*models.WorkstationDevice)
	}
	physMap, physErr := s.deviceService.GetPhysicalDevicesByWorkstations(deviceCtx, workstationIDs)
	if physErr != nil {
		logger.Warnf("[ExcelExport] 批量查询物理链路设备失败: %v", physErr)
		physMap = make(map[string][]*models.WorkstationDevice)
	}

	// 扁平化: map[wsID] → []WorkstationDevice
	var adDevices, assetDevices, physicalDevices []*models.WorkstationDevice
	for _, wsID := range workstationIDs {
		adDevices = append(adDevices, adMap[wsID]...)
		assetDevices = append(assetDevices, assetMap[wsID]...)
		physicalDevices = append(physicalDevices, physMap[wsID]...)
	}
	logger.Infof("[ExcelExport] 设备收集完成: AD=%d 资产=%d 物理=%d",
		len(adDevices), len(assetDevices), len(physicalDevices))

	// 3. 批量 enrichment (workstation_name / ad_computer OS+LastLogon / asset IP)
	//
	// 关键: enrichment 必须复用 deviceCtx (与本函数第 2 步真批量设备查询一致),
	// 而非上层 ctx. 上层 ctx 来自 gin.Request.Context(), 客户端断开 (browser / proxy
	// 超时) 会立即 cancel, 导致三个 enrichment 查询全部报 context canceled,
	// 但 deviceCtx 是 context.Background() + 60s timeout, 不受 HTTP 客户端断影响.
	//
	// 借用 deviceCtx 也是安全的: enrichment 阶段所有数据已经从 deviceCtx 步骤取出,
	// 这里只是再补 3 个字典级查询, 数据量小 (UUID 列表 648/1643/1662), 60s 之内必完成.
	wsNameMap := s.batchGetWorkstationNames(deviceCtx, workstationIDs)
	adEnrichment := s.batchGetADEnrichment(deviceCtx, adDevices)
	assetEnrichment := s.batchGetAssetEnrichment(deviceCtx, assetDevices)

	// 4. 写入 3 个 sheet
	if err := s.writeDeviceSheet(f, "AD设备",
		[]string{"工位名称", "ComputerName", "OS", "MAC", "Serial", "LastLogon"},
		adDevices,
		func(d *models.WorkstationDevice) []string {
			os := ""
			lastLogon := ""
			if d.ADComputerID != nil {
				if e, ok := adEnrichment[*d.ADComputerID]; ok {
					os = e.os
					lastLogon = e.lastLogon
				}
			}
			return []string{
				wsNameMap[d.WorkstationID],
				stringValueOrEmpty(d.DeviceName),
				os,
				stringValueOrEmpty(d.MACAddress),
				stringValueOrEmpty(d.DeviceSerial),
				lastLogon,
			}
		}); err != nil {
		return fmt.Errorf("写 AD sheet 失败: %w", err)
	}

	if err := s.writeDeviceSheet(f, "资产设备",
		[]string{"工位名称", "DeviceName", "Model", "Type", "IP", "ResponsibleUser"},
		assetDevices,
		func(d *models.WorkstationDevice) []string {
			ip := ""
			if d.AssetID != nil {
				ip = assetEnrichment[*d.AssetID]
			}
			return []string{
				wsNameMap[d.WorkstationID],
				stringValueOrEmpty(d.DeviceName),
				stringValueOrEmpty(d.DeviceModel),
				stringValueOrEmpty(d.DeviceType),
				ip,
				stringValueOrEmpty(d.ResponsibleUser),
			}
		}); err != nil {
		return fmt.Errorf("写资产 sheet 失败: %w", err)
	}

	if err := s.writeDeviceSheet(f, "物理链路设备",
		[]string{"工位名称", "设备名称", "序列号", "型号", "类型", "MAC", "IP地址", "责任人", "Port", "InfoPoint", "LastSeen", "Confidence"},
		physicalDevices,
		func(d *models.WorkstationDevice) []string {
			port, infoPoint := parsePhysicalPortInfo(d.Description)
			lastSeen := ""
			if d.HistoryLastSeen != nil && !d.HistoryLastSeen.IsZero() {
				lastSeen = d.HistoryLastSeen.Format("2006-01-02 15:04:05")
			}
			confidence := ""
			if d.Confidence != nil {
				confidence = fmt.Sprintf("%.2f", *d.Confidence)
			}
			return []string{
				wsNameMap[d.WorkstationID],
				stringValueOrEmpty(d.DeviceName),
				stringValueOrEmpty(d.DeviceSerial),
				stringValueOrEmpty(d.DeviceModel),
				stringValueOrEmpty(d.DeviceType),
				stringValueOrEmpty(d.MACAddress),
				stringValueOrEmpty(d.IPAddress),
				stringValueOrEmpty(d.ResponsibleUser),
				port,
				infoPoint,
				lastSeen,
				confidence,
			}
		}); err != nil {
		return fmt.Errorf("写物理链路 sheet 失败: %w", err)
	}

	logger.Infof("[ExcelExport] 工位导出追加设备 sheet 完成: AD=%d 资产=%d 物理=%d 工位数=%d",
		len(adDevices), len(assetDevices), len(physicalDevices), len(workstationIDs))
	return nil
}

// queryWorkstationIDsForExport 提取当前过滤条件下的工位 ID 列表。
//
// 实现策略:
//   - 不复用 WorkstationQueryBuilder (其带 ops_floors/ops_buildings LEFT JOIN 与 ::uuid 强制类型转换,
//     在 floor_id/building_id 为空字符串时会导致整个 query 失败, 进而本函数无返回)
//   - 改用直接 sys_workstation 表查询 + 同源 FilterMapping, 避免 JOIN 副作用
//   - Pluck 到 *[]string 在 PG/SQLite 上对非空 UUID 列均安全
func (s *ExcelService) queryWorkstationIDsForExport(
	ctx context.Context,
	exportService *excelExportServiceImpl,
	params map[string]any,
) ([]string, error) {
	config, exists := GetExportConfig("workstation")
	if !exists {
		return nil, fmt.Errorf("工位导出配置缺失")
	}
	_ = exportService // 当前实现不再依赖 exportService, 保留形参以兼容调用方

	// 直接查询 sys_workstation 表, 应用 FilterMapping 过滤 (避免 JOIN 副作用)
	query := s.db.WithContext(ctx).
		Table("sys_workstation").
		Where("deleted_at IS NULL")

	for paramKey, paramValue := range params {
		if paramValue == nil || paramValue == "" {
			continue
		}
		dbField, hasMapping := config.FilterMapping[paramKey]
		if !hasMapping {
			dbField = paramKey
		}
		// FilterMapping 中的字段不带表前缀, 补 sys_workstation.
		if !strings.Contains(dbField, ".") {
			dbField = "sys_workstation." + dbField
		}
		switch v := paramValue.(type) {
		case string:
			query = query.Where(dbField+" LIKE ?", "%"+v+"%")
		case int, int64, float64:
			query = query.Where(dbField+" = ?", v)
		case bool:
			query = query.Where(dbField+" = ?", v)
		case []interface{}:
			if len(v) > 0 {
				query = query.Where(dbField+" IN ?", v)
			}
		}
	}

	var wsIDs []string
	if err := query.Pluck("sys_workstation.id", &wsIDs).Error; err != nil {
		return nil, fmt.Errorf("查询 sys_workstation.id 失败: %w", err)
	}
	return wsIDs, nil
}

// adEnrichmentInfo AD 设备的 enrichment 数据 (OS + LastLogon)。
type adEnrichmentInfo struct {
	os        string
	lastLogon string
}

// batchGetWorkstationNames 按 ID 批量获取工位名称映射 (sys_workstation.workstation_name)。
func (s *ExcelService) batchGetWorkstationNames(ctx context.Context, ids []string) map[string]string {
	result := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return result
	}
	var rows []struct {
		ID              string
		WorkstationName string
	}
	if err := s.db.WithContext(ctx).
		Table("sys_workstation").
		Select("id, workstation_name").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Scan(&rows).Error; err != nil {
		logger.Warnf("[ExcelExport] 批量获取工位名称失败: %v", err)
		return result
	}
	for _, r := range rows {
		result[r.ID] = r.WorkstationName
	}
	return result
}

// batchGetADEnrichment 按 ADComputerID 批量获取 sys_ad_computer 的 OS + LastLogon。
func (s *ExcelService) batchGetADEnrichment(ctx context.Context, devices []*models.WorkstationDevice) map[string]adEnrichmentInfo {
	result := make(map[string]adEnrichmentInfo)
	if len(devices) == 0 {
		return result
	}
	idSet := make(map[string]bool)
	ids := make([]string, 0)
	for _, d := range devices {
		if d.ADComputerID != nil && *d.ADComputerID != "" && !idSet[*d.ADComputerID] {
			idSet[*d.ADComputerID] = true
			ids = append(ids, *d.ADComputerID)
		}
	}
	if len(ids) == 0 {
		return result
	}
	var rows []struct {
		ID              string
		OperatingSystem string
		LastLogon       *time.Time
	}
	if err := s.db.WithContext(ctx).
		Table("sys_ad_computer").
		Select("id, operating_system, last_logon").
		Where("id IN ?", ids).
		Scan(&rows).Error; err != nil {
		logger.Warnf("[ExcelExport] 批量获取 AD enrichment 失败: %v", err)
		return result
	}
	for _, r := range rows {
		lastLogon := ""
		if r.LastLogon != nil && !r.LastLogon.IsZero() {
			lastLogon = r.LastLogon.Format("2006-01-02 15:04:05")
		}
		result[r.ID] = adEnrichmentInfo{os: r.OperatingSystem, lastLogon: lastLogon}
	}
	return result
}

// batchGetAssetEnrichment 按 AssetID 批量获取 ops_asset 的 IP 地址。
//
// 关键: ops_asset 表实际只有 `machine_ip` 列(对应 Asset.MachineIP 字段,语义"加域IP"),
// 没有 `ip_address` 列 (SQLSTATE 42703). 早期实现误用 `ip_address` → 被前面
// 三轮 enrichment context canceled 掩盖,2026-07-24 context cancel 修复后才浮出.
func (s *ExcelService) batchGetAssetEnrichment(ctx context.Context, devices []*models.WorkstationDevice) map[string]string {
	result := make(map[string]string)
	if len(devices) == 0 {
		return result
	}
	idSet := make(map[string]bool)
	ids := make([]string, 0)
	for _, d := range devices {
		if d.AssetID != nil && *d.AssetID != "" && !idSet[*d.AssetID] {
			idSet[*d.AssetID] = true
			ids = append(ids, *d.AssetID)
		}
	}
	if len(ids) == 0 {
		return result
	}
	var rows []struct {
		ID        string
		MachineIP string
	}
	if err := s.db.WithContext(ctx).
		Table("ops_asset").
		Select("id, machine_ip").
		Where("id IN ? AND deleted_at IS NULL", ids).
		Scan(&rows).Error; err != nil {
		logger.Warnf("[ExcelExport] 批量获取资产 IP enrichment 失败: %v", err)
		return result
	}
	for _, r := range rows {
		result[r.ID] = r.MachineIP
	}
	return result
}

// writeDeviceSheet 写入设备 sheet (固定格式: 1 行表头 + N 行数据)。
func (s *ExcelService) writeDeviceSheet(
	f *excelize.File,
	sheetName string,
	headers []string,
	devices []*models.WorkstationDevice,
	rowMapper func(*models.WorkstationDevice) []string,
) error {
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("创建 sheet %s 失败: %w", sheetName, err)
	}
	f.SetActiveSheet(index)

	headerStyle, err := createHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("创建表头样式失败: %w", err)
	}

	// 写表头
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheetName, cell, h)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// 列宽
	for i, h := range headers {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheetName, colName, colName, calculateColumnWidth(h))
	}

	// 写数据行 (允许 devices 为空, 表头保留)
	for i, d := range devices {
		row := rowMapper(d)
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}

	_ = f.SetPanes(sheetName, &excelize.Panes{
		Freeze: true,
		XSplit: 1,
		YSplit: 1,
	})

	return nil
}

// parsePhysicalPortInfo 从 WorkstationDevice.Description 解析 Port + InfoPoint。
//
// Description 格式:
//   - "端口 GE2/6 (信息点 WH-04F-130)"
//   - "端口 GE2/6"
//   - "端口 GE2/6 (信息点 WH-04F-130)\n历史关联 (最后上线时间: 2026-07-21 12:00:00)"
//
// 返回值: port (如 "GE2/6"), infoPoint (如 "WH-04F-130"), 无匹配时为空字符串。
func parsePhysicalPortInfo(desc *string) (port, infoPoint string) {
	if desc == nil {
		return "", ""
	}
	// 匹配 "端口 <port>" (直到空白 / 左括号 / 换行)
	portRe := regexp.MustCompile(`端口\s+([^\s(\n]+)`)
	if m := portRe.FindStringSubmatch(*desc); len(m) > 1 {
		port = m[1]
	}
	// 匹配 "信息点 <name>" (直到右括号 / 换行)
	infoRe := regexp.MustCompile(`信息点\s+([^)\n]+)`)
	if m := infoRe.FindStringSubmatch(*desc); len(m) > 1 {
		infoPoint = m[1]
	}
	return port, infoPoint
}

// stringValueOrEmpty 把 *string 安全转为字符串 (nil 返回空)。
func stringValueOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
