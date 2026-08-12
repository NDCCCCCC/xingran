package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	"gorm.io/gorm"
)

// SelectorLearner 选择器学习服务接口
type SelectorLearner interface {
	RecordSuccess(ctx context.Context, record *SelectorSuccessRecord) error
	RecordFailure(ctx context.Context, record *SelectorFailureRecord) error
	GetBestSelector(ctx context.Context, pageURL, elementID string) (*SelectorRecommendation, error)
	LearnFromExecution(ctx context.Context, executionID string) error
	ScoreSelector(ctx context.Context, selector string, pageURL string) (float64, error)
	GetSelectorAlternatives(ctx context.Context, selector string, pageURL string) ([]string, error)
}

// selectorLearnerImpl 选择器学习服务实现
type selectorLearnerImpl struct {
	db     *gorm.DB
	cache  cache.Cache
	config *config.Config
	mu     sync.RWMutex
}

// NewSelectorLearner 创建选择器学习服务
func NewSelectorLearner(db *gorm.DB, cache cache.Cache, cfg *config.Config) SelectorLearner {
	return &selectorLearnerImpl{
		db:     db,
		cache:  cache,
		config: cfg,
	}
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

		if err := l.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新成功记录失败: %w", err)
		}
	} else {
		// 创建新记录
		if err := l.db.WithContext(ctx).Create(record).Error; err != nil {
			return fmt.Errorf("创建成功记录失败: %w", err)
		}
	}

	// 清除缓存
	cacheKey := l.getCacheKey(record.PageURL, record.ElementID)
	_ = l.cache.Delete(ctx, cacheKey)

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
		if err := l.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
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
	// 先检查缓存
	cacheKey := l.getCacheKey(pageURL, elementID)
	if cached, err := l.cache.Get(ctx, cacheKey); err == nil {
		var result SelectorRecommendation
		if err := json.Unmarshal([]byte(cached), &result); err == nil {
			return &result, nil
		}
	}

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

	// 缓存结果
	if best != nil {
		if data, err := json.Marshal(best); err == nil {
			_ = l.cache.Set(ctx, cacheKey, string(data), 30*time.Minute)
		}
	}

	return best, nil
}

// LearnFromExecution 从执行记录中学习
func (l *selectorLearnerImpl) LearnFromExecution(ctx context.Context, executionID string) error {
	// TODO: 从执行记录中提取选择器使用情况
	// 这个功能需要在执行记录中存储详细的选择器信息
	return nil
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
func (l *selectorLearnerImpl) getCacheKey(pageURL, elementID string) string {
	return fmt.Sprintf("rpa:selector:best:%s:%s", pageURL, elementID)
}

// markSelectorForUpdate 标记选择器需要更新
func (l *selectorLearnerImpl) markSelectorForUpdate(ctx context.Context, record *SelectorFailureRecord) {
	// TODO: 实现选择器更新通知机制
	// 可以通过 WebSocket 通知前端，或创建待处理任务
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
