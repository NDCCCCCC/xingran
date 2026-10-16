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

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models/system/requests"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// pushQueueBatchSize 批量入队分批大小（对齐 internal/services/system/user_sync_service.go
// 的 batchWriteSize = 500，兼顾 PostgreSQL 参数上限）。
const pushQueueBatchSize = 500

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
//  3. 谓词文本与 migration_215 的 uq_ad_push_queue_active 索引谓词文本级一致，
//     且**不加 ad_push_queue. 表名限定**（sqlite 对 partial index 的冲突目标是结构化比较，
//     加限定符会失配）。
//
// 项目内已验证先例：internal/services/asset/reconciliation_detection.go:426。
func adPushOnConflictClause(now time.Time) clause.OnConflict {
	// 活跃行谓词 = uq_ad_push_queue_active 的索引谓词（文本级一致，无表名限定）
	activeRowPredicate := clause.Where{Exprs: []clause.Expression{
		clause.Expr{SQL: "status IN ('pending', 'failed')"},
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
	if err := tx.Clauses(adPushOnConflictClause(time.Now())).Create(&row).Error; err != nil {
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
//   - 重复 id 先去重（保持首次出现顺序）：PG 对同一条 INSERT 内重复命中的行 DO UPDATE
//     会报 "ON CONFLICT DO UPDATE command cannot affect row a second time"
//   - 无启用的 AD 配置 → WARN + 返回 nil
//   - 配置启用 → 批量 upsert，冲突命中即合并（同实体恒一行活跃）
//   - INSERT 失败 → 返回 error，让调用方事务回滚
func (e *ADPushEnqueuer) EnqueueDisable(tx *gorm.DB, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	deduped := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		deduped = append(deduped, id)
	}
	if len(deduped) == 0 {
		return nil
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

	now := time.Now()
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
