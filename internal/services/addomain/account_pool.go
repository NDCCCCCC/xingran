package addomain

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 账号池状态机常量（Phase 69 DICT-01：值唯一真相源位于 models.ADAccountStatus*，
// 此处为包内别名，保持既有调用方零改动；值语义由 status_constants_test.go 锁定）
const (
	AccountStatusAvailable     = models.ADAccountStatusAvailable
	AccountStatusDisabled      = models.ADAccountStatusDisabled
	AccountStatusCircuitBroken = models.ADAccountStatusBreaker
)

// 阈值常量（与 PLAN.md 对齐）
const (
	FailureThreshold       = 3                // 连续失败次数触发熔断
	CircuitBreakerDuration = 30 * time.Minute // 熔断持续时间
	DefaultMaxHops         = 10               // FailoverClient 单次最多尝试账号数（Issue 18）
	BreakerRecoveryCron    = "*/5 * * * *"    // 每 5 分钟恢复过期熔断
)

// 哨兵错误
var (
	ErrAllAccountsUnavailable = errors.New("账号池无可用账号")
	ErrAccountNotFound        = errors.New("账号不存在")
	ErrInvalidOperator        = errors.New("operator 必填")
	ErrInvalidUnlockReason    = errors.New("解锁原因必填且不少于10字符")
	// ErrADTargetNotFound debug session ad-update-attr-no-such-object Fix 2:
	// 标记目标 AD 对象不存在（LDAP code 32 No Such Object）。handler 层据此
	// 短路重试，避免 3 次重试 + MarkFailure 累加触发应用层 breaker 熔断。
	ErrADTargetNotFound = errors.New("AD 目标对象不存在")
)

// AccountPool AD 服务账号池接口
//
// 设计要点（Phase 36）：
//   - 行级悲观锁（SELECT FOR UPDATE）保证并发 MarkSuccess/MarkFailure 不丢失更新（Issue 2）
//   - ListAvailable 纯读，不修改 DB（Issue 4）
//   - RecoverExpiredBreakers 显式副作用，由 cron 任务每 5 分钟调用（Issue 4/11）
//   - 不再使用冷却期（Issue 14）：避免池子饥饿
//   - ManualUnlock 校验下沉到 service 层（Issue 13）
type AccountPool interface {
	// PickAvailable 随机选一个可用账号（用于 FailoverClient 早期版本）
	// Wave 3 改用 ListAvailable + 顺序遍历更优
	PickAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error)

	// ListAvailable **纯读**：返回过滤后的可用账号列表（status=0 或熔断已到期）
	ListAvailable(ctx context.Context, configID string) ([]models.ADServiceAccount, error)

	// ListAll 包含停用/熔断账号（供 UI 展示用）
	ListAll(ctx context.Context, configID string, page, pageSize int, statusFilter *int) ([]models.ADServiceAccount, int64, error)

	// CountByStatus 按状态分组统计（供 Stats 用，O(状态数) 而非 O(账号数)）
	CountByStatus(ctx context.Context, configID string) (total, available, disabled, circuitBroken int64, err error)

	// PickFirstAvailable 取第一个可用账号（供 Stats "当前活跃"）
	PickFirstAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error)

	// Create 新增账号
	Create(ctx context.Context, account *models.ADServiceAccount) error

	// Update 更新账号（不更新 password 时 password_ciphertext 字段保持不变）
	Update(ctx context.Context, account *models.ADServiceAccount) error

	// Delete 软删除
	Delete(ctx context.Context, accountID string) error

	// MarkSuccess 成功上报（重置 failure_count + 写 last_success_at）
	MarkSuccess(ctx context.Context, accountID string) error

	// MarkFailure 失败上报（累加 failure_count，达阈值触发熔断）
	// Issue 2: 行锁 + 事务
	// Issue 14: 不再写 recently_failed_until
	MarkFailure(ctx context.Context, accountID, reason string) error

	// ManualUnlock 手动解锁（清除熔断状态 + 记录解锁审计字段）
	// Issue 13: operator 非空 + reason ≥10 字符（service 层 invariant）
	ManualUnlock(ctx context.Context, accountID, operator, reason string) error

	// SetEnabled 启用/停用（不影响熔断状态）
	SetEnabled(ctx context.Context, accountID string, enabled bool) error

	// RecoverExpiredBreakers 显式副作用：恢复所有已到期的熔断账号
	// Issue 11: 写完后主动 InvalidateCache
	// Issue 17: 写完后通过 Redis pub/sub 广播（如果 redisPubSub 非 nil）
	RecoverExpiredBreakers(ctx context.Context) (int, error)

	// InvalidateCache 清本地内存缓存（pool 内部 cache）
	InvalidateCache(configID string)

	// StartHotReload 启动 Redis pub/sub 订阅（监听其他进程的失效事件）
	// Issue 17: 包括 breaker_recovered 事件
	StartHotReload(ctx context.Context) error
}

// accountPoolImpl 默认实现（基于 GORM + 内存 cache）
type accountPoolImpl struct {
	db          *gorm.DB
	mu          sync.RWMutex
	cache       map[string][]models.ADServiceAccount // configID → 可用账号快照
	cacheTTL    map[string]time.Time                 // configID → 缓存到期时间
	redisPubSub RedisPublisher                       // 可选：用于跨进程广播（Issue 17）
	cacheTTLDur time.Duration                        // 缓存有效期
	rand        *rand.Rand                           // P2-M1: 随机源（替代 time.Now() 伪随机）
	randMu      sync.Mutex                           // P2-M1: math/rand 不是并发安全的
}

// RedisPublisher 是 pub/sub 发布的最小接口（便于测试 mock）
type RedisPublisher interface {
	Publish(ctx context.Context, channel, message string) error
	Subscribe(ctx context.Context, channel string) (<-chan string, error)
}

// NewAccountPool 创建账号池（redisPubSub 可传 nil 表示不启用跨进程广播）
func NewAccountPool(db *gorm.DB, redisPubSub RedisPublisher) AccountPool {
	return &accountPoolImpl{
		db:          db,
		cache:       make(map[string][]models.ADServiceAccount),
		cacheTTL:    make(map[string]time.Time),
		redisPubSub: redisPubSub,
		cacheTTLDur: 30 * time.Second,                                // 短 TTL + pub/sub 推送（兜底 Issue 17）
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())), // P2-M1
	}
}

// ListAvailable **纯读**：返回过滤后的可用账号（status≠1 且 (status≠2 或 熔断到期)）
func (p *accountPoolImpl) ListAvailable(ctx context.Context, configID string) ([]models.ADServiceAccount, error) {
	// 1. 查缓存
	p.mu.RLock()
	if cached, ok := p.cache[configID]; ok {
		if exp, ok := p.cacheTTL[configID]; ok && time.Now().Before(exp) {
			p.mu.RUnlock()
			return cached, nil
		}
	}
	p.mu.RUnlock()

	// 2. 走 DB（不加锁，纯读）
	var all []models.ADServiceAccount
	if err := p.db.WithContext(ctx).
		Where("config_id = ? AND deleted_at IS NULL", configID).
		Find(&all).Error; err != nil {
		return nil, fmt.Errorf("查询账号池失败: %w", err)
	}

	now := time.Now()
	available := make([]models.ADServiceAccount, 0, len(all))
	for _, a := range all {
		if a.Status == AccountStatusDisabled {
			continue
		} // 停用
		if a.Status == AccountStatusCircuitBroken {
			// 熔断中：未到期则跳过；到期但不修改（由 cron 恢复）
			if a.CircuitBreakerUntil == nil || now.Before(*a.CircuitBreakerUntil) {
				continue
			}
		}
		// Issue 14: 不再过滤 recently_failed_until
		available = append(available, a)
	}

	// 3. 写缓存
	p.mu.Lock()
	p.cache[configID] = available
	p.cacheTTL[configID] = time.Now().Add(p.cacheTTLDur)
	p.mu.Unlock()

	return available, nil
}

// PickAvailable 从可用账号随机选一个
//
// P2-M1 修复：用 math/rand 替代 time.Now().UnixNano()（Windows 15ms 精度，高并发分布不均）
func (p *accountPoolImpl) PickAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	available, err := p.ListAvailable(ctx, configID)
	if err != nil {
		return nil, err
	}
	if len(available) == 0 {
		return nil, ErrAllAccountsUnavailable
	}
	// 注：Wave 3 FailoverClient 改用顺序遍历，此方法保留用于单点场景
	p.randMu.Lock()
	idx := p.rand.Intn(len(available))
	p.randMu.Unlock()
	return &available[idx], nil
}

// ListAll 包含停用/熔断账号（分页）
func (p *accountPoolImpl) ListAll(ctx context.Context, configID string, page, pageSize int, statusFilter *int) ([]models.ADServiceAccount, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	q := p.db.WithContext(ctx).Model(&models.ADServiceAccount{}).
		Where("config_id = ? AND deleted_at IS NULL", configID)
	if statusFilter != nil {
		q = q.Where("status = ?", *statusFilter)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []models.ADServiceAccount
	if err := q.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// CountByStatus 按状态分组统计（供 Stats 用，避免 pageSize=9999 全量扫描）
//
// claude 问题 8 修复
func (p *accountPoolImpl) CountByStatus(ctx context.Context, configID string) (total, available, disabled, circuitBroken int64, err error) {
	rows, err := p.db.WithContext(ctx).Model(&models.ADServiceAccount{}).
		Select("status, COUNT(*) as cnt").
		Where("config_id = ? AND deleted_at IS NULL", configID).
		Group("status").
		Rows()
	if err != nil {
		return 0, 0, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var status, cnt int64
		if scanErr := rows.Scan(&status, &cnt); scanErr != nil {
			return 0, 0, 0, 0, scanErr
		}
		total += cnt
		switch status {
		case AccountStatusAvailable:
			available = cnt
		case AccountStatusDisabled:
			disabled = cnt
		case AccountStatusCircuitBroken:
			circuitBroken = cnt
		}
	}
	return total, available, disabled, circuitBroken, rows.Err()
}

// PickFirstAvailable 返回当前可用账号（best-effort，供 Stats 显示"当前活跃账号"）
func (p *accountPoolImpl) PickFirstAvailable(ctx context.Context, configID string) (*models.ADServiceAccount, error) {
	var a models.ADServiceAccount
	err := p.db.WithContext(ctx).Where("config_id = ? AND deleted_at IS NULL AND status = ?",
		configID, AccountStatusAvailable).
		Order("created_at ASC").
		Limit(1).
		Take(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// Create 新增账号
func (p *accountPoolImpl) Create(ctx context.Context, account *models.ADServiceAccount) error {
	if err := p.db.WithContext(ctx).Create(account).Error; err != nil {
		return err
	}
	p.InvalidateCache(account.ConfigID)
	applogger.Infof("[ADPool] 创建账号成功 [ID=%s, username=%s, configID=%s]",
		account.ID, account.Username, account.ConfigID)
	return nil
}

// Update 更新账号
func (p *accountPoolImpl) Update(ctx context.Context, account *models.ADServiceAccount) error {
	if err := p.db.WithContext(ctx).Save(account).Error; err != nil {
		return err
	}
	p.InvalidateCache(account.ConfigID)
	return nil
}

// Delete 软删除
func (p *accountPoolImpl) Delete(ctx context.Context, accountID string) error {
	var acct models.ADServiceAccount
	if err := p.db.WithContext(ctx).First(&acct, "id = ?", accountID).Error; err != nil {
		return ErrAccountNotFound
	}
	if err := p.db.WithContext(ctx).Delete(&acct).Error; err != nil {
		return err
	}
	p.InvalidateCache(acct.ConfigID)
	applogger.Infof("[ADPool] 删除账号 [ID=%s, username=%s]", acct.ID, acct.Username)
	return nil
}

// MarkSuccess 成功上报（行锁 + 事务）
//
// P2-M4 修复：缓存失效移到事务外（defer 在 commit 后执行）
func (p *accountPoolImpl) MarkSuccess(ctx context.Context, accountID string) error {
	var configID string
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		res := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Model(&models.ADServiceAccount{}).
			Where("id = ?", accountID).
			Updates(map[string]interface{}{
				"failure_count":   0,
				"last_success_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		// 拿到 configID 用于缓存失效（事务内 select 是安全的）
		var row struct{ ConfigID string }
		if err := tx.Raw("SELECT config_id FROM sys_ad_service_accounts WHERE id = ?", accountID).
			Scan(&row).Error; err == nil {
			configID = row.ConfigID
		}
		return nil
	})
	if err == nil && configID != "" {
		p.InvalidateCache(configID)
	}
	return err
}

// sanitizeFailureReason 清洗失败原因字符串，确保可安全写入 PostgreSQL text 列。
// PostgreSQL 拒绝 0x00（null 字节）与无效 UTF-8 序列；AD/LDAP 错误消息可能含这些
// （Windows AcceptSecurityContext 上下文数据），未清洗会使 UPDATE 报 SQLSTATE 22021，
// 致 failure_count 不累加、坏账号永不熔断、FailoverClient 反复重试同一坏账号。
func sanitizeFailureReason(reason string) string {
	cleaned := strings.ToValidUTF8(reason, "")        // 移除无效 UTF-8 序列
	cleaned = strings.ReplaceAll(cleaned, "\x00", "") // 移除 null 字节（UTF-8 合法但 PG text 列拒绝）
	if len(cleaned) > 1000 {                          // 截断超长错误，避免撑爆字段
		cleaned = cleaned[:1000]
	}
	return cleaned
}

// MarkFailure 失败上报（行锁 + 事务 + 自动熔断）
//
// Issue 14 修复：不再设置 recently_failed_until（冷却期），避免池子饥饿
// P2-M4 修复：缓存失效移到事务外
// ad-update-attr-no-such-object Fix 3：按 reason 前缀区分语义
//   - "dial:" 前缀 → connect/bind 失败，账号健康问题 → 计入 failure_count → 触发熔断
//   - "operation:" 前缀 → Modify/Search/Move 等单操作失败（如 LDAP code 32
//     No Such Object、code 53 等），目标对象/数据问题，不是账号问题 →
//     仍记录 last_failure_reason 供审计，**不计入** failure_count，
//     不触发熔断。这避免了：单条 stale-DN modify 失败 → handler 3 次重试
//     → 同一账号被打 3 次 MarkFailure → 熔断 → 全池 bind 失败 → 用户看到
//     "管理员账号被锁"。
//   - 其他前缀 → 兼容旧调用，按 dial 语义计入（保守处理）
func (p *accountPoolImpl) MarkFailure(ctx context.Context, accountID, reason string) error {
	reason = sanitizeFailureReason(reason) // 清洗 0x00/无效 UTF-8，避免 PG UPDATE SQLSTATE 22021

	// 语义分流：operation: 前缀不计入 failure_count（仍记录原因）
	countsTowardBreaker := !strings.HasPrefix(reason, "operation:")

	var configID string
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var a models.ADServiceAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&a, "id = ?", accountID).Error; err != nil {
			return err
		}

		now := time.Now()
		if countsTowardBreaker {
			a.FailureCount++
		} else {
			applogger.Debugf("[ADPool] operation 失败不计入熔断计数 [ID=%s, reason=%s]",
				a.ID, reason)
		}
		a.LastFailureAt = &now
		a.LastFailureReason = reason

		// 触发熔断（仅 countsTowardBreaker 路径）
		if countsTowardBreaker && a.FailureCount >= FailureThreshold && a.Status != AccountStatusCircuitBroken {
			a.Status = AccountStatusCircuitBroken
			breakerUntil := now.Add(CircuitBreakerDuration)
			a.CircuitBreakerUntil = &breakerUntil
			applogger.Warnf("[ADPool] 账号 %s 进入熔断状态 [ID=%s, failureCount=%d, reason=%s]",
				a.Username, a.ID, a.FailureCount, reason)
		}

		if err := tx.Save(&a).Error; err != nil {
			return err
		}
		configID = a.ConfigID
		return nil
	})
	if err == nil && configID != "" {
		p.InvalidateCache(configID)
	}
	return err
}

// ManualUnlock 手动解锁（service 层 invariant 校验）
//
// Issue 13: operator 非空 + reason ≥10 字符（service 层）
// P2-M4 修复：缓存失效移到事务外
func (p *accountPoolImpl) ManualUnlock(ctx context.Context, accountID, operator, reason string) error {
	operator = strings.TrimSpace(operator)
	reason = strings.TrimSpace(reason)
	if operator == "" {
		return ErrInvalidOperator
	}
	if reason == "" || len(reason) < 10 {
		return ErrInvalidUnlockReason
	}

	var configID string
	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var a models.ADServiceAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&a, "id = ?", accountID).Error; err != nil {
			return ErrAccountNotFound
		}

		now := time.Now()
		a.Status = AccountStatusAvailable
		a.FailureCount = 0
		a.CircuitBreakerUntil = nil
		a.ManualUnlockedBy = operator
		a.ManualUnlockedAt = &now
		a.ManualUnlockReason = reason

		if err := tx.Save(&a).Error; err != nil {
			return err
		}
		configID = a.ConfigID
		applogger.Infof("[ADPool] 管理员手动解锁 [ID=%s, username=%s, operator=%s, reason=%s]",
			a.ID, a.Username, operator, reason)
		return nil
	})
	if err == nil && configID != "" {
		p.InvalidateCache(configID)
	}
	return err
}

// SetEnabled 启用/停用
//
// P2-claude 问题 7 修复：启用时同时清除熔断状态（避免 status=0 但 circuit_breaker_until 未到期）
func (p *accountPoolImpl) SetEnabled(ctx context.Context, accountID string, enabled bool) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var acct models.ADServiceAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&acct, "id = ?", accountID).Error; err != nil {
			return ErrAccountNotFound
		}

		updates := map[string]interface{}{}
		if enabled {
			updates["status"] = AccountStatusAvailable
			// 启用时一并清除熔断状态，避免 status=0 但 circuit_breaker_until 未到期的矛盾
			updates["circuit_breaker_until"] = nil
			updates["failure_count"] = 0
		} else {
			updates["status"] = AccountStatusDisabled
		}

		if err := tx.Model(&acct).Updates(updates).Error; err != nil {
			return err
		}
		// P2-M4 修复：事务提交后失效缓存（不是事务内部）
		defer p.InvalidateCache(acct.ConfigID)
		applogger.Infof("[ADPool] 账号 %s 状态变更 [ID=%s, enabled=%v]", acct.Username, acct.ID, enabled)
		return nil
	})
}

// RecoverExpiredBreakers 恢复已到期熔断账号 + 跨进程广播
//
// Issue 11: 主动 InvalidateCache
// Issue 17: Redis pub/sub 广播（如果 redisPubSub 非 nil）
func (p *accountPoolImpl) RecoverExpiredBreakers(ctx context.Context) (int, error) {
	now := time.Now()

	// 1. 查询受影响的 configID 列表（用于精准失效缓存 + 广播）
	var configIDs []string
	if err := p.db.WithContext(ctx).
		Model(&models.ADServiceAccount{}).
		Where("status = ? AND circuit_breaker_until <= ? AND deleted_at IS NULL",
			AccountStatusCircuitBroken, now).
		Distinct("config_id").
		Pluck("config_id", &configIDs).Error; err != nil {
		return 0, err
	}

	// 2. 批量恢复
	result := p.db.WithContext(ctx).Model(&models.ADServiceAccount{}).
		Where("status = ? AND circuit_breaker_until <= ? AND deleted_at IS NULL",
			AccountStatusCircuitBroken, now).
		Updates(map[string]interface{}{
			"status":                AccountStatusAvailable,
			"circuit_breaker_until": nil,
			"failure_count":         0,
		})
	if result.Error != nil {
		return 0, result.Error
	}

	// 3. 本地失效
	for _, cid := range configIDs {
		p.InvalidateCache(cid)
	}

	// 4. Issue 17: Redis pub/sub 广播（其他进程收到后失效本地缓存）
	if len(configIDs) > 0 && p.redisPubSub != nil {
		msg := strings.Join(configIDs, ",")
		if err := p.redisPubSub.Publish(ctx, "ad:account_pool:breaker_recovered", msg); err != nil {
			applogger.Warnf("[ADPool] 广播 breaker 恢复事件失败（不影响主流程）: %v", err)
		}
	}

	if n := int(result.RowsAffected); n > 0 {
		applogger.Infof("[ADPool] cron 恢复 %d 个过期熔断账号", n)
	}
	return int(result.RowsAffected), nil
}

// InvalidateCache 清本地缓存
func (p *accountPoolImpl) InvalidateCache(configID string) {
	p.mu.Lock()
	delete(p.cache, configID)
	delete(p.cacheTTL, configID)
	p.mu.Unlock()
}

// StartHotReload 启动 Redis pub/sub 订阅
//
// 监听 "ad:account_pool:breaker_recovered" 事件（Issue 17）
// 收到消息后，解析 configID 列表并失效本地缓存
func (p *accountPoolImpl) StartHotReload(ctx context.Context) error {
	if p.redisPubSub == nil {
		// 没有 pub/sub 时跳过（单机部署场景）
		return nil
	}
	ch, err := p.redisPubSub.Subscribe(ctx, "ad:account_pool:breaker_recovered")
	if err != nil {
		return fmt.Errorf("订阅 AD 账号池事件失败: %w", err)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				for _, cid := range strings.Split(msg, ",") {
					cid = strings.TrimSpace(cid)
					if cid != "" {
						p.InvalidateCache(cid)
					}
				}
				applogger.Debugf("[ADPool] 收到跨进程缓存失效事件: %s", msg)
			}
		}
	}()
	return nil
}
