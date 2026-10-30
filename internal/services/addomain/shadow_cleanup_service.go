package addomain

// Phase 61 SHADOW-01/SHADOW-03: 影子表删除清理。
//
// 全量同步四类 upsert 全部成功后，按类别对 AD 端已消失的 sys_ad_user/ou/computer
// 活行软删除（GORM 默认作用域 Delete = UPDATE deleted_at，行保留 Unscoped 可审计）。
// 两道闸门逐字复用 handleDeletedGroups 先例
// （internal/services/addomain/group_sync_service.go:346-405）：
//   - Gate 1: 该类 LDAP entries 为 0 → 拒绝清理（AD 不可达/BaseDN 配错会伪装成"全部消失"）
//   - Gate 2: 拟删数*2 > 存量活行数 → 拒绝清理（LDAP 返回不完整/分页中断），判式严格大于、恰 50% 放行
//
// 两道闸门拒绝均为 WARN + return nil 非阻断；清理内部可恢复失败由调用方降级 WARN，
// 整轮同步仍 success（Pitfall 4：四类 upsert 已各自提交不可回滚）。
//
// 挂载点唯一：syncDataInternal 成功路径末尾（syncComputers 成功之后、last_sync_at 更新
// 之前），处于 singleflight.Do 闭包内 + syncType 守卫共同构成 SHADOW-03（清理与全量同步
// 互斥不并发、增量分支 no-op）。禁止独立后台执行体（sys_job / 独立 goroutine）。

import (
	"context"
	"fmt"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// cleanupAfterFullSync 全量同步成功路径末尾的影子表清理编排入口（SHADOW-03）。
//
// SHADOW-03 守卫：syncType 与全量类型不相等时直接 no-op（incremental 等分支不清理）——
// 判据用 models.ADSyncTypeFull 常量，禁裸字符串比较。清理与全量同步同处
// singleflight.Do 闭包内，二者不并发；singleflight key 含 syncType 导致的
// full/incremental 不互斥是既有行为，本守卫只承诺「清理只在全量分支执行」。
//
// 三类依次独立判定闸门：单类拒绝（return nil）不影响其它类别继续执行
// （per-category 语义）。
func (s *SyncService) cleanupAfterFullSync(ctx context.Context, config *models.ADConfig, syncType string, ous, users, computers []*ldap.Entry, result *SyncResult) error {
	if syncType != string(models.ADSyncTypeFull) {
		return nil
	}
	if err := s.cleanupVanishedOUs(ctx, config, ous, result); err != nil {
		return err
	}
	if err := s.cleanupVanishedUsers(ctx, config, users, result); err != nil {
		return err
	}
	return s.cleanupVanishedComputers(ctx, config, computers, result)
}

// cleanupVanishedUsers 软删 AD 端已消失的用户行（SHADOW-01 user 类别）。
//
// 五段结构逐字对齐 handleDeletedGroups（group_sync_service.go:346-405）：
// Gate 1（LDAP 0 条拒绝）→ LDAP 管理集 → 存量活行差集 → Gate 2（拟删 >50% 拒绝）→ 软删。
//
// P1-3 补充：软删成功**之后**联动停用对应的 sys_user 账号（AD 端消失 = 本地账号
// 不得再登录）。两道闸门任一拒绝都在联动之前 return，故闸门拒绝时**不会**
// 停用任何账号（AD 抖动 ≠ 全员停用）。
func (s *SyncService) cleanupVanishedUsers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry, result *SyncResult) error {
	// Gate 1: 安全闸门 — 如果 LDAP 返回为空,直接拒绝任何删除。
	// 否则 AD 不可达/网络抖动/搜索 BaseDN 配错都会伪装成「用户全部消失」,
	// 导致整表软删除,恢复极其困难。
	if len(entries) == 0 {
		applogger.Warnf("[用户同步] 跳过过期用户清理: LDAP 返回 0 条记录,可能 AD 不可达或 BaseDN 配置错误")
		return nil
	}

	// LDAP 管理集：仅 sAMAccountName 非空的 entry 入集 —— 与 syncUsers 的
	// userDNs 构造（sync.go:443-449）同口径。清理集合必须等于本轮真正被
	// upsert 管理的行集,否则会把本轮被跳过的行误判为已消失（Pitfall 2）。
	managedUserDNs := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.GetAttributeValue("sAMAccountName") != "" {
			managedUserDNs[entry.DN] = true
		}
	}

	// 存量活行:默认作用域（deleted_at IS NULL）统计 —— 闸门分母恒为活行数,禁 Unscoped（Pitfall 3）。
	var staleUsers []models.ADUser
	s.db.WithContext(ctx).
		Where("ad_config_id = ? AND deleted_at IS NULL", config.ID).
		Find(&staleUsers)

	var vanishedDNs []string
	for _, user := range staleUsers {
		if !managedUserDNs[user.UserDN] {
			vanishedDNs = append(vanishedDNs, user.UserDN)
		}
	}

	if len(vanishedDNs) > 0 {
		// Gate 2: 阈值保护 — 若拟删超过存量活行的 50%,拒绝并要求人工确认（判式严格大于,
		// 恰 50% 放行,与先例逐字同构）。防止 LDAP 返回严重不完整(分页中断/部分搜索失败)
		// 时误删大量数据。拒绝日志只含数量、不含 DN 明文（Phase 60 SyncError 同口径）。
		if len(staleUsers) > 0 && len(vanishedDNs)*2 > len(staleUsers) {
			applogger.Warnf(
				"[用户同步] 跳过过期用户清理: 拟删除 %d/%d 个用户(>50%%),"+
					"可能是 LDAP 返回不完整。请人工核实后手动清理",
				len(vanishedDNs), len(staleUsers))
			return nil
		}

		// 软删:GORM 默认作用域 Delete = UPDATE deleted_at,禁 Unscoped 硬删（行保留供审计）。
		if err := s.db.WithContext(ctx).
			Where("ad_config_id = ? AND user_dn IN ?", config.ID, vanishedDNs).
			Delete(&models.ADUser{}).Error; err != nil {
			return fmt.Errorf("删除过期用户失败: %w", err)
		}

		result.DeletedUserCount = len(vanishedDNs)
		applogger.Infof("[用户同步] 清理了 %d 个已不存在的用户", len(vanishedDNs))

		// P1-3 联动:AD 端已消失（离职/被删）的用户,系统里对应的账号也必须停用,
		// 否则 AD 账号没了 sys_user 仍是启用态 → 登录鉴权只查 sys_user.Status
		// (internal/api/v1/auth.go:229) → 该账号照样能登录。
		//
		// 位置约束（重要）:
		//   - 挂在**软删成功之后**:影子行删掉了但本地账号还活着 = 漏停用;
		//   - 位于两道闸门**之后**:Gate 1(LDAP 0 条)/Gate 2(拟删>50%)拒绝时
		//     上面已 return,根本走不到这里 —— AD 抖动时不会"全员停用"。
		//   - 联动失败只记 WARN 不阻断:影子清理本身是软删(可恢复),
		//     清理主流程不应因联动失败而回滚/中断。
		if disabled, linkErr := disableSysUsersForADDNs(ctx, s.db, config, vanishedDNs); linkErr != nil {
			applogger.Warnf("[用户同步] 联动停用 sys_user 失败（已清理 %d 个消失用户）: %v",
				len(vanishedDNs), linkErr)
		} else if disabled > 0 {
			applogger.Infof("[用户同步] 联动停用 %d 个 sys_user 账号（AD 端已消失）", disabled)
		}
	}

	return nil
}

// cleanupVanishedOUs 软删 AD 端已消失的 OU 行（SHADOW-01 ou 类别）。
//
// 五段结构对齐 handleDeletedGroups 先例；LDAP 管理集 = 全部 entry.DN
// （syncOUs 无过滤,extractDNs 同口径,sync.go:179-203）。差集键为模型 OUN 列
// （存储完整 OU DN）。
func (s *SyncService) cleanupVanishedOUs(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry, result *SyncResult) error {
	// Gate 1: 安全闸门 — LDAP 返回为空直接拒绝任何删除（AD 不可达/BaseDN 配错伪装成「OU 全部消失」）。
	if len(entries) == 0 {
		applogger.Warnf("[OU同步] 跳过过期OU清理: LDAP 返回 0 条记录,可能 AD 不可达或 BaseDN 配置错误")
		return nil
	}

	// LDAP 管理集 = 全部 entry（syncOUs 对 entry 无过滤）
	managedOUNDs := make(map[string]bool, len(entries))
	for _, entry := range entries {
		managedOUNDs[entry.DN] = true
	}

	// 存量活行:默认作用域统计,禁 Unscoped（Pitfall 3）
	var staleOUs []models.ADOU
	s.db.WithContext(ctx).
		Where("ad_config_id = ? AND deleted_at IS NULL", config.ID).
		Find(&staleOUs)

	var vanishedDNs []string
	for _, ou := range staleOUs {
		if !managedOUNDs[ou.OUN] {
			vanishedDNs = append(vanishedDNs, ou.OUN)
		}
	}

	if len(vanishedDNs) > 0 {
		// Gate 2: 阈值保护 — 判式严格大于（恰 50% 放行）,与先例逐字同构;
		// 拒绝日志只含数量、不含 DN 明文。
		if len(staleOUs) > 0 && len(vanishedDNs)*2 > len(staleOUs) {
			applogger.Warnf(
				"[OU同步] 跳过过期OU清理: 拟删除 %d/%d 个OU(>50%%),"+
					"可能是 LDAP 返回不完整。请人工核实后手动清理",
				len(vanishedDNs), len(staleOUs))
			return nil
		}

		// 软删:GORM 默认作用域 Delete = UPDATE deleted_at,禁 Unscoped 硬删。
		if err := s.db.WithContext(ctx).
			Where("ad_config_id = ? AND ou_dn IN ?", config.ID, vanishedDNs).
			Delete(&models.ADOU{}).Error; err != nil {
			return fmt.Errorf("删除过期OU失败: %w", err)
		}

		result.DeletedOUCount = len(vanishedDNs)
		applogger.Infof("[OU同步] 清理了 %d 个已不存在的OU", len(vanishedDNs))
	}

	return nil
}

// cleanupVanishedComputers 软删 AD 端已消失的电脑行（SHADOW-01 computer 类别）。
//
// LDAP 管理集 = 仅 cn 非空的 entry（与 syncComputers 的跳过口径同源,
// computer.go:419-423,Pitfall 2）。差集键为 DistinguishedName 列
// （syncComputers 按 DN 管理:extractDNs + queryExistingComputers 按
// distinguished_name 查询,computer.go:383/:388）。
func (s *SyncService) cleanupVanishedComputers(ctx context.Context, config *models.ADConfig, entries []*ldap.Entry, result *SyncResult) error {
	// Gate 1: 安全闸门 — LDAP 返回为空直接拒绝任何删除。
	if len(entries) == 0 {
		applogger.Warnf("[电脑同步] 跳过过期电脑清理: LDAP 返回 0 条记录,可能 AD 不可达或 BaseDN 配置错误")
		return nil
	}

	// LDAP 管理集：仅 cn 非空的 entry 入集（本轮真正被 upsert 管理的行集,Pitfall 2）
	managedComputerDNs := make(map[string]bool, len(entries))
	for _, entry := range entries {
		if entry.GetAttributeValue("cn") != "" {
			managedComputerDNs[entry.DN] = true
		}
	}

	// 存量活行:默认作用域统计,禁 Unscoped（Pitfall 3）
	var staleComputers []models.ADComputer
	s.db.WithContext(ctx).
		Where("ad_config_id = ? AND deleted_at IS NULL", config.ID).
		Find(&staleComputers)

	var vanishedDNs []string
	for _, computer := range staleComputers {
		if !managedComputerDNs[computer.DistinguishedName] {
			vanishedDNs = append(vanishedDNs, computer.DistinguishedName)
		}
	}

	if len(vanishedDNs) > 0 {
		// Gate 2: 阈值保护 — 判式严格大于（恰 50% 放行）,与先例逐字同构;
		// 拒绝日志只含数量、不含 DN 明文。
		if len(staleComputers) > 0 && len(vanishedDNs)*2 > len(staleComputers) {
			applogger.Warnf(
				"[电脑同步] 跳过过期电脑清理: 拟删除 %d/%d 台电脑(>50%%),"+
					"可能是 LDAP 返回不完整。请人工核实后手动清理",
				len(vanishedDNs), len(staleComputers))
			return nil
		}

		// 软删:GORM 默认作用域 Delete = UPDATE deleted_at,禁 Unscoped 硬删。
		if err := s.db.WithContext(ctx).
			Where("ad_config_id = ? AND distinguished_name IN ?", config.ID, vanishedDNs).
			Delete(&models.ADComputer{}).Error; err != nil {
			return fmt.Errorf("删除过期电脑失败: %w", err)
		}

		result.DeletedComputerCount = len(vanishedDNs)
		applogger.Infof("[电脑同步] 清理了 %d 台已不存在的电脑", len(vanishedDNs))
	}

	return nil
}
