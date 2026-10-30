package addomain

// Phase 57 D-01/D-02/D-04/D-06 入队层（outbox-lite 同事务入队）。
//
// ⚠ 使用红线：本文件所有入队函数只在**调用方的业务事务内**使用，必须显式传入 tx。
//   - 禁接收 service 自持的 *gorm.DB（否则入队与业务写不在同一事务，违背 outbox-lite 核心）
//   - 禁 goroutine / 禁 defer 中入队（事务提交后写入即失去原子性）
//   - 前置条件查询（IsADSyncEnabled）必须用同一个 tx，保证与队列表 INSERT 同事务同 snapshot
//     （T-57-02，杜绝"事务外查 → 事务内插"的 TOCTOU 窗口）
//
// 三条语义红线：
//   - D-01 diff 入队：仅 nickname / email / phone / deptId 四字段实际变更才入队；
//     纯角色 / 岗位 / 备注 / 状态编辑不产生队列行。diff 是纯函数（无 DB、无副作用）可单测。
//   - D-02 降级：① 无启用的 AD 配置 → 不入队 + WARN 放行业务编辑；
//     ② 队列表 INSERT 真失败 → 原样返回 error 让调用方事务回滚（禁止"跳过+WARN"降级）。
//   - D-06 合并：合并只由 uq_ad_push_queue_active 部分唯一索引夺锁（ON CONFLICT）实现，
//     禁用"先查后写"应用层合并（net_device_enrichment_task 并发窗口 + 僵尸前科）。

import (
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/models/system/requests"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// pushQueueBatchSize 批量入队分批大小（对齐 internal/services/system/user_sync_service.go
// 的 batchWriteSize = 500，兼顾 PostgreSQL 参数上限）。
const pushQueueBatchSize = 500

// adPushSupersededByDisableReason 被 disable 压制的 update_user 行写入 last_error 的原因串
// （F-01）。相位 58 排水器/排障 SQL 可据此区分"人工/代码压制"与"AD 推送失败"。
const adPushSupersededByDisableReason = "superseded by disable"

// IsADSyncEnabled 前置条件检查（D-02 第一层 / T-57-02）：是否存在
// sync_enabled = true AND status = enabled 的 sys_ad_config 行（软删行自动被作用域排除）。
//
// 必须传入调用方的 tx，与后续队列表 INSERT 同事务同 snapshot。
// 查询错误原样返回（不做"兜底 false"降级），由调用方决定是否回滚业务事务。
func IsADSyncEnabled(tx *gorm.DB) (bool, error) {
	var cnt int64
	if err := tx.Model(&models.ADConfig{}).
		Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).
		Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

// HasADFieldChanged 四字段 diff 纯函数（D-01）。
//
// 参与 diff 的字段仅 nickname / email / phone / deptId —— 与
// internal/api/v1/system/user_handler.go buildADSyncMap 的四字段契约一致。
// 工号 / 性别 / 状态 / 角色 / 岗位 / 备注一律不参与 diff。
//
// 规则：
//   - req 对应字段为 nil → 该字段视为"无变更"，跳过
//   - 非 nil 时与旧值比较，旧值为 nil 视作空串，双方先 TrimSpace 再比较
//   - 任一字段不相等 → return true
//   - 四个字段都无变更 → return false
func HasADFieldChanged(oldUser *models.User, req *requests.UserUpdateRequest) bool {
	if oldUser == nil || req == nil {
		return false
	}
	if adFieldDiffers(oldUser.Nickname, req.Nickname) {
		return true
	}
	if adFieldDiffers(oldUser.Email, req.Email) {
		return true
	}
	if adFieldDiffers(oldUser.Phone, req.Phone) {
		return true
	}
	if adFieldDiffers(oldUser.DeptID, req.DeptID) {
		return true
	}
	return false
}

// adFieldDiffers 单字段比较：req 值为 nil = 无变更（false）；否则 TrimSpace 后比较。
func adFieldDiffers(oldVal *string, newVal *string) bool {
	if newVal == nil {
		return false
	}
	oldStr := ""
	if oldVal != nil {
		oldStr = *oldVal
	}
	return strings.TrimSpace(oldStr) != strings.TrimSpace(*newVal)
}

// DedupeEntityIDs 实体 id 列表去重纯函数（QUEUE-06 前置）。
//
// 规则：丢弃空串（脏入参不应占队列行），其余按**首次出现顺序**保留 —— 顺序稳定让
// 批量 INSERT 的入参与排障 SQL 的 entity_id 顺序一致。
//
// 为什么要抽成纯函数（F-05）：去重唯一的真实故障模式是 PG 的
// "ON CONFLICT DO UPDATE command cannot affect row a second time"（同一条 INSERT 内
// 重复命中冲突目标行）。sqlite 对该情形不报错，会逐行处理并合并 —— 也就是说，
// 直接对 EnqueueDisable 断言"行数 == 去重后个数"，在 sqlite 上把去重循环整个删掉也照样
// 变绿（假断言，其 RED 状态不可达）。把逻辑提成可直接单测的纯函数，
// TestPhase57_DedupeEntityIDs_PureFunction 才是真正能红的断言。
func DedupeEntityIDs(ids []string) []string {
	deduped := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}
	return deduped
}

// ADPushEnqueuer AD 推送入队器（无状态，零参构造）。
type ADPushEnqueuer struct{}

// NewADPushEnqueuer 创建入队器。无状态，可安全并发复用。
func NewADPushEnqueuer() *ADPushEnqueuer {
	return &ADPushEnqueuer{}
}

// adPushOnConflictClause 构造 QUEUE-02 合并子句（D-06）。
//
// 三重防线（错任何一条合并语义即彻底失效，Task 2 的
// TestPhase57_EnqueueUpdate_MergeOnSecondEdit 会直接红）：
//
//  1. 谓词必须放 **TargetWhere**（gorm v1.30.5 clause/on_conflict.go:35，冲突目标推断谓词），
//     而非 Where（:49，DO UPDATE 的尾随谓词）。放 Where → 部分唯一索引不被推断 →
//     PG 报 42P10 / sqlite 报 "ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint"。
//  2. 裸 SQL 表达式类型是 **clause.Expr**（clause.Literal 在 gorm v1.30.5 中并不存在）。
//  3. 谓词文本取自单一事实源 models.ADPushQueueActivePredicate（migration_215 的
//     uq_ad_push_queue_active 索引 DDL 复用同一常量），
//     且**不加 ad_push_queue. 表名限定**（sqlite 对 partial index 的冲突目标是结构化比较，
//     加限定符会失配）。
//
// 项目内已验证先例：internal/services/asset/reconciliation_detection.go:426。
func adPushOnConflictClause(now time.Time) clause.OnConflict {
	// 活跃行谓词 = uq_ad_push_queue_active 的索引谓词（同一常量，文本级一致，无表名限定）
	activeRowPredicate := clause.Where{Exprs: []clause.Expression{
		clause.Expr{SQL: models.ADPushQueueActivePredicate},
	}}

	return clause.OnConflict{
		Columns: []clause.Column{
			{Name: "entity_type"},
			{Name: "entity_id"},
			{Name: "action_type"},
		},
		TargetWhere: activeRowPredicate,
		// 合并语义：把命中的 failed 行打回 pending 并清空退避状态，
		// 让 Phase 58 排水器重新拾取（D-06 + D-07）；created_at/updated_at 刷新为
		// 最近一次编辑时间（队列表按 created_at 排序时体现"最后一次编辑"）。
		DoUpdates: clause.Assignments(map[string]interface{}{
			"status":        string(models.ADPushStatusPending),
			"attempts":      0,
			"last_error":    gorm.Expr("NULL"),
			"next_retry_at": gorm.Expr("NULL"),
			"failed_at":     gorm.Expr("NULL"),
			"created_at":    now,
			"updated_at":    now,
		}),
	}
}

// upsert 入队单行（D-06：合并完全交给部分唯一索引夺锁，无先查后写）。
// 出错时打 ERROR 日志并原样返回 error（D-02 第二层：禁止吞错降级）。
func (e *ADPushEnqueuer) upsert(tx *gorm.DB, action models.ADPushActionType, entityID string) error {
	row := models.ADPushQueue{
		EntityType: models.ADPushEntityUser,
		EntityID:   entityID,
		ActionType: action,
		Status:     models.ADPushStatusPending,
	}
	// Phase 58 F-12：入队路径时间戳统一 UTC（排水器是 pushed_at/failed_at 的首个写入方，
	// 若写入本地时区而 PG 列是 timestamptz，排障 SQL 会出现 8 小时偏移）。
	if err := tx.Clauses(adPushOnConflictClause(time.Now().UTC())).Create(&row).Error; err != nil {
		applogger.Errorf("[ad_push_queue] 入队失败 entity_type=%s entity_id=%s action=%s: %v",
			models.ADPushEntityUser, entityID, action, err)
		return err
	}
	return nil
}

// EnqueueUpdateIfChanged 用户编辑入队入口（D-01 diff + D-02 两层降级）。
//
// oldUser = 调用方在事务内 UPDATE 之前加载的旧值快照；req = 本次编辑请求。
//
// 行为：
//   - 四字段无变更 → 直接返回 nil（连 sys_ad_config 查询都不发）
//   - 有变更但无启用的 AD 配置 → WARN + 返回 nil（放行业务编辑）
//   - 有变更且配置启用 → upsert 一行 action=update_user
//   - 队列表 INSERT 失败 → 返回 error，让调用方事务回滚
func (e *ADPushEnqueuer) EnqueueUpdateIfChanged(tx *gorm.DB, oldUser *models.User, req *requests.UserUpdateRequest) error {
	if !HasADFieldChanged(oldUser, req) {
		return nil
	}
	enabled, err := IsADSyncEnabled(tx)
	if err != nil {
		return err
	}
	if !enabled {
		applogger.Warnf("[ad_push_queue] 无启用的 AD 配置(sync_enabled=true AND status=enabled)，跳过入队: entity_id=%s action=update_user",
			oldUser.ID)
		return nil
	}
	return e.upsert(tx, models.ADPushActionUpdateUser, oldUser.ID)
}

// EnqueueDisableOnStatusTransition 用户**被停用**（status 0→1）时入队 disable 动作行（F-02 / GAP-01）。
//
// 为什么需要它：停用有**两个**入口，两个都必须接，否则留下安全语义缺口 ——
// 系统里用户已停用、AD 账号仍启用，域内照常可登录。
//   - Update()          —— 用户管理**编辑表单**保存时携带 status（F-02 修复时接入）
//   - UpdateStatus()    —— 用户列表页 **Switch 开关**（POST /system/users/:id/status）
//     ← 列表页开关才是最高频的停用入口，GAP-01 修复前未接入；两个入口现已同时接线。
//
// 与 EnqueueDisable 的区别：
//   - 不做 F-01 压制。用户行仍在（只是 status=1），残留的 update_user 行仍可被排水器
//     正常重读推送，两条动作按 created_at 先后处理（先推属性、再禁用）语义正确；
//   - 语义仅限"停用"：0→1 入队；1→1 / 0→0 无操作。**启用**（1→0）刻意不入队 ——
//     models.ADPushActionEnable 在 v1.21 不存在（见 ad_push_queue.go 常量注释）。
//
// D-02 降级与 EnqueueDisable 一致：队列表写失败返回 error 让调用方事务回滚；
// 无启用的 AD 配置则 WARN + 放行。
func (e *ADPushEnqueuer) EnqueueDisableOnStatusTransition(tx *gorm.DB, oldStatus, newStatus models.UserStatus, userID string) error {
	if userID == "" || oldStatus == newStatus || newStatus != models.UserStatusDisabled {
		return nil
	}
	enabled, err := IsADSyncEnabled(tx)
	if err != nil {
		return err
	}
	if !enabled {
		applogger.Warnf("[ad_push_queue] 无启用的 AD 配置(sync_enabled=true AND status=enabled)，跳过停用入队: entity_id=%s", userID)
		return nil
	}
	return e.upsert(tx, models.ADPushActionDisable, userID)
}

// EnqueueDisable 软删用户入队 disable 动作行（QUEUE-06 / D-04）。
//
// 由 Delete 与 BatchDelete 两条软删事务路径调用。Phase 58 排水器按
// entity_id → sys_user.ad_dn 路由到 ldap_client.go:337 DisableUser（UAC 514 Replace，
// 幂等可重放）。Phase 57 只入队，不做任何 LDAP 写。
//
// ⚠ 禁止改为 delete AD 账号：addomain 全库无 DeleteUser 先例，delete 销毁 group membership
// 且不可逆恢复，与 Phase 60 SAFE-01 的 ad_dn 校验恢复不兼容（D-04）。
//
// 行为：
//   - 空切片 → 直接返回 nil
//   - 重复 id 先去重（DedupeEntityIDs，保持首次出现顺序）：PG 对同一条 INSERT 内重复
//     命中的行 DO UPDATE 会报 "ON CONFLICT DO UPDATE command cannot affect row a second time"
//   - **压制同实体活跃的 update_user 行**（F-01）
//   - 无启用的 AD 配置 → WARN + 返回 nil
//   - 配置启用 → 批量 upsert，冲突命中即合并（同实体恒一行活跃）
//   - INSERT 失败 → 返回 error，让调用方事务回滚
func (e *ADPushEnqueuer) EnqueueDisable(tx *gorm.DB, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	deduped := DedupeEntityIDs(userIDs)
	if len(deduped) == 0 {
		return nil
	}

	now := time.Now().UTC() // Phase 58 F-12：全链路 UTC

	// F-01 压制：让"删除"在队列层面对"编辑"形成显式压制。
	//
	// 部分唯一索引的一维是 action_type，故同一用户可同时存在 pending update_user 与
	// pending disable。sys_user 行被软删后，Phase 58 排水器处理残留的 update_user 行时会
	// 按 D-05 重读 sys_user 全量推送 → 默认软删作用域读不到该行 → 要么永远推不出去（僵尸行），
	// 要么读到零值后把 nickname/email/phone/dept 清空写进 AD 账号（破坏性）。
	// 故在入队 disable 的同一事务里把该实体所有活跃 update_user 行打终态。
	//
	// 为什么用 dead 而不新增 superseded 状态：终态必须退出
	// uq_ad_push_queue_active 的索引谓词 (models.ADPushQueueActivePredicate =
	// status IN ('pending','failed'))，否则被打死的行仍占用唯一索引，同实体的新
	// disable 行直接撞唯一约束。dead 是既有终态，谓词一行不动 —— DDL 与 ON CONFLICT
	// TargetWhere 复用的常量不因本修复而变化（漂移由 TestPhase57_ActiveRowPredicate_MatchesIndexDDL 守护）。
	//
	// 为什么放在 D-02 配置门控之前：压制的是"已失效的队列行"这一安全性不变量，与"要不要
	// 入队 disable"的降级策略无关。若 AD 配置当时关闭而跳过压制，AD 重新启用后排水器仍会
	// 读到这行并复现同一破坏性故障。
	if err := tx.Model(&models.ADPushQueue{}).
		Where("entity_type = ? AND entity_id IN ? AND action_type = ? AND status IN ?",
			string(models.ADPushEntityUser), deduped, string(models.ADPushActionUpdateUser),
			[]string{string(models.ADPushStatusPending), string(models.ADPushStatusFailed)},
		).
		Updates(map[string]interface{}{
			"status":        string(models.ADPushStatusDead),
			"last_error":    adPushSupersededByDisableReason,
			"next_retry_at": nil,
			"failed_at":     nil,
			"updated_at":    now,
		}).Error; err != nil {
		applogger.Errorf("[ad_push_queue] 压制 update_user 行失败 (%d 个实体): %v", len(deduped), err)
		return err
	}

	enabled, err := IsADSyncEnabled(tx)
	if err != nil {
		return err
	}
	if !enabled {
		applogger.Warnf("[ad_push_queue] 无启用的 AD 配置(sync_enabled=true AND status=enabled)，跳过 disable 入队: %d 个用户",
			len(deduped))
		return nil
	}

	rows := make([]models.ADPushQueue, 0, len(deduped))
	for _, id := range deduped {
		rows = append(rows, models.ADPushQueue{
			EntityType: models.ADPushEntityUser,
			EntityID:   id,
			ActionType: models.ADPushActionDisable,
			Status:     models.ADPushStatusPending,
		})
	}
	if err := tx.Clauses(adPushOnConflictClause(now)).CreateInBatches(&rows, pushQueueBatchSize).Error; err != nil {
		applogger.Errorf("[ad_push_queue] 批量 disable 入队失败 (%d 行): %v", len(rows), err)
		return err
	}
	return nil
}

// EnqueueUpdateBatch 批量入队 update_user（D-01 导入路径专用变体）。
//
// ⚠ 与 EnqueueUpdateIfChanged 的语义差异（务必读完再调用）：
// EnqueueUpdateIfChanged 做**四字段 diff**（nickname/email/phone/deptId），只有实际变更才入队；
// 本变体**不做 diff** —— 导入场景没有"导入前旧值快照"（被替换掉的
// BatchSyncUsersToAD 同样在注释里承认拿不到旧值所以不做 OU 移动），
// "全量入队"才是正确语义：排水器闭包会重读 DB 后推最新值。
//
// 行为链（与 EnqueueDisable 同款，只是少了 F-01 压制那一步）：
//   - 空切片 → 直接返回 nil（连 sys_ad_config 查询都不发）
//   - 重复 id 先去重（DedupeEntityIDs，保持首次出现顺序）：PG 对同一条 INSERT 内重复
//     命中的行 DO UPDATE 会报 "ON CONFLICT DO UPDATE command cannot affect row a second time"
//   - 无启用的 AD 配置 → WARN + 返回 nil（放行导入，D-02 第一层）
//   - 配置启用 → 批量 upsert，冲突命中即合并（同实体恒一行活跃，adPushOnConflictClause 单一句面）
//   - INSERT 失败 → 原样返回 error，由调用方决定回滚还是降级（禁吞错）
//
// ⚠ 不做 F-01 压制：压制只发生在 disable 打 update_user 上，update 与 update 之间不存在压制关系。
//
// ⚠ 唯一调用方是 api/v1/system/user_import_handler.go 的 enqueueImportUsersToAD，
//   它自开短事务（ExcelService 的 Transaction 已提交）—— 这是 outbox-lite
//   「同生共死」不变量的**一处已接受例外**，详见调用点注释。
func (e *ADPushEnqueuer) EnqueueUpdateBatch(tx *gorm.DB, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	deduped := DedupeEntityIDs(userIDs)
	if len(deduped) == 0 {
		return nil
	}

	enabled, err := IsADSyncEnabled(tx)
	if err != nil {
		return err
	}
	if !enabled {
		applogger.Warnf("[ad_push_queue] 无启用的 AD 配置(sync_enabled=true AND status=enabled)，跳过批量 update 入队: %d 个用户",
			len(deduped))
		return nil
	}

	now := time.Now().UTC() // Phase 58 F-12：全链路 UTC

	rows := make([]models.ADPushQueue, 0, len(deduped))
	for _, id := range deduped {
		rows = append(rows, models.ADPushQueue{
			EntityType: models.ADPushEntityUser,
			EntityID:   id,
			ActionType: models.ADPushActionUpdateUser,
			Status:     models.ADPushStatusPending,
		})
	}
	if err := tx.Clauses(adPushOnConflictClause(now)).CreateInBatches(&rows, pushQueueBatchSize).Error; err != nil {
		applogger.Errorf("[ad_push_queue] 批量 update 入队失败 (%d 行): %v", len(rows), err)
		return err // 原样上抛，调用方决定是否降级
	}
	applogger.Infof("[ad_push_queue] 批量 update 入队成功: %d 行", len(rows))
	return nil
}

// EnqueueUpdateManager 经理重算入队入口：为一批用户 upsert action=update_manager 行。
//
// 触发点（同事务调用，QUEUE-01 同生共死）：
//   - user_service.Update：用户部门/岗位变更（经理解析的输入变了）
//   - department_service.Update：部门 leader 变更（dept_leader 来源的经理可能变）
//
// 行为与 EnqueueUpdateBatch 一致（去重 / D-02 两层降级 / 冲突合并），仅 action 不同。
func (e *ADPushEnqueuer) EnqueueUpdateManager(tx *gorm.DB, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	deduped := DedupeEntityIDs(userIDs)
	if len(deduped) == 0 {
		return nil
	}

	enabled, err := IsADSyncEnabled(tx)
	if err != nil {
		return err
	}
	if !enabled {
		applogger.Warnf("[ad_push_queue] 无启用的 AD 配置，跳过 update_manager 入队: %d 个用户", len(deduped))
		return nil
	}

	now := time.Now().UTC()
	rows := make([]models.ADPushQueue, 0, len(deduped))
	for _, id := range deduped {
		rows = append(rows, models.ADPushQueue{
			EntityType: models.ADPushEntityUser,
			EntityID:   id,
			ActionType: models.ADPushActionUpdateManager,
			Status:     models.ADPushStatusPending,
		})
	}
	if err := tx.Clauses(adPushOnConflictClause(now)).CreateInBatches(&rows, pushQueueBatchSize).Error; err != nil {
		applogger.Errorf("[ad_push_queue] update_manager 入队失败 (%d 行): %v", len(rows), err)
		return err
	}
	applogger.Infof("[ad_push_queue] update_manager 入队成功: %d 行", len(rows))
	return nil
}
