package operations

import (
	"context"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/pkg/query"
	"gorm.io/gorm"
)

const (
	floorTable            = "ops_floors"
	workstationJoinSelect = "sys_workstation.*, ops_floors.name as floor_name, ops_floors.floor_no as floor_code, ops_buildings.name as building_name, ops_buildings.id as building_id, sys_dept.dept_name as dept_name, sys_user.nickname as user_name, (SELECT device_serial FROM ops_workstation_device WHERE workstation_id = CAST(sys_workstation.id AS TEXT) AND deleted_at IS NULL AND is_primary = true ORDER BY priority DESC, created_at ASC LIMIT 1) as primary_device_serial"
	// uuid 列(varchar 外键列对 uuid 主键)用标准 SQL CAST(... AS TEXT) 统一转 text 比较,
	// PG/SQLite 双方言行为一致(PG 专有 ::uuid/::text 在 SQLite 报 unrecognized token ':')。
	// 与 GetWorkstationDeptOptions / location_alias_service 的 CAST 范式一致。
	workstationJoinClause = "LEFT JOIN ops_floors ON CAST(ops_floors.id AS TEXT) = sys_workstation.floor_id LEFT JOIN ops_buildings ON CAST(ops_buildings.id AS TEXT) = ops_floors.building_id LEFT JOIN sys_dept ON CAST(sys_dept.id AS TEXT) = sys_workstation.dept_id LEFT JOIN sys_user ON CAST(sys_user.id AS TEXT) = sys_workstation.user_id"
	// workstationOrgColumn（工位机构筛选「有效机构」表达式）见下方 var：
	// 运行时经 effectiveOrgExpr 生成，单一源自 dept_filter.go（审计 P2-6）。
)

// validateTableName 验证表名是否在白名单中，防止 SQL 注入
func validateTableName(tableName string) bool {
	allowedTables := map[string]bool{
		"ops_buildings": true,
		"sys_buildings": true,
		"ops_floors":    true,
		"sys_floors":    true,
	}
	return allowedTables[tableName]
}

// workstationOrgColumn 工位机构筛选使用的「有效机构」表达式。
//
// ops_floors.org_id 为楼层级机构覆盖值（可空），NULL 时继承 ops_buildings.org_id。
// 用于「一栋楼跨机构办公」：例如浙商大厦 org_id=分公司本部，但 1/3 楼
// org_id=武汉中支，选武汉中支时必须能筛出 1/3 楼的工位，反之亦然。
//
// 由 workstationOrgExistsSub 保证子查询里 f/b 两个表别名都存在（JOIN
// ops_floors f → ops_buildings b），列名均为常量、不含用户输入。
// NULLIF 把表单清空写入的空串也归一为 NULL，避免 COALESCE 命中空串。
// ⚠ 两侧都 CAST AS TEXT：org_id 若与 uuid 漂移（同 building_id 先例），
// COALESCE(text, uuid) 会被 PG 拒绝（types cannot be matched）；CAST 对
// varchar 是 no-op，SQLite 动态类型不受影响。
// 定义单一源自 dept_filter.go 的 effectiveOrgExpr（审计 P2-6）。
var workstationOrgColumn = effectiveOrgExpr("f.org_id", "b.org_id")

// CountsByFloorResult 批量楼层工位计数结果
type CountsByFloorResult struct {
	Counts map[string]int `json:"counts"` // floorId -> workstation count
}

// WorkstationService 工位服务接口。
//
// D-05（Phase 91-02）：List 与 SearchWorkstationOptions 的参数从
// map[string]interface{} 改为 typed request requests.WorkstationListRequest；
// 其余方法签名保持不变。Statistics 保持 map 参数（D-04 留 service，仍需
// 弱类型字符串提取）。
type WorkstationService interface {
	Create(ctx context.Context, workstation *models.Workstation) error
	Update(ctx context.Context, workstation *models.Workstation) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*models.Workstation, error)
	// List 查询工位列表（D-05: typed request）
	List(ctx context.Context, req requests.WorkstationListRequest) (*PageResult, error)
	BatchDelete(ctx context.Context, ids []string) error
	BatchUpdatePositions(ctx context.Context, items []PositionUpdateItem) error
	// Statistics 工位统计(专用 COUNT 聚合,不依赖分页列表;支持 orgId 部门筛选含子部门,替代前端 4 次 list 拼 total)。
	Statistics(ctx context.Context, params map[string]interface{}) (*WorkstationStatisticsResult, error)
	// GetWorkstationDeptOptions 工位编辑"所属部门"下拉数据源(orgId 子孙 + alias union)
	GetWorkstationDeptOptions(ctx context.Context, orgId string) ([]DeptOption, error)
	// BackfillDeptFromUser 按工位绑定用户反推部门，批量回填存量工位的 dept_id。
	// dryRun=true 只返回预览不写库；fixMismatch=true 额外纠正「已有部门但与
	// 用户部门不一致」的工位（默认 false = 仅填补空缺）。详见 quick-261106-wsb。
	BackfillDeptFromUser(ctx context.Context, dryRun, fixMismatch bool) (*BackfillDeptResult, error)
	// SearchWorkstationOptions 工位下拉数据源(LIKE 模糊 + 多维筛选,LIMIT 50)。
	// 设计给前端 Select/AutoComplete 远程搜索。workstation 高频变更,不缓存。
	// D-05: typed request,复用 WorkstationListRequest（去 map，不新增重复 struct）。
	SearchWorkstationOptions(ctx context.Context, req requests.WorkstationListRequest) ([]DropdownOption, error)
	// GetFloorWorkstationsAll 楼层全部工位(CAD/3D/平面图全集专用,无分页;V130R-09 D-03-6/D-03-7)。
	GetFloorWorkstationsAll(ctx context.Context, floorId string) ([]models.Workstation, error)
	// CountsByFloor 批量获取各楼层工位数量(READ,一个 SQL GROUP BY floor_id,替代前端 N+1 模式)。
	CountsByFloor(ctx context.Context, floorIds []string) (*CountsByFloorResult, error)
}

// DeptOption 工位编辑"所属部门"下拉选项(union: orgId 子孙 + alias 映射)
type DeptOption struct {
	DeptID   string `json:"deptId"`   // 部门 ID (UUID text)
	DeptName string `json:"deptName"` // 部门名称 (短名,由前端 trimTitleToLastSegment 进一步收窄)
	IsAlias  bool   `json:"isAlias"`  // 是否为 alias 映射条目; true → 前端追加 [映射] 后缀
}

// BackfillDeptSample 反推预览/执行的单条样例（上限 20 条，防大批量明细泄露）。
type BackfillDeptSample struct {
	WorkstationID   string `json:"workstationId"`
	WorkstationName string `json:"workstationName"`
	CurrentDeptID   string `json:"currentDeptId,omitempty"`
	CurrentDeptName string `json:"currentDeptName,omitempty"`
	TargetDeptID    string `json:"targetDeptId"`
	TargetDeptName  string `json:"targetDeptName,omitempty"`
	UserName        string `json:"userName,omitempty"`
	Action          string `json:"action"` // "fill" | "correct" | "skip"
}

// BackfillDeptResult 「按人员反推部门」预览/执行结果。
//
// dryRun=true 时 ToFill/ToCorrect 表示「将」变更条数且不写库；
// dryRun=false 时表示「实际」更新条数。第二次执行天然幂等（ToFill=0）。
type BackfillDeptResult struct {
	DryRun              bool                 `json:"dryRun"`
	FixMismatch         bool                 `json:"fixMismatch"`
	Candidates          int                  `json:"candidates"`          // 命中「有用户且用户部门有效」的工位总数
	ToFill              int                  `json:"toFill"`              // dept_id 为空 → 填补
	ToCorrect           int                  `json:"toCorrect"`           // dept_id 非空且不一致 → 纠正（仅 fixMismatch=true）
	SkippedNoUser       int                  `json:"skippedNoUser"`       // 无绑定用户
	SkippedBadUserDept  int                  `json:"skippedBadUserDept"`  // 用户部门 NULL/脏/指向不存在部门
	SkippedAlreadyValid int                  `json:"skippedAlreadyValid"` // 已有部门且一致（或不纠正时不一致亦计此）
	Samples             []BackfillDeptSample `json:"samples"`             // 前 20 条
}

// WorkstationStatisticsResult 工位统计结果(status: 0=可用 1=占用 2=维护)。
type WorkstationStatisticsResult struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"` // models.WorkstationStatusAvailable
	Occupied  int64 `json:"occupied"`  // models.WorkstationStatusOccupied
	Maintain  int64 `json:"maintain"`  // models.WorkstationStatusMaintain
}

// Statistics 统计工位(按 status 聚合,排除软删除;支持 orgId 部门筛选含子部门,与 List orgId 过滤一致)。
func (s *workstationService) Statistics(ctx context.Context, params map[string]interface{}) (*WorkstationStatisticsResult, error) {
	var result WorkstationStatisticsResult
	query := s.db.WithContext(ctx).Model(&models.Workstation{})
	if orgId := extractStringParam(params, "orgId"); orgId != "" {
		// 部门递归四条件统一走 BuildDeptRecursiveFilter（V130R-08）；
		// EXISTS 骨架（工位 → 楼层 f → 楼宇 b）见 workstationOrgExistsSub。
		// 有效机构 = 楼层覆盖值优先、NULL 时继承楼宇值（workstationOrgColumn）。
		sub := workstationOrgExistsSub(query, "CAST(f.id AS TEXT) = sys_workstation.floor_id")
		query = query.Where("EXISTS (?)", BuildDeptRecursiveFilter(orgId, workstationOrgColumn)(sub))
	}
	err := query.
		Select(
			"COUNT(*) AS total",
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS available", int(models.WorkstationStatusAvailable)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS occupied", int(models.WorkstationStatusOccupied)),
			fmt.Sprintf("COALESCE(SUM(CASE WHEN status = %d THEN 1 ELSE 0 END), 0) AS maintain", int(models.WorkstationStatusMaintain)),
		).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// workstationOrgExistsSub 工位 orgId 部门筛选的 EXISTS 子查询骨架
// （工位 → 楼层 f → 楼宇 b；部门递归条件由调用方经
// BuildDeptRecursiveFilter(orgId, "b.org_id") 注入后以 EXISTS (?) 挂接）：
//
//	sub := workstationOrgExistsSub(query, "CAST(f.id AS TEXT) = sys_workstation.floor_id")
//	query = query.Where("EXISTS (?)", BuildDeptRecursiveFilter(orgId, "b.org_id")(sub))
//
// Session(NewDB) 从外层链派生全新 statement（保留 ConnPool/Context），
// 子查询条件不会污染外层查询；原内联实现的 JOIN sys_dept d 已由 helper 的
// IN 子查询承接，此处不再需要。Statistics / List(filterScope) /
// SearchWorkstationOptions 三处共用。
func workstationOrgExistsSub(db *gorm.DB, entityFloorLink string) *gorm.DB {
	return db.Session(&gorm.Session{NewDB: true}).
		Table("ops_floors f").
		Joins("JOIN ops_buildings b ON CAST(b.id AS TEXT) = f.building_id").
		Where(entityFloorLink).
		Where("b.deleted_at IS NULL").
		Select("1")
}

// GetWorkstationDeptOptions 工位编辑"所属部门"下拉数据源 (D-06 单 query union)
// union: orgId 子孙节点 (is_alias=false) + alias 命中节点 (is_alias=true)
func (s *workstationService) GetWorkstationDeptOptions(ctx context.Context, orgId string) ([]DeptOption, error) {
	if orgId == "" {
		return []DeptOption{}, nil
	}
	var result []DeptOption
	// 单 query union: sys_dept 子孙节点 (is_alias=false) + alias 命中节点 (is_alias=true)
	// 1) 子孙节点: BuildDeptRecursiveFilter 四条件口径（orgId 自身 + 全部子孙，
	//    V130R-08），排除外部机构本身 —— 与 List/Statistics orgId 过滤同一 helper
	// 2) alias 节点: scope='workstation' AND location_id=orgId, JOIN sys_dept 取 dept_name
	//
	// 关于 sys_dept.id (uuid) 与 alias.dept_id/location_id (varchar) 的列对列比较:
	// 用标准 SQL CAST(... AS TEXT) 而非 PG 专有 ::text。
	//   - PG: CAST(uuid AS TEXT) 把 uuid 转成 text,与 varchar 比较(text/varchar 同族有 =)
	//   - SQLite: uuid 列实际存为 TEXT,CAST 是 no-op,TEXT = TEXT 正常
	// 故 CAST 写法在 PG/SQLite 双 DB 行为一致(避免 PG 专有 ::text 在 SQLite 语法错误)。
	// 与 location_alias_service.go 的 aliasListJoinClause 处理方式一致。
	// 子查询经 (?) 实参内联为 UNION ALL 左支（GORM 渲染 *gorm.DB 实参并合并其绑定参数）。
	descendants := s.db.WithContext(ctx).
		Table("sys_dept").
		Select("CAST(id AS TEXT) AS dept_id, dept_name AS dept_name, false AS is_alias").
		Where("deleted_at IS NULL AND is_external_org = 0").
		Scopes(BuildDeptRecursiveFilter(orgId, "id"))
	err := s.db.WithContext(ctx).Raw(`(?) UNION ALL
		SELECT a.dept_id, d.dept_name, true AS is_alias
		FROM sys_dept_location_alias a
		JOIN sys_dept d ON CAST(d.id AS TEXT) = a.dept_id
		WHERE a.deleted_at IS NULL
		  AND a.scope = 'workstation'
		  AND a.location_id = ?
	`, descendants, orgId).Scan(&result).Error
	return result, err
}

type workstationService struct {
	// repo 承接六方法 CRUD 管道（Phase 91-02 repo 化，D-02 纯 struct 组合）
	repo *base.GORMRepository[models.Workstation]
	// db 保留给 Statistics / GetWorkstationDeptOptions / BatchUpdatePositions /
	// SearchWorkstationOptions / Update 的 First 回填（D-04 留 service 的直用场景）
	db *gorm.DB
	// cache 可选注入：BackfillDeptFromUser 执行成功后失效 workstation 缓存。
	// nil 时跳过（测试/未注入场景），与 ExcelService.WithDeviceService 同款 setter 模式。
	cache system.CacheProvider
}

// workstationAllowedSortFields 工位可排序字段白名单。
// 因 List 有 LEFT JOIN floors/buildings/dept/user,字段值带表别名。
// key 须与前端列 dataIndex 一致(useServerSort 按 sorter.field === dataIndex 匹配 sorterMeta)。
//
// 注:workstationNo 列在 legacy-2026-06-15/034_remove_workstation_code_and_capacity.sql
// 已删除,白名单不再映射 — 否则客户端发 orderByColumn=workstationNo 会触发
// SQLSTATE 42703 (column workstation_no does not exist)。
var workstationAllowedSortFields = map[string]string{
	"workstationName": "sys_workstation.workstation_name",
	"name":            "sys_workstation.workstation_name", // 与前端 dataIndex="name" 对齐
	"status":          "sys_workstation.status",
	"workstationType": "sys_workstation.workstation_type",
	"type":            "sys_workstation.workstation_type", // 与前端 dataIndex="type" 对齐
	"buildingName":    "ops_buildings.name",
	"floorName":       "ops_floors.name",
	"deptName":        "sys_dept.dept_name",
	"userName":        "sys_user.nickname",
	"createdAt":       "sys_workstation.created_at",
	"updatedAt":       "sys_workstation.updated_at",
}

// PositionUpdateItem 位置更新项
type PositionUpdateItem struct {
	ID        string `json:"id"`
	PositionX int    `json:"positionX"`
	PositionY int    `json:"positionY"`
	Rotation  *int   `json:"rotation,omitempty"` // 旋转角度（可选）
	Width     *int   `json:"width,omitempty"`    // 工位宽度（毫米，可选）
	Depth     *int   `json:"depth,omitempty"`    // 工位深度（毫米，可选）
	DeskType  *int   `json:"deskType,omitempty"` // 桌型（可选）：0=一字型, 1=L型
}

// NewWorkstationService 创建工位服务实例
func NewWorkstationService(db *gorm.DB) WorkstationService {
	return &workstationService{
		repo: base.NewGORMRepository[models.Workstation](db),
		db:   db,
	}
}

// NewWorkstationServiceWithCache 创建带缓存失效能力的工位服务。
// 供 router 生产路径使用：BackfillDeptFromUser 成功后需失效 workstation 缓存
// （工位 CRUD 本身不缓存，但导入/批量写后列表统计依赖该失效）。
// 保留 NewWorkstationService 单参签名不变，避免影响 14 个既有调用点。
func NewWorkstationServiceWithCache(db *gorm.DB, cache system.CacheProvider) WorkstationService {
	return &workstationService{
		repo:  base.NewGORMRepository[models.Workstation](db),
		db:    db,
		cache: cache,
	}
}

// WithCacheProvider 注入缓存提供者，供 BackfillDeptFromUser 成功后失效工位缓存。
// 可选：不注入时该 service 仍可正常工作（缓存清理被跳过），与
// ExcelService.WithDeviceService 同款 setter 模式，避免改构造函数签名影响 14 个调用点。
//
// ⚠ 审计 P2-2 备注：当前 workstation 域无缓存写入者（CRUD 直读 DB），
// 此失效链路属预埋——为「工位数据被缓存化」的未来演进预先就位正确语义；
// 现阶段执行它是对失效模式的无害空转。删除或启用请与导入侧 CachePatterns
// （workstation:* / workstations:*）一并裁决。
func (s *workstationService) WithCacheProvider(cache system.CacheProvider) *workstationService {
	s.cache = cache
	return s
}

// Create 创建工位
//
// 新建工位默认 status=0（空闲）。如果用户在创建时同时指定了 user_id，则
// applyWorkstationOccupancyLink 会把 status 联动为 1（占用）。Maintain 状态
// 仅由管理员手动设置，新建路径不会触发（因为新工位入库前 status 来自表单，
// 通常是 Available）。
func (s *workstationService) Create(ctx context.Context, workstation *models.Workstation) error {
	// ✨ 联动 user_id ↔ status(占用/空闲)
	applyWorkstationOccupancyLink(workstation, workstation.Status)
	return s.repo.Create(ctx, workstation)
}

// Update 更新工位
//
// 实现说明(handler 反序列化的请求体只含表单字段,不含 createdAt/updatedAt):
// handler 把请求体反序列化到一个零值 models.Workstation,其 BaseModel.CreatedAt
// 是非指针 time.Time(见 internal/models/base.go),零值即 0001-01-01。若直接 Save,
// GORM 会用零值全量覆盖 created_at,导致工位创建时间被清零。
// 修复:Save 前先 First 查出现有记录,把审计字段 CreatedAt/CreatedBy 回填到入参,
// 再 Save。参考 internal/services/system/user_service.go 的 Update 模式。
func (s *workstationService) Update(ctx context.Context, workstation *models.Workstation) error {
	var existing models.Workstation
	if err := s.db.WithContext(ctx).Where("id = ?", workstation.ID).First(&existing).Error; err != nil {
		return err
	}
	// 保留创建时间和创建人,避免 Save 用零值覆盖(0001-01-01 bug 的根因)
	workstation.CreatedAt = existing.CreatedAt
	workstation.CreatedBy = existing.CreatedBy

	// ✨ 联动 user_id ↔ status(占用/空闲),Maintain(2) 保留维护语义
	applyWorkstationOccupancyLink(workstation, existing.Status)

	return s.repo.Update(ctx, workstation)
}

// applyWorkstationOccupancyLink 联动工位状态与所属人员。
//
// 规则:
//   - 如果 user_id 有值 → status = Occupied(1)
//   - 如果 user_id 为空 → status = Available(0)
//   - 如果当前 status = Maintain(2)，**保留维护状态不变**(运维流程强语义)
//
// 调用入口: Create + Update(在 Save 前)。Excel 导入路径单独在
// excel_service.prepareRecordsForUpsert 里覆盖 status(参见 n71)。
func applyWorkstationOccupancyLink(w *models.Workstation, currentStatus models.WorkstationStatus) {
	if currentStatus == models.WorkstationStatusMaintain {
		// 维护状态不受 user_id 联动影响
		return
	}
	if w.UserID != nil && *w.UserID != "" {
		w.Status = models.WorkstationStatusOccupied
	} else {
		w.Status = models.WorkstationStatusAvailable
	}
}

func (s *workstationService) Delete(ctx context.Context, id string) error {
	// repo.Delete 经 Model(new(T)) 软删除,与迁移前 .Table(workstationTable) 形式行为等价
	return s.repo.Delete(ctx, id)
}

// GetByID 根据ID获取工位（6 表 JOIN 管道经 repo 执行）。
// join scope 非空 → repo 经 Tabler 断言生成 sys_workstation.id = ?（models.Workstation
// TableName() 返回 sys_workstation，与迁移前 :231 的表限定逐字一致）。
func (s *workstationService) GetByID(ctx context.Context, id string) (*models.Workstation, error) {
	return s.repo.GetByID(ctx, id, s.joinScope())
}

// joinScope List/GetByID 共用的 Select + 6 表 LEFT JOIN scope（常量逐字平移）。
//
// 行为等价说明（Phase 91-02）：迁移前 Select+Joins 仅在 Find 前追加（Count 无 join）；
// repo 化后 joinScope 前置于 Count——workstationJoinClause 全部为按主键的 N:1 LEFT JOIN，
// 无行增殖，Count 结果不变（91-01 spike 实证 + TestImp77 Total 断言守护）；
// Count 时自定义 Select 被临时替换为 count(*) 并在 Find 恢复（TestBase91 契约锁定）。
func (s *workstationService) joinScope() base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Select(workstationJoinSelect).Joins(workstationJoinClause)
	}
}

// filterScope List 的六个过滤条件（条件字符串与 ? 占位符逐字平移自迁移前实现）。
//
// P6 -1 跳过语义保留：
//   - status: req.GetStatus(-1)——nil/缺失 → -1 → 跳过（与迁移前 map 路径
//     "status 缺省 -1，仅 >=0 才过滤" 语义等价）
//   - type:   req.Type == nil 或 *req.Type < 0 → 跳过
//
// buildingId/code 不消费：WorkstationListRequest 保留声明但服务从不读取
// （行为与迁移前一致，前端传入被静默忽略）。
func (s *workstationService) filterScope(req requests.WorkstationListRequest) base.Scope {
	return func(db *gorm.DB) *gorm.DB {
		if req.Name != "" {
			db = db.Where("sys_workstation.workstation_name LIKE ?", "%"+req.Name+"%")
		}
		if req.FloorID != "" {
			db = db.Where("sys_workstation.floor_id = ?", req.FloorID)
		}
		if req.FloorCode != "" {
			db = db.Where("EXISTS (SELECT 1 FROM "+floorTable+" WHERE CAST("+floorTable+".id AS TEXT) = sys_workstation.floor_id AND "+floorTable+".floor_no = ?)", req.FloorCode)
		}
		if st := req.GetStatus(-1); st >= 0 {
			db = db.Where("sys_workstation.status = ?", st)
		}
		if req.Type != nil && *req.Type >= 0 {
			db = db.Where("sys_workstation.workstation_type = ?", *req.Type)
		}
		// 通过关联机构筛选工位（包含子部门）：工位 -> 楼层 f -> 楼宇 b。
		// 有效机构 = COALESCE(NULLIF(f.org_id, ''), b.org_id)：楼层可覆盖楼宇归属，
		// 未覆盖（NULL/空串）则继承（workstationOrgColumn，一栋楼跨机构办公场景）。
		// 使用 EXISTS 子查询避免与现有 JOIN 冲突；部门递归四条件统一走
		// BuildDeptRecursiveFilter（V130R-08），其内部对 uuid/varchar 混比
		// 统一 CAST(... AS TEXT)（PG 对二者无隐式互转）；EXISTS 骨架见
		// workstationOrgExistsSub（Session(NewDB) 派生子查询，不污染外层链）。
		if req.OrgID != "" {
			sub := workstationOrgExistsSub(db, "CAST(f.id AS TEXT) = sys_workstation.floor_id")
			db = db.Where("EXISTS (?)", BuildDeptRecursiveFilter(req.OrgID, workstationOrgColumn)(sub))
		}
		return db
	}
}

// List 查询工位列表（D-05 typed request；CRUD 管道经 base.GORMRepository 执行）。
//
// floorCode 白名单守卫保留在 scope 闭包外（floorTable 为编译期常量，
// validateTableName(floorTable) 恒真——防御死分支按原样保留，错误路径逐字一致）。
//
// 分页口径（V130R-09 D-03-8）：统一走 query.NormalizePagination（cap=
// pkg/constants.MaxPageSize=200）；平面图/3D 全集场景由专用端点
// GET /ops/workstation/:floorId/workstations-all（Plan 99-05）承接，
// List 不再承担全集下拉职责。
func (s *workstationService) List(ctx context.Context, req requests.WorkstationListRequest) (*PageResult, error) {
	// 验证表名是否在白名单中，防止 SQL 注入（保留迁移前守卫与错误文案）
	if req.FloorCode != "" && !validateTableName(floorTable) {
		return nil, fmt.Errorf("invalid table name: %s", floorTable)
	}

	current, pageSize := query.NormalizePagination(req.Current, req.PageSize)
	return s.repo.List(ctx, base.PageParams{Current: current, PageSize: pageSize},
		s.filterScope(req),
		s.joinScope(),
		base.SortScope(req.BaseListRequest, workstationAllowedSortFields, "sys_workstation.created_at DESC"),
	)
}

// GetFloorWorkstationsAll 楼层全部工位（CAD/3D/平面图全集专用端点，无分页；
// V130R-09 D-03-6/D-03-7）。与 List 同一 6 表 JOIN/Select 口径
// （floor_name/dept_name/user_name 等 CAD 视图所需字段照常返回）；
// 软删过滤由 Model 起链自动附带（与 99-01 的 Total 口径一致）。
func (s *workstationService) GetFloorWorkstationsAll(ctx context.Context, floorId string) ([]models.Workstation, error) {
	var workstations []models.Workstation
	if err := s.db.WithContext(ctx).
		Model(&models.Workstation{}).
		Scopes(s.joinScope()).
		Where("sys_workstation.floor_id = ?", floorId).
		Order("sys_workstation.created_at DESC").
		Find(&workstations).Error; err != nil {
		return nil, err
	}
	return workstations, nil
}

func (s *workstationService) BatchDelete(ctx context.Context, ids []string) error {
	// 空 ids 早退已由 repo 承接（91-01 P1 语义反转：空 ids 返回 nil），
	// 与迁移前 `if len(ids) == 0 { return nil }` 行为一致
	return s.repo.BatchDelete(ctx, ids)
}

// BatchUpdatePositions 批量更新工位位置和尺寸
// 遵循 Go 最佳实践：使用 CASE WHEN 批量更新，避免 N+1 问题
func (s *workstationService) BatchUpdatePositions(ctx context.Context, items []PositionUpdateItem) error {
	if len(items) == 0 {
		return nil
	}

	sql, args := buildBatchPositionUpdateSQL(items)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(sql, args...).Error; err != nil {
			return fmt.Errorf("批量更新工位位置失败: %w", err)
		}

		return nil
	})
}

// backfillDeptCandidate 反推候选行（工位 + 有效用户部门）。
type backfillDeptCandidate struct {
	WorkstationID   string  `gorm:"column:workstation_id"`
	WorkstationName string  `gorm:"column:workstation_name"`
	CurrentDeptID   *string `gorm:"column:current_dept_id"`
	CurrentDeptName *string `gorm:"column:current_dept_name"`
	TargetDeptID    string  `gorm:"column:target_dept_id"`
	TargetDeptName  *string `gorm:"column:target_dept_name"`
	UserName        *string `gorm:"column:user_name"`
}

// BackfillDeptFromUser 对存量工位按绑定用户反推部门（quick-261106-wsb）。
//
// 与 Excel 导入兜底 inferDeptFromUser 的判定规则一致，但作用于**已落库**数据：
//   - INNER JOIN sys_dept 保证用户部门有效（过滤 NULL / 部门名称 / 悬空 ID）；
//   - JOIN 两侧 CAST(... AS TEXT)：sys_dept.id 是 uuid、sys_user.dept_id 是
//     varchar(64)，PG 无 uuid = varchar 运算符（migration_217 既有范式）；
//   - 只写 dept_id + updated_at，**绝不碰 status**（存量状态是真实业务状态，
//     与导入 prepareRecordsForUpsert 的 user_id↔status 联动不同）；
//   - dryRun=true 只预览不写库、不清缓存；
//   - fixMismatch=false（默认）只填补空缺；true 额外纠正「已有部门但与用户
//     部门不一致」的行（高危：工位部门≠人员部门可能是有意的跨机构配置，
//     故由前端二次确认 + 默认关闭）。
//
// 双方言：不用 PG 的 UPDATE ... FROM（SQLite 不支持），改为
// 「一次 SELECT 候选 → 内存分类 → 事务内批量 CASE UPDATE」，CASE 的 THEN
// 分支显式 CAST 避免 PG 42804（见 buildBatchPositionUpdateSQL 注释）。
func (s *workstationService) BackfillDeptFromUser(ctx context.Context, dryRun, fixMismatch bool) (*BackfillDeptResult, error) {
	result := &BackfillDeptResult{
		DryRun:      dryRun,
		FixMismatch: fixMismatch,
		Samples:     make([]BackfillDeptSample, 0),
	}

	// 1) 候选工位：有绑定用户且该用户的部门有效。
	var candidates []backfillDeptCandidate
	err := s.db.WithContext(ctx).
		Table("sys_workstation AS w").
		Select(`w.id AS workstation_id, w.workstation_name,
			w.dept_id AS current_dept_id, d2.dept_name AS current_dept_name,
			su.dept_id AS target_dept_id, d.dept_name AS target_dept_name,
			su.nickname AS user_name`).
		Joins("JOIN sys_user su ON CAST(su.id AS TEXT) = CAST(w.user_id AS TEXT)").
		Joins("JOIN sys_dept d ON CAST(d.id AS TEXT) = CAST(su.dept_id AS TEXT)").
		Joins("LEFT JOIN sys_dept d2 ON CAST(d2.id AS TEXT) = CAST(w.dept_id AS TEXT)").
		// ⚠ 空串判断必须 CAST 后再 <> ''：sys_user.dept_id 在生产 PG 是原生 uuid
		// 列（user_service.go:463 的 P4 fix 记载），裸写 su.dept_id <> '' 会让 PG
		// 尝试把 '' 隐式转 uuid → invalid input syntax 运行时报错；SQLite 动态
		// 类型放行，故该类缺陷只有 PG contract 测试能拦（2026-11-07 实测 500）。
		Where("w.deleted_at IS NULL AND su.deleted_at IS NULL AND su.dept_id IS NOT NULL AND CAST(su.dept_id AS TEXT) <> ''").
		Find(&candidates).Error
	if err != nil {
		return nil, fmt.Errorf("查询反推候选工位失败: %w", err)
	}
	result.Candidates = len(candidates)

	// 2) 未命中候选的工位按原因归类：一条条件聚合替代两次全表 COUNT（审计 P2-4）。
	var stats []struct {
		Total    int64
		WithUser int64
	}
	if err := s.db.WithContext(ctx).Table("sys_workstation").
		Select("COUNT(*) AS total, " +
			"COALESCE(SUM(CASE WHEN user_id IS NOT NULL AND CAST(user_id AS TEXT) <> '' THEN 1 ELSE 0 END), 0) AS with_user").
		Where("deleted_at IS NULL").Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("统计工位总数失败: %w", err)
	}
	if len(stats) == 0 {
		return result, nil
	}
	result.SkippedNoUser = int(stats[0].Total - stats[0].WithUser)
	result.SkippedBadUserDept = int(stats[0].WithUser) - len(candidates)

	// 3) 内存分类 + 收集待更新。样例收集策略（审计 P2-5）：fill/correct 优先
	// 占满上限，skip 只在槽位剩余时补位 —— 幂等二跑（ToFill=0）时预览表不再
	// 被 20 行「跳过」挤占，审计目的可见。
	type updateItem struct{ id, deptID string }
	toUpdate := make([]updateItem, 0)
	const maxSamples = 20
	for _, c := range candidates {
		current := ""
		if c.CurrentDeptID != nil {
			current = strings.TrimSpace(*c.CurrentDeptID)
		}
		// 用户部门同样 trim + 忽略大小写比较：UUID 文本在 varchar 列里可能
		// 存在大小写/空格差异，避免 fixMismatch 模式下误判为「不一致」而覆盖。
		target := strings.TrimSpace(c.TargetDeptID)
		action := "skip"
		switch {
		case current == "":
			// dept_id 空（NULL 或空串）→ 填补（两种模式都做）。
			result.ToFill++
			action = "fill"
			toUpdate = append(toUpdate, updateItem{c.WorkstationID, target})
		case !strings.EqualFold(current, target) && fixMismatch:
			// 已有部门但与用户部门不一致，且显式开启纠正 → 覆盖。
			result.ToCorrect++
			action = "correct"
			toUpdate = append(toUpdate, updateItem{c.WorkstationID, target})
		default:
			// 已有部门且一致（或不一致但未开启纠正）→ 跳过，绝不覆盖。
			result.SkippedAlreadyValid++
		}
		// skip 样例只补位：把前 20 个槽位尽量留给 fill/correct（审计 P2-5）。
		if action == "skip" && len(result.Samples) >= maxSamples-1 {
			continue
		}
		if len(result.Samples) < maxSamples {
			sample := BackfillDeptSample{
				WorkstationID:   c.WorkstationID,
				WorkstationName: c.WorkstationName,
				CurrentDeptID:   current,
				TargetDeptID:    target,
				Action:          action,
			}
			if c.CurrentDeptName != nil {
				sample.CurrentDeptName = *c.CurrentDeptName
			}
			if c.TargetDeptName != nil {
				sample.TargetDeptName = *c.TargetDeptName
			}
			if c.UserName != nil {
				sample.UserName = *c.UserName
			}
			result.Samples = append(result.Samples, sample)
		}
	}

	// 4) dryRun 到此为止：不写库、不清缓存、不记 operlog。
	if dryRun || len(toUpdate) == 0 {
		return result, nil
	}

	// 5) 事务内批量 CASE UPDATE，分块走 chunksOf 共享 helper（审计 P2-11）。
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, chunk := range chunksOf(toUpdate, 400) {
			var sb strings.Builder
			// CASE 表达式内两侧 CAST：id 是 uuid、dept_id 是 varchar，PG 无
			// uuid = varchar 运算符；THEN 分支显式 CAST 避免 PG 42804（CASE
			// 结果类型退化为 text）。
			sb.WriteString("UPDATE sys_workstation SET dept_id = CASE CAST(id AS TEXT) ")
			args := make([]interface{}, 0, len(chunk)*2)
			for _, it := range chunk {
				sb.WriteString("WHEN CAST(? AS TEXT) THEN CAST(? AS TEXT) ")
				args = append(args, it.id, it.deptID)
			}
			// WHERE 走参数化 id IN (?)：pgx 按列类型推断把 string 编码为 uuid，
			// 不包 CAST 才能命中 PK 索引（审计 P2-4；包裹 CAST 会全表扫描）。
			sb.WriteString("END, updated_at = CURRENT_TIMESTAMP WHERE id IN (")
			for j := range chunk {
				if j > 0 {
					sb.WriteString(",")
				}
				sb.WriteString("?")
				args = append(args, chunk[j].id)
			}
			sb.WriteString(") AND deleted_at IS NULL")
			if err := tx.Exec(sb.String(), args...).Error; err != nil {
				return fmt.Errorf("批量回填工位部门失败: %w", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 6) 缓存失效必须在事务提交**之后**（cache 写不进事务，与
	// ExcelService.ImportData 的 post-commit 语义一致）。
	if s.cache != nil {
		base.InvalidatePattern(ctx, s.cache, []string{"workstation:*", "workstations:*"}, "workstation")
	}

	return result, nil
}

// buildBatchPositionUpdateSQL 构造批量更新工位位置的 SQL 与参数。
//
// 参数化沿用 F-19(消除 SQL 注入): ID 全部走占位符,按 SQL 中 ? 的出现顺序追加 args。
//
// 【类型安全 — 42804 的根因与修复】
//
// 原写法 `CASE id WHEN ? THEN ? END` 在 PostgreSQL 上必然失败:
// THEN 分支的占位符在 Parse 阶段**无法推断类型** —— CASE 表达式的结果类型
// 由所有 THEN/ELSE 分支共同决定,与赋值目标列(position_x integer)的类型不是
// 同一个上下文,PG 只能把未类型化参数落到 text,于是整个 CASE 退化为 text,
// 赋值时报:
//
//	ERROR: column "position_x" is of type integer but expression is of type text
//	(SQLSTATE 42804)
//
// SQLite 无严格类型推断(动态类型),因此同一段 SQL 在 SQLite 单测里全绿,
// 缺陷只在真实 PostgreSQL 上暴露。
//
// 修复:THEN 分支统一写成 `CAST(? AS int)`。CAST 显式给出参数类型后,
// PG 推断 CASE 结果类型为 integer,int4 列赋值成立;SQLite 同样支持 CAST(… AS int)。
// ID 仍是占位符,F-19 的防注入目标不变。
//
// 抽成独立纯函数的原因:42804 只有 PostgreSQL 会报,SQLite 复现不了,
// 把 SQL 构造独立出来才能写不依赖数据库的回归测试(见
// workstation_position_sql_test.go)。
func buildBatchPositionUpdateSQL(items []PositionUpdateItem) (string, []interface{}) {
	ids := make([]string, 0, len(items))
	args := make([]interface{}, 0, len(items)*12)

	// 必填: position_x / position_y 的 CASE WHEN
	var positionXClause, positionYClause strings.Builder
	positionXClause.WriteString("position_x = CASE id ")
	positionYClause.WriteString("position_y = CASE id ")
	for _, item := range items {
		ids = append(ids, item.ID)
		positionXClause.WriteString("WHEN ? THEN CAST(? AS int) ")
	}
	positionXClause.WriteString("END")
	for range items {
		positionYClause.WriteString("WHEN ? THEN CAST(? AS int) ")
	}
	positionYClause.WriteString("END")

	// 按 SQL 模板中 ? 的出现顺序追加 args
	for _, item := range items {
		args = append(args, item.ID, item.PositionX)
	}
	for _, item := range items {
		args = append(args, item.ID, item.PositionY)
	}

	setClauses := []string{positionXClause.String(), positionYClause.String()}

	// 可选字段:rotation / width / depth / desk_type
	// 提取生成 CASE 子句和 args 的公共逻辑
	appendOptionalClause := func(col string, valuer func(PositionUpdateItem) (int, bool)) {
		var b strings.Builder
		b.WriteString(col + " = CASE id ")
		added := 0
		for _, item := range items {
			if _, ok := valuer(item); ok {
				b.WriteString("WHEN ? THEN CAST(? AS int) ")
				added++
			}
		}
		if added == 0 {
			return
		}
		b.WriteString("ELSE " + col + " END")
		setClauses = append(setClauses, b.String())
		for _, item := range items {
			if v, ok := valuer(item); ok {
				args = append(args, item.ID, v)
			}
		}
	}
	appendOptionalClause("rotation", func(it PositionUpdateItem) (int, bool) {
		if it.Rotation == nil {
			return 0, false
		}
		return *it.Rotation, true
	})
	appendOptionalClause("width", func(it PositionUpdateItem) (int, bool) {
		if it.Width == nil {
			return 0, false
		}
		return *it.Width, true
	})
	appendOptionalClause("depth", func(it PositionUpdateItem) (int, bool) {
		if it.Depth == nil {
			return 0, false
		}
		return *it.Depth, true
	})
	appendOptionalClause("desk_type", func(it PositionUpdateItem) (int, bool) {
		if it.DeskType == nil {
			return 0, false
		}
		return *it.DeskType, true
	})

	// WHERE id IN (?) — GORM 接受 []string 参数展开
	sql := fmt.Sprintf("UPDATE sys_workstation SET %s WHERE id IN ?", strings.Join(setClauses, ", "))
	args = append(args, ids)

	return sql, args
}

// SearchWorkstationOptions 工位下拉数据源(name LIKE 模糊 + floorId/floorCode/status/type/orgId 多维筛选,LIMIT 50)。
// 设计给前端 Select/AutoComplete 远程搜索(原 bug 位置: info-points/index.tsx 「所属工位」下拉)。
//
// 工位高频变更(占用/释放/调位),不缓存避免 staleness。
// D-05: typed request；查询形状逐字保留（Select 两列 + DropdownMaxRows LIMIT + Order），
// 仅 map 读取替换为 req 字段读取（status/type 的 -1 跳过规则与 List 一致，P6）。
func (s *workstationService) SearchWorkstationOptions(ctx context.Context, req requests.WorkstationListRequest) ([]DropdownOption, error) {
	var result []DropdownOption

	query := s.db.WithContext(ctx).Table("sys_workstation").
		Select("sys_workstation.id AS value, sys_workstation.workstation_name AS label").
		Limit(DropdownMaxRows)

	if req.Name != "" {
		query = query.Where("sys_workstation.workstation_name LIKE ?", "%"+req.Name+"%")
	}
	if req.FloorID != "" {
		query = query.Where("sys_workstation.floor_id = ?", req.FloorID)
	}
	if req.FloorCode != "" {
		if !validateTableName(floorTable) {
			return nil, fmt.Errorf("invalid table name: %s", floorTable)
		}
		query = query.Where("EXISTS (SELECT 1 FROM "+floorTable+" WHERE CAST("+floorTable+".id AS TEXT) = sys_workstation.floor_id AND "+floorTable+".floor_no = ?)", req.FloorCode)
	}
	if st := req.GetStatus(-1); st >= 0 {
		query = query.Where("sys_workstation.status = ?", st)
	}
	if req.Type != nil && *req.Type >= 0 {
		query = query.Where("sys_workstation.workstation_type = ?", *req.Type)
	}
	// orgId 部门筛选含子部门:与 List 同款 EXISTS 子查询,避免类型转换问题。
	// 有效机构 = COALESCE(NULLIF(f.org_id, ''), b.org_id)（楼层覆盖优先）。
	if req.OrgID != "" {
		sub := workstationOrgExistsSub(query, "CAST(f.id AS TEXT) = sys_workstation.floor_id")
		query = query.Where("EXISTS (?)", BuildDeptRecursiveFilter(req.OrgID, workstationOrgColumn)(sub))
	}

	if err := query.Order("sys_workstation.workstation_name ASC").Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

// CountsByFloor 批量获取各楼层工位数量(READ,一个 SQL GROUP BY floor_id)。
// 0=正常工位(status=enabled),软删除排除;空 floorIds 返回空 map。
func (s *workstationService) CountsByFloor(ctx context.Context, floorIds []string) (*CountsByFloorResult, error) {
	counts := make(map[string]int)
	for _, id := range floorIds {
		counts[id] = 0 // 初始化:确保所有请求的 floorId 都有键(哪怕 count=0)
	}
	if len(floorIds) == 0 {
		return &CountsByFloorResult{Counts: counts}, nil
	}
	var results []struct {
		FloorID string `gorm:"column:floor_id"`
		Count   int    `gorm:"column:cnt"`
	}
	err := s.db.WithContext(ctx).
		Model(&models.Workstation{}).
		Select("floor_id, COUNT(*) AS cnt").
		Where("floor_id IN ? AND status = ? AND deleted_at IS NULL", floorIds, models.WorkstationStatusAvailable).
		Group("floor_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	for _, r := range results {
		counts[r.FloorID] = r.Count
	}
	return &CountsByFloorResult{Counts: counts}, nil
}
