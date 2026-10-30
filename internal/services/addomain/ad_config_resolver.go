package addomain

// Phase 58 D-02 / D-03 配置解析单一真实源。
//
//   - MatchADConfigByDN：最长后缀唯一匹配纯函数（QUEUE-04 compare-and-move 的路由依据）
//   - FirstEnabledADConfigID / FirstSyncEnabledADConfigID：HYGIENE-01 双份 helper 合并后的
//     **两个显式入口**（研究 F-D：语义不同，禁合并成一个）
//
// 纯函数部分无 DB、无副作用、可单测；两个 First* 是薄查询封装。

import (
	"context"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// MatchADConfigByDN 按 sys_user.ad_dn 归属解析所属 AD 配置（Phase 58 D-02 契约）。
//
// 纯函数：无 DB、无副作用、可单测。调用方（排水器）负责把"无匹配/歧义"映射为队列行 dead。
//
// 算法：
//  1. 目标 DN 归一：NormalizeDN(dn)
//  2. 候选 = configs 中 status == models.ADConfigStatusEnabled 的配置（软删由调用方查库时过滤）
//  3. 命中判定**必须带逗号边界**：strings.HasSuffix(norm, ","+NormalizeDN(cfg.BaseDN))
//     —— 裸 HasSuffix 会让 "CN=a,DC=xcorp,DC=com" 被 "DC=corp,DC=com" 误命中（P6）
//  4. 取 base_dn **最长**者（最长后缀唯一匹配）
//  5. 最长长度并列 → 歧义 → error；无任何候选命中 → error
//  6. 跳过 BaseDN 为空的配置（配置未填 base_dn 时不能参与匹配，否则空串后缀恒命中）
//
// ⚠ 返回的指针指向 configs 切片的元素（调用方持有整个 slice，生命周期安全）；
// **不得**返回副本后再被调用方误当作独立对象修改。
//
// ⚠ 逗号边界推论：DN 恰好等于 base_dn 时不算命中（需要 ",<base>" 前缀），
// 这是刻意的——sys_user.ad_dn 恒为完整用户 DN（含 CN=），不会退化成域根。
func MatchADConfigByDN(dn string, configs []models.ADConfig) (*models.ADConfig, error) {
	norm := NormalizeDN(dn)

	var best *models.ADConfig
	bestLen := 0
	// 记录"与当前最长候选等长"的全部 base_dn：>1 即为歧义。
	// 等长且不同的两个 base 字符串不可能同时是同一字符串的逗号边界后缀，
	// 现实中触发歧义的形态是**两条配置指向同一域**（归一后 base 相同）——
	// 属真实误配置，必须报错而不是静默取第一条。
	var tiedBaseDNs []string

	for i := range configs {
		cfg := &configs[i]
		if cfg.Status != models.ADConfigStatusEnabled {
			continue
		}
		base := NormalizeDN(cfg.BaseDN)
		if base == "" {
			// 空 base_dn 恒命中任何 DN，必须显式跳过（T-58-02-06）
			continue
		}
		if !strings.HasSuffix(norm, ","+base) {
			continue
		}
		switch {
		case len(base) > bestLen:
			best = cfg
			bestLen = len(base)
			tiedBaseDNs = []string{base}
		case len(base) == bestLen:
			tiedBaseDNs = append(tiedBaseDNs, base)
		}
	}

	if best == nil {
		return nil, fmt.Errorf("无匹配 AD 配置: DN %s 未命中任何启用配置的 base_dn", dn)
	}
	if len(tiedBaseDNs) > 1 {
		return nil, fmt.Errorf("AD 配置歧义: DN %s 同时匹配多个等长 base_dn (%s)",
			dn, strings.Join(tiedBaseDNs, " | "))
	}
	return best, nil
}

// FirstEnabledADConfigID 取第一条启用（status=enabled）的 AD 配置 ID，ORDER BY created_at ASC。
//
// ⚠ 只判 status，**刻意不判 sync_enabled** —— 保留 internal/scheduler 两处自动挑选的
// 现行行为（研究 F-D）：若合并时统一加上 sync_enabled，"配置启用但同步关闭"的场景下
// 手动同步任务会从"能同步"变成"报未找到配置"，属行为变更。
// 需要 sync_enabled 双条件的场景请用 FirstSyncEnabledADConfigID。
func FirstEnabledADConfigID(ctx context.Context, db *gorm.DB) (string, error) {
	var cfg models.ADConfig
	err := db.WithContext(ctx).
		Where("status = ?", models.ADConfigStatusEnabled).
		Order("created_at ASC").
		First(&cfg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 文案与收敛前的 scheduler 两份旧实现逐字一致
			// （既有 scheduler 测试断言 "未找到启用的AD配置"）
			return "", fmt.Errorf("未找到启用的AD配置，请先在AD域配置中启用至少一个配置")
		}
		return "", fmt.Errorf("查询AD配置失败: %w", err)
	}
	applogger.Infof("自动使用AD配置: %s (ID: %s)", cfg.ConfigName, cfg.ID)
	return cfg.ID, nil
}

// FirstSyncEnabledADConfigID 取第一条 sync_enabled=true 且 status=enabled 的 AD 配置 ID。
//
// ⚠ 与 FirstEnabledADConfigID 的差异是**本质**的（研究 F-D）：本入口服务于
// "当前操作的 AD 配置"（无 DN 可依据的场景，如 OU 部门映射手工绑定），
// 必须与 IsADSyncEnabled 的前置门控同为双条件，否则会把一个"同步关闭"的配置绑给映射行。
//
// ⚠ 不带 ORDER BY：原 ou_mapping_handler.go 就是无序 First —— 收敛只消除重复实现、
// 不改语义；加 ORDER BY 会静默改变取到的配置，属行为变更。
func FirstSyncEnabledADConfigID(ctx context.Context, db *gorm.DB) (string, error) {
	var cfg models.ADConfig
	err := db.WithContext(ctx).
		Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).
		First(&cfg).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("查询 AD 配置失败: 未找到 sync_enabled=true 且 status=enabled 的配置")
		}
		return "", fmt.Errorf("查询AD配置失败: %w", err)
	}
	return cfg.ID, nil
}
