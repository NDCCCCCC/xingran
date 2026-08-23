package addomain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/xingran-next/xingran-go-backend/internal/constants"
	"github.com/xingran-next/xingran-go-backend/internal/models"
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

// SyncUserUpdateToAD 同步用户更新到AD域控
// 在系统更新成功后调用，降级处理：AD同步失败不回滚系统更新
func (s *UserADSyncService) SyncUserUpdateToAD(ctx context.Context, userID string, updateReq map[string]interface{}) error {
	// 1. 获取用户信息
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 如果用户没有AD DN，跳过同步
	if user.AdDn == nil || *user.AdDn == "" {
		applogger.Infof("用户 %s 无AD DN，跳过AD同步", user.Username)
		return nil
	}

	// 2. 获取AD配置
	var adConfig models.ADConfig
	if err := s.db.WithContext(ctx).Where("sync_enabled = ? AND status = ?", true, 0).First(&adConfig).Error; err != nil {
		return fmt.Errorf("获取AD配置失败: %w", err)
	}
	// Phase 36 后 admin_password 为 SM4 密文存储，绑定前必须解密
	// （遗漏会导致密文当明文绑定 → LDAP 49 data 52e → 重试触发账号锁定 775）

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	// 所有 LDAP 操作（MoveUser + UpdateUserAttribute）必须在闭包内完成（Pitfall 3）
	fc := NewFailoverClient(s.pool, &adConfig)
	if err := fc.ExecuteWithFailover(ctx, func(ldapClient LDAPClientIface) error {
		// 3. 如果部门变更，需要移动用户到新OU
		if newDeptID, ok := updateReq["deptId"]; ok {
			if newDeptIDStr, ok := newDeptID.(string); ok && newDeptIDStr != "" {
				currentDeptID := ""
				if user.DeptID != nil {
					currentDeptID = *user.DeptID
				}
				if newDeptIDStr != currentDeptID {
					if err := s.moveUserToNewOU(ctx, ldapClient, userID, newDeptIDStr); err != nil {
						applogger.Errorf("移动用户 %s 到新OU失败: %v", user.Username, err)
						// 继续处理属性更新，不因OU移动失败而中断
					}
				}
			}
		}

		// 4. 更新其他属性到AD
		if err := s.syncUserAttributes(ctx, ldapClient, &user, updateReq); err != nil {
			return fmt.Errorf("同步用户属性失败: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return fmt.Errorf("连接AD失败: %w", err)
	}

	// 5. 更新同步时间戳
	if err := s.updateSyncTimestamp(ctx, userID); err != nil {
		applogger.Warnf("更新同步时间戳失败: %v", err)
	}

	applogger.Infof("用户 %s 信息同步到AD成功", user.Username)
	return nil
}

// moveUserToNewOU 移动用户到新OU
func (s *UserADSyncService) moveUserToNewOU(ctx context.Context, ldapClient LDAPClientIface, userID, newDeptID string) error {
	// 1. 查找新部门的OU DN
	ouDN, err := s.mapper.FindOUDNByDeptID(ctx, newDeptID)
	if err != nil {
		return fmt.Errorf("查找部门OU失败: %w", err)
	}

	// 2. 获取用户信息
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	if user.AdDn == nil || *user.AdDn == "" {
		return fmt.Errorf("用户无AD DN，无法移动")
	}

	// 3. 移动用户到新OU
	if err := ldapClient.MoveUser(*user.AdDn, ouDN); err != nil {
		return fmt.Errorf("移动用户到新OU失败: %w", err)
	}

	// 4. 更新用户的ad_ou_dn
	if err := s.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("ad_ou_dn", ouDN).Error; err != nil {
		applogger.Warnf("更新用户ad_ou_dn失败: %v", err)
	}

	applogger.Infof("用户 %s 已移动到新OU: %s", user.Username, ouDN)
	return nil
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

// BatchMoveUsersToNewOU 批量移动用户到新OU
// 为未来功能预留的批量操作接口
func (s *UserADSyncService) BatchMoveUsersToNewOU(ctx context.Context, userIDs []string, newDeptID string) error {
	// 1. 查找新部门的OU DN
	ouDN, err := s.mapper.FindOUDNByDeptID(ctx, newDeptID)
	if err != nil {
		return fmt.Errorf("查找部门OU失败: %w", err)
	}

	// 2. 获取AD配置
	var adConfig models.ADConfig
	if err := s.db.WithContext(ctx).Where("sync_enabled = ? AND status = ?", true, 0).First(&adConfig).Error; err != nil {
		return fmt.Errorf("获取AD配置失败: %w", err)
	}
	// Phase 36 后 admin_password 为 SM4 密文存储，绑定前必须解密
	// （遗漏会导致密文当明文绑定 → LDAP 49 data 52e → 重试触发账号锁定 775）

	// Phase 38 Wave 1: 改走 FailoverClient.ExecuteWithFailover（账号池故障切换）
	// operation 边界 = 整个批量任务（SP-3：单次批量一个 operation，非每用户）
	// 所有 LDAP 操作（MoveUser）必须在闭包内（Pitfall 3）；单用户失败计入 failedCount 不中断批量
	successCount := 0
	failedCount := 0
	fc := NewFailoverClient(s.pool, &adConfig)
	if err := fc.ExecuteWithFailover(ctx, func(ldapClient LDAPClientIface) error {
		// 批量移动用户（分批处理，每批10个）
		batchSize := 10
		for i := 0; i < len(userIDs); i += batchSize {
			end := i + batchSize
			if end > len(userIDs) {
				end = len(userIDs)
			}

			batch := userIDs[i:end]
			for _, userID := range batch {
				if err := s.moveSingleUserToOU(ctx, ldapClient, userID, ouDN); err != nil {
					applogger.Errorf("移动用户失败 [ID=%s]: %v", userID, err)
					failedCount++
				} else {
					successCount++
				}
			}

			// 每批之间暂停1秒，避免AD压力过大
			if i+batchSize < len(userIDs) {
				time.Sleep(1 * time.Second)
			}
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return fmt.Errorf("连接AD失败: %w", err)
	}

	applogger.Infof("批量移动用户完成: 成功=%d, 失败=%d, 目标OU=%s", successCount, failedCount, ouDN)

	if failedCount > 0 {
		return fmt.Errorf("批量移动完成，部分失败: 成功=%d, 失败=%d", successCount, failedCount)
	}
	return nil
}

// moveSingleUserToOU 移动单个用户到指定OU
func (s *UserADSyncService) moveSingleUserToOU(ctx context.Context, ldapClient LDAPClientIface, userID, ouDN string) error {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	if user.AdDn == nil || *user.AdDn == "" {
		return fmt.Errorf("用户 %s 无AD DN，跳过移动", user.Username)
	}

	if err := ldapClient.MoveUser(*user.AdDn, ouDN); err != nil {
		return fmt.Errorf("LDAP移动失败: %w", err)
	}

	// 更新用户的ad_ou_dn
	if err := s.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Update("ad_ou_dn", ouDN).Error; err != nil {
		applogger.Warnf("更新用户 %s ad_ou_dn失败: %v", user.Username, err)
	}

	return nil
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
		Where("sync_enabled = ? AND status = ?", true, 0).
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

	// 3. 预加载所有用户所在部门（含 ancestors + leader）
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

	// 4. 解析每个用户的 leaderID，收集所有 leader user.id（用于批量查 ad_dn）
	type userLeader struct {
		user     models.User
		leaderID string
	}
	resolved := make([]userLeader, 0, len(users))
	leaderIDSet := make(map[string]bool)
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
		resolved = append(resolved, userLeader{user: u, leaderID: leaderID})
		if leaderID != "" && leaderID != u.ID {
			leaderIDSet[leaderID] = true
		}
	}

	// 5. 批量查 leader 用户的 ad_dn（manager 值需为 DN 格式）
	leaderAdDnMap := make(map[string]string)
	if len(leaderIDSet) > 0 {
		ids := make([]string, 0, len(leaderIDSet))
		for id := range leaderIDSet {
			ids = append(ids, id)
		}
		type leaderRow struct {
			ID   string
			AdDn *string
		}
		var leaders []leaderRow
		if err := s.db.WithContext(ctx).
			Table("sys_user").
			Select("id, ad_dn").
			Where("id IN ?", ids).
			Scan(&leaders).Error; err != nil {
			return nil, fmt.Errorf("查询 leader 用户失败: %w", err)
		}
		for _, l := range leaders {
			if l.AdDn != nil && *l.AdDn != "" {
				leaderAdDnMap[l.ID] = *l.AdDn
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
				if r.leaderID == "" {
					result.Skipped++ // 部门链均无 leader
					continue
				}
				if r.leaderID == r.user.ID {
					result.Skipped++ // 自指，保持原值
					continue
				}
				leaderAdDn, ok := leaderAdDnMap[r.leaderID]
				if !ok || leaderAdDn == "" {
					result.Skipped++ // leader 无 ad_dn，无法设 manager
					applogger.Warnf("[AD-MANAGER-SYNC] 用户 %s 的 leader(%s) 无 ad_dn，跳过", r.user.Username, r.leaderID)
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
					err := updateAttr(userDN, map[string]string{"manager": leaderAdDn})

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
			if r.leaderID == "" {
				result.Skipped++
				continue
			}
			if r.leaderID == r.user.ID {
				result.Skipped++
				continue
			}
			leaderAdDn, ok := leaderAdDnMap[r.leaderID]
			if !ok || leaderAdDn == "" {
				result.Skipped++
				applogger.Warnf("[AD-MANAGER-SYNC] 用户 %s 的 leader(%s) 无 ad_dn，跳过", r.user.Username, r.leaderID)
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
				err := updateAttr(userDN, map[string]string{"manager": leaderAdDn})
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

// BatchSyncResult 批量同步结果统计
type BatchSyncResult struct {
	Total   int      `json:"total"`
	Synced  int      `json:"synced"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}

// BatchSyncUsersToAD 批量同步用户属性到 AD（导入后/全量场景）。
//
// 性能：复用单个已绑定 LDAP 连接（ExecuteWithFailover 拿一个账号绑定），
// errgroup + 信号量（MaxConcurrentADSync）并发，避免逐用户新建连接的
// ~100ms 拨号+绑定开销。2274 用户从 ~10 分钟降到 ~10 秒。
//
// 行为（属性同步；不移动 OU）：
//   - 只处理 ad_dn 非空的用户（无 ad_dn 的本地用户跳过）
//   - 同步属性到 AD：displayName（nickname）/mail（email）/telephoneNumber（phone）/department（deptId→deptName）
//   - 单用户失败不中断批量（收集到 Errors）
//   - 成功后更新 ad_synced_at
//
// 为什么不在这里移动 OU（ModifyDN）：
//
//	批量场景无"导入前旧 dept"快照，updateReq 只能从用户当前字段构造，
//	与 DB 中的 dept_id 恒等，无法判断是否变更。OU 移动留给单用户编辑路径
//	SyncUserUpdateToAD（请求体携带新旧值，由调用方判定）。
//
// 与 SyncUserUpdateToAD 关系：单用户编辑用 SyncUserUpdateToAD（含 OU 移动判断），
// 导入/全量用本方法（仅属性同步）。
func (s *UserADSyncService) BatchSyncUsersToAD(ctx context.Context, userIDs []string) (*BatchSyncResult, error) {
	result := &BatchSyncResult{}

	// 1. ADConfig（无启用配置则跳过，返回空 result）
	var adConfig models.ADConfig
	if err := s.db.WithContext(ctx).
		Where("sync_enabled = ? AND status = ?", true, 0).
		First(&adConfig).Error; err != nil {
		applogger.Warnf("[AD-BATCH-SYNC] 未找到启用的 AD 配置，跳过同步: %v", err)
		return result, nil
	}

	// 2. 批量查用户（ad_dn 非空，一次 DB；无 ad_dn 的本地用户跳过）
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

	// 3. ExecuteWithFailover 闭包内 errgroup 并发（operation 边界 = 整个批量，同 SyncManagersToAD）。
	//    errgroup 在此仅用于 gctx 取消传播 + SetLimit 信号量；错误聚合走 result struct
	//    （每个 goroutine 必返 nil，不中断批量），故 _ = g.Wait() 是有意的。
	fc := NewFailoverClient(s.pool, &adConfig)
	if err := fc.ExecuteWithFailover(ctx, func(ldapClient LDAPClientIface) error {
		g, gctx := errgroup.WithContext(ctx)
		g.SetLimit(constants.MaxConcurrentADSync)
		var mu sync.Mutex

		for i := range users {
			u := users[i] // 显式捕获循环变量（读起来意图清晰；Go 1.22+ 实际不必要）

			g.Go(func() error {
				// syncErr 捕获本用户同步结果，goroutine 结束时由聚合 defer 写入 result
				var syncErr error

				// defer 顺序 LIFO：先 panic-recover（设 syncErr），再聚合（写 result）。
				// 这样 panic 正确计入 Failed 而非 Synced。
				defer func() {
					mu.Lock()
					if syncErr != nil {
						result.Failed++
						result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", u.Username, syncErr))
					} else {
						result.Synced++
					}
					mu.Unlock()
				}()

				defer func() {
					if rec := recover(); rec != nil {
						applogger.Errorf("[AD-BATCH-SYNC] panic [user=%s]: %v", u.Username, rec)
						syncErr = fmt.Errorf("panic: %v", rec)
					}
				}()

				// 构造 updateReq（从 user 字段）。导入场景下 updateReq["deptId"] == u.DeptID，
				// 无"导入前旧 dept"快照，无法判断是否变更，所以批量路径不做 OU 移动——
				// OU 移动交给单用户编辑路径 SyncUserUpdateToAD（请求体携带新旧值）。
				updateReq := make(map[string]interface{})
				if u.DeptID != nil {
					updateReq["deptId"] = *u.DeptID
				}
				if u.Email != nil {
					updateReq["email"] = *u.Email
				}
				if u.Phone != nil {
					updateReq["phone"] = *u.Phone
				}
				if u.Nickname != nil {
					updateReq["nickname"] = *u.Nickname
				}

				// 同步属性（复用 ldapClient；department 文本由 syncUserAttributes 内部查 dept_name 填充）
				if err := s.syncUserAttributes(gctx, ldapClient, &u, updateReq); err != nil {
					syncErr = err
					return nil
				}
				// 时间戳失败仅 WARN（属性已成功，不算同步失败）
				if tsErr := s.updateSyncTimestamp(gctx, u.ID); tsErr != nil {
					applogger.Warnf("[AD-BATCH-SYNC] 更新时间戳失败 [user=%s]: %v", u.Username, tsErr)
				}
				return nil
			})
		}

		_ = g.Wait() // 错误聚合已在各 goroutine 的 defer 里完成；此处仅等待
		return nil
	}); err != nil {
		if errors.Is(err, ErrAllAccountsUnavailable) {
			return nil, fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
		}
		return nil, fmt.Errorf("连接 AD 失败: %w", err)
	}

	applogger.Infof("[AD-BATCH-SYNC] 完成: total=%d synced=%d failed=%d",
		result.Total, result.Synced, result.Failed)

	// 失败明细：逐条 WARN 输出，便于 grep 定位失败用户 / AD 错误。
	// 不设上限：业务上需要看到全部失败（如 129/2274）；如未来批次 >10000
	// 需考虑分批 / 文件落盘，但当前规模直接日志即可。
	if result.Failed > 0 {
		applogger.Warnf("[AD-BATCH-SYNC] 失败明细（共 %d 条）:", len(result.Errors))
		for _, e := range result.Errors {
			applogger.Warnf("[AD-BATCH-SYNC]   - %s", e)
		}
	}
	return result, nil
}
