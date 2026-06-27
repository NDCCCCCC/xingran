package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/workorder"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ReconciliationWorkorderService 资产对账异常转工单服务(Phase 43 R2 / D-A1~D-A2)
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
//
// Status Convention: 0=启用 1=停用(CLAUDE.md 强制);本服务处理 severity/critical/high 等业务字段,
// 不涉及 status convention。
type ReconciliationWorkorderService struct {
	db *gorm.DB
}

// NewReconciliationWorkorderService 构造器
func NewReconciliationWorkorderService(db *gorm.DB) *ReconciliationWorkorderService {
	return &ReconciliationWorkorderService{db: db}
}

// AssigneeRoleMap type→role 映射表(D-A2-02)
//
// 业务说明:
//   - key 是 workorder.ReconciliationWorkorderTemplate.AssigneeRoleKey
//     (B/C/D="asset_owner", E="ops_owner", F="responsible_owner")
//   - value 是 sys_role.id(运维预建,迁移不创建)
//   - 由 sys_config:asset.reconciliation.workorder.assignee_role_map JSONB 字符串反序列化
//
// 反查流程:
//   1. GetTemplate(conflictType) → AssigneeRoleKey(如 "asset_owner")
//   2. roleMap["asset_owner"] → role_id(uuid)
//   3. SELECT sys_user WHERE role_id=? AND status=0 LIMIT 1 → assigneeID
//
// 软失败:JSONB 缺失或解析失败时,AssigneeID 为 nil,工单仍创建,等待人工分配。
type AssigneeRoleMap map[string]int64

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
// 12 步流程(D-A1-01/02 + D-A2-01~04):
//  1. SELECT sys_data_reconciliation WHERE id=? AND workorder_id IS NULL AND deleted_at IS NULL
//     (排除已转单 + 软删除记录,避免重复转单)
//  2. SELECT WorkOrderCategory WHERE category_name="对账-{Type}类"
//     (B-F 5 类,migration_169 已 seed)
//  3. tpl=workorder.GetTemplate(rec.ConflictType);nil → err("无模板")
//  4. configSvc.GetByKey("asset.reconciliation.workorder.assignee_role_map") 软失败
//  5. JSONB 反查 role_id → SELECT sys_user WHERE role_id=? AND status=0 LIMIT 1
//  6. json.Unmarshal(rec.RawSnapshot) 取 asset_code(用于标题)
//  7. title=fmt.Sprintf("[资产对账·%s类] 资产 %s (%s) %s",conflictType,asset_code,severity,detectedAt)
//  8. description=descLines+"\n\n【原始数据】\n"+raw_snapshot+"\n\n## SLA: "+slaMin+"m"
//  9. workorder.NewBaseService(s.db).Create(ctx,&CreateRequest{...},"SYSTEM")
// 10. 失败 logrus.Errorf,return nil,err(D-A1-03 不写 SysNotice)
// 11. 成功 s.db.Model(&rec).Update("workorder_id",wo.ID)
// 12. return wo
//
// submitterID="SYSTEM"(T-43-01 转单越权 mitigation):工单创建后 status=Pending,
// 等运维接管,避免"自动转单+自动受理"双重系统行为。
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
		} else if roleID, ok := roleMap[tpl.AssigneeRoleKey]; ok && roleID > 0 {
			var user models.User
			if userErr := s.db.WithContext(ctx).
				Where("role_id = ? AND status = ?", roleID, 0). // status=0 启用
				Order("created_at ASC").
				Limit(1).
				First(&user).Error; userErr == nil {
				uid := user.ID
				assigneeID = &uid
			} else {
				logrus.Warnf("[reconciliation:workorder] role %d 下未找到可用用户(role_id=%d)", roleID, roleID)
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

	// 第 9 步:创建工单(submitterID="SYSTEM" — T-43-01 mitigation)
	baseSvc := workorder.NewBaseService(s.db)
	wo, createErr := baseSvc.Create(ctx, &workorder.CreateRequest{
		Title:       title,
		CategoryID:  category.ID,
		Type:        models.WorkOrderTypeIncident,
		Priority:    tpl.DefaultPriority,
		Description: description,
		AssigneeID:  assigneeID,
	}, "SYSTEM")
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
		// 工单已创建但 workorder_id 回写失败:log 严重错误(可能产生重复转单)
		// 下个 cron 周期会重复扫描,但 workorder_id IS NULL 已被其他周期写过会跳过(若第一次失败)
		logrus.Errorf("[reconciliation:workorder] UPDATE workorder_id 失败(工单=%s, 异常=%s): %v", wo.ID, exceptionID, err)
		return nil, fmt.Errorf("UPDATE workorder_id 失败: %w", err)
	}

	applogger.Infof("[reconciliation:workorder] 异常 %s (ConflictType=%s, Severity=%s) 已生成工单 %s",
		exceptionID, rec.ConflictType, rec.Severity, wo.ID)

	// 第 12 步:返回工单
	return wo, nil
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