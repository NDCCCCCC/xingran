package addomain

// Phase 134 / FIX-11：AD 同步漂移常态化告警通道（可配置 / 可观测 / 有触发记录）。
//
// 背景（2026-11-08 开放项盘点 #A7 / REQUIREMENTS.md FIX-11 / RETROSPECTIVE.md:418）：
//   scripts/uat61-t1-poller-v5.ps1 是一次性 UAT 重放窗口采样脚本（3025 次采样，
//   0.887s 窗口），**不是告警通道**。本服务把"周期性采样 + 漂移判定"产品化，
//   方向与 A8① 裁定「加告警通道而非改回整批 403」一致：不改动任何既有读接口的
//   403 行为，只新增一条常态化告警通道。
//
// 采样口径（对齐 uat61-t1-poller-v5.ps1 的两列）：
//   - sys_ad_group_member 总行数（member_count）
//   - 去重 group_dn 数（distinct_groups）
// 漂移定义：相邻两次采样之间，member_count 或 distinct_groups 发生**下降**，
// 或偏差超过可配置阈值（默认 0 = 任何变化即告警），视为一次同步漂移。
// 全量同步窗口内逐条写入导致短时抖动，故用「基线 + 容差」+「节流」双层降噪
// （复用 reconciliation_tasks.go checkPortStatusDrift 的自适应基线思想 +
//  fix_suggestion_monitor.go 的节流思想），避免把一次正常同步拆成告警风暴。
//
// 三通道（全部复用项目既有设施，零新依赖）：
//   1. applogger.Warnf —— 结构化可观测日志（含全部诊断字段）；
//   2. sys_oper_log    —— 触发记录（OperTypeSync=14，direct INSERT，无需 gin.Context）；
//   3. sys_notice      —— 面向运维的通知（notice_type='2' 警告 / 立即发布 / TargetAll）。
//
// 配置（sys_config，热重载，每轮读一次）：
//   ad_sync.drift_alert.enabled          默认 true
//   ad_sync.drift_alert.tolerance        默认 0（任一计数变化即告警）
//   ad_sync.drift_alert.baseline_key     基线存储用的 config_key（默认见常量）
//   ad_sync.drift_alert.throttle_minutes 默认 60（同一持续漂移 1h 最多 1 次通知）
//   ad_sync.drift_alert.notice_enabled   默认 true（是否写 sys_notice）
//
// ⚠ 边界（交付说明会明确）：本服务 = 可复用检测服务 + 配置 + 单测 + cron 接入点，
//   未做前端告警页 / WS 推送 / 企业微信 webhook（属后续产品化增量，不伪造完成）。
//   软失败：任何一步失败仅 applogger.Warnf，绝不阻断 cron 主流程（error 返回不 panic）。

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// ==================== 配置旋钮 ====================

const (
	adDriftKeyEnabled        = "ad_sync.drift_alert.enabled"
	adDriftKeyTolerance      = "ad_sync.drift_alert.tolerance"
	adDriftKeyThrottle       = "ad_sync.drift_alert.throttle_minutes"
	adDriftKeyNoticeEnabled  = "ad_sync.drift_alert.notice_enabled"
	adDriftBaselineConfigKey = "ad_sync.drift_alert.baseline" // 基线存 sys_config（member_count,distinct_groups）

	adDriftDefaultEnabled        = true
	adDriftDefaultTolerance      = 0
	adDriftDefaultThrottleMin    = 60
	adDriftDefaultNoticeEnabled  = true
	adDriftAlertTitle            = "AD 同步漂移告警"
	adDriftAlertContentPrefix    = "[ad_sync_drift_alert]"
	adDriftAlertMethod           = "ADSyncDriftAlertService.CheckAndAlert"
	adDriftAlertNoticeType       = "2" // 警告
	adDriftAlertOperBusinessType = 14  // OperTypeSync
)

// adDriftConfig 漂移告警配置。
type adDriftConfig struct {
	Enabled       bool
	Tolerance     int // 计数变化绝对值 <= 容差 视为噪声，不告警
	ThrottleMin   int // 同一持续漂移的告警节流分钟数
	NoticeEnabled bool
}

func defaultADDriftConfig() adDriftConfig {
	return adDriftConfig{
		Enabled:       adDriftDefaultEnabled,
		Tolerance:     adDriftDefaultTolerance,
		ThrottleMin:   adDriftDefaultThrottleMin,
		NoticeEnabled: adDriftDefaultNoticeEnabled,
	}
}

// ==================== 采样 / 判定纯数据结构 ====================

// adDriftSample 一次采样的两个计数（对齐 uat61 脚本 CSV 的两列）。
type adDriftSample struct {
	MemberCount    int64
	DistinctGroups int64
}

// adDriftVerdict 漂移判定结果（纯数据，便于单测）。
type adDriftVerdict struct {
	Drift       bool
	FirstSample bool // 无历史基线（首次观测，只记基线不告警）
	Recovered   bool // 计数恢复/上涨（基线下降，log "数据治理见效"）
	MemberDelta int64
	GroupDelta  int64
}

// detectADDrift 纯函数：与基线比较，判定是否漂移。
//
// 规则（与 reconciliation_tasks.go checkPortStatusDrift 自适应基线同构）：
//   - 无基线（exists=false）→ 首次观测，不告警；
//   - 任一计数**下降**超过容差 → 漂移（同步异常导致成员关系丢失，最危险）；
//   - 任一计数**变化**绝对值超过容差（含上涨）→ 漂移（同步窗口抖动/漂移）；
//   - 计数下跌后回升 / 持平容差内 → 不告警，基线按"历史最低"自适应更新。
//
// tolerance=0 时任何计数变化即告警（最敏感，适配 uat61「零偏差」验收口径）。
func detectADDrift(cur adDriftSample, baseline adDriftSample, baselineExists bool, tolerance int) adDriftVerdict {
	if !baselineExists {
		return adDriftVerdict{FirstSample: true}
	}
	dm := cur.MemberCount - baseline.MemberCount
	dg := cur.DistinctGroups - baseline.DistinctGroups
	abs := func(v int64) int64 {
		if v < 0 {
			return -v
		}
		return v
	}
	tol := int64(tolerance)
	if abs(dm) <= tol && abs(dg) <= tol {
		return adDriftVerdict{MemberDelta: dm, GroupDelta: dg}
	}
	return adDriftVerdict{
		Drift:       true,
		Recovered:   dm > 0 || dg > 0,
		MemberDelta: dm,
		GroupDelta:  dg,
	}
}

// ==================== 服务 ====================

// ADSyncDriftAlertService AD 同步漂移常态化告警服务。
//
// 依赖注入：db（*gorm.DB）+ 可选 now / cfg 注入点（单测用；生产零值走真实 time.Now /
// sys_config）。无 WS / 无 NoticeService 依赖，直接写 sys_notice + sys_oper_log。
//
// 状态：baseline 存 sys_config（跨进程/重启可观测，复用 checkPortStatusDrift 先例），
// 节流状态存内存（进程重启后重新计时，可接受）。
type ADSyncDriftAlertService struct {
	db  *gorm.DB
	now func() time.Time // 测试注入；生产 nil → time.Now

	mu               sync.Mutex
	lastAlertAt      time.Time
	lastAlertDrifted bool
	alertCount       int // 累计触发次数（可观测 / 单测断言）
}

func NewADSyncDriftAlertService(db *gorm.DB) *ADSyncDriftAlertService {
	return &ADSyncDriftAlertService{db: db}
}

func (s *ADSyncDriftAlertService) timeNow() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

// CheckAndAlert 单轮检测入口（cron / 手动端点均可调用）。
//
// 流程：读配置 → 采样 → 读基线 → 纯函数判定 → 首次观测记基线 / 漂移则节流+三通道告警
// / 恢复则更新基线 → 写回基线。软失败：任何错误仅 WARN，返回 nil（不阻断 cron）。
func (s *ADSyncDriftAlertService) CheckAndAlert(ctx context.Context) error {
	if s.db == nil {
		applogger.Warnf("[ad_sync:drift] db 未注入，跳过本轮漂移检测")
		return nil
	}
	cfg := s.loadConfig()
	if !cfg.Enabled {
		applogger.Debugf("[ad_sync:drift] 告警通道已关闭（%s=false），跳过", adDriftKeyEnabled)
		return nil
	}

	cur, err := s.sample(ctx)
	if err != nil {
		applogger.Warnf("[ad_sync:drift] 采样失败(忽略，不阻断): %v", err)
		return nil
	}
	baseline, exists := s.readBaseline(ctx)
	verdict := detectADDrift(cur, baseline, exists, cfg.Tolerance)

	if verdict.FirstSample {
		applogger.Infof("[ad_sync:drift] 首次观测 member_count=%d distinct_groups=%d，记录为基线(不告警)",
			cur.MemberCount, cur.DistinctGroups)
		s.writeBaseline(ctx, cur)
		return nil
	}

	if verdict.Drift {
		// 漂移期间基线**不随上涨更新**（持续告警直到恢复/人工处理，同 checkPortStatusDrift）
		shouldAlert := s.shouldAlert(s.timeNow(), cfg.ThrottleMin)
		if shouldAlert {
			s.emit(ctx, cur, baseline, verdict, cfg)
		} else {
			applogger.Debugf("[ad_sync:drift] 漂移持续但节流中(member_delta=%d group_delta=%d)，跳过通知",
				verdict.MemberDelta, verdict.GroupDelta)
		}
		return nil
	}

	// 持平 / 恢复：自适应更新基线为"历史最低"（数据治理见效，不锁死在高值）
	if cur.MemberCount < baseline.MemberCount || cur.DistinctGroups < baseline.DistinctGroups {
		applogger.Infof("[ad_sync:drift] 计数下降 member=%d→%d groups=%d→%d，基线自适应更新",
			baseline.MemberCount, cur.MemberCount, baseline.DistinctGroups, cur.DistinctGroups)
	}
	s.resetDriftState() // 漂移结束：清节流标记，下一次新漂移可立即触发"首次告警"
	s.writeBaseline(ctx, cur)
	applogger.Infof("[ad_sync:drift] 健康 member_count=%d distinct_groups=%d", cur.MemberCount, cur.DistinctGroups)
	return nil
}

// sample 采样 sys_ad_group_member（双方言纯 GORM，无 ::text 等 PG 专属语法）。
// 表缺失（未初始化环境）→ 返回 error 由调用方软降级，不 panic。
func (s *ADSyncDriftAlertService) sample(ctx context.Context) (adDriftSample, error) {
	var out adDriftSample
	if !s.db.Migrator().HasTable(&models.ADGroupMember{}) {
		return out, fmt.Errorf("sys_ad_group_member 表不存在（AD 同步尚未初始化）")
	}
	row := s.db.WithContext(ctx).Model(&models.ADGroupMember{}).
		Select("COUNT(*) AS member_count, COUNT(DISTINCT group_dn) AS distinct_groups").
		Row()
	if row == nil {
		return out, fmt.Errorf("采样返回空行")
	}
	if err := row.Scan(&out.MemberCount, &out.DistinctGroups); err != nil {
		return out, fmt.Errorf("扫描采样结果失败: %w", err)
	}
	return out, nil
}

// ==================== 配置 / 基线（sys_config 热重载） ====================

func (s *ADSyncDriftAlertService) loadConfig() adDriftConfig {
	cfg := defaultADDriftConfig()
	if !s.db.Migrator().HasTable("sys_config") {
		return cfg // 表缺失 → 全默认，零噪音
	}
	read := func(key string) (string, bool) {
		var v string
		if err := s.db.Raw(
			"SELECT config_value FROM sys_config WHERE config_key = ? AND deleted_at IS NULL", key,
		).Scan(&v).Error; err != nil || v == "" {
			return "", false
		}
		return v, true
	}
	if v, ok := read(adDriftKeyEnabled); ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "0", "false", "no", "off":
			cfg.Enabled = false
		case "1", "true", "yes", "on":
			cfg.Enabled = true
		}
	}
	if v, ok := read(adDriftKeyTolerance); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
			cfg.Tolerance = n
		}
	}
	if v, ok := read(adDriftKeyThrottle); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
			cfg.ThrottleMin = n
		}
	}
	if v, ok := read(adDriftKeyNoticeEnabled); ok {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "0", "false", "no", "off":
			cfg.NoticeEnabled = false
		case "1", "true", "yes", "on":
			cfg.NoticeEnabled = true
		}
	}
	return cfg
}

// baseline 编码 "member_count,distinct_groups" 存入 sys_config，避免加表/加列/迁移。
func (s *ADSyncDriftAlertService) readBaseline(ctx context.Context) (adDriftSample, bool) {
	var v string
	if err := s.db.WithContext(ctx).Raw(
		"SELECT config_value FROM sys_config WHERE config_key = ? AND deleted_at IS NULL",
		adDriftBaselineConfigKey,
	).Scan(&v).Error; err != nil || v == "" {
		return adDriftSample{}, false
	}
	parts := strings.Split(v, ",")
	if len(parts) != 2 {
		return adDriftSample{}, false
	}
	m, errM := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	g, errG := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	if errM != nil || errG != nil {
		return adDriftSample{}, false
	}
	return adDriftSample{MemberCount: m, DistinctGroups: g}, true
}

func (s *ADSyncDriftAlertService) writeBaseline(ctx context.Context, sample adDriftSample) {
	value := fmt.Sprintf("%d,%d", sample.MemberCount, sample.DistinctGroups)
	const name = "AD 同步漂移基线(member_count,distinct_groups)"

	// 显式「先查后 UPDATE / 无则 INSERT」：不用 FirstOrCreate 的 Assign 变体 ——
	// 后者在 sys_config.config_key 唯一索引下会生成 INSERT 并撞 UNIQUE(2067)，
	// 导致基线永远停在首轮值、后续每轮重复误告警（软失败掩盖了真实缺陷）。
	var existing models.Config
	err := s.db.WithContext(ctx).
		Where("config_key = ? AND deleted_at IS NULL", adDriftBaselineConfigKey).
		First(&existing).Error
	if err == nil {
		if uerr := s.db.WithContext(ctx).Model(&models.Config{}).
			Where("id = ?", existing.ID).
			Update("config_value", value).Error; uerr != nil {
			applogger.Warnf("[ad_sync:drift] 写回基线失败(忽略): %v", uerr)
		}
		return
	}
	if cerr := s.db.WithContext(ctx).Create(&models.Config{
		ConfigKey:   adDriftBaselineConfigKey,
		ConfigName:  name,
		ConfigValue: value,
	}).Error; cerr != nil {
		applogger.Warnf("[ad_sync:drift] 创建基线失败(忽略): %v", cerr)
	}
}

// ==================== 节流 ====================

// shouldAlert 状态变化（非漂移→漂移）必告警；持续漂移按 throttleMin 节流。
// 返回 true 时已推进 lastAlertAt / alertCount（可观测）。
func (s *ADSyncDriftAlertService) shouldAlert(now time.Time, throttleMin int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	stateChange := !s.lastAlertDrifted
	s.lastAlertDrifted = true
	interval := time.Duration(throttleMin) * time.Minute
	if stateChange || now.Sub(s.lastAlertAt) >= interval {
		s.lastAlertAt = now
		s.alertCount++
		return true
	}
	return false
}

// resetDriftState 漂移恢复时清理节流标记，使下一次新漂移能立即触发"首次告警"。
func (s *ADSyncDriftAlertService) resetDriftState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAlertDrifted = false
}

// ==================== 三通道告警 ====================

// emit 触发记录的三通道产出：applogger + sys_oper_log + sys_notice。
// 任一通道失败仅 WARN，不影响其它通道，也不返回 error（软失败）。
func (s *ADSyncDriftAlertService) emit(ctx context.Context, cur, baseline adDriftSample, v adDriftVerdict, cfg adDriftConfig) {
	content := fmt.Sprintf(
		"%s\nAD 同步漂移：member_count %d→%d (Δ%d)，distinct_groups %d→%d (Δ%d)。请核对最近一轮全量同步与 sys_ad_group_member 重放窗口。",
		adDriftAlertContentPrefix,
		baseline.MemberCount, cur.MemberCount, v.MemberDelta,
		baseline.DistinctGroups, cur.DistinctGroups, v.GroupDelta)

	// 通道 1：结构化 WARN（可观测，含全量诊断字段）
	applogger.Warnf("[ad_sync:drift] %s member_count=%d→%d distinct_groups=%d→%d member_delta=%d group_delta=%d tolerance=%d",
		"检测到同步漂移", baseline.MemberCount, cur.MemberCount,
		baseline.DistinctGroups, cur.DistinctGroups, v.MemberDelta, v.GroupDelta, cfg.Tolerance)

	// 通道 2：sys_oper_log 触发记录（OperTypeSync=14；direct INSERT 无需 gin.Context）
	now := s.timeNow()
	jsonResult := fmt.Sprintf(`{"memberCount":%d,"distinctGroups":%d,"baseline":[%d,%d],"memberDelta":%d,"groupDelta":%d}`,
		cur.MemberCount, cur.DistinctGroups, baseline.MemberCount, baseline.DistinctGroups, v.MemberDelta, v.GroupDelta)
	operLog := &models.OperLog{
		Title:        adDriftAlertTitle,
		BusinessType: adDriftAlertOperBusinessType,
		Method:       adDriftAlertMethod,
		Status:       int(models.OperLogStatusSuccess),
		OperTime:     now,
		JsonResult:   &jsonResult,
	}
	if err := s.db.WithContext(ctx).Create(operLog).Error; err != nil {
		applogger.Warnf("[ad_sync:drift] operlog 触发记录写入失败(忽略): %v", err)
	} else {
		applogger.Infof("[ad_sync:drift] 触发记录已写 sys_oper_log id=%s", operLog.ID)
	}

	// 通道 3：sys_notice 通知（notice_type='2' 警告 / 立即发布 / TargetAll）
	if cfg.NoticeEnabled && s.db.Migrator().HasTable(&models.Notice{}) {
		publishAt := now
		notice := &models.Notice{
			NoticeTitle:   adDriftAlertTitle,
			NoticeType:    adDriftAlertNoticeType,
			NoticeContent: content,
			Status:        int(models.NoticeStatusNormal),
			Priority:      models.PriorityUrgent,
			PublishStatus: models.PublishStatusPublished,
			PublishTime:   &publishAt,
			TargetType:    models.TargetAll,
			CreatedByName: "SYSTEM",
		}
		if err := s.db.WithContext(ctx).Create(notice).Error; err != nil {
			applogger.Warnf("[ad_sync:drift] sys_notice 写入失败(忽略): %v", err)
		} else {
			applogger.Infof("[ad_sync:drift] 告警通知已发布 sys_notice id=%s", notice.ID)
		}
	}
}

// AlertCount 累计告警触发次数（排障 / 单测 / 可观测用）。
func (s *ADSyncDriftAlertService) AlertCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.alertCount
}
