package addomain

// Phase 144 · 同步漂移告警查询/配置/触发服务（后端端点数据源）。
//
// 端点：POST /system/ad-drift-alert/{status,logs,config,config/update,trigger}。
// 全部走 sys_config（热重载旋钮），不新增 yaml / config struct（避免双源冲突）。
// 分页 current 从 1；软失败：查询错误由 handler 用 response.Error 包装。

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// adDriftConfigKeys 9 个 sys_config 旋钮（5 既有 + 4 新增，PLAN 任务4 seed 同源）。
// 供 GetConfig / UpdateConfig 白名单校验，避免写任意 sys_config key。
var adDriftConfigKeys = []string{
	adDriftKeyEnabled,
	adDriftKeyTolerance,
	adDriftKeyThrottle,
	adDriftKeyNoticeEnabled,
	adDriftBaselineConfigKey,
	adDriftKeyWSEnabled,
	adDriftKeyWebhookEnabled,
	adDriftKeyWebhookConfigID,
	adDriftKeyBaselineKeyAlias,
}

// adDriftKeyBaselineKeyAlias 第 9 行 seed：原 FIX-11 注释提及的「基线存储用
// config_key」配置项。当前代码用常量 adDriftBaselineConfigKey 硬编码读基线，
// seed 该别名保持 9 行口径一致（前向兼容，不影响运行时）。
const adDriftKeyBaselineKeyAlias = "ad_sync.drift_alert.baseline_key"

// DriftAlertStatus 告警通道运行状态（GET /status 响应）。
type DriftAlertStatus struct {
	Enabled           bool   `json:"enabled"`
	Tolerance         int    `json:"tolerance"`
	ThrottleMinutes   int    `json:"throttleMinutes"`
	NoticeEnabled     bool   `json:"noticeEnabled"`
	WSEnabled         bool   `json:"wsEnabled"`
	WebhookEnabled    bool   `json:"webhookEnabled"`
	WebhookConfigID   string `json:"webhookConfigId"`
	Baseline          string `json:"baseline"`
	BaselineKey       string `json:"baselineKey"`
	LastAlertAt       string `json:"lastAlertAt"`
	LastAlertDrifted  bool   `json:"lastAlertDrifted"`
	AlertCount        int    `json:"alertCount"`
}

// DriftAlertConfigView 9 个旋钮的当前值（GET /config 响应）。
type DriftAlertConfigView map[string]string

// DriftAlertTriggerLog 一条触发记录（sys_oper_log 投影）。
type DriftAlertTriggerLog struct {
	ID           string `json:"id" gorm:"column:id"`
	Title        string `json:"title" gorm:"column:title"`
	Method       string `json:"method" gorm:"column:method"`
	OperTime     string `json:"operTime" gorm:"column:oper_time"`
	JsonResult   string `json:"jsonResult" gorm:"column:json_result"`
	Status       int    `json:"status" gorm:"column:status"`
}

// DriftAlertTriggerLogPage 分页响应（list/total/current/pageSize，current 从 1）。
type DriftAlertTriggerLogPage struct {
	List     []DriftAlertTriggerLog `json:"list"`
	Total    int64                  `json:"total"`
	Current  int                    `json:"current"`
	PageSize int                    `json:"pageSize"`
}

// GetStatus 暴露通道运行状态（mu 保护内存态；配置每轮热重载）。
func (s *ADSyncDriftAlertService) GetStatus(ctx context.Context) (DriftAlertStatus, error) {
	cfg := s.loadConfig()
	baseline, _ := s.readBaseline(ctx)
	baselineStr := ""
	// 直接读原始 config_value（含首次观测前的空值），保留可观测性
	var raw string
	if err := s.db.WithContext(ctx).Raw(
		"SELECT config_value FROM sys_config WHERE config_key = ? AND deleted_at IS NULL",
		adDriftBaselineConfigKey,
	).Scan(&raw).Error; err == nil {
		baselineStr = raw
	}
	_ = baseline

	s.mu.Lock()
	lastAt := s.lastAlertAt
	lastDrifted := s.lastAlertDrifted
	count := s.alertCount
	s.mu.Unlock()

	lastAtStr := ""
	if !lastAt.IsZero() {
		lastAtStr = lastAt.Format("2006-01-02 15:04:05")
	}
	return DriftAlertStatus{
		Enabled:          cfg.Enabled,
		Tolerance:        cfg.Tolerance,
		ThrottleMinutes:  cfg.ThrottleMin,
		NoticeEnabled:    cfg.NoticeEnabled,
		WSEnabled:        cfg.WSEnabled,
		WebhookEnabled:   cfg.WebhookEnabled,
		WebhookConfigID:  cfg.WebhookConfigID,
		Baseline:         baselineStr,
		BaselineKey:      adDriftBaselineConfigKey,
		LastAlertAt:      lastAtStr,
		LastAlertDrifted: lastDrifted,
		AlertCount:       count,
	}, nil
}

// ListTriggerLogs 分页查询触发记录（method=ADSyncDriftAlertService.CheckAndAlert）。
// current <1 归 1；pageSize 限 [1,200]。
func (s *ADSyncDriftAlertService) ListTriggerLogs(ctx context.Context, current, pageSize int) (DriftAlertTriggerLogPage, error) {
	if current < 1 {
		current = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}
	page := DriftAlertTriggerLogPage{
		List:     []DriftAlertTriggerLog{},
		Current:  current,
		PageSize: pageSize,
	}
	if !s.db.Migrator().HasTable(&models.OperLog{}) {
		return page, nil // 表缺失 → 空页，软降级
	}

	q := s.db.WithContext(ctx).Model(&models.OperLog{}).
		Where("method = ?", adDriftAlertMethod)
	if err := q.Count(&page.Total).Error; err != nil {
		return page, fmt.Errorf("统计漂移告警触发记录失败: %w", err)
	}
	var rows []DriftAlertTriggerLog
	if err := q.Select("id, title, method, oper_time, json_result, status").
		Order("oper_time DESC").
		Offset((current - 1) * pageSize).Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return page, fmt.Errorf("查询漂移告警触发记录失败: %w", err)
	}
	page.List = rows
	return page, nil
}

// GetConfig 读 9 个 sys_config 旋钮（缺行回落到默认值，不依赖 seed 才能跑）。
func (s *ADSyncDriftAlertService) GetConfig(ctx context.Context) (DriftAlertConfigView, error) {
	view := DriftAlertConfigView{}
	defaults := defaultADDriftConfigView()
	for _, key := range adDriftConfigKeys {
		view[key] = defaults[key]
	}
	if !s.db.Migrator().HasTable("sys_config") {
		return view, nil
	}
	var rows []struct {
		ConfigKey   string `gorm:"column:config_key"`
		ConfigValue string `gorm:"column:config_value"`
	}
	if err := s.db.WithContext(ctx).Raw(
		"SELECT config_key, config_value FROM sys_config WHERE config_key IN ? AND deleted_at IS NULL",
		adDriftConfigKeys,
	).Scan(&rows).Error; err != nil {
		return view, fmt.Errorf("读取漂移告警配置失败: %w", err)
	}
	for _, r := range rows {
		view[r.ConfigKey] = r.ConfigValue
	}
	return view, nil
}

// defaultADDriftConfigView 9 个旋钮的缺省值（与 loadConfig / migration seed 一致）。
func defaultADDriftConfigView() DriftAlertConfigView {
	return DriftAlertConfigView{
		adDriftKeyEnabled:          "true",
		adDriftKeyTolerance:        "0",
		adDriftKeyThrottle:         "60",
		adDriftKeyNoticeEnabled:    "true",
		adDriftBaselineConfigKey:   "",
		adDriftKeyWSEnabled:        "true",
		adDriftKeyWebhookEnabled:   "false",
		adDriftKeyWebhookConfigID:  "",
		adDriftKeyBaselineKeyAlias: adDriftBaselineConfigKey,
	}
}

// UpdateConfig 写 9 个 sys_config 旋钮（白名单过滤 + 幂等 upsert）。
// 只接受字符串值；数值/布尔合法性由下一轮 loadConfig 兜底（非法值回落默认）。
// 返回 error 由 handler 用 response.Error 包装。
func (s *ADSyncDriftAlertService) UpdateConfig(ctx context.Context, updates map[string]string) error {
	if len(updates) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(adDriftConfigKeys))
	for _, k := range adDriftConfigKeys {
		allowed[k] = true
	}
	// 显式事务：避免半更新；双方言纯 GORM（无 PG 专属语法）。
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for key, value := range updates {
			if !allowed[key] {
				continue // 白名单外 key（含 webhookUrl 等请求体字段）静默忽略
			}
			value = strings.TrimSpace(value)
			var existing models.Config
			err := tx.Where("config_key = ? AND deleted_at IS NULL", key).First(&existing).Error
			if err == nil {
				if uerr := tx.Model(&models.Config{}).Where("id = ?", existing.ID).
					Update("config_value", value).Error; uerr != nil {
					return fmt.Errorf("更新配置 %s 失败: %w", key, uerr)
				}
				continue
			}
			if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("查询配置 %s 失败: %w", key, err)
			}
			if cerr := tx.Create(&models.Config{
				ConfigKey:   key,
				ConfigName:  "同步漂移告警配置",
				ConfigValue: value,
			}).Error; cerr != nil {
				return fmt.Errorf("创建配置 %s 失败: %w", key, cerr)
			}
		}
		return nil
	})
}

// TriggerOnce 手动跑一轮 CheckAndAlert（写操作，供 trigger 端点）。
// 保持软失败契约：CheckAndAlert 任何错误仅内部 WARN 并返回 nil。
func (s *ADSyncDriftAlertService) TriggerOnce(ctx context.Context) error {
	return s.CheckAndAlert(ctx)
}

// parseADDriftInt 小工具：把 sys_config 字符串转 int（非法返回 fallback）。
func parseADDriftInt(v string, fallback int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
		return n
	}
	return fallback
}
