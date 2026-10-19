package addomain

// Phase 58 D-05 排水器。in-process 定时调用（cron 接线在 internal/scheduler/ad_sync_tasks.go，
// 禁注册 sys_job —— 里程碑锁定）。禁止自持账号池：必须由调用方注入
// （研究 Pitfall 4：独立缓存 → 熔断后仍选到坏账号）。
//
// 执行面（本 plan 范围）：
//   - 拾取：status IN ('pending','failed') AND (next_retry_at IS NULL OR next_retry_at <= now)
//   - 分组：按 sys_user.ad_dn 归属解析 AD 配置（D-02），每组一次账号池故障切换
//   - 推送：disable（UAC 514）与 update_user（属性）
//   - 状态机：pending/failed → pushed | failed(退避) | dead（终态留表可查）
//
// ⚠ OU 移动（compare-and-move）是 Phase 58 Plan 04 的内容，本文件只预留接口位。

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

const (
	// drainBatchLimit 单轮批量上限（D-05）。LDAP 单写 10-100ms → 单轮 1-10s 天然有界。
	drainBatchLimit = 100
	// maxDrainAttempts 连续失败达此次数转 dead（D-05）。总重试窗 ~61min，
	// 刻意桥接账号池 30min 熔断周期（避免熔断窗内密集重试继续喂 MarkFailure）。
	maxDrainAttempts = 5
	// backoffCap 退避封顶（D-05）。
	backoffCap = 30 * time.Minute
)

// backoffDuration 退避曲线纯函数（Phase 58 D-05：60s→2min→4min→8min→16min，封顶 30min）。
// 纯函数：无 DB、无副作用、可单测。attempts 为"本轮失败后的累计连续失败次数"。
// attempts <= 0 → 60s；1→60s，2→2min，3→4min，4→8min，5→16min，>=6→30min（封顶）。
func backoffDuration(attempts int) time.Duration {
	switch attempts {
	case 1:
		return 60 * time.Second
	case 2:
		return 2 * time.Minute
	case 3:
		return 4 * time.Minute
	case 4:
		return 8 * time.Minute
	case 5:
		return 16 * time.Minute
	default:
		if attempts < 1 {
			return 60 * time.Second
		}
		return backoffCap
	}
}

// drainDeadRow 在"配置分组阶段"就被判定为不可恢复的队列行，附带终态原因。
//
// 为什么不用裸 models.ADPushQueue：deadRows 内的三类原因（user-not-found /
// no-ad-dn / no-ad-config）必须分别落库，否则排障 SQL 无法区分根因。
type drainDeadRow struct {
	models.ADPushQueue
	reason string
}

// ADPushDrainer AD 推送队列排水器（Phase 58）。
type ADPushDrainer struct {
	db   *gorm.DB
	pool AccountPool
	// now 测试注入点；生产恒为 time.Now。⚠ 一轮只取一次并透传（P8），
	// 闭包内多次取时间会让 pushed_at / next_retry_at 与行内容不对应。
	now func() time.Time
	// mu 轮次互斥的**事实源**：cron 的 SkipIfStillRunning 互斥是闭包私有 channel，
	// 无法被手动触发路径复用，故此处自持（Go >=1.18 提供 TryLock）。
	mu sync.Mutex
	// adSync 复用 UserADSyncService 的 syncUserAttributes（属性推送 + code 32 语义，
	// 不复制粘贴 —— P11）。mapper 供 Plan 04 的 OU 映射查询使用，本 plan 先注入。
	adSync *UserADSyncService
	mapper *DeptOUmapper
	// clientFactoryOverride **仅测试用**的 LDAP 客户端工厂注入点（未导出，不构成生产 API 面）。
	//
	// 为什么必须在本 plan（wave 2）就位、不能推迟到 Plan 07：executeGroup 内
	// NewFailoverClient(...) 构造出的 FailoverClient，其 clientFactory 恒为 nil →
	// newClient 走 NewLDAPClient **真拨号**；而 clientFactory 是未导出字段，外部拿不到 fc
	// 句柄，唯一可注入处就是这里。若推迟，本 plan Task 2 的 12 个与 Plan 04 的 10 个
	// DrainOnce 用例会真连域控并落 MarkFailure 而必红。生产路径恒 nil → 行为不变。
	clientFactoryOverride func(*models.ADConfig, *models.ADServiceAccount) LDAPClientIface
}

// NewADPushDrainer 构造排水器。⚠ 禁在本函数内 NewAccountPool —— 调用方必须传
// 全局账号池单例（dept_sync_tasks.go:16-21 的同款约束）。
func NewADPushDrainer(db *gorm.DB, pool AccountPool) *ADPushDrainer {
	mapper := NewDeptOUmapper(db)
	return &ADPushDrainer{
		db:     db,
		pool:   pool,
		now:    func() time.Time { return time.Now().UTC() },
		mapper: mapper,
		adSync: NewUserADSyncService(db, pool, nil, mapper),
	}
}

// DrainOnce 执行一轮排水（D-05）。cron 与手动触发共用本入口，互斥在内部统一处理。
// 返回 error 只表示"本轮基础设施层失败"（如查库失败），行级失败一律写回队列状态、不上抛。
func (d *ADPushDrainer) DrainOnce(ctx context.Context) error {
	if !d.mu.TryLock() { // 上一轮未结束，本轮跳过（30s 间隔下单轮可能超 30s）
		applogger.Debugf("[ad_push_queue] 上一轮排水未结束，本轮跳过")
		return nil
	}
	defer d.mu.Unlock()
	now := d.now() // 一轮一次，全轮透传（P8）

	rows, err := d.pickRows(ctx, now)
	if err != nil {
		return fmt.Errorf("拾取待排水行失败: %w", err)
	}
	if len(rows) == 0 {
		return nil
	}

	// 按解析出的 AD 配置分组（P12：每组一次账号池故障切换，而不是每行一次 bind）
	groups, deadRows, err := d.groupByResolvedConfig(ctx, rows)
	if err != nil {
		return fmt.Errorf("解析 AD 配置分组失败: %w", err)
	}
	// deadRows 只含"单用户匹配不到配置 / 无 ad_dn / 用户已不存在"三类；
	// "全局零启用配置"在 groupByResolvedConfig 内部已 defer 成空结果，不会走到这里。
	for _, row := range deadRows {
		// 不可恢复分支**不烧 attempts**：传 row.Attempts 原值写回（不 +1）
		d.markDead(ctx, row.ID, now, row.Attempts, row.reason)
	}
	for cfgID, group := range groups {
		if err := d.executeGroup(ctx, cfgID, group, now); err != nil {
			applogger.Warnf("[ad_push_queue] 配置组 %s 本轮执行异常: %v", cfgID, err)
		}
	}
	return nil
}

// pickRows 拾取本轮待排水行（D-05）。
//
// 命中 idx_ad_push_queue_status_retry；**行级不认领**（不引入 processing 状态 ——
// 单机部署 + 本结构体的 TryLock 已覆盖轮次重叠；且任何中间态都绝不能进
// ADPushQueueActivePredicate，否则占用部分唯一索引阻塞同实体新入队，命中即 42P10）。
func (d *ADPushDrainer) pickRows(ctx context.Context, now time.Time) ([]models.ADPushQueue, error) {
	var rows []models.ADPushQueue
	err := d.db.WithContext(ctx).
		Where(models.ADPushQueueActivePredicate).                  // 只读红线常量（P7）
		Where("next_retry_at IS NULL OR next_retry_at <= ?", now). // NULL = 立即可拾取（D-05），该分支不可省
		Order("created_at ASC").                                   // F-01 压制后仍按入队先后
		Limit(drainBatchLimit).
		Find(&rows).Error
	return rows, err
}

// groupByResolvedConfig 按 ad_dn 归属把行分到 AD 配置（D-02）。
// 第二个返回值是"解析不出配置"的行 —— 契约规定它们转 dead 不推（无匹配/多匹配/无 ad_dn/用户已不存在）。
func (d *ADPushDrainer) groupByResolvedConfig(ctx context.Context, rows []models.ADPushQueue) (map[string][]models.ADPushQueue, []drainDeadRow, error) {
	var configs []models.ADConfig
	if err := d.db.WithContext(ctx).
		Where("status = ?", models.ADConfigStatusEnabled).
		Find(&configs).Error; err != nil {
		return nil, nil, err
	}

	// Q3(i)：全局零启用配置 → defer，不动任何行。
	// ⚠ 与"单用户 ad_dn 匹配不到"（Q3(ii)，转 dead）必须区分：配置可能被临时停用，
	// 此时把整批行打成 dead 会误伤已入队的推送。两种情形不得合并处理。
	if len(configs) == 0 {
		applogger.Warnf("[ad_push_queue] 无启用 AD 配置，本轮 defer（%d 行留待下轮）", len(rows))
		return nil, nil, nil
	}

	groups := make(map[string][]models.ADPushQueue)
	var deadRows []drainDeadRow
	for _, row := range rows {
		user, err := d.loadUserUnscoped(ctx, row.EntityID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// F-07 孤儿行：与 code 32 同一条终态机制（不可恢复 → dead）。
				deadRows = append(deadRows, drainDeadRow{ADPushQueue: row, reason: "unrecoverable:user-not-found"})
				continue
			}
			// 读库失败属基础设施层问题：本轮跳过该行（不烧 attempts，下轮仍可拾取）。
			applogger.Warnf("[ad_push_queue] 读取用户 %s 失败，本轮跳过: %v", row.EntityID, err)
			continue
		}

		if user.AdDn == nil || *user.AdDn == "" {
			// 不可恢复：⚠ 不可当成成功，否则 Phase 59 守卫会误判"无待推送 → 一致"。
			deadRows = append(deadRows, drainDeadRow{ADPushQueue: row, reason: "unrecoverable:no-ad-dn"})
			continue
		}

		cfg, err := MatchADConfigByDN(*user.AdDn, configs)
		if err != nil {
			// D-02 契约：无匹配 / 歧义 → dead（不推）。
			deadRows = append(deadRows, drainDeadRow{ADPushQueue: row, reason: "unrecoverable:no-ad-config"})
			continue
		}
		groups[cfg.ID] = append(groups[cfg.ID], row)
	}

	return groups, deadRows, nil
}

// loadUserUnscoped 读取用户。
// ⚠ P2 红线：必须 Unscoped —— disable 行对应软删用户，作用域查询读不到会永久失败烧 attempts。
func (d *ADPushDrainer) loadUserUnscoped(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	err := d.db.WithContext(ctx).Unscoped().Where("id = ?", userID).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// executeGroup 一个 AD 配置一次 bind，闭包内串行处理该组全部行（P12）。
//
// ⚠ P1 红线：闭包内**吞掉行级错误、恒返回 nil** —— 返回 error 会让账号池故障切换
// 上报 MarkFailure("operation:...") 累加并换账号**重跑整组**（一次 Modify 失败被放大成
// 重复 LDAP 写 + 3 次即熔断 30min，前科 ad-modify-fail-double-counts-breaker）。
func (d *ADPushDrainer) executeGroup(ctx context.Context, cfgID string, group []models.ADPushQueue, now time.Time) error {
	var cfg models.ADConfig
	if err := d.db.WithContext(ctx).Where("id = ?", cfgID).First(&cfg).Error; err != nil {
		// 配置在分组后被删/改状态：该组全 dead（不可恢复，不烧 attempts → 传原值）。
		for _, row := range group {
			d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:no-ad-config")
		}
		return nil
	}

	fc := NewFailoverClient(d.pool, &cfg)
	// 测试注入点（生产恒 nil → 走 newClient 的 NewLDAPClient 真拨号，行为不变）。
	// ⚠ 必须构造之后、故障切换执行之前赋值，否则 DrainOnce 无法零网络测试。
	if d.clientFactoryOverride != nil {
		fc.clientFactory = d.clientFactoryOverride
	}

	err := fc.ExecuteWithFailover(ctx, func(c LDAPClientIface) error {
		for _, row := range group {
			d.drainRow(ctx, c, row, cfgID, now) // 行级结果写回队列，上抛即破坏 P1
		}
		return nil // ← 关键：恒 nil
	})
	if err != nil {
		// 空账号列表（池熔断/未配置）在调用闭包之前返回 → 可重试，不直接 dead。
		poolExhausted := errors.Is(err, ErrAllAccountsUnavailable)
		for _, row := range group {
			d.markFailed(ctx, row, now, poolExhausted, "pool:"+err.Error())
		}
	}
	return nil
}

// markPushed 成功终态。
// ⚠ next_retry_at / failed_at 显式写 nil（清空）—— 用 map 形式而非 struct 形式，
// 否则零值语义混乱（struct 的零值会被 GORM 忽略为"不变"）。
func (d *ADPushDrainer) markPushed(ctx context.Context, id string, now time.Time) error {
	return d.db.WithContext(ctx).Model(&models.ADPushQueue{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        string(models.ADPushStatusPushed),
			"pushed_at":     now,
			"next_retry_at": nil,
			"failed_at":     nil,
			"last_error":    "",
			"updated_at":    now,
		}).Error
}

// markFailed 可重试失败：attempts+1 + 退避。
//
// ⚠ 池熔断（ErrAllAccountsUnavailable）不判 dead（D-05：不可恢复错误才短路；
// 熔断是可恢复的，退避窗正是为它设计的）。
func (d *ADPushDrainer) markFailed(ctx context.Context, row models.ADPushQueue, now time.Time, poolExhausted bool, msg string) error {
	newAttempts := row.Attempts + 1
	// F-11 红线：dead 判定**只能**发生在本轮实际失败分支内，且按 attempts+1 >= maxDrainAttempts。
	// attempts 的语义是"自上次编辑以来的连续失败次数"（Phase 57 合并子句每次编辑清零 attempts），
	// 因此它是正确的连续失败计数器 —— 但**禁止**在拾取阶段 / 成功分支 / 任何非失败路径上
	// 读 attempts 判死。
	// 已知并接受的后果：用户持续编辑某个持续失败的属性时，每次编辑清零 attempts，该行永不
	// 终态（"编辑即续命"）。这是有意设计 —— 用户还在管这件事，反复重试是期望行为。
	if !poolExhausted && newAttempts >= maxDrainAttempts {
		// ⚠ 传 newAttempts：本次失败已计入连续失败计数，DB 必须落到 5。
		return d.markDead(ctx, row.ID, now, newAttempts, "dead:max-attempts")
	}
	return d.db.WithContext(ctx).Model(&models.ADPushQueue{}).Where("id = ?", row.ID).
		Updates(map[string]interface{}{
			"status":        string(models.ADPushStatusFailed),
			"attempts":      newAttempts,
			"next_retry_at": now.Add(backoffDuration(newAttempts)),
			"failed_at":     now,
			"last_error":    msg,
			"updated_at":    now,
		}).Error
}

// markDead 不可恢复终态（留表可查，QUEUE-03）。
//
// ⚠ attempts 必须显式入参（否则达上限用例断言 attempts == 5 不可达）：
//   - 不可恢复分支传 row.Attempts（**不 +1**，D-05「不可恢复不烧 attempts」）；
//   - 达上限分支（markFailed 内的连续失败阈值判定命中时）传新计数。
//
// reason 的可 grep 前缀约定：unrecoverable:* / dead:* / retry:* / pool:*。
func (d *ADPushDrainer) markDead(ctx context.Context, id string, now time.Time, attempts int, reason string) error {
	return d.db.WithContext(ctx).Model(&models.ADPushQueue{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        string(models.ADPushStatusDead),
			"attempts":      attempts,
			"next_retry_at": nil,
			"failed_at":     now,
			"last_error":    reason,
			"updated_at":    now,
		}).Error
}

// drainRow 处理单行。⚠ 行级错误一律写回队列状态、绝不上抛（P1：上抛会触发
// 账号池故障切换换账号重跑整组 + MarkFailure 累加熔断）。
// ⚠ 签名已含 cfgID，与 Plan 04 Task 2 终稿**同一形态** —— Plan 04 只把标记行
// 替换为真实 compare-and-move 调用，不再动签名与调用点。
func (d *ADPushDrainer) drainRow(ctx context.Context, c LDAPClientIface, row models.ADPushQueue, cfgID string, now time.Time) {
	user, err := d.loadUserUnscoped(ctx, row.EntityID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// F-07 孤儿，与 code 32 同一条分支；不可恢复 → 不烧 attempts（传原值）。
			d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:user-not-found")
			return
		}
		d.markFailed(ctx, row, now, false, "retry:读取用户失败 "+err.Error())
		return
	}
	if user.AdDn == nil || *user.AdDn == "" {
		// 不可恢复：⚠ 不可当成成功（否则 Phase 59 守卫误判"无待推送 → 一致"）。
		d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:no-ad-dn")
		return
	}

	switch row.ActionType {
	case models.ADPushActionDisable:
		// UAC 514 Replace，幂等可重放；⚠ 禁 delete AD 账号（Phase 57 D-04 红线：
		// 销毁 group membership 且与 Phase 60 SAFE-01 不兼容）。
		if err := c.DisableUser(*user.AdDn); err != nil {
			d.handlePushError(ctx, row, now, err, "disable")
			return
		}
		d.markPushed(ctx, row.ID, now)

	case models.ADPushActionUpdateUser:
		// 闭包内重读到的最新值（不依赖入队时刻的旧值 —— compare-and-move 的前提）。
		updateReq := buildDrainUpdateReq(user)
		// 属性推送（复用 syncUserAttributes：DNExists 预检 + code 32 清 ad_dn 已封装）。
		if err := d.adSync.syncUserAttributes(ctx, c, user, updateReq); err != nil {
			d.handlePushError(ctx, row, now, err, "update_user")
			return
		}
		// ⚠ syncUserAttributes 在 DNExists=false 时清空 ad_dn 后 **return nil**（不报错）。
		//   故必须重新 loadUserUnscoped 判空：目标在 AD 端不存在时终态是 dead 而不是
		//   pushed（QUEUE-05）。缺这一步 → DNExists=false 用例会把"AD 端不存在"误报成功。
		if refreshed, err := d.loadUserUnscoped(ctx, row.EntityID); err == nil {
			if refreshed.AdDn == nil || *refreshed.AdDn == "" {
				d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:target-not-found") // 不烧 attempts
				return
			}
			user = refreshed
		}
		// Phase 58 Plan 04: 此处补 compare-and-move OU 移动分支
		d.markPushed(ctx, row.ID, now)

	default:
		d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:unknown-action")
	}
}

// buildDrainUpdateReq 从闭包内重读的最新用户构造属性推送载荷。
//
// 键名契约（syncUserAttributes 内部做 string 类型断言，nil 指针会 panic）：
// nickname / email / phone / deptId，值必须是 string；指针为 nil 的字段不写入。
// deptId 为空串时同样跳过（内部会把空串当"不变"）。
func buildDrainUpdateReq(user *models.User) map[string]interface{} {
	updateReq := make(map[string]interface{})
	if user.Nickname != nil {
		updateReq["nickname"] = *user.Nickname
	}
	if user.Email != nil {
		updateReq["email"] = *user.Email
	}
	if user.Phone != nil {
		updateReq["phone"] = *user.Phone
	}
	if user.DeptID != nil && *user.DeptID != "" {
		updateReq["deptId"] = *user.DeptID
	}
	return updateReq
}

// handlePushError 错误分流（QUEUE-05 核心）：不可恢复 → dead 且不烧 attempts；其余 → 退避重试。
func (d *ADPushDrainer) handlePushError(ctx context.Context, row models.ADPushQueue, now time.Time, err error, action string) {
	// 1) 属性路径的哨兵错误（syncUserAttributes 已清空 ad_dn/ad_ou_dn/ad_synced_at）。
	if errors.Is(err, ErrADTargetNotFound) {
		d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:target-not-found")
		return
	}
	// 2) LDAP code 32（errors.As 穿透 %w 包装；OU 路径的 code 32 在 Plan 04 复用此分支）。
	if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
		d.clearUserADRefs(ctx, row.EntityID) // 清 ad_dn/ad_ou_dn/ad_synced_at
		d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:target-not-found")
		return
	}
	// 3) 账号池熔断（同步面可能直接冒泡到此处）→ 可重试，不判 dead。
	if errors.Is(err, ErrAllAccountsUnavailable) {
		d.markFailed(ctx, row, now, true, "pool:"+err.Error())
		return
	}
	// 4) 其余可恢复错误 → 退避重试（死判定只在这里、且只按 attempts+1，F-11）。
	d.markFailed(ctx, row, now, false, "retry:"+action+":"+err.Error())
}

// clearUserADRefs 清空用户的 AD 引用三列（语义照抄 user_ad_sync_service.go 的
// DNExists=false / code 32 分支）。
// ⚠ 必须 Unscoped()（disable 行对应软删用户）。
func (d *ADPushDrainer) clearUserADRefs(ctx context.Context, userID string) {
	if err := d.db.WithContext(ctx).Unscoped().Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{
			"ad_dn":        nil,
			"ad_ou_dn":     nil,
			"ad_synced_at": nil,
		}).Error; err != nil {
		applogger.Warnf("[ad_push_queue] 清空用户 %s 的 ad_dn 失败: %v", userID, err)
	}
}
