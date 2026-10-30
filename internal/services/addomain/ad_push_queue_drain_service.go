package addomain

// Phase 58 D-05 排水器。in-process 定时调用（cron 接线在 internal/scheduler/ad_sync_tasks.go，
// 禁注册 sys_job —— 里程碑锁定）。禁止自持账号池：必须由调用方注入
// （研究 Pitfall 4：独立缓存 → 熔断后仍选到坏账号）。
//
// 执行面（本 plan 范围）：
//   - 拾取：status IN ('pending','failed') AND (next_retry_at IS NULL OR next_retry_at <= now)
//   - 分组：按 sys_user.ad_dn 归属解析 AD 配置（D-02），每组一次账号池故障切换
//   - 推送：disable（UAC 514）与 update_user（属性 + compare-and-move OU 移动）
//   - 状态机：pending/failed → pushed | failed(退避) | dead（终态留表可查）
//
// OU 移动（QUEUE-04 compare-and-move）由本文件的 compareAndMoveOU 承担：
// 闭包内重读最新值 → 归一化比较当前 ad_ou_dn 与配置收敛查到的目标 OU →
// 一致则**不调** MoveUser（幂等点）→ 不一致才移动并同时回写 ad_dn + ad_ou_dn。

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

const (
	// drainBatchLimit 单轮批量上限（D-05）。LDAP 单写 10-100ms → 单轮 1-10s 天然有界。
	drainBatchLimit = 100
	// maxDrainAttempts 连续失败达此次数转 dead（D-05）。
	// 业务失败路径实际总重试窗 = 60s+2m+4m+8m = **15min**（第 5 次失败即转 dead，
	// 第 5/6+ 档 16min/30min 生产不可达）—— D-05 原文"~61min 桥接账号池 30min 熔断"
	// 的意图在该路径不成立，逻辑已决定暂不改（见 backoffDuration "生产不可达档位"
	// 注释与 58-REVIEW H-03）。池侧故障路径退避不递增，行不会逼近终态阈值。
	maxDrainAttempts = 5
	// backoffCap 退避封顶（D-05）。
	backoffCap = 30 * time.Minute
)

// backoffDuration 退避曲线纯函数（Phase 58 D-05 名义曲线 60s→2min→4min→8min→16min，封顶 30min；
// 实际业务失败路径只走前四档，见上方"生产不可达档位"说明）。
// 纯函数：无 DB、无副作用、可单测。attempts 为"本轮失败后的累计连续失败次数"。
// attempts <= 0 → 60s；1→60s，2→2min，3→4min，4→8min，5→16min，>=6→30min（封顶）。
//
// ⚠ 生产不可达档位（Phase 58 审查 H-03 + goal-verification MISSED-01/02，2026-10-19 决定：
// **不改退避逻辑**，仅记录事实）：
//
//   【业务失败路径】markFailed 在 newAttempts >= maxDrainAttempts(=5) 时直接转 dead，
//     故本函数只会被以 1/2/3/4 调用 —— 实际重试窗是 60s+2m+4m+8m = **15min**，
//     而非 D-05 原文声明的 ~61min。后果：D-05「退避窗桥接账号池 30min 熔断周期」
//     的设计意图在**业务失败**路径上仍不成立。
//
//   【池侧故障路径】MISSED-02 修复后 poolExhausted 分支**不自增 attempts**，故本函数
//     以该行原值调用并保持不变 —— 故障期间退避**不递增**，固定在 attempts=0 时的 60s。
//     这反而让"任何时长的 AD 断连都不把行推到终态阈值边上"成立（比 D-05 的 ~61min
//     更强），代价是池侧故障期间退避不随时间拉长（可接受：池全熔断时一次 LDAP 拨号
//     都不会发生）。
//
//   【为何两档不可达是加载承重的】它曾一度变得可达：H-04 修好"池侧故障不判死"后，
//   池路径仍在自增 attempts，于是能爬到 5+ 并用上 16min/30min；MISSED-02 把那次自增
//   去掉后，两档重新不可达。**这两处修复是一对**：单独修 H-04 会得到"不判死但持续计数"
//   的半修状态（见 markFailed 注释）。任何改动必须同时维持两者。
//
// ⚠ TestPhase58_BackoffDuration_Curve / _CappedAt30m 直接对纯函数断言 16min 与封顶档，
//   因此它们**测不出**这个断层（纯函数本身是对的，错的是调用方永不传 5/6+）。
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
	// mu 轮次互斥（Go >=1.18 提供 TryLock）：保证**同一个实例**上的并发 DrainOnce
	// 至多跑一个。
	//
	// ⚠ 单实例语义：互斥只在同一个 ADPushDrainer 内成立 —— 两个实例各有一把锁，
	//   彼此不阻塞。cron 侧因此必须在 AddJob 之前构造单个排水器并由闭包捕获
	//   （internal/scheduler/ad_sync_tasks.go）；写在闭包体内等于每轮一把新锁。
	// ⚠ 当前仓库**不存在**手动触发路径，本字段是"跨调用方互斥"这一设计意图的载体，
	//   现在真正拦住的是同一实例上的并发调用；将来新增触发端点必须复用同一实例。
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

// DrainOnce 执行一轮排水（D-05）。轮次互斥在本入口内部统一处理（mu.TryLock）。
// ⚠ 该互斥是**单实例**的：调用方必须复用同一个排水器实例（见 mu 字段注释），
//   每次调用都 New 一个新实例等于没有互斥。
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
		// poolExhausted = "池侧故障，闭包根本没跑起来"：账号列表为空（熔断/未配置，
		// ErrAllAccountsUnavailable）或全部账号拨号失败（域控宕机/网络中断，
		// ErrAllAccountsDialFailed）—— 两者都是可恢复的基础设施故障，
		// 纯退避重试、不烧 attempts、不判死。
		// ⚠ H-04：修复前只认前者。"全部拨号失败"返回的是包着拨号错误的普通 error →
		//   poolExhausted=false → 一次 AD 宕机会让整批（单轮最多 100 行）在 ~15min 内
		//   全部转 dead 终态，且 dead 行不会被任何后续编辑复活。
		//   两个哨兵**不合并**（见 account_pool.go 的注释），只在本处同款处理。
		// ⚠ 闭包执行过（账号拨通了、只是业务操作全失败）时**不得**归为 poolExhausted：
		//   code 32 等不可恢复语义必须原样流转到行级 handlePushError。
		poolExhausted := errors.Is(err, ErrAllAccountsUnavailable) || errors.Is(err, ErrAllAccountsDialFailed)
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
	// ⚠ 缺陷X（2026-10-21 UAT 实测 12:15-12:50 刻度循环）：msg 可能来自 LDAP err.Error()
	//   直拼（:581/:298/:390 三处调用点），含 0x00 / 无效 UTF-8 → PG 拒收 SQLSTATE 22021
	//   → UPDATE 整体失败 → 行留 pending/attempts=0/next_retry_at=NULL 每 tick 无限重试。
	//   与账号池 MarkFailure 同款清洗（sanitizeFailureReason），必须在任何 DB 写之前。
	msg = sanitizeFailureReason(msg)
	// ⚠ MISSED-02（2026-10-19 goal-verification 补修）：attempts 是**业务失败**的计数，
	//   池侧/基础设施故障（账号全被熔断、域控宕机、网络中断）**不得**计入。
	//   原实现无条件 `row.Attempts + 1`，H-04 修好后虽然池侧故障不再判死，却仍在自增 ——
	//   一次约 2 小时的 AD 断连会把 attempts 推到 12+，此后**任何一次**业务失败立刻
	//   `attempts >= maxDrainAttempts` 判死，而 last_error 已被覆盖成 "dead:max-attempts"，
	//   运维完全看不出前面 12 次不是业务失败。H-04 修了"别死"，这里补上"别计数"。
	newAttempts := row.Attempts
	if !poolExhausted {
		newAttempts++
	}
	// F-11 红线：dead 判定**只能**发生在本轮实际业务失败分支内，且按 attempts+1 >= maxDrainAttempts。
	// attempts 的语义是"自上次编辑以来的连续**业务**失败次数"（Phase 57 合并子句每次编辑清零
	// attempts），因此它是正确的连续失败计数器 —— 但**禁止**在拾取阶段 / 成功分支 /
	// 池侧故障分支 / 任何非失败路径上读 attempts 判死。
	// 已知并接受的后果：用户持续编辑某个持续失败的属性时，每次编辑清零 attempts，该行永不
	// 终态（"编辑即续命"）。这是有意设计 —— 用户还在管这件事，反复重试是期望行为。
	if !poolExhausted && newAttempts >= maxDrainAttempts {
		// ⚠ 传 newAttempts：本次失败已计入连续失败计数，DB 必须落到 5。
		return d.markDead(ctx, row.ID, now, newAttempts, "dead:max-attempts")
	}
	// 池侧故障分支：attempts 保持原值，next_retry_at 由该值派生 → 故障期间退避**不递增**，
	// 固定在 attempts=0 时的 60s。代价与理由：池全被熔断时 ListAvailable 直接返回空、
	// 一次 LDAP 拨号都不会发生，60s 空转开销可忽略；换来的是基础设施故障永不把行
	// 推到终态阈值边上。
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
	// 缺陷X 同源：reason 也可能含 0x00 / 无效 UTF-8（dead:retry: 前缀拼链），
	// 同款清洗保证 last_error 恒为合法 UTF-8。
	reason = sanitizeFailureReason(reason)
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

		// compare-and-move（QUEUE-04）：目标 OU 归一化一致则跳过 MoveUser，重复排水幂等。
		terminal, err := d.compareAndMoveOU(ctx, c, row, user, cfgID, now)
		if err != nil {
			// ⚠ 有意相对 SyncUserUpdateToAD 收紧：后者 OU 移动失败仅 ERROR 日志并继续，
			//   这里上抛 → failed + 退避。否则"OU 永远移不动"的行会被属性成功掩盖成 pushed。
			d.handlePushError(ctx, row, now, err, "move_ou")
			return
		}
		// ⚠ 终态不覆盖不变量：行已被 compareAndMoveOU 写成终态（no-ou-mapping → dead）时
		//   **必须**在此处短路返回，绝不能落到下面的 markPushed —— 那会把 dead 覆盖成
		//   pushed（生产上"需要重跑部门同步"被误报成功 + 终态不可查）。
		if terminal {
			return
		}
		d.markPushed(ctx, row.ID, now)

	case models.ADPushActionUpdateManager:
		// 经理同步规则引擎（Phase：ad-sync-fixes）：按 sys_manager_rule 解析经理，
		// 回写 AD manager 属性；解析结果为空 → 清空属性（与系统侧"无经理"一致）。
		d.drainUpdateManager(ctx, c, row, user, now)

	default:
		d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:unknown-action")
	}
}

// drainUpdateManager update_manager 动作执行面：
//  1. 规则引擎解析经理（ResolveManagerDN 内部处理无规则/无经理等跳过语义）
//  2. 有经理 DN → Replace AD manager；无 → DeleteUserAttribute 清空
//  3. LDAP 失败 → handlePushError（failed + 退避）；规则引擎内部错误 → 同样退避
//
// 幂等性：Replace 同一 DN 重复执行无副作用；每次排水实时解析（规则/任职人可能已变）。
func (d *ADPushDrainer) drainUpdateManager(ctx context.Context, c LDAPClientIface, row models.ADPushQueue, user *models.User, now time.Time) {
	svc := NewManagerRuleService(d.db)
	managerDN, reason, err := svc.ResolveManagerDN(ctx, user)
	if err != nil {
		// 规则引擎内部故障（DB 查询失败等）→ 与业务失败同路径退避重试
		d.handlePushError(ctx, row, now, err, "update_manager")
		return
	}
	if managerDN == "" {
		// 无规则命中或解析不出经理 → 清空 AD manager（保持系统侧一致性）
		applogger.Infof("[ad_push_queue] update_manager 无经理可写，清空属性: user=%s, reason=%s", user.Username, reason)
		if err := c.DeleteUserAttribute(*user.AdDn, "manager"); err != nil {
			// 属性本就不存在等场景 AD 可能报错；按失败退避一次后人工排查
			d.handlePushError(ctx, row, now, err, "update_manager:clear")
			return
		}
		d.updateShadowManager(ctx, user, "")
		d.markPushed(ctx, row.ID, now)
		return
	}
	if err := c.UpdateUserAttribute(*user.AdDn, map[string]string{"manager": managerDN}); err != nil {
		d.handlePushError(ctx, row, now, err, "update_manager")
		return
	}
	applogger.Infof("[ad_push_queue] update_manager 推送成功: user=%s, managerDN=%s", user.Username, managerDN)
	d.updateShadowManager(ctx, user, managerDN)
	d.markPushed(ctx, row.ID, now)
}

// updateShadowManager 同步 AD 用户影子表 sys_ad_user.manager。
//
// 目的：AD 用户页的 manager 列读的是影子表，影子同步全量跑之前若不回写，
// 页面看不到刚推送的经理值（manager 列永远是空的）。
//
// 失败只 WARN 不阻断：影子表回写失败不影响 AD 侧已生效的属性，下次全量同步会纠正。
func (d *ADPushDrainer) updateShadowManager(ctx context.Context, user *models.User, managerDN string) {
	if user.AdDn == nil || *user.AdDn == "" {
		return
	}
	if !d.db.Migrator().HasTable(&models.ADUser{}) {
		return
	}
	q := d.db.WithContext(ctx).Model(&models.ADUser{}).
		Where("user_dn = ?", *user.AdDn)
	if user.AdConfigID != nil && *user.AdConfigID != "" {
		q = q.Where("ad_config_id = ?", *user.AdConfigID)
	}
	if err := q.Update("manager", managerDN).Error; err != nil {
		applogger.Warnf("[ad_push_queue] 回写影子表 manager 失败: user=%s, err=%v", user.Username, err)
	}
}

// compareAndMoveOU OU 移动的幂等执行面（Phase 58 QUEUE-04）。
//
// 三步，全部基于**闭包内重读到的最新值**（不是入队时刻的旧值 —— 禁新旧值比较，
// 那会激活 MoveUser 的未验证重试分支）：
//  1. 取目标 OU：FindOUDNByDeptIDForConfig(ctx, *user.DeptID, cfgID)（F-B）
//     无映射 → 不可恢复，dead（Q2：需运营动作「跑部门同步」，重试无意义）
//  2. 归一化比较：当前 ad_ou_dn 与目标 OU **双侧**过 NormalizeDN 后相等 → 直接返回
//     （幂等点。对同一 DN 重复 MoveUser 会 code 32，见 ldap_client.go:350-361，
//      ModifyDN 的 deleteOldRDN=true 会先删掉旧 RDN）
//  3. 不一致 → MoveUser(*user.AdDn, targetOUDN)；成功后**同时**回写
//     ad_dn = extractRDNFromDN(*user.AdDn) + "," + targetOUDN 与 ad_ou_dn = targetOUDN
//     （F-A 阻断级：ModifyDN 后 AD 侧 DN 已变，ad_dn 不回写会让 D-02 下一轮按陈旧 DN
//     解析配置，跨域移动后落到另一个 base_dn → 误判「配置无匹配」→ 误转 dead；
//     属性 Modify 也会打到旧 DN → 二次 code 32）
//
// ⚠ 与 SyncUserUpdateToAD 的语义差异（有意收紧）：后者 OU 移动失败仅 ERROR 日志并继续推属性；
// 本函数把 OU 失败上抛给 handlePushError → 行转 failed 并退避重试。理由：否则"OU 永远移不动"
// 的行会被属性成功掩盖成 pushed，SC2「重复排水幂等、失败可查」的可观测性就没了。
//
// ⚠ 返回值契约（terminal 是本函数存在的理由，**不得**退回单 error 返回）：
//
//	(false, nil) = 本函数未对行做任何终态写 → 调用方**可以** markPushed
//	(true,  nil) = 本函数已把行写成终态（当前仅 no-ou-mapping → dead）→ 调用方**必须**跳过 markPushed
//	(false, err) = 本函数失败 → 调用方交给 handlePushError 分流
//
// 把 (true, nil) 当成 (false, nil) 处理会让 dead 被 markPushed 覆盖成 pushed，
// TestPhase58_Drain_NoOUMappingGoesDead 必红，且生产上"需要重跑部门同步"被误报为成功。
func (d *ADPushDrainer) compareAndMoveOU(ctx context.Context, c LDAPClientIface, row models.ADPushQueue, user *models.User, cfgID string, now time.Time) (terminal bool, err error) {
	// 1) 用户无部门 → 无 OU 概念：既不是失败也不是终态，调用方照常 markPushed。
	if user.DeptID == nil || *user.DeptID == "" {
		return false, nil
	}

	// 2) 目标 OU 按 AD 配置收敛（F-B）。查不到 → 不可恢复：映射不会自己出现，
	//    重试只会无限堆积 pending；转 dead 并由 last_error 前缀引导管理员跑部门同步。
	targetOUDN, mapErr := d.mapper.FindOUDNByDeptIDForConfig(ctx, *user.DeptID, cfgID)
	if mapErr != nil {
		// ⚠ terminal=true：本函数已经写了终态，调用方不得再 markPushed。
		// ⚠ 第三个实参是 row.Attempts（不可恢复不烧 attempts，与 markDead 契约一致）。
		d.markDead(ctx, row.ID, now, row.Attempts, "unrecoverable:no-ou-mapping")
		return true, nil
	}

	// 3) 幂等点：当前 OU 与目标 OU 归一化后相等 → 不调 MoveUser。
	//    ⚠ 必须走 NormalizeDN（attr 名与值双侧 ToLower + 去空格），裸比较会把
	//    `OU=IT,DC=corp,DC=com` 与 `ou=it , dc=corp , dc=com` 判为不等 →
	//    对已搬移的源 DN 再调一次 ModifyDN → code 32。
	if user.AdOuDn != nil && NormalizeDN(*user.AdOuDn) == NormalizeDN(targetOUDN) {
		return false, nil
	}

	// 防御：drainRow 已判过 ad_dn 非空；此处再兜一层，避免 nil 解引用 panic（P1：闭包内不得 panic）。
	if user.AdDn == nil || *user.AdDn == "" {
		return false, nil
	}

	// 4) 不一致才移动。%w 包装让 handlePushError 的 ldap.IsErrorWithCode 能穿透判定 code 32。
	if moveErr := c.MoveUser(*user.AdDn, targetOUDN); moveErr != nil {
		return false, fmt.Errorf("移动用户到新 OU 失败 [%s → %s]: %w", *user.AdDn, targetOUDN, moveErr)
	}

	// 5) F-A：同时回写 ad_dn 与 ad_ou_dn。
	//    RDN 用同包 extractRDNFromDN 取（MoveUser 内部用的也是它，保证拼接一致）。
	//    ⚠ 必须 Unscoped()（disable 场景用户已软删，作用域 Updated 会命中 0 行且不报错）。
	//    ⚠ 用 Updates(map[...]) 而非 Save（Phase 57 F-06 收窄写路径的同款决策：显式字段）。
	//
	//    ⚠ H-05：回写失败**必须上抛**。此前这里只 Warn 并返回 (false, nil)，调用方
	//    直接 markPushed —— 队列行以"成功"终结，而本地 ad_dn 停在旧 OU，AD 侧对象
	//    已经在新 OU。下一次该用户被编辑再入队时，syncUserAttributes 的
	//    DNExists(旧 DN)=false → 清空 ad_dn/ad_ou_dn/ad_synced_at → 行转 dead
	//    "unrecoverable:target-not-found"：AD 账号明明还在，系统却永久丢失了绑定。
	//    注释里"本地回写失败下轮会重算"是错的 —— pushed 是终态，根本没有"下轮"。
	//    上抛后该行走 markFailed + 退避，last_error 立刻暴露真实原因（而不是被
	//    markPushed 掩盖成成功，Phase 59 的信号守卫也就不会把这种丢绑定读成"一致"）。
	newAdDn := fmt.Sprintf("%s,%s", extractRDNFromDN(*user.AdDn), targetOUDN)
	if err := d.db.WithContext(ctx).Unscoped().Model(&models.User{}).Where("id = ?", user.ID).
		Updates(map[string]interface{}{"ad_dn": newAdDn, "ad_ou_dn": targetOUDN}).Error; err != nil {
		applogger.Errorf("[ad_push_queue] 回写用户 %s 的 ad_dn/ad_ou_dn 失败 [%s]（MoveUser 已在 AD 侧生效）: %v",
			user.ID, newAdDn, err)
		// ⚠ 已知残余风险（非退避可自愈）：下一轮该行重读到的仍是旧 ad_dn，
		//   syncUserAttributes 的 DNExists(旧 DN) 会返回 false → 清空三列 → dead。
		//   即"重试"并不会自动修复绑定，只是不再被误报成 pushed。要真正自愈需要
		//   在重放时识别"对象已在目标 OU"并只补写回，该设计待单独决策
		//   （见 58-REVIEW H-05 的修复建议段与本次修复报告）。
		return false, fmt.Errorf("回写用户 %s 的 ad_dn/ad_ou_dn 失败（MoveUser 已生效，本地 DN 已陈旧）: %w", user.ID, err)
	}

	// 终态由调用方的 markPushed 写，本函数不重复写。
	return false, nil
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

// drainIntervalSysConfigKey 排水间隔单旋钮的 sys_config key（Phase 58 D-05）。
// 启动时读取一次，改后需重启后端生效（同 network.connection_pool.* 先例）。
const drainIntervalSysConfigKey = "ad_push_queue.drain_interval_seconds"

// DefaultDrainIntervalSeconds 默认排水间隔（D-05：30s）。管理员编辑 → AD 生效感知 ≤1min；
// 给"域控吃不消"留放宽到 5min 的逃生门。
// 导出：scheduler 排水门控（方案A热重载）需要引用同一默认值。
const DefaultDrainIntervalSeconds = 30

// defaultDrainIntervalSeconds 包内兼容别名（历史引用点不改名）。
const defaultDrainIntervalSeconds = DefaultDrainIntervalSeconds

// LoadDrainIntervalSeconds 读取排水间隔（秒）。读不到/解析失败/非正数 → 回落 30。
// 方案A热重载（2026-10-21）：由 scheduler 排水闭包**每 tick（30s）调用一次**，
// 改 sys_config 无需重启；成功路径不打日志（每 30s 一条会刷屏），值变化日志由
// scheduler 的 drainGate.shouldDrain 负责。
func LoadDrainIntervalSeconds(db *gorm.DB) int {
	const fallback = defaultDrainIntervalSeconds
	var v string
	if err := db.Raw(
		"SELECT config_value FROM sys_config WHERE config_key = ? AND deleted_at IS NULL",
		drainIntervalSysConfigKey,
	).Scan(&v).Error; err != nil || v == "" {
		applogger.Warnf("[ad_push_queue] 读取 %s 失败(%v)，回落默认 %ds", drainIntervalSysConfigKey, err, fallback)
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		applogger.Warnf("[ad_push_queue] %s 值非法(%q)，回落默认 %ds", drainIntervalSysConfigKey, v, fallback)
		return fallback
	}
	return n
}
