package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	systemServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// SelectorLearner 选择器学习服务接口
type SelectorLearner interface {
	RecordSuccess(ctx context.Context, record *SelectorSuccessRecord) error
	RecordFailure(ctx context.Context, record *SelectorFailureRecord) error
	GetBestSelector(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error)
	// TIDY-03（v1.51）：返回消费的事件数——0 且无错表示"无可学事件"调用方可
	// 降级；执行记录缺失/无明细返回明确错误（不静默吞）。
	LearnFromExecution(ctx context.Context, executionID string) (consumed int, err error)
	ScoreSelector(ctx context.Context, selector string, pageURL string) (float64, error)
	GetSelectorAlternatives(ctx context.Context, selector string, pageURL string) ([]string, error)
}

// selectorLearnerImpl 选择器学习服务实现
type selectorLearnerImpl struct {
	db     *gorm.DB
	cache  base.CacheProvider
	config *config.Config
	mu     sync.RWMutex
	// noticeHub TIDY-03（v1.51）：可选 WebSocket 通知枢纽（nil 安全）。
	noticeHub *websocket.NoticeHub
}

// NewSelectorLearner 创建选择器学习服务
// Phase 103 CONV-03 (D-103-2): cache cache.Cache → cache base.CacheProvider
// TIDY-03（v1.51）：可选变参注入 NoticeHub（nil 安全）——选择器连续失败时经
// WebSocket 广播更新提醒；不传 = 维持旧行为（仅本地记录）。
func NewSelectorLearner(db *gorm.DB, cache base.CacheProvider, cfg *config.Config, noticeHub ...*websocket.NoticeHub) SelectorLearner {
	impl := &selectorLearnerImpl{
		db:     db,
		cache:  cache,
		config: cfg,
	}
	for _, h := range noticeHub {
		impl.noticeHub = h
	}
	return impl
}

// SelectorSuccessRecord 选择器成功记录
type SelectorSuccessRecord struct {
	ID           string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PageURL      string    `json:"pageUrl" gorm:"index;not null"`
	ElementID    string    `json:"elementId" gorm:"index;not null"`
	Selector     string    `json:"selector" gorm:"not null"`
	SelectorType string    `json:"selectorType"` // css, xpath, text, aria, data-testid
	SuccessCount int       `json:"successCount" gorm:"default:1"`
	AvgDuration  int64     `json:"avgDuration"` // 平均查找时长（毫秒）
	LastUsedAt   time.Time `json:"lastUsedAt" gorm:"autoUpdateTime"`
	CreatedAt    time.Time `json:"createdAt" gorm:"autoCreateTime"`
	Metadata     string    `json:"metadata"` // JSON 格式的额外信息
}

// SelectorFailureRecord 选择器失败记录
type SelectorFailureRecord struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	PageURL      string     `json:"pageUrl" gorm:"index;not null"`
	ElementID    string     `json:"elementId" gorm:"index;not null"`
	Selector     string     `json:"selector" gorm:"not null"`
	ErrorType    string     `json:"errorType"` // timeout, not_found, multiple_matches
	ErrorMessage string     `json:"errorMessage"`
	FailureCount int        `json:"failureCount" gorm:"default:1"`
	CreatedAt    time.Time  `json:"createdAt" gorm:"autoCreateTime"`
	ResolvedAt   *time.Time `json:"resolvedAt"`   // 是否已解决
	ResolvedWith string     `json:"resolvedWith"` // 解决方案
}

// SelectorRecommendation 选择器推荐
type SelectorRecommendation struct {
	Selector     string    `json:"selector"`
	SelectorType string    `json:"selectorType"`
	Score        float64   `json:"score"`
	SuccessRate  float64   `json:"successRate"`
	UsageCount   int       `json:"usageCount"`
	AvgDuration  int64     `json:"avgDuration"`
	LastUsedAt   time.Time `json:"lastUsedAt"`
}

// SelectorStats 选择器统计
type SelectorStats struct {
	SuccessCount int       `json:"successCount"`
	FailureCount int       `json:"failureCount"`
	SuccessRate  float64   `json:"successRate"`
	LastUsedAt   time.Time `json:"lastUsedAt"`
}

// RecordSuccess 记录选择器成功
func (l *selectorLearnerImpl) RecordSuccess(ctx context.Context, record *SelectorSuccessRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查是否已存在
	var existing SelectorSuccessRecord
	err := l.db.WithContext(ctx).
		Where("page_url = ? AND element_id = ? AND selector = ?", record.PageURL, record.ElementID, record.Selector).
		First(&existing).Error

	if err == nil {
		// 更新现有记录
		updates := map[string]interface{}{
			"success_count": existing.SuccessCount + 1,
			"last_used_at":  time.Now(),
		}

		// 更新平均时长
		if record.AvgDuration > 0 {
			totalDuration := existing.AvgDuration*int64(existing.SuccessCount) + record.AvgDuration
			updates["avg_duration"] = totalDuration / int64(existing.SuccessCount+1)
		}

		// 同 RecordFailure：显式三元组条件（防 sqlite 空 id 丢 WHERE）
		if err := l.db.WithContext(ctx).Model(&SelectorSuccessRecord{}).
			Where("page_url = ? AND element_id = ? AND selector = ?", record.PageURL, record.ElementID, record.Selector).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("更新成功记录失败: %w", err)
		}
	} else {
		// 创建新记录
		if err := l.db.WithContext(ctx).Create(record).Error; err != nil {
			return fmt.Errorf("创建成功记录失败: %w", err)
		}
	}

	// 清除缓存
	// Phase 103 CONV-03 (D-103-14): cache.Delete → base.Invalidate（nil 防护 + 统一日志）
	base.Invalidate(ctx, l.cache, []string{l.getCacheKey(record.PageURL, record.ElementID)}, "SelectorLearner")

	return nil
}

// RecordFailure 记录选择器失败
func (l *selectorLearnerImpl) RecordFailure(ctx context.Context, record *SelectorFailureRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 检查是否已存在
	var existing SelectorFailureRecord
	err := l.db.WithContext(ctx).
		Where("page_url = ? AND element_id = ? AND selector = ?", record.PageURL, record.ElementID, record.Selector).
		First(&existing).Error

	if err == nil {
		// 更新现有记录
		updates := map[string]interface{}{
			"failure_count": existing.FailureCount + 1,
		}
		// ⚠ 显式三元组条件而非主键：sqlite 测试库的 id 由手写表承载时可能为空
		// （PG 下 gen_random_uuid 兜底），空主键会让 Updates 丢失 WHERE。
		if err := l.db.WithContext(ctx).Model(&SelectorFailureRecord{}).
			Where("page_url = ? AND element_id = ? AND selector = ?", record.PageURL, record.ElementID, record.Selector).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("更新失败记录失败: %w", err)
		}
		record = &existing
	} else {
		// 创建新记录
		if err := l.db.WithContext(ctx).Create(record).Error; err != nil {
			return fmt.Errorf("创建失败记录失败: %w", err)
		}
	}

	// 如果失败次数过多，标记该选择器需要更新
	if record.FailureCount >= 3 {
		l.markSelectorForUpdate(ctx, record)
	}

	return nil
}

// GetBestSelector 获取最佳选择器
func (l *selectorLearnerImpl) GetBestSelector(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error) {
	// Phase 103 CONV-03 (D-103-8): 手写 JSON cache-aside 收敛 getBestSelectorCached
	// wrapper，best-effort 静默语义保留（cache 层任何错误 → nil, nil 视为 miss）
	return l.getBestSelectorCached(ctx, pageURL, elementID)
}

// getBestSelectorCached 读穿透缓存 wrapper (Phase 103 CONV-03, D-103-8)。
//
// 语义等价说明（WR-01/WR-02 review 修正后的准确口径）：
//   - base.GetOrSetJSON 透传的错误只可能是 computeBestSelector 的 DB 查询错误
//     （provider 层读错误吞为 miss、写失败仅 warn），因此 err 直接一次传播
//     （等价原实现 DB 错误 `查询成功记录失败` 一次返回），不重算
//   - DB 查询返回 (nil, nil)（无记录）时静默返回，不缓存占位：GetOrSetJSON
//     会把 nil 序列化为 "null" 写缓存（与原实现 `if best != nil` 语义漂移，
//     WR-02），故 nil 结果主动失效该键恢复原语义
//   - cache 读失败被 provider 吞为 miss（原实现 cache.Get err → 静默走 DB 等价）
//
// TTL 保持 30*time.Minute 字面量（D-103-12）。
func (l *selectorLearnerImpl) getBestSelectorCached(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error) {
	cacheKey := l.getCacheKey(pageURL, elementID)

	best, err := base.GetOrSetJSON[*SelectorRecommendation](ctx, l.cache, cacheKey, 30*time.Minute, func() (*SelectorRecommendation, error) {
		return l.computeBestSelector(ctx, pageURL, elementID)
	})
	if err != nil {
		return nil, err
	}
	if best == nil {
		// 无记录不缓存占位（等价原实现 `if best != nil` 才 Set）
		base.Invalidate(ctx, l.cache, []string{cacheKey}, "SelectorLearner")
	}
	return best, nil
}

// computeBestSelector DB 直查最佳选择器（原 GetBestSelector 主体查询逻辑）。
// 返回 (nil, nil) 表示无记录；DB 查询错误透传。
func (l *selectorLearnerImpl) computeBestSelector(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 查询成功记录
	var successes []SelectorSuccessRecord
	if err := l.db.WithContext(ctx).
		Where("page_url = ? AND element_id = ?", pageURL, elementID).
		Order("success_count DESC, avg_duration ASC").
		Find(&successes).Error; err != nil {
		return nil, fmt.Errorf("查询成功记录失败: %w", err)
	}

	// 查询失败记录
	failuresMap := make(map[string]int)
	var failures []SelectorFailureRecord
	if err := l.db.WithContext(ctx).
		Where("page_url = ? AND element_id = ?", pageURL, elementID).
		Find(&failures).Error; err == nil {
		for _, f := range failures {
			failuresMap[f.Selector] += f.FailureCount
		}
	}

	// 计算每个选择器的得分
	var best *SelectorRecommendation
	maxScore := 0.0

	for _, success := range successes {
		stats := l.calculateSelectorStats(success, failuresMap[success.Selector])
		score := l.calculateScore(stats)

		rec := &SelectorRecommendation{
			Selector:     success.Selector,
			SelectorType: success.SelectorType,
			Score:        score,
			SuccessRate:  stats.SuccessRate,
			UsageCount:   stats.SuccessCount + stats.FailureCount,
			AvgDuration:  stats.LastUsedAt.Sub(success.CreatedAt).Milliseconds(),
			LastUsedAt:   success.LastUsedAt,
		}

		if score > maxScore {
			maxScore = score
			best = rec
		}
	}

	return best, nil
}

// selectorEventEnvelope worker 上报明细的解析信封（TIDY-03 :256 方案 A 契约）。
//
// 契约（worker 上报侧适配依据，零 migration）：执行记录的 Logs 字段（自由文本）
// 中允许嵌入一行合法 JSON：
//
//	{"selectorEvents":[{"pageUrl":"...","elementId":"...","selector":"...",
//	  "selectorType":"css|xpath|text|aria|data-testid","success":true,
//	  "durationMs":120,"errorType":"not_found","errorMessage":"..."}]}
//
// 解析规则：Logs 按 行 切分，逐行尝试 unmarshal；含 selectorEvents 的行被消费，
// 其余行（普通日志）静默跳过——两种形态共存安全。durationMs 仅 success 事件
// 有意义；errorType ∈ timeout/not_found/multiple_matches（与 FailureRecord 对齐）。
type selectorEventEnvelope struct {
	SelectorEvents []selectorEvent `json:"selectorEvents"`
}

type selectorEvent struct {
	PageURL      string `json:"pageUrl"`
	ElementID    string `json:"elementId"`
	Selector     string `json:"selector"`
	SelectorType string `json:"selectorType"`
	Success      bool   `json:"success"`
	DurationMs   int64  `json:"durationMs"`
	ErrorType    string `json:"errorType"`
	ErrorMessage string `json:"errorMessage"`
}

// LearnFromExecution 从执行记录中学习（TIDY-03 :256 实现，方案 A）。
//
// 流程：读 sys_rpa_executions.logs → 逐行解析 selectorEvents 信封 →
// success 事件走 RecordSuccess（复用既有 upsert + 缓存失效）、失败事件走
// RecordFailure（失败 ≥3 次自动触发 :389 的 selector_update_required 广播，
// 学习→通知闭环在此汇合）。
//
// 结果语义：返回 (consumed, error)——consumed=消费的事件数。执行记录不存在
// 或不含任何选择器明细时返回明确错误（调用方降级，不静默吞）。
// 单事件解析/记录失败不中断整批（记 WARN 继续），与「学习失败不阻断主同步」
// 的既有容忍语义一致。
func (l *selectorLearnerImpl) LearnFromExecution(ctx context.Context, executionID string) (int, error) {
	if strings.TrimSpace(executionID) == "" {
		return 0, fmt.Errorf("executionID 不能为空")
	}
	var exec rpamodels.Execution
	if err := l.db.WithContext(ctx).Where("id = ?", executionID).First(&exec).Error; err != nil {
		return 0, fmt.Errorf("查询执行记录失败: %w", err)
	}
	if strings.TrimSpace(exec.Logs) == "" {
		return 0, fmt.Errorf("执行记录 %s 的 logs 为空，无选择器明细可学习", executionID)
	}

	consumed := 0
	var firstErr error
	for _, line := range strings.Split(exec.Logs, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var envelope selectorEventEnvelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil || len(envelope.SelectorEvents) == 0 {
			continue
		}
		for _, ev := range envelope.SelectorEvents {
			if ev.PageURL == "" || ev.ElementID == "" || ev.Selector == "" {
				applogger.Warnf("[RPA-SELECTOR] 跳过不完整的选择器事件: page=%q element=%q selector=%q",
					ev.PageURL, ev.ElementID, ev.Selector)
				continue
			}
			var err error
			if ev.Success {
				err = l.RecordSuccess(ctx, &SelectorSuccessRecord{
					PageURL:      ev.PageURL,
					ElementID:    ev.ElementID,
					Selector:     ev.Selector,
					SelectorType: ev.SelectorType,
					AvgDuration:  ev.DurationMs,
				})
			} else {
				err = l.RecordFailure(ctx, &SelectorFailureRecord{
					PageURL:      ev.PageURL,
					ElementID:    ev.ElementID,
					Selector:     ev.Selector,
					ErrorType:    ev.ErrorType,
					ErrorMessage: ev.ErrorMessage,
				})
			}
			if err != nil {
				applogger.Warnf("[RPA-SELECTOR] 事件记录失败（继续处理其余事件）: %v", err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			consumed++
		}
	}

	if consumed == 0 && firstErr == nil {
		return 0, fmt.Errorf("执行记录 %s 的 logs 未解析出任何选择器明细（worker 未按契约上报 selectorEvents）", executionID)
	}
	return consumed, firstErr
}

// ScoreSelector 对选择器进行评分
func (l *selectorLearnerImpl) ScoreSelector(ctx context.Context, selector string, pageURL string) (float64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var success SelectorSuccessRecord
	err := l.db.WithContext(ctx).
		Where("page_url = ? AND selector = ?", pageURL, selector).
		First(&success).Error

	if err != nil {
		return 0.0, fmt.Errorf("选择器未找到")
	}

	// 查询失败次数
	var totalFailures int64
	l.db.WithContext(ctx).
		Model(&SelectorFailureRecord{}).
		Where("page_url = ? AND selector = ?", pageURL, selector).
		Select("COALESCE(SUM(failure_count), 0)").
		Scan(&totalFailures)

	stats := SelectorStats{
		SuccessCount: success.SuccessCount,
		FailureCount: int(totalFailures),
	}

	return l.calculateScore(stats), nil
}

// GetSelectorAlternatives 获取选择器的替代方案
func (l *selectorLearnerImpl) GetSelectorAlternatives(ctx context.Context, selector string, pageURL string) ([]string, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// 获取原始选择器的元素信息
	var record SelectorSuccessRecord
	err := l.db.WithContext(ctx).
		Where("selector = ? AND page_url = ?", selector, pageURL).
		First(&record).Error

	if err != nil {
		return nil, fmt.Errorf("未找到选择器记录")
	}

	// 查找同一元素的其他选择器
	var alternatives []SelectorSuccessRecord
	err = l.db.WithContext(ctx).
		Where("page_url = ? AND element_id = ? AND selector != ?", pageURL, record.ElementID, selector).
		Order("success_count DESC").
		Find(&alternatives).Error

	if err != nil {
		return nil, fmt.Errorf("查询替代选择器失败: %w", err)
	}

	result := make([]string, 0, len(alternatives))
	for _, alt := range alternatives {
		// 计算得分
		stats := l.calculateSelectorStats(alt, 0)
		score := l.calculateScore(stats)

		// 只返回得分较高的选择器
		if score > 0.5 {
			result = append(result, alt.Selector)
		}
	}

	return result, nil
}

// calculateSelectorStats 计算选择器统计信息
func (l *selectorLearnerImpl) calculateSelectorStats(success SelectorSuccessRecord, failureCount int) SelectorStats {
	total := success.SuccessCount + failureCount
	successRate := 0.0
	if total > 0 {
		successRate = float64(success.SuccessCount) / float64(total)
	}

	return SelectorStats{
		SuccessCount: success.SuccessCount,
		FailureCount: failureCount,
		SuccessRate:  successRate,
		LastUsedAt:   success.LastUsedAt,
	}
}

// calculateScore 计算选择器综合得分
func (l *selectorLearnerImpl) calculateScore(stats SelectorStats) float64 {
	if stats.SuccessCount == 0 {
		return 0.0
	}

	// 成功率权重 60%
	successRateWeight := 0.6
	// 使用频率权重 20%
	usageWeight := 0.2
	// 最近使用权重 20%
	recencyWeight := 0.2

	// 归一化使用次数（假设最大100次）
	usageScore := float64(stats.SuccessCount) / 100.0
	if usageScore > 1.0 {
		usageScore = 1.0
	}

	// 计算最近使用得分（30天内）
	daysSinceLastUse := time.Since(stats.LastUsedAt).Hours() / 24
	recencyScore := 1.0 - (daysSinceLastUse / 30.0)
	if recencyScore < 0.0 {
		recencyScore = 0.0
	}

	score := stats.SuccessRate*successRateWeight +
		usageScore*usageWeight +
		recencyScore*recencyWeight

	return score
}

// getCacheKey 获取缓存键
// D-102-3: fmt.Sprintf 改 systemServices.GetRpaSelectorBestKey helper
func (l *selectorLearnerImpl) getCacheKey(pageURL, elementID string) string {
	return systemServices.GetRpaSelectorBestKey(pageURL, elementID)
}

// markSelectorForUpdate 标记选择器需要更新（TIDY-03 :389 实现）。
//
// 通知机制（复用既有 NoticeHub WebSocket 链路，不新增通知基建）：
//   - noticeHub 注入时：BroadcastToAll 广播 selector_update_required 消息，
//     管理端（RPA AI 面板）据此提示「该选择器连续失败 ≥3 次，建议重新学习」；
//   - 未注入（旧构造签名/单测）：仅 WARN 日志，行为与 TODO 之前一致。
//
// 刻意不做的事：不落库「待处理任务」——SelectorFailureRecord 本身就是持久
// 待办（FailureCount + ResolvedAt 已表达未解决状态），再建任务表是第二真相。
func (l *selectorLearnerImpl) markSelectorForUpdate(ctx context.Context, record *SelectorFailureRecord) {
	if record == nil {
		return
	}
	applogger.Warnf("[RPA-SELECTOR] 选择器连续失败 %d 次，需要更新: page=%s element=%s selector=%s lastError=%s",
		record.FailureCount, record.PageURL, record.ElementID, record.Selector, record.ErrorMessage)

	if l.noticeHub == nil {
		return
	}
	l.noticeHub.BroadcastToAll(websocket.NoticeMessage{
		Type:  "selector_update_required",
		Title: "RPA 选择器需要更新",
		Content: fmt.Sprintf("选择器连续失败 %d 次（页面 %s / 元素 %s），建议在 RPA AI 面板重新学习。",
			record.FailureCount, record.PageURL, record.ElementID),
		Priority:  2,
		Timestamp: time.Now().Unix(),
	})
}

// GetSelectorHistory 获取选择器历史记录
func (l *selectorLearnerImpl) GetSelectorHistory(ctx context.Context, pageURL, elementID string, limit int) ([]SelectorSuccessRecord, error) {
	var records []SelectorSuccessRecord
	err := l.db.WithContext(ctx).
		Where("page_url = ? AND element_id = ?", pageURL, elementID).
		Order("last_used_at DESC").
		Limit(limit).
		Find(&records).Error

	return records, err
}

// AnalyzeSelectorTrends 分析选择器趋势
func (l *selectorLearnerImpl) AnalyzeSelectorTrends(ctx context.Context, pageURL string, days int) (*SelectorTrendAnalysis, error) {
	since := time.Now().AddDate(0, 0, -days)

	var successes []SelectorSuccessRecord
	if err := l.db.WithContext(ctx).
		Where("page_url = ? AND last_used_at >= ?", pageURL, since).
		Find(&successes).Error; err != nil {
		return nil, err
	}

	var failures []SelectorFailureRecord
	if err := l.db.WithContext(ctx).
		Where("page_url = ? AND created_at >= ?", pageURL, since).
		Find(&failures).Error; err != nil {
		return nil, err
	}

	analysis := &SelectorTrendAnalysis{
		PageURL:        pageURL,
		AnalyzedPeriod: days,
		TotalSuccesses: len(successes),
		TotalFailures:  len(failures),
	}

	// 计算成功率和最常用的选择器类型
	typeCount := make(map[string]int)
	for _, s := range successes {
		typeCount[s.SelectorType]++
	}

	analysis.MostUsedTypes = make([]SelectorTypeStat, 0, len(typeCount))
	for selectorType, count := range typeCount {
		analysis.MostUsedTypes = append(analysis.MostUsedTypes, SelectorTypeStat{
			Type:  selectorType,
			Count: count,
		})
	}

	if analysis.TotalSuccesses+analysis.TotalFailures > 0 {
		analysis.OverallSuccessRate = float64(analysis.TotalSuccesses) /
			float64(analysis.TotalSuccesses+analysis.TotalFailures)
	}

	return analysis, nil
}

// SelectorTrendAnalysis 选择器趋势分析
type SelectorTrendAnalysis struct {
	PageURL            string             `json:"pageUrl"`
	AnalyzedPeriod     int                `json:"analyzedPeriod"`
	TotalSuccesses     int                `json:"totalSuccesses"`
	TotalFailures      int                `json:"totalFailures"`
	OverallSuccessRate float64            `json:"overallSuccessRate"`
	MostUsedTypes      []SelectorTypeStat `json:"mostUsedTypes"`
}

// SelectorTypeStat 选择器类型统计
type SelectorTypeStat struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}
