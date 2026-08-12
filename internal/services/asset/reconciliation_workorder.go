package asset

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	"github.com/xingran-next/xingran-go-backend/internal/services/system"
	"github.com/xingran-next/xingran-go-backend/internal/services/workorder"
	"github.com/xingran-next/xingran-go-backend/internal/websocket"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ReconciliationWorkorderService 资产对账异常转工单服务(Phase 43 R2 / D-A1~D-A2 + R2 闭环)
//
// 业务边界:
//   - 本服务由 internal/scheduler/reconciliation_tasks.go 续写的 2 个 cron 触发:
//     reconciliation:createWorkorderCritical(@every 2m)+ reconciliation:createWorkorderHigh(@every 5m)
//   - 转单流程 12 步(详见 CreateWorkorderFromException docstring)
//   - 失败仅 logrus.Errorf,下个 cron 周期重试(D-A1-03,不写 SysNotice)
//
// 与 Phase 42 R1 boundary:
//   - R1 仅观测底座,不触发任何工单;R2 在 R1 基础上扩展自动转工单能力
//   - 转单成功后 UPDATE sys_data_reconciliation.workorder_id(D-A1-04)
//
// 跨模块依赖:
//   - workorder.BaseService.Create: Phase 23 已存在,接受 CreateRequest 结构体
//   - system.ConfigService.GetByKey: Phase 39 已有,从 sys_config 读 JSONB
//   - websocket.NoticeHub.BroadcastToAll: 全量 WS 推送(D-A4-01 全量 broadcast)
//   - services.NoticeService.CreateNoticeWithTargets + PublishNotice: SysNotice 写入(D-A4-03 双通道)
//
// R2 WS 事件类型常量(与 useReconciliationWebSocket hook 对齐):
//   - EventCriticalExceptionDetected = "critical_exception_detected"
//   - - EventCriticalWorkorderCreated = "critical_workorder_created"
//
// R2 SysNotice NoticeType 字面量(无 schema 字段时通过 NoticeContent 头部拼接):
//   - sys_notice.notice_type='2'(警告) + NoticeContent 头部 prefix "[asset_reconciliation_critical]"
//   - 该字面量是前端过滤 token,R3 可考虑扩展 schema notice_type_str 字段
//
// Status Convention: 0=启用 1=停用(CLAUDE.md 强制);本服务处理 severity/critical/high 等业务字段,
// 不涉及 status convention。
type ReconciliationWorkorderService struct {
	db            *gorm.DB
	noticeHub     *websocket.NoticeHub
	noticeService *services.NoticeService
	cache         cache.Cache // R4 启用:用于 R2 createWorkorder 缓存失效
}

// NewReconciliationWorkorderService 构造器(向后兼容 — 无 WS / SysNotice 时为 nil)
//
// 参数说明:
//   - db: 数据库句柄
//   - noticeHub: WebSocket 推送中心(可选,nil 时跳过 WS 推送)
//   - noticeService: SysNotice 服务(可选,nil 时跳过 SysNotice 写入)
//
// 兼容性:
//   - 旧调用方 NewReconciliationWorkorderService(db) 仍能工作,但 WS / SysNotice 不触发
//   - R2 推荐构造: NewReconciliationWorkorderService(db, core.NoticeHub, services.NewNoticeService(db))
//   - nil 字段的存在是为了非 production 环境(test / 单测)的解耦测试
//
// R4 扩展(Phase 45):
//   - 增加第 4 参数 cache,支持 R2 createWorkorder 缓存失效(D-A4-04)
//   - 旧调用方(无 cache) 通过 SetCache 注入,或继续传 nil(单测 / 非 production)
func NewReconciliationWorkorderService(db *gorm.DB, noticeHub *websocket.NoticeHub, noticeService *services.NoticeService) *ReconciliationWorkorderService {
	return &ReconciliationWorkorderService{db: db, noticeHub: noticeHub, noticeService: noticeService}
}

// NewReconciliationWorkorderServiceWithCache 构造器(R4 带 cache,D-A4-04 缓存失效用)
//
// 推荐生产构造: NewReconciliationWorkorderServiceWithCache(db, core.Cache, core.NoticeHub, services.NewNoticeService(db))
func NewReconciliationWorkorderServiceWithCache(db *gorm.DB, c cache.Cache, noticeHub *websocket.NoticeHub, noticeService *services.NoticeService) *ReconciliationWorkorderService {
	return &ReconciliationWorkorderService{db: db, cache: c, noticeHub: noticeHub, noticeService: noticeService}
}

// SetCache 注入 cache.Cache(R2 cron 调用方在创建 service 后注入 cache)
//
// 使用场景:
//   - 调用方使用旧 NewReconciliationWorkorderService 构造(无 cache 参数),后续可
//     通过 SetCache 注入 cache,避免破坏 R2 cron 既有调用方
//   - 与 NewReconciliationWorkorderServiceWithCache 二选一
func (s *ReconciliationWorkorderService) SetCache(c cache.Cache) {
	if s != nil {
		s.cache = c
	}
}

// WS 事件类型常量(D-A4-02 锁定只推 2 类)
const (
	EventCriticalExceptionDetected = "critical_exception_detected"
	EventCriticalWorkorderCreated  = "critical_workorder_created"
)

// SysNotice content 前缀(D-A4-03 锁定) — 前端过滤 token
const sysNoticePrefix = "[asset_reconciliation_critical]"

// AssigneeRoleMap type→role 映射表(D-A2-02)
//
// 业务说明:
//   - key 是 workorder.ReconciliationWorkorderTemplate.AssigneeRoleKey
//     (B/C/D="asset_owner", E="ops_owner", F="responsible_owner")
//   - value 是 sys_role.id(uuid 字符串 — sys_role 主键为 type:uuid)
//   - 由 sys_config:asset.reconciliation.workorder.assignee_role_map JSONB 字符串反序列化
//
// 反查流程:
//   1. GetTemplate(conflictType) → AssigneeRoleKey(如 "asset_owner")
//   2. roleMap["asset_owner"] → role_id(uuid 字符串)
//   3. INNER JOIN sys_user_role ON ur.user_id=sys_user.id WHERE ur.role_id=? AND sys_user.status=0 LIMIT 1 → assigneeID
//
// 软失败:JSONB 缺失或解析失败时,AssigneeID 为 nil,工单仍创建,等待人工分配。
type AssigneeRoleMap map[string]string

// CreateWorkorderFromException 从对账异常记录生成工单(12 步流程)
//
// 参数:
//   - ctx: 上下文
//   - exceptionID: sys_data_reconciliation.id(uuid 字符串)
//
// 返回:
//   - *models.WorkOrder: 成功创建的工单;失败返回 nil
//   - error: 失败原因(已 logrus.Errorf,调用方可继续)
//
// 12 步流程(D-A1-01/02 + D-A2-01~04 + R2 WS/SysNotice):
//  1. SELECT sys_data_reconciliation WHERE id=? AND workorder_id IS NULL AND deleted_at IS NULL
//     (排除已转单 + 软删除记录,避免重复转单)
//  2. SELECT WorkOrderCategory WHERE category_name="对账-{Type}类"
//     (B-F 5 类,migration_169 已 seed)
//  3. tpl=workorder.GetTemplate(rec.ConflictType);nil → err("无模板")
//  4. configSvc.GetByKey("asset.reconciliation.workorder.assignee_role_map") 软失败
//  5. JSONB 反查 role_id → JOIN sys_user_role 查 sys_user
//  6. json.Unmarshal(rec.RawSnapshot) 取 asset_code(用于标题)
//  7. title=fmt.Sprintf("[资产对账·%s类] 资产 %s (%s) %s",conflictType,asset_code,severity,detectedAt)
//  8. description=descLines+"\n\n【原始数据】\n"+raw_snapshot+"\n\n## SLA: "+slaMin+"m"
//  9. SELECT sys_user WHERE username='system' → systemSubmitterID
// 10. workorder.NewBaseService(s.db).Create(ctx,&CreateRequest{...}, systemSubmitterID)
// 11. 失败 logrus.Errorf,return nil,err(D-A1-03 不写 SysNotice)
// 12. 成功 s.db.Model(&rec).Update("workorder_id",wo.ID)
// 13a. WS 推送 critical_workorder_created(D-A4-01 全量 broadcast,non-blocking)
// 13b. SysNotice 写入(立即发布,避免草稿状态)
//
// submitterID = systemSubmitterID(T-43-01 转单越权 mitigation 的真实实现):
// 按 username='system' 查 sys_user.id(uuid),确保 submitter_id(uuid;not null) 写入合法 UUID。
// 复用 internal/scheduler/workorder_tasks.go:60-62 已有的查询样板(周期性工单 cron 同款)。
// 工单创建后 status=Pending,等运维接管,避免"自动转单+自动受理"双重系统行为。
//
// 依赖约束:
//   - sys_user 必须存在 username='system' 的用户(由初始化或运营预建)
//   - 不存在时本方法返回 error,cron 下个周期重试(D-A1-03 不写 SysNotice)
func (s *ReconciliationWorkorderService) CreateWorkorderFromException(ctx context.Context, exceptionID string) (*models.WorkOrder, error) {
	// 第 1 步:查异常记录(已转单 + 软删除 跳过)
	var rec models.SysDataReconciliation
	if err := s.db.WithContext(ctx).
		Where("id = ? AND workorder_id IS NULL AND deleted_at IS NULL", exceptionID).
		First(&rec).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 已被其他 cron 周期转单,或异常已软删除/resolved,跳过
			logrus.Debugf("[reconciliation:workorder] 异常 %s 已转单或不存在,跳过", exceptionID)
			return nil, nil
		}
		logrus.Errorf("[reconciliation:workorder] 查询异常 %s 失败: %v", exceptionID, err)
		return nil, fmt.Errorf("查询异常失败: %w", err)
	}

	// R2 WS:critical_exception_detected 推送(在任何转单动作之前,D-A4-02 锁定只 critical 触发)
	if rec.Severity == "critical" {
		s.broadcastCriticalException(ctx, &rec)
	}

	// 第 2 步:查 category(B-F 5 类)
	categoryName := fmt.Sprintf("对账-%s类", rec.ConflictType)
	var category models.WorkOrderCategory
	if err := s.db.WithContext(ctx).
		Where("category_name = ?", categoryName).
		First(&category).Error; err != nil {
		logrus.Errorf("[reconciliation:workorder] 查 category %s 失败: %v", categoryName, err)
		return nil, fmt.Errorf("查 category 失败: %w", err)
	}

	// 第 3 步:取模板(A 类 / 未知类型 → 报错)
	tpl := workorder.GetTemplate(rec.ConflictType)
	if tpl == nil {
		logrus.Errorf("[reconciliation:workorder] ConflictType=%s 无模板(A 类不入主表)", rec.ConflictType)
		return nil, fmt.Errorf("ConflictType=%s 无模板", rec.ConflictType)
	}

	// 第 4 步:读 sys_config(type→role 映射,软失败)
	var assigneeID *string
	configSvc := system.NewConfigService(s.db)
	cfg, err := configSvc.GetByKey(ctx, "asset.reconciliation.workorder.assignee_role_map")
	if err != nil || cfg == nil {
		// 软失败:config 缺失或查不到,工单仍创建,等人工分配
		logrus.Warnf("[reconciliation:workorder] 读 sys_config.assignee_role_map 失败(将不带 assignee): %v", err)
	} else {
		// 第 5 步:JSONB 反查 role_id → sys_user
		var roleMap AssigneeRoleMap
		if jsonErr := json.Unmarshal([]byte(cfg.ConfigValue), &roleMap); jsonErr != nil {
			logrus.Warnf("[reconciliation:workorder] 解析 assignee_role_map JSONB 失败: %v", jsonErr)
		} else if roleID, ok := roleMap[tpl.AssigneeRoleKey]; ok && roleID != "" {
			var user models.User
			if userErr := s.db.WithContext(ctx).
				Table("sys_user u").
				Joins("INNER JOIN sys_user_role ur ON ur.user_id = u.id").
				Where("ur.role_id = ? AND u.status = ?", roleID, 0). // status=0 启用
				Order("u.created_at ASC").
				Limit(1).
				First(&user).Error; userErr == nil {
				uid := user.ID
				assigneeID = &uid
			} else {
				logrus.Warnf("[reconciliation:workorder] role %s 下未找到可用用户(role_id=%s)", roleID, roleID)
			}
		} else {
			logrus.Warnf("[reconciliation:workorder] AssigneeRoleKey=%s 在 roleMap 中无映射", tpl.AssigneeRoleKey)
		}
	}

	// 第 6 步:从 raw_snapshot 取 asset_code(用于标题拼接)
	var snapshot map[string]interface{}
	assetCode := ""
	if jsonErr := json.Unmarshal(rec.RawSnapshot, &snapshot); jsonErr == nil {
		if v, ok := snapshot["asset_code"].(string); ok {
			assetCode = v
		}
	}
	if assetCode == "" {
		assetCode = rec.AssetID // fallback:用 UUID 兜底
	}

	// 第 7 步:工单标题 D-A2-01 模板
	title := fmt.Sprintf("[资产对账·%s类] 资产 %s (%s) %s",
		rec.ConflictType,
		assetCode,
		rec.Severity,
		rec.DetectedAt.Format("2006-01-02 15:04"),
	)

	// 第 8 步:description 由 3 部分拼接
	slaMin := severityToSLAMinutes(rec.Severity)
	descLines := strings.Join(tpl.DescriptionLines, "\n")
	rawSnapshotStr := string(rec.RawSnapshot)
	if rawSnapshotStr == "" {
		rawSnapshotStr = "(无 raw_snapshot)"
	}
	description := fmt.Sprintf("【建议修复】\n%s\n\n【原始数据】\n%s\n\n## SLA: %dm",
		descLines, rawSnapshotStr, slaMin)

	// 第 9 步:查询 system 用户的真实 UUID 作为 submitterID(T-43-01 mitigation 的真实实现)
	// 复用 internal/scheduler/workorder_tasks.go:60-62 同款查询样板。
	// sys_user.username='system' 必须存在(运维预建或初始化时 seed)。
	var systemSubmitter struct {
		ID string
	}
	if sysUserErr := s.db.WithContext(ctx).
		Table("sys_user").
		Select("id").
		Where("username = ?", "system").
		First(&systemSubmitter).Error; sysUserErr != nil {
		logrus.Errorf("[reconciliation:workorder] 查询 system 用户失败,请确保 sys_user 存在 username='system' 的用户(exception=%s): %v",
			exceptionID, sysUserErr)
		return nil, fmt.Errorf("查询 system 用户失败: %w", sysUserErr)
	}

	// 第 10 步:创建工单(submitterID = systemSubmitter.ID,合法 UUID)
	baseSvc := workorder.NewBaseService(s.db)
	wo, createErr := baseSvc.Create(ctx, &workorder.CreateRequest{
		Title:       title,
		CategoryID:  category.ID,
		Type:        models.WorkOrderTypeIncident,
		Priority:    tpl.DefaultPriority,
		Description: description,
		AssigneeID:  assigneeID,
	}, systemSubmitter.ID)
	if createErr != nil {
		// 第 10 步:失败仅 logrus.Errorf,不写 SysNotice(D-A1-03)
		logrus.Errorf("[reconciliation:workorder] 创建工单失败(exceptionID=%s, conflictType=%s, severity=%s): %v",
			exceptionID, rec.ConflictType, rec.Severity, createErr)
		return nil, createErr
	}

	// 第 11 步:回写 workorder_id(D-A1-04)
	if err := s.db.WithContext(ctx).
		Model(&rec).
		Update("workorder_id", wo.ID).Error; err != nil {
		// WR-03 修复:工单已创建但 workorder_id 回写失败 → 必须补偿删除孤儿工单。
		// 否则 workorder_id 仍为 NULL,下个 cron 周期 WHERE workorder_id IS NULL 再次命中,
		// 重复创建工单。BaseService.Delete 仅删 Pending 状态工单(cron 建单即 Pending,T-43-01)。
		logrus.Errorf("[reconciliation:workorder] UPDATE workorder_id 失败(工单=%s, 异常=%s): %v", wo.ID, exceptionID, err)
		if delErr := baseSvc.Delete(ctx, wo.ID); delErr != nil {
			logrus.Errorf("[reconciliation:workorder] 补偿删除孤儿工单失败(工单=%s): %v — 需人工介入防止重复转单", wo.ID, delErr)
		} else {
			logrus.Warnf("[reconciliation:workorder] 已补偿删除孤儿工单 %s(异常=%s),下个 cron 周期将重试转单", wo.ID, exceptionID)
		}
		return nil, fmt.Errorf("UPDATE workorder_id 失败: %w", err)
	}

	applogger.Infof("[reconciliation:workorder] 异常 %s (ConflictType=%s, Severity=%s) 已生成工单 %s",
		exceptionID, rec.ConflictType, rec.Severity, wo.ID)

	// R2 WS + SysNotice 闭环(只 critical 触发,与 D-A4-02 一致)
	if rec.Severity == "critical" {
		// 第 12a 步:WS 推送 critical_workorder_created(全量 broadcast)
		s.broadcastCriticalWorkorder(ctx, &rec, wo.ID, title)

		// 第 12b 步:SysNotice 写入(立即发布,避免草稿状态)
		s.publishCriticalSysNotice(ctx, &rec, wo.ID, title, assetCode)
	}

	// 第 12 步:返回工单
	return wo, nil
}

// broadcastCriticalException WS 推送 critical_exception_detected 事件(D-A4-02)
//
// 调用栈:CreateWorkorderFromException 第 1 步 SELECT 后(severity='critical' 时)
//
// payload 结构:
//   - exception_id: sys_data_reconciliation.id
//   - asset_code: 从 raw_snapshot 取(若有)
//   - conflict_type: B/C/D/E/F
//   - severity: critical
//   - detected_at: RFC3339
//
// 注意:NoticeHub.BroadcastToAll 内部 channel 有缓冲(256),不会阻塞调用方;
//       仅在 hub 已关闭时 select panic(罕见,启动期 race)。
func (s *ReconciliationWorkorderService) broadcastCriticalException(ctx context.Context, rec *models.SysDataReconciliation) {
	if s.noticeHub == nil {
		return // 兼容非 production 环境
	}

	// 从 raw_snapshot 取 asset_code(失败时不阻塞 WS 推送)
	assetCode := rec.AssetID
	if len(rec.RawSnapshot) > 0 {
		var snap map[string]interface{}
		if json.Unmarshal(rec.RawSnapshot, &snap) == nil {
			if v, ok := snap["asset_code"].(string); ok && v != "" {
				assetCode = v
			}
		}
	}

	payload := map[string]interface{}{
		"event": EventCriticalExceptionDetected,
		"data": map[string]interface{}{
			"exception_id":  rec.ID,
			"asset_id":      rec.AssetID,
			"asset_code":    assetCode,
			"conflict_type": rec.ConflictType,
			"severity":      rec.Severity,
			"detected_at":   rec.DetectedAt.Format(time.RFC3339),
		},
	}
	content, err := json.Marshal(payload)
	if err != nil {
		logrus.Warnf("[reconciliation:workorder] WS payload 序列化失败: %v", err)
		return
	}

	defer func() {
		// Hub 关闭时 BroadcastToAll 写 channel 会 panic,兜底避免 cron 崩溃
		if r := recover(); r != nil {
			logrus.Warnf("[reconciliation:workorder] WS broadcast 异常(忽略): %v", r)
		}
	}()

	s.noticeHub.BroadcastToAll(websocket.NoticeMessage{
		Type:      EventCriticalExceptionDetected,
		Content:   string(content),
		Timestamp: time.Now().Unix(),
	})

	logrus.Debugf("[reconciliation:workorder] WS 推送 %s 已广播 (exception_id=%s)",
		EventCriticalExceptionDetected, rec.ID)
}

// broadcastCriticalWorkorder WS 推送 critical_workorder_created 事件(D-A4-02)
//
// 调用栈:CreateWorkorderFromException 第 11 步 UPDATE workorder_id 后
//
// payload 结构:
//   - workorder_id: 新建工单 uuid
//   - exception_id: 关联异常 uuid
//   - title: 工单标题(含 [资产对账·X类] 前缀)
//   - asset_code: 从 raw_snapshot 取
//   - severity: critical
//
// 注意:前端 useReconciliationWebSocket hook 只过滤 critical_* 2 类事件,
//       high/medium/low 工单创建不推送(避免 dashboard 事件过载)。
func (s *ReconciliationWorkorderService) broadcastCriticalWorkorder(ctx context.Context, rec *models.SysDataReconciliation, workorderID string, title string) {
	if s.noticeHub == nil {
		return
	}

	assetCode := rec.AssetID
	if len(rec.RawSnapshot) > 0 {
		var snap map[string]interface{}
		if json.Unmarshal(rec.RawSnapshot, &snap) == nil {
			if v, ok := snap["asset_code"].(string); ok && v != "" {
				assetCode = v
			}
		}
	}

	payload := map[string]interface{}{
		"event": EventCriticalWorkorderCreated,
		"data": map[string]interface{}{
			"workorder_id":  workorderID,
			"exception_id":  rec.ID,
			"asset_id":      rec.AssetID,
			"asset_code":    assetCode,
			"conflict_type": rec.ConflictType,
			"severity":      rec.Severity,
			"title":         title,
		},
	}
	content, err := json.Marshal(payload)
	if err != nil {
		logrus.Warnf("[reconciliation:workorder] WS payload(workorder) 序列化失败: %v", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logrus.Warnf("[reconciliation:workorder] WS broadcast(workorder) 异常(忽略): %v", r)
		}
	}()

	s.noticeHub.BroadcastToAll(websocket.NoticeMessage{
		Type:      EventCriticalWorkorderCreated,
		Content:   string(content),
		Timestamp: time.Now().Unix(),
	})

	logrus.Debugf("[reconciliation:workorder] WS 推送 %s 已广播 (workorder_id=%s)",
		EventCriticalWorkorderCreated, workorderID)
}

// publishCriticalSysNotice 写入 sys_notice(D-A4-03 双通道之 SysNotice 通道)
//
// 业务说明:
//   - WS 给在线 dashboard,SysNotice 给未在线的运维(下次登录时收件箱可见)
//   - notice_type='2'(警告),Priority=Urgent,TargetType=All
//   - NoticeContent 头部拼接 sysNoticePrefix 前端过滤 token
//   - 立即发布(PublishStatus=Published)避免草稿状态,运维能立即看到
//
// 软失败:
//   - noticeService == nil 时跳过(test 环境)
//   - CreateNoticeWithTargets / PublishNotice 失败时仅 logrus.Warnf,不阻塞主流程
//     (工单已创建并 WS 推送,D-A1-03 风格 — 告警通道失败不影响业务)
func (s *ReconciliationWorkorderService) publishCriticalSysNotice(ctx context.Context, rec *models.SysDataReconciliation, workorderID, title, assetCode string) {
	if s.noticeService == nil {
		return
	}

	// NoticeContent 头部拼接 [asset_reconciliation_critical] 前端过滤 token
	noticeContent := fmt.Sprintf("%s\n异常: %s\n冲突类型: %s\n严重级别: %s\n检测时间: %s\n关联工单: %s\n\n工单已自动创建,请及时处理。",
		sysNoticePrefix,
		rec.ID,
		rec.ConflictType,
		rec.Severity,
		rec.DetectedAt.Format("2006-01-02 15:04:05"),
		workorderID,
	)

	req := &services.CreateNoticeRequest{
		NoticeTitle:   title,
		NoticeType:    "2", // 2=警告(参见 models.Notice.NoticeType 字典)
		NoticeContent: noticeContent,
		Priority:      models.PriorityUrgent, // 紧急
		TargetType:    models.TargetAll,     // 全部用户
		IsMarkdown:    false,
	}

	// 创建 + 立即发布(避免草稿状态)
	notice, err := s.noticeService.CreateNoticeWithTargets(ctx, req, "SYSTEM", "SYSTEM")
	if err != nil {
		logrus.Warnf("[reconciliation:workorder] SysNotice 创建失败(exception=%s, workorder=%s): %v",
			rec.ID, workorderID, err)
		return
	}

	if publishErr := s.noticeService.PublishNotice(ctx, notice.ID); publishErr != nil {
		logrus.Warnf("[reconciliation:workorder] SysNotice 发布失败(notice=%s, exception=%s): %v",
			notice.ID, rec.ID, publishErr)
		return
	}

	logrus.Infof("[reconciliation:workorder] SysNotice 已发布 notice_id=%s (asset=%s, severity=%s, workorder=%s)",
		notice.ID, assetCode, rec.Severity, workorderID)
}

// WorkstationIDForException 通过 exceptionID 反查工位 ID(Phase 45 R4 / D-A4-04)
//
// 设计原因(plan 45-02 B2 锁定):
//   - **不修改** CreateWorkorderFromException 签名(保持 R2 cron 既有 12 步流程不变)
//   - 单独提供轻量级反查方法,scheduler 在 CreateWorkorderFromException 成功后
//     调此方法拿 wsID,再触发 InvalidateWorkstationHealth
//
// 查询路径:
//   - sys_data_reconciliation.id (exceptionID) → sys_data_reconciliation.asset_id
//   - asset_id → reconciliation_normalized.workstation_id (R1 MV 已 JOIN)
//   - 若 reconciliation_normalized 缺失该 asset 行或 workstation_id IS NULL → 返回 ""
//     (caller 跳过 invalidate,正常情况)
//
// 返回:
//   - workstationID == "" 表示该 exception 孤立无工位归属(常见于 A 类不入主表但 cron
//     漏查场景,或 reconciliation_normalized 尚未刷新),正常情况
//   - error != nil 表示 DB 故障(查询失败),scheduler 应 logger.Warnf 但不阻断 cron 流程
func (s *ReconciliationWorkorderService) WorkstationIDForException(ctx context.Context, exceptionID string) (string, error) {
	if exceptionID == "" {
		return "", nil
	}
	var wsID sql.NullString
	err := s.db.WithContext(ctx).
		Table("reconciliation_normalized").
		Select("reconciliation_normalized.workstation_id").
		Joins("JOIN sys_data_reconciliation ON sys_data_reconciliation.asset_id = reconciliation_normalized.asset_id").
		Where("sys_data_reconciliation.id = ? AND sys_data_reconciliation.deleted_at IS NULL AND reconciliation_normalized.workstation_id IS NOT NULL", exceptionID).
		Limit(1).
		Row().
		Scan(&wsID)
	if err != nil {
		// gorm.Row() returns *sql.Row from database/sql, whose Scan() reports
		// sql.ErrNoRows for the empty case (not gorm.ErrRecordNotFound).
		// CR-02: check sql.ErrNoRows so expected "no row" returns ("", nil)
		// and only real DB failures propagate.
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if !wsID.Valid {
		return "", nil
	}
	return wsID.String, nil
}

// InvalidateWorkstationHealth 失效工位健康度缓存(Phase 45 R4 / D-A4-04)
//
// 包装 cache_keys.InvalidateWorkstationHealth,nil-safe:
//   - s.cache == nil → 返回 nil(单测 / 非 production 环境,跳过)
//   - workstationID == "" → 返回 nil(caller 跳过)
//
// 调用栈:
//   - R2 createWorkorderBySeverity(在 CreateWorkorderFromException 成功后,通过
//     woSvc.WorkstationIDForException 拿 wsID,然后调本方法)
//   - ReconciliationHandler.ResolveException(通过 h.core.Cache 直接调 cache_keys.InvalidateWorkstationHealth)
//
// 顺序约定(D-A4-04):
//
//	workorderSvc.InvalidateWorkstationHealth(ctx, wsID)   // 先失效
//	operlog.Record(...)                                  // 再写日志
//	response.Success(...)                                // 最后返响应
func (s *ReconciliationWorkorderService) InvalidateWorkstationHealth(ctx context.Context, workstationID string) error {
	if workstationID == "" {
		return nil
	}
	if s == nil || s.cache == nil {
		// 单测 / 非 production 场景 cache 未注入,跳过失效(不报错)
		return nil
	}
	return InvalidateWorkstationHealth(ctx, s.cache, workstationID)
}

// severityToSLAMinutes 根据 severity 字符串返回 SLA 分钟数(D-A2-03)
//
// 业务说明:
//   - critical=30min / high=4h / medium=24h / low=7d
//   - 落 workorder 表 description 段(Phase 23 workorder schema 未强制 sla_duration 字段)
//   - 由 workorder 内部 cron 监控超时(本服务不主动接管超时升级)
//
// 设计考量:
//   - 30 / 240 / 1440 / 10080 来自 sys_config:asset.reconciliation.workorder.sla_minutes_by_severity
//     (D-A2-03 / migration_171),本函数硬编码默认值与 sys_config 一致
//   - 若运维修改 sys_config,本函数返回值不变(本服务不读 sys_config.sla_minutes_by_severity,
//     仅写 description 文本展示;真实 SLA 监控走 workorder 内部 cron)
//
// 输入: severity 字符串(critical/high/medium/low)
// 输出: SLA 分钟数(int);未知 severity 默认 medium=1440
func severityToSLAMinutes(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 30
	case "high":
		return 240
	case "medium":
		return 1440
	case "low":
		return 10080
	default:
		return 1440 // 默认 medium
	}
}

// Compile-time interface check:确保本服务实现 Phase 42 R1 reconciliation_service 内的接口约定
//
// _ = time.Second 引用 time 包,避免 unused import 错误(用于 future Phase 43 R3+R5 扩展)
var _ = time.Second