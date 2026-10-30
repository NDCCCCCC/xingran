//go:build archive_skip


package migrations

import (
	"log"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate171ReconciliationWorkorderAssigneeRole Phase 43 R2: 资产对账 workorder
// assignee role 映射 + SLA minutes seed(D-A2-02 + D-A2-03)
//
// 业务背景:
//   - Phase 43 R2 引入 critical/high 异常自动转工单(43-01 plan)
//   - type→role 映射从 sys_config JSONB 字符串读(D-A2-02),运维可改
//   - SLA 按 severity 分级:D-A2-03(critical=30min / high=4h / medium=24h / low=7d)
//   - 本迁移 seed 2 条 sys_config,作为 R2 转单 service 的默认配置
//
// 2 个 key:
//   - asset.reconciliation.workorder.assignee_role_map
//     value: {"asset_owner":1,"ops_owner":2,"responsible_owner":3}
//     role_id 1/2/3 是占位 — 运维按实际 sys_role.id 修改,默认值无用户匹配(assigneeID=nil,
//     工单仍创建,等人工分配)
//   - asset.reconciliation.workorder.sla_minutes_by_severity
//     value: {"critical":30,"high":240,"medium":1440,"low":10080}
//     R2 service 硬编码默认值与本 config 一致(本服务不读此 config,仅写 description 展示;
//     真实 SLA 监控走 workorder 内部 cron)
//
// 幂等: count-then-insert 模式(migration_165/169 风格),重复运行不会报错或产生重复行。
func Migrate171ReconciliationWorkorderAssigneeRole(db *gorm.DB) error {
	log.Println("Running migration 171: Phase 43 R2 reconciliation workorder assignee_role + SLA config seeds")

	configs := []struct {
		configName  string
		configKey   string
		configValue string
		remark      string
	}{
		{
			configName:  "对账工单 assignee role 映射",
			configKey:   "asset.reconciliation.workorder.assignee_role_map",
			configValue: `{"asset_owner":1,"ops_owner":2,"responsible_owner":3}`,
			remark:      "R2: type→role 映射 JSONB。role_id 1/2/3 是占位 — 运维按实际 sys_role.id 修改。B/C/D=asset_owner, E=ops_owner, F=responsible_owner",
		},
		{
			configName:  "对账工单 SLA minutes by severity",
			configKey:   "asset.reconciliation.workorder.sla_minutes_by_severity",
			configValue: `{"critical":30,"high":240,"medium":1440,"low":10080}`,
			remark:      "R2: SLA 分级 minutes(critical=30min / high=4h / medium=24h / low=7d)。由 workorder 内部 cron 监控超时",
		},
	}

	for _, c := range configs {
		var existingCount int64
		if err := db.Model(&models.Config{}).Where("config_key = ?", c.configKey).Count(&existingCount).Error; err != nil {
			log.Printf("Migration 171: count config %s failed: %v", c.configKey, err)
			continue
		}
		if existingCount > 0 {
			applogger.Debugf("[迁移] sys_config %s 已存在,跳过", c.configKey)
			continue
		}

		cfg := &models.Config{
			ConfigName:  c.configName,
			ConfigKey:   c.configKey,
			ConfigValue: c.configValue,
			ConfigType:  models.ConfigTypeYes, // Y = 系统参数,前端不允许编辑
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      c.remark,
		}
		if err := db.Create(cfg).Error; err != nil {
			log.Printf("Migration 171: create config %s failed: %v", c.configKey, err)
			continue
		}
		applogger.Infof("[迁移] sys_config seed: %s = %s", c.configKey, c.configValue)
	}

	log.Println("Migration 171 completed: 2 reconciliation workorder config seeds done")
	return nil
}