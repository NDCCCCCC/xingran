package addomain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/constants"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// UserADSyncService 用户AD同步服务
// 负责在用户信息修改时同步变更到AD域控
type UserADSyncService struct {
	db     *gorm.DB
	pool   AccountPool // Phase 38 Wave 1 DI 脚手架（38-02 将用于 FailoverClient 闭包改造）
	ldap   *LDAPClient
	mapper *DeptOUmapper
	// updateUserAttributeFn 测试注入钩子：非 nil 时 SyncManagersToAD 直接调用它，
	// 绕过 NewLDAPClient + Connect（避免单测依赖真实 AD 服务器）。生产路径为 nil。
	updateUserAttributeFn func(userDN string, attrs map[string]string) error
}

// NewUserADSyncService 创建UserADSyncService实例
// Phase 38 Wave 1: 注入 AccountPool 字段（38-02 将用于 FailoverClient 闭包改造）。
func NewUserADSyncService(db *gorm.DB, pool AccountPool, ldapClient *LDAPClient, mapper *DeptOUmapper) *UserADSyncService {
	return &UserADSyncService{
		db:     db,
		pool:   pool,
		ldap:   ldapClient,
		mapper: mapper,
	}
}

// syncUserAttributes 同步用户属性到AD
func (s *UserADSyncService) syncUserAttributes(ctx context.Context, ldapClient LDAPClientIface, user *models.User, updateReq map[string]interface{}) error {
	attributes := make(map[string]string)

	// 映射系统字段到AD属性
	if nickname, ok := updateReq["nickname"]; ok {
		if nickStr, ok := nickname.(string); ok && nickStr != "" {
			attributes["displayName"] = nickStr
		}
	}

	if email, ok := updateReq["email"]; ok {
		if emailStr, ok := email.(string); ok {
			attributes["mail"] = emailStr
		}
	}

	if phone, ok := updateReq["phone"]; ok {
		if phoneStr, ok := phone.(string); ok {
			attributes["telephoneNumber"] = phoneStr
		}
	}

	// 如果部门变更，同步department属性
	if deptID, ok := updateReq["deptId"]; ok {
		if deptIDStr, ok := deptID.(string); ok && deptIDStr != "" {
			var dept models.Department
			if err := s.db.WithContext(ctx).Where("id = ?", deptIDStr).First(&dept).Error; err == nil {
				attributes["department"] = dept.DeptName
			}
		}
	}

	if len(attributes) == 0 {
		applogger.Infof("用户 %s 无需同步的属性变更", user.Username)
		return nil
	}

	// Debug session ad-update-attr-no-such-object Fix 1：
	// 预检 sys_user.ad_dn 在 AD 端是否仍然存在，避免对已删除/移走的对象
	// 执行 Modify → LDAP code 32 → handler 3 次重试 → MarkFailure 累加
	// → 应用层 breaker 熔断 30 分钟 → 全池 bind 失败 → 用户看到"管理员账号被锁"。
	//
	// DNExists 返回 false（code 32 语义）→ 清空 ad_dn，让下次 login sync
	// 重新拉取（走 SyncADUser 的 read path）。这条路径不返回 error，
	// 上层 SyncUserUpdateToAD 视为同步成功，避免后续 handler 重试放大。
	exists, err := ldapClient.DNExists(*user.AdDn)
	if err != nil {
		// 网络/认证错误等非 code 32 情况：上抛让 FailoverClient 走下一账号重试
		return fmt.Errorf("DN 存在性预检失败 [%s]: %w", *user.AdDn, err)
	}
	if !exists {
		applogger.Warnf("用户 %s 的 AD DN 在 AD 端不存在 [dn=%s]，清空 ad_dn 等待下次 login sync 重新拉取",
			user.Username, *user.AdDn)
		// 清空过期 DN（不阻塞系统主流程；ad_synced_at 也清，避免 UI 误显示"已同步"）
		s.db.WithContext(ctx).
			Model(&models.User{}).
			Where("id = ?", user.ID).
			Updates(map[string]interface{}{
				"ad_dn":        nil,
				"ad_ou_dn":     nil,
				"ad_synced_at": nil,
			})
		return nil
	}

	// 更新AD属性
	if err := ldapClient.UpdateUserAttribute(*user.AdDn, attributes); err != nil {
		// 区分 code 32（对象在预检后被删/移走，竞态）与其它错误：
		// code 32 视为"过期 DN"语义，做同样的清空 + INFO，让 handler 短路
		// 不再重试（debug session Fix 2）。返回哨兵错误 ErrADTargetNotFound
		// 让 handler 用 errors.Is 判定并 break 出重试循环。
		if ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) {
			applogger.Warnf("用户 %s 的 AD DN 在 modify 时返回 code 32（预检后被删/移走）[dn=%s]，清空 ad_dn",
				user.Username, *user.AdDn)
			s.db.WithContext(ctx).
				Model(&models.User{}).
				Where("id = ?", user.ID).
				Updates(map[string]interface{}{
					"ad_dn":        nil,
					"ad_ou_dn":     nil,
					"ad_synced_at": nil,
				})
			return ErrADTargetNotFound
		}
		return fmt.Errorf("更新AD用户属性失败: %w", err)
	}

	applogger.Debugf("用户 %s 属性同步到AD: %v", user.Username, attributes)
	return nil
}

// updateSyncTimestamp 更新同步时间戳
func (s *UserADSyncService) updateSyncTimestamp(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("ad_synced_at", time.Now()).Error
}

// ManagerSyncResult AD manager 同步结果统计
type ManagerSyncResult struct {
	Total   int      `json:"total"`            // 候选用户总数（有 ad_dn）
	Synced  int      `json:"synced"`           // 成功写入 AD manager 数
	Skipped int      `json:"skipped"`          // 跳过数（无 leader / 自指 / leader 无 ad_dn / 无部门）
	Failed  int      `json:"failed"`           // 失败数（LDAP Modify 出错）
	Errors  []string `json:"errors,omitempty"` // 失败明细（username: reason）
}

// maxLeaderDepth 递归查找部门 leader 的最大深度（防祖先链异常导致无限遍历）
const maxLeaderDepth = 20

// splitAncestorIDs 解析 ancestors 字段（"0,rootID,parentID"，从根到父，含 "0" 根占位符）。
// 过滤 "0" 根占位符、空值、当前部门自身。返回原始顺序（从根到父，远→近）的祖先 ID 列表。
func splitAncestorIDs(ancestors string, selfID string) []string {
	if ancestors == "" {
		return nil
	}
	parts := strings.Split(ancestors, ",")
	var result []string // nil 初始：无匹配时返回 nil（区别于空切片，便于调用方判空）
	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" || id == "0" || id == selfID {
			continue
		}
		result = append(result, id)
	}
	return result
}

// resolveLeaderByAncestors 解析用户经理：当前部门 leader，无则递归祖先链。
// 返回 leader 的 user.id（UUID，来自 sys_dept.leader）。
// 按从近到远顺序（父→根）找第一个有非空 leader 的部门，深度上限 maxLeaderDepth。
func (s *UserADSyncService) resolveLeaderByAncestors(ctx context.Context, dept *models.Department) (string, error) {
	// 当前部门有 leader
	if dept.Leader != nil && *dept.Leader != "" {
		return *dept.Leader, nil
	}

	// 解析祖先链（过滤 "0" 根占位符、空值、自身）
	ancestors := splitAncestorIDs(dept.Ancestors, dept.ID)
	if len(ancestors) == 0 {
		return "", nil
	}

	// 一次性查这些祖先部门的 leader（只查有 leader 的，避免 N+1）
	type deptLeaderRow struct {
		ID     string
		Leader string
	}
	var rows []deptLeaderRow
	if err := s.db.WithContext(ctx).
		Table("sys_dept").
		Select("id, leader").
		Where("id IN ?", ancestors).
		Where("leader IS NOT NULL AND leader <> ''").
		Find(&rows).Error; err != nil {
		return "", fmt.Errorf("查询祖先部门 leader 失败: %w", err)
	}

	leaderMap := make(map[string]string, len(rows))
	for _, r := range rows {
		leaderMap[r.ID] = r.Leader
	}

	// 从近到远遍历（ancestors 为根→父序，反向即父→根），找第一个有 leader 的
	for i := len(ancestors) - 1; i >= 0; i-- {
		depth := len(ancestors) - 1 - i
		if depth >= maxLeaderDepth {
			break
		}
		if leader, ok := leaderMap[ancestors[i]]; ok {
			return leader, nil
		}
	}

	return "", nil
}

// SyncManagersToAD 批量同步用户 AD manager 属性（= 部门 leader 的 ad_dn）。
//
// 同步规则：
//   - 候选 = 所有 ad_dn 非空的用户（userIDs 非空时仅限这些用户）
//   - 部门无 leader → 递归祖先链找第一个有 leader 的部门（深度上限 maxLeaderDepth）
//   - leader 是用户自己 → 跳过（保持原值，不同步 manager）
//   - leader 无 ad_dn → 跳过 + WARN
//   - 复用单个 LDAP 连接（一次 connect/bind，遍历所有 UpdateUserAttribute），
//     不同于 SyncUserUpdateToAD 每用户一个连接
//   - 信号量限并发（MaxConcurrentADSync=3），单失败不中断批量
//
// 降级：无启用的 AD 配置时返回空 result（不报错），便于在未配置 AD 的环境调用。
func (s *UserADSyncService) SyncManagersToAD(ctx context.Context, userIDs []string) (*ManagerSyncResult, error) {
	result := &ManagerSyncResult{}

	// 1. ADConfig（无启用配置则跳过，返回空 result）
	var adConfig models.ADConfig
	if err := s.db.WithContext(ctx).
		Where("sync_enabled = ? AND status = ?", true, models.ADConfigStatusEnabled).
		First(&adConfig).Error; err != nil {
		applogger.Warnf("[AD-MANAGER-SYNC] 未找到启用的 AD 配置，跳过同步: %v", err)
		return result, nil
	}
	// Phase 36 后 admin_password 为 SM4 密文存储，绑定前必须解密（同 SyncUserUpdateToAD）

	// 2. 查候选用户（ad_dn 非空）
	query := s.db.WithContext(ctx).
		Model(&models.User{}).
		Where("ad_dn IS NOT NULL AND ad_dn <> ''")
	if len(userIDs) > 0 {
		query = query.Where("id IN ?", userIDs)
	}
	var users []models.User
	if err := query.Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	result.Total = len(users)
	if len(users) == 0 {
		return result, nil
	}

	// 3. 经理解析：**规则优先**，无规则时回退旧「部门 leader 递归祖先」逻辑。
	//
	// ⚠ 规则优先是必需的：手动「同步经理」与排水器 update_manager 必须用同一套判定，
	//   否则手动全量会用 leader 逻辑覆盖规则引擎写入的 manager（规则形同虚设）。
	//   排水器侧的同款判定见 ManagerRuleService.ResolveManagerDN。
	hasRules, err := NewManagerRuleService(s.db).HasEnabledRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询经理同步规则失败: %w", err)
	}

	// userManager：待写 AD manager 属性的用户 + 其经理的 sys_user.id
	type userManager struct {
		user      models.User
		managerID string
	}
	resolved := make([]userManager, 0, len(users))
	managerIDSet := make(map[string]bool)

	if hasRules {
		// 3a. 规则引擎：一次性预加载（规则/机构/部门/用户/岗位），之后纯内存解析
		resolver, err := NewManagerRuleService(s.db).LoadResolver(ctx)
		if err != nil {
			return nil, fmt.Errorf("加载经理同步规则失败: %w", err)
		}
		for _, u := range users {
			res := resolver.Resolve(&u)
			if res.Skipped || res.ManagerID == "" {
				result.Skipped++
				applogger.Infof("[AD-MANAGER-SYNC] 用户 %s 跳过（规则「%s」）：%s",
					u.Username, res.RuleName, res.Reason)
				continue
			}
			for _, w := range res.Warnings {
				applogger.Warnf("[AD-MANAGER-SYNC] 用户 %s 规则告警: %s", u.Username, w)
			}
			resolved = append(resolved, userManager{user: u, managerID: res.ManagerID})
			managerIDSet[res.ManagerID] = true
		}
	} else {
		// 3b. 回退旧逻辑：预加载所有用户所在部门（含 ancestors + leader）
		deptIDs := make([]string, 0, len(users))
		for _, u := range users {
			if u.DeptID != nil && *u.DeptID != "" {
				deptIDs = append(deptIDs, *u.DeptID)
			}
		}
		deptMap := make(map[string]*models.Department)
		if len(deptIDs) > 0 {
			var depts []models.Department
			if err := s.db.WithContext(ctx).
				Table("sys_dept").
				Where("id IN ?", deptIDs).
				Find(&depts).Error; err != nil {
				return nil, fmt.Errorf("查询部门失败: %w", err)
			}
			for i := range depts {
				deptMap[depts[i].ID] = &depts[i]
			}
		}

		for _, u := range users {
			if u.DeptID == nil || *u.DeptID == "" {
				result.Skipped++
				continue
			}
			dept, ok := deptMap[*u.DeptID]
			if !ok {
				result.Skipped++
				continue
			}
			leaderID, err := s.resolveLeaderByAncestors(ctx, dept)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("%s: 解析 leader 失败: %v", u.Username, err))
				continue
			}
			resolved = append(resolved, userManager{user: u, managerID: leaderID})
			if leaderID != "" && leaderID != u.ID {
				managerIDSet[leaderID] = true
			}
		}
	}

	// 4. 批量查经理用户的 ad_dn（manager 值需为 DN 格式）
	managerAdDnMap := make(map[string]string)
	if len(managerIDSet) > 0 {
		ids := make([]string, 0, len(managerIDSet))
		for id := range managerIDSet {
			ids = append(ids, id)
		}
		type managerRow struct {
			ID   string
			AdDn *string
		}
		var managers []managerRow
		if err := s.db.WithContext(ctx).
			Table("sys_user").
			Select("id, ad_dn").
			Where("id IN ?", ids).
			Scan(&managers).Error; err != nil {
			return nil, fmt.Errorf("查询经理用户失败: %w", err)
		}
		for _, m := range managers {
			if m.AdDn != nil && *m.AdDn != "" {
				managerAdDnMap[m.ID] = *m.AdDn
			}
		}
	}

	// 6. 构造 updateAttr：测试钩子优先（SHA-5），否则走 FailoverClient 账号池故障切换
	var updateAttr func(string, map[string]string) error
	if s.updateUserAttributeFn != nil {
		// SHA-5: 测试钩子分支保留（7 个 TestSyncManagersToAD_* 回归测试依赖，绕过 FailoverClient）
		updateAttr = s.updateUserAttributeFn
	} else {
		// Phase 38 Wave 1: operation 边界 = 整个 errgroup 批量（SP-3）
		// FailoverClient 闭包内启动 errgroup，g.Wait() 必须在闭包内（Pitfall 3+5）
		fc := NewFailoverClient(s.pool, &adConfig)
		if err := fc.ExecuteWithFailover(ctx, func(ldapClient LDAPClientIface) error {
			updateAttr = ldapClient.UpdateUserAttribute
			// 7. 信号量并发同步（单失败不中断：g.Go 始终返回 nil）
			g, gctx := errgroup.WithContext(ctx)
			g.SetLimit(constants.MaxConcurrentADSync)
			var mu sync.Mutex
			for _, r := range resolved {
				r := r
				// 跳过逻辑（主 goroutine 内，无并发）
				if r.managerID == "" {
					result.Skipped++ // 部门链均无 leader
					continue
				}
				if r.managerID == r.user.ID {
					result.Skipped++ // 自指，保持原值
					continue
				}
				managerAdDn, ok := managerAdDnMap[r.managerID]
				if !ok || managerAdDn == "" {
					result.Skipped++ // 经理无 ad_dn，无法设 manager
					applogger.Warnf("[AD-MANAGER-SYNC] 用户 %s 的经理(%s) 无 ad_dn，跳过", r.user.Username, r.managerID)
					continue
				}

				g.Go(func() error {
					defer func() {
						if rec := recover(); rec != nil {
							applogger.Errorf("[AD-MANAGER-SYNC] panic [user=%s]: %v", r.user.Username, rec)
						}
					}()

					userDN := ""
					if r.user.AdDn != nil {
						userDN = *r.user.AdDn
					}
					err := updateAttr(userDN, map[string]string{"manager": managerAdDn})

					// 成功才更新时间戳（持锁外执行，减少临界区）
					if err == nil {
						if tsErr := s.updateSyncTimestamp(gctx, r.user.ID); tsErr != nil {
							applogger.Warnf("[AD-MANAGER-SYNC] 更新时间戳失败 [user=%s]: %v", r.user.Username, tsErr)
						}
					}

					mu.Lock()
					defer mu.Unlock()
					if err != nil {
						result.Failed++
						result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", r.user.Username, err))
						applogger.Warnf("[AD-MANAGER-SYNC] 同步 manager 失败 [user=%s]: %v", r.user.Username, err)
						return nil
					}
					result.Synced++
					return nil
				})
			}
			_ = g.Wait()
			return nil
		}); err != nil {
			if errors.Is(err, ErrAllAccountsUnavailable) {
				return nil, fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
			}
			return nil, fmt.Errorf("连接 AD 失败: %w", err)
		}
	}

	// 测试钩子分支仍需在外部跑 errgroup（钩子模式下闭包未执行）
	if s.updateUserAttributeFn != nil {
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(constants.MaxConcurrentADSync)
		var mu sync.Mutex
		for _, r := range resolved {
			r := r
			if r.managerID == "" {
				result.Skipped++
				continue
			}
			if r.managerID == r.user.ID {
				result.Skipped++
				continue
			}
			managerAdDn, ok := managerAdDnMap[r.managerID]
			if !ok || managerAdDn == "" {
				result.Skipped++
				applogger.Warnf("[AD-MANAGER-SYNC] 用户 %s 的经理(%s) 无 ad_dn，跳过", r.user.Username, r.managerID)
				continue
			}
			g.Go(func() error {
				defer func() {
					if rec := recover(); rec != nil {
						applogger.Errorf("[AD-MANAGER-SYNC] panic [user=%s]: %v", r.user.Username, rec)
					}
				}()
				userDN := ""
				if r.user.AdDn != nil {
					userDN = *r.user.AdDn
				}
				err := updateAttr(userDN, map[string]string{"manager": managerAdDn})
				if err == nil {
					if tsErr := s.updateSyncTimestamp(gctx, r.user.ID); tsErr != nil {
						applogger.Warnf("[AD-MANAGER-SYNC] 更新时间戳失败 [user=%s]: %v", r.user.Username, tsErr)
					}
				}
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					result.Failed++
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", r.user.Username, err))
					applogger.Warnf("[AD-MANAGER-SYNC] 同步 manager 失败 [user=%s]: %v", r.user.Username, err)
					return nil
				}
				result.Synced++
				return nil
			})
		}
		_ = g.Wait()
	}

	applogger.Infof("[AD-MANAGER-SYNC] 完成: total=%d synced=%d skipped=%d failed=%d",
		result.Total, result.Synced, result.Skipped, result.Failed)
	return result, nil
}
