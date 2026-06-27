package asset

import (
	"context"
	"errors"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/base"
	"gorm.io/gorm"
)

// ============================================================================
// R1 边界说明 — 资产对账异常查询服务(Phase 42 R1 plan 02 / 任务 1)
//
// 本文件实现 R1 范围:仅"只读观察"。具体而言:
//
//   - ListExceptions:分页查询异常列表,支持多维度过滤 + 白名单排序
//   - GetByID:按主键查询单条异常
//
// R1 **不**提供的操作(后续 plan 接管):
//   - MarkResolved (D-18):由 R2 ReconciliationResolutionService 实现
//   - Create / Update / Delete 异常记录:由 Layer 3 检测引擎写入,
//     handler 不暴露写入口,运维只能"查看 + 解决"两类行为
//   - BulkResolve:由 R2 实现
//   - 例外规则 CRUD:由 R3 ReconciliationExceptionService 实现
//   - 统计/Count 聚合:由 R2 ReconciliationStatistics 实现
//
// 数据源:
//
//   - 主表 sys_data_reconciliation(R1 异常元数据)
//   - LEFT JOIN ops_asset 取 asset_code(devicesn) / machine_ip(显示用)
//   - LEFT JOIN reconciliation_normalized(MV,phase 42-01 已创建)取
//     physical_username (R1=NULL,R2 填充) / ad_username (R1 已有 AD 链)
//   - LEFT JOIN sys_user 取 responsible_username:此处按"asset 维度"理解
//     "责任人 username",即 ops_asset.user_id 关联 sys_user.username
//
// 关联约定 (来自 phase 42-01 物化视图定义):
//   - ops_asset.id (uuid) = sys_data_reconciliation.asset_id (uuid) —— 类型一致,可直接 JOIN
//   - reconciliation_normalized.asset_id = ops_asset.id (一对一) —— 同一资产
//   - sys_user.id (uuid) = ops_asset.user_id (varchar(64) per asset model) ——
//     严格说 sys_user.id 是 uuid,ops_asset.user_id 是 varchar(64),本查询
//     仅按 uuid = uuid 比较(假设 ops_asset.user_id 已规范化为 uuid 字符串;
//     历史上脏数据见 migration_145_fix_bound_user_id_uuid,本服务假设已完成)。
// ============================================================================

// ExceptionListParams 异常列表查询参数
//
// 嵌入 base.BaseListRequest 自动获得 current / pageSize / orderByColumn / isAsc 四个字段。
// 所有过滤字段均为可选;零值/空值跳过对应过滤。
type ExceptionListParams struct {
	base.BaseListRequest

	ConflictType string     `json:"conflictType"`           // 冲突类型(A/B/C/D/E/F,精确匹配)
	Severity     string     `json:"severity"`               // 严重度(low/medium/high/critical,精确匹配)
	AssetCode    string     `json:"assetCode"`              // 资产编号(devicesn),LIKE 模糊匹配
	DetectedFrom *time.Time `json:"detectedFrom,omitempty"` // 检测时间区间起点(包含)
	DetectedTo   *time.Time `json:"detectedTo,omitempty"`   // 检测时间区间终点(包含)
}

// ExceptionListItem 异常列表展示项(查询 DTO,非表模型)
//
// 嵌入 SysDataReconciliation 复用其全部字段;额外字段是 JOIN 出来的展示列
// (asset_code / asset_ip_display / physical_username / responsible_username / ad_username),
// 不对应 sys_data_reconciliation 真实列(故不放进 model,避免 AutoMigrate 误建列)。
//
// GORM 列名映射说明:
//   - SysDataReconciliation.ID 等基础字段由嵌入体自身的 gorm tag 处理
//   - "asset_code" / "asset_ip" / "physical_username" / "ad_username" 由 gorm
//     column tag 显式映射;SELECT 子句使用同名 AS alias 让 GORM 顺利 Scan
//   - "responsible_username" 来自 sys_user.username,无 gorm column tag,
//     需在 SELECT 子句中显式 AS responsible_username
type ExceptionListItem struct {
	models.SysDataReconciliation

	// AssetCode 资产编号(来自 ops_asset.devicesn)
	AssetCode string `gorm:"column:asset_code" json:"assetCode,omitempty"`

	// AssetIPDisplay 资产 IP 显示值(来自 ops_asset.machine_ip,COALESCE 转 text)
	// 字段名加 Display 后缀,避免与 SysDataReconciliation.AssetIP(*net.IP)同名字段冲突。
	AssetIPDisplay string `gorm:"column:asset_ip" json:"assetIpDisplay,omitempty"`

	// PhysicalUsername 物理链路反推用户(R1 物化视图固定 NULL,R2 填充)
	PhysicalUsername *string `gorm:"column:physical_username" json:"physicalUsername,omitempty"`

	// ResponsibleUsername 责任人 username(来自 sys_user.username via ops_asset.user_id)
	// 命名遵循 CLAUDE.md json convention(驼峰);gorm column 与物理列同名(responsible_username)
	ResponsibleUsername *string `gorm:"column:responsible_username" json:"responsibleUsername,omitempty"`

	// AdUsername AD 域账号(来自 MV reconciliation_normalized.ad_username)
	AdUsername *string `gorm:"column:ad_username" json:"adUsername,omitempty"`
}

// reconAllowedSortFields 异常列表排序白名单
//
// key 为前端传入的 orderByColumn,value 为安全的物理列名(列名白名单是 SQL 注入防线:
// 即使恶意构造 orderByColumn,也只能命中本 map 内的值,无法拼接任意 SQL)。
//
// Phase 42-01 物化视图已建,Phase 42-02 R1 只读查询,排序字段仅限主表
// sys_data_reconciliation 的列;MV 列(physical_username / ad_username)暂不
// 开放给前端排序(R2 视情况补充)。
var reconAllowedSortFields = map[string]string{
	"detectedAt":      "sys_data_reconciliation.detected_at",
	"conflictType":    "sys_data_reconciliation.conflict_type",
	"severity":        "sys_data_reconciliation.severity",
	"confidenceScore": "sys_data_reconciliation.confidence_score",
}

// ReconciliationService 资产对账异常查询服务接口
//
// R1 范围:ListExceptions(分页过滤 + 排序) + GetByID(单条详情)。
// 写操作 / 标记已解决 / 例外规则 CRUD 见 R2/R3 的独立 service。
type ReconciliationService interface {
	// ListExceptions 列出对账异常(分页 + 过滤 + 白名单排序 + 多表 JOIN)
	ListExceptions(ctx context.Context, params *ExceptionListParams) (*base.PageResult, error)

	// GetByID 按主键查询单条异常(不带 JOIN,纯主表元数据)
	GetByID(ctx context.Context, id string) (*models.SysDataReconciliation, error)
}

// reconciliationServiceImpl ReconciliationService 私有实现
type reconciliationServiceImpl struct {
	db *gorm.DB
}

// NewReconciliationService 构造 ReconciliationService 实例
func NewReconciliationService(db *gorm.DB) ReconciliationService {
	return &reconciliationServiceImpl{db: db}
}

// exceptionListJoinSelect ListExceptions 的 SELECT 子句
//
// 关键设计:
//   - "sys_data_reconciliation.*" 显式限定表名前缀,避免与 JOIN 表的列名冲突
//   - a.devicesn AS asset_code —— 把 devicesn 别名为业务字段 asset_code,
//     配合 ExceptionListItem.AssetCode 的 gorm column tag 让 GORM 自动 Scan
//   - COALESCE(a.machine_ip::text, '') AS asset_ip —— machine_ip 是 inet 类型,
//     PG 强类型 inet 不能直接走 json Marshal,转 text 安全(SQLite 端 inet 即 TEXT,
//     COALESCE(text, '') 在两库行为一致)
//   - rn.physical_username / rn.ad_username —— 来自 reconciliation_normalized 物化视图
//   - ru.username AS responsible_username —— sys_user.username 别名为 responsible_username
//     匹配 ExceptionListItem.ResponsibleUsername 的 column tag
const exceptionListJoinSelect = `sys_data_reconciliation.*,
	a.devicesn AS asset_code,
	COALESCE(a.machine_ip::text, '') AS asset_ip,
	rn.physical_username,
	ru.username AS responsible_username,
	rn.ad_username`

// exceptionListJoinClause ListExceptions 的 JOIN 子句
//
// 关联关系:
//   - sys_data_reconciliation.asset_id (uuid) → ops_asset.id (uuid) —— 类型一致,直接 =
//   - ops_asset.id (uuid) → reconciliation_normalized.asset_id (uuid) ——
//     MV 与 ops_asset 一对一(见 migration_168 定义:一个 asset 一行 MV)
//   - ops_asset.user_id (varchar(64) in asset model) → sys_user.id (uuid) ——
//     理论上应一致(已被 migration_145 规范化);如遇脏数据,本 JOIN 会留空
//
// 软删除条件:各表都加 deleted_at IS NULL 兜底,避免已删除的资产/用户污染对账展示。
const exceptionListJoinClause = `
LEFT JOIN ops_asset a ON a.id = sys_data_reconciliation.asset_id AND a.deleted_at IS NULL
LEFT JOIN reconciliation_normalized rn ON rn.asset_id = a.id
LEFT JOIN sys_user ru ON ru.id = a.user_id AND ru.deleted_at IS NULL`

// ListExceptions 列出对账异常
//
// 行为:
//  1. 校验 params 非 nil
//  2. 默认 current=1, pageSize=10
//  3. 基础查询:Model(SysDataReconciliation{}) + deleted_at IS NULL
//  4. 过滤:conflictType / severity (精确) + assetCode (LIKE) + detectedFrom/To (区间)
//  5. Count + Find,Find 携带 JOIN + SELECT 子句
//  6. 排序:base.ApplySort 白名单;无显式排序则默认 detected_at DESC
//  7. 分页:Offset + Limit
//  8. 拼装 base.PageResult 返回
func (s *reconciliationServiceImpl) ListExceptions(ctx context.Context, params *ExceptionListParams) (*base.PageResult, error) {
	if params == nil {
		return nil, errors.New("查询参数不能为空")
	}

	// 默认分页
	current := params.Current
	if current <= 0 {
		current = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	// 基础查询
	query := s.db.WithContext(ctx).
		Model(&models.SysDataReconciliation{}).
		Where("sys_data_reconciliation.deleted_at IS NULL")

	// 过滤:精确匹配
	if params.ConflictType != "" {
		query = query.Where("sys_data_reconciliation.conflict_type = ?", params.ConflictType)
	}
	if params.Severity != "" {
		query = query.Where("sys_data_reconciliation.severity = ?", params.Severity)
	}
	// 过滤:资产编号 LIKE 模糊匹配
	// 注意:LIKE 操作符未走白名单过滤,值通过 ? 占位符 + 参数化,无 SQL 注入风险
	if params.AssetCode != "" {
		query = query.Where("a.devicesn LIKE ?", "%"+params.AssetCode+"%")
	}
	// 过滤:检测时间区间
	if params.DetectedFrom != nil {
		query = query.Where("sys_data_reconciliation.detected_at >= ?", *params.DetectedFrom)
	}
	if params.DetectedTo != nil {
		query = query.Where("sys_data_reconciliation.detected_at <= ?", *params.DetectedTo)
	}

	// Count(COUNT 不需要 JOIN 出来的列;基础查询 + 过滤已足够)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Find 携带 SELECT + JOIN
	var list []ExceptionListItem
	findQuery := query.
		Select(exceptionListJoinSelect).
		Joins(exceptionListJoinClause)

	// 排序:白名单过滤;无显式排序则默认 detected_at DESC(操作员最关心"最新告警")
	findQuery = base.ApplySort(findQuery, params.BaseListRequest, reconAllowedSortFields)
	if findQuery.Statement.SQL.String() != "" {
		// base.ApplySort 在 OrderByColumn 为空时不会追加 Order,这里手动判断
		// 简化做法:始终先调用 ApplySort,然后无条件追加默认排序作为 fallback
		// 但 GORM 多个 Order() 调用后者会覆盖前者,所以仅在未指定排序时追加
		if params.OrderByColumn == "" {
			findQuery = findQuery.Order("sys_data_reconciliation.detected_at DESC")
		}
	} else {
		findQuery = findQuery.Order("sys_data_reconciliation.detected_at DESC")
	}

	offset := (current - 1) * pageSize
	if err := findQuery.Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, err
	}

	return &base.PageResult{
		List:     list,
		Total:    total,
		Current:  current,
		PageSize: pageSize,
	}, nil
}

// GetByID 按主键查询单条异常
//
// 行为:
//   - 软删除兜底:deleted_at IS NULL
//   - 记录不存在返回 nil(nil + nil 是该方法的正常"未找到"语义,
//     区别于"数据库出错"返回的 nil + err)
func (s *reconciliationServiceImpl) GetByID(ctx context.Context, id string) (*models.SysDataReconciliation, error) {
	var recon models.SysDataReconciliation
	err := s.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&recon).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &recon, nil
}
