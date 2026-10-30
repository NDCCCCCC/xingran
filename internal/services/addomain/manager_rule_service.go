package addomain

// 通用经理同步方案 —— 规则引擎 + 规则/机构配置 CRUD。
//
// ============================ 概念模型 ============================
//
// 机构（org）  : sys_manager_org 定义。dept_id 指向 sys_dept 上的机构节点，
//                parent_org_id 指上级机构，org_type 0=分公司本部 / 1=分支机构。
// 用户所属机构  : org(U) = U 的部门 ancestors 链上**最近的**、命中 sys_manager_org
//                的部门节点（含 U 自己所在的部门）。
// 部门负责人    : 某部门的 sys_dept.leader 指向的用户 id。
// 一把手        : 在「一把手岗位」任职、且其所属机构 == 目标机构的那名用户。
// 岗位任职人    : 岗位模块（sys_post + user_posts 关联）里的任职用户。
//
// ============================ 规则语义 ============================
//
// 对每个用户 U：
//  1. 算 org(U)；算不出 → 跳过（Skipped + WARN）。
//  2. 按 priority 升序遍历启用规则 R，找**第一条**「主体匹配 + 范围匹配」的规则：
//     - 主体匹配（subject_type）：
//         employee     = U 不是任何部门的 leader，且不是任何机构的一把手
//         dept_leader  = U 是至少一个部门的 leader
//         org_head     = U 是至少一个机构的一把手（一把手岗位取 R.SubjectPostID，
//                        为空时取 R.ManagerPostID）
//         post_holder  = U 是 R.SubjectPostID 岗位的任职人
//         若 R.SubjectOrgID 非空：主体还必须满足「所属机构 == R.SubjectOrgID」
//     - 范围匹配（scope_type）：
//         0 ManagerScopeAll      = 总是匹配
//         1 ManagerScopeDeptTree = U 的部门在 R.ScopeID 的子树内（含自身）
//         2 ManagerScopeOrgTree  = org(U) 在 R.ScopeID 所指机构的子树内
//  3. 经理取值（manager_type）：
//         dept_leader = U 所在部门的 leader（从 U 的部门起沿 ancestors 由近到远回溯）
//         org_leader  = org(U) 对应部门节点的 sys_dept.leader
//         post_holder / org_head = R.ManagerPostID 岗位的任职人，按 R.ManagerOrgID 过滤机构：
//             R.ManagerOrgID 为空    → org(U) 内的任职人
//             R.ManagerOrgID="parent" → org(U) 的**上级机构**内的任职人
//             R.ManagerOrgID=某 UUID  → 该机构内的任职人
//             任职人筛选：未软删、status=0 正常、且其所属机构 == 目标机构
//             多个任职人：取 ad_dn 非空、user id 升序的第一个，并记 WARN
//  4. 经理 = 目标用户的 ad_dn；目标用户无 ad_dn → 跳过 + WARN；
//     经理 == U（自指）→ 跳过 + WARN。
//
// ============================ 配置示例 ============================
//
// 设 O0=总公司(org_type=0 本部)，O1=武汉中支(org_type=1, parent=O0)；
// 岗位 P_gm=总经理（一把手岗），P_vp=分管副总：
//
//	| 场景                        | 规则                                                                 |
//	|-----------------------------|----------------------------------------------------------------------|
//	| 分公司本部普通员工 → 部门负责人 | subject=employee, manager=dept_leader, scope=OrgTree(O0)              |
//	| 分公司部门经理 → 分管总       | subject=dept_leader, manager=post_holder(P_vp), manager_org=空(自身机构) |
//	| 非分公司普通员工 → 机构一把手   | subject=employee, manager=org_head(P_gm), manager_org=空(自身机构)        |
//	| 非分公司一把手 → 分公司总经理   | subject=org_head(subject_post=P_gm), manager=org_head(P_gm), manager_org="parent" |
//	| 兜底                        | subject=employee, manager=dept_leader, scope=All                       |
//
// ============================ 向后兼容 ============================
//
// sys_manager_rule 表不存在（migration_220 未执行 / 旧部署）或没有任何启用规则时，
// SyncManagersToAD **回退**到既有的「部门 leader 递归祖先」逻辑 —— 既有行为与
// 既有回归测试（manager_sync_test.go）必须继续通过。

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// managerRuleLogPrefix 规则引擎日志前缀（与既有 [AD-MANAGER-SYNC] 区分，便于排障 grep）。
const managerRuleLogPrefix = "[AD-MANAGER-RULE]"

// managerUserPost 岗位任职关系行（user_posts / sys_user_post 关联表）。
type managerUserPost struct {
	UserID string
	PostID string
}

// ============================ 解析结果类型 ============================

// ManagerResolution 单个用户的经理解析结果。
//
// Skipped=true 表示本次不写 AD（无机构 / 无匹配规则 / 自指 / 经理无 ad_dn 等），
// Reason 给出人类可读原因，Warnings 收集需要管理员关注的提示（如岗位多人）。
type ManagerResolution struct {
	UserID    string   `json:"userId"`
	ManagerID string   `json:"managerId,omitempty"` // 经理的 sys_user.id
	RuleID    string   `json:"ruleId,omitempty"`    // 命中的规则 id
	RuleName  string   `json:"ruleName,omitempty"`  // 命中的规则名
	Skipped   bool     `json:"skipped"`
	Reason    string   `json:"reason,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

// ============================ 规则引擎 ============================

// ManagerRuleService 经理同步规则引擎 + 规则/机构配置服务。
type ManagerRuleService struct {
	db *gorm.DB
}

// NewManagerRuleService 创建规则引擎服务。
func NewManagerRuleService(db *gorm.DB) *ManagerRuleService {
	return &ManagerRuleService{db: db}
}

// HasEnabledRules 判断是否存在启用的经理同步规则。
//
// 表不存在（migration_220 未执行 / 旧部署）→ (false, nil)：调用方据此回退到
// 「部门 leader 递归祖先」的旧同步逻辑，保证升级后不会突然没有 manager。
func (s *ManagerRuleService) HasEnabledRules(ctx context.Context) (bool, error) {
	if s.db == nil || !s.db.Migrator().HasTable(&models.ManagerRule{}) {
		return false, nil
	}
	var cnt int64
	if err := s.db.WithContext(ctx).
		Model(&models.ManagerRule{}).
		Where("enabled = ?", true).
		Count(&cnt).Error; err != nil {
		return false, fmt.Errorf("查询启用的经理同步规则失败: %w", err)
	}
	return cnt > 0, nil
}

// ============================ Resolver 进程内缓存 ============================

// ManagerResolver 是全量只读索引（规则/机构/部门/用户/岗位任职），
// Resolve 及各 match/resolve 方法只读 map，可安全跨请求共享。
// 预览端点等高频读路径复用缓存实例，避免每次全表预加载；
// 规则/机构任一写操作成功后由 invalidateResolverCache 主动失效，TTL 兜底。
var (
	resolverCacheMu     sync.Mutex
	resolverCache       *ManagerResolver
	resolverCacheLoaded time.Time
)

// resolverCacheTTL 缓存存活时间。配置页改动会主动失效，TTL 仅兜底
// （如 DB 侧手工改数据）。预览展示允许该量级的延迟。
const resolverCacheTTL = 60 * time.Second

// invalidateResolverCache 使 resolver 进程内缓存失效（规则/机构写路径调用）。
func invalidateResolverCache() {
	resolverCacheMu.Lock()
	resolverCache = nil
	resolverCacheMu.Unlock()
}

// enqueueAllManagersRecheck 规则/机构配置变更后，为全部有 AD 绑定的在职用户入队
// update_manager，排水器会按最新规则重算每个人的经理（30s 内生效）。
// 重复入队由 ad_push_queue 的部分唯一索引 + adPushOnConflictClause 合并，不会堆积；
// 入队失败不回滚已成功的配置变更，仅 WARN（管理员可手动「同步经理」全量兜底）。
func enqueueAllManagersRecheck(ctx context.Context, db *gorm.DB) {
	var ids []string
	if err := db.WithContext(ctx).Model(&models.User{}).
		Where("deleted_at IS NULL AND ad_dn IS NOT NULL AND ad_dn <> ''").
		Pluck("id", &ids).Error; err != nil {
		applogger.Warnf("%s 规则变更后查询待重算用户失败: %v", managerRuleLogPrefix, err)
		return
	}
	if len(ids) == 0 {
		return
	}
	if err := NewADPushEnqueuer().EnqueueUpdateManager(db.WithContext(ctx), ids); err != nil {
		applogger.Warnf("%s 规则变更后入队经理重算失败（共 %d 人）: %v", managerRuleLogPrefix, len(ids), err)
		return
	}
	applogger.Infof("%s 规则/机构已变更，已为 %d 个AD绑定用户入队经理重算", managerRuleLogPrefix, len(ids))
}

// LoadResolver 一次性预加载规则引擎所需的全部只读数据（带进程内 TTL 缓存）。
func (s *ManagerRuleService) LoadResolver(ctx context.Context) (*ManagerResolver, error) {
	resolverCacheMu.Lock()
	if resolverCache != nil && time.Since(resolverCacheLoaded) < resolverCacheTTL {
		r := resolverCache
		resolverCacheMu.Unlock()
		return r, nil
	}
	resolverCacheMu.Unlock()

	// 查库在锁外执行（全表预加载耗时），并发 miss 容忍重复查询，
	// 结果一致（同表快照级别差异可忽略）。
	r, err := s.loadResolverUncached(ctx)
	if err != nil {
		return nil, err
	}
	resolverCacheMu.Lock()
	resolverCache = r
	resolverCacheLoaded = time.Now()
	resolverCacheMu.Unlock()
	return r, nil
}

// loadResolverUncached 实际执行全量预加载。
//
// 为什么要「一次性预加载」：规则数 × 用户数的朴素实现会产生 N+1 放大
// （每个用户都要遍历规则并回查部门/岗位/机构）。本方法把只读数据全部装进内存，
// 之后的解析是纯内存计算。
func (s *ManagerRuleService) loadResolverUncached(ctx context.Context) (*ManagerResolver, error) {
	r := &ManagerResolver{
		orgByID:           make(map[string]models.ManagerOrg),
		orgByDept:         make(map[string]models.ManagerOrg),
		depts:             make(map[string]models.Department),
		users:             make(map[string]models.User),
		postNames:         make(map[string]string),
		deptNames:         make(map[string]string),
		orgNames:          make(map[string]string),
		deptLeaderUserIDs: make(map[string]bool),
		postHolders:       make(map[string][]string),
		userPosts:         make(map[string][]string),
	}

	// 1) 启用规则（priority 升序；同优先级按创建时间升序，保证结果稳定）
	if s.db.Migrator().HasTable(&models.ManagerRule{}) {
		if err := s.db.WithContext(ctx).
			Model(&models.ManagerRule{}).
			Where("enabled = ?", true).
			Order("priority ASC, created_at ASC").
			Find(&r.rules).Error; err != nil {
			return nil, fmt.Errorf("查询经理同步规则失败: %w", err)
		}
	}

	// 2) 启用机构
	if s.db.Migrator().HasTable(&models.ManagerOrg{}) {
		if err := s.db.WithContext(ctx).
			Model(&models.ManagerOrg{}).
			Where("enabled = ?", true).
			Find(&r.orgs).Error; err != nil {
			return nil, fmt.Errorf("查询机构定义失败: %w", err)
		}
	}
	for _, org := range r.orgs {
		r.orgByID[org.ID] = org
		r.orgByDept[org.DeptID] = org
		r.orgNames[org.ID] = org.OrgName
	}

	// 3) 部门（id / dept_name / parent_id / ancestors / leader）
	var depts []models.Department
	if err := s.db.WithContext(ctx).
		Table("sys_dept").
		Select("id, dept_name, parent_id, ancestors, leader").
		Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询部门失败: %w", err)
	}
	for _, d := range depts {
		r.depts[d.ID] = d
		r.deptNames[d.ID] = d.DeptName
		if d.Leader != nil && *d.Leader != "" {
			r.deptLeaderUserIDs[*d.Leader] = true
		}
	}

	// 4) 用户（id / username / nickname / dept_id / ad_dn / status）
	//    ⚠ 精简测试库可能没有 status 列 → 降级为最小列集（status 视为 0 正常）。
	//    ⚠ 不能用 map dest（Find(&map[string]models.User)）：Table+Select 精简列
	//    场景下 GORM map 扫描报 "Scan called without calling Next"，必须 slice 查询后手工建 map。
	//    nickname 供经理预览端点显示「姓名(账号)」（有 nickname 列即有，无降级风险）。
	var userRows []models.User
	if err := s.db.WithContext(ctx).
		Table("sys_user").
		Select("id, username, nickname, dept_id, ad_dn, status").
		Find(&userRows).Error; err != nil {
		applogger.Warnf("%s 查询用户（含 status 列）失败，降级为最小列集: %v", managerRuleLogPrefix, err)
		userRows = nil
		if err2 := s.db.WithContext(ctx).
			Table("sys_user").
			Select("id, username, dept_id, ad_dn").
			Find(&userRows).Error; err2 != nil {
			return nil, fmt.Errorf("查询用户失败: %w", err2)
		}
	}
	r.users = make(map[string]models.User, len(userRows))
	for _, u := range userRows {
		r.users[u.ID] = u
	}

	// 5) 岗位任职关系
	//    真实关联表是 sys_user_post（写入路径见 services/system/user_service.go
	//    的 models.UserPost.TableName()）。user_posts 是 many2many 标签写错时
	//    产生的幻影表，已由 migration_222 清理；此处保留兜底查询以兼容尚未
	//    执行该迁移的旧部署（HasTable 守卫 + 去重，不会重复计入）。
	for _, table := range []string{"sys_user_post", "user_posts"} {
		if !s.db.Migrator().HasTable(table) {
			continue
		}
		var rows []managerUserPost
		if err := s.db.WithContext(ctx).
			Table(table).
			Select("user_id, post_id").
			Find(&rows).Error; err != nil {
			return nil, fmt.Errorf("查询岗位任职关系失败[%s]: %w", table, err)
		}
		for _, row := range rows {
			if row.UserID == "" || row.PostID == "" {
				continue
			}
			r.addPostHolder(row.PostID, row.UserID)
		}
	}
	r.sortPostHolders()

	// 6) 岗位名称（列表翻译用）
	if s.db.Migrator().HasTable(&models.Post{}) {
		var posts []models.Post
		if err := s.db.WithContext(ctx).
			Model(&models.Post{}).
			Select("id, post_name, post_code").
			Find(&posts).Error; err != nil {
			return nil, fmt.Errorf("查询岗位失败: %w", err)
		}
		for _, p := range posts {
			r.postNames[p.ID] = p.PostName
		}
	}

	return r, nil
}

// ResolveManagers 批量解析用户的经理。
func (s *ManagerRuleService) ResolveManagers(ctx context.Context, users []models.User) ([]ManagerResolution, error) {
	resolver, err := s.LoadResolver(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ManagerResolution, 0, len(users))
	for i := range users {
		out = append(out, resolver.Resolve(&users[i]))
	}
	return out, nil
}

// ResolveOne 解析单个用户的经理（排水器 update_manager 动作、单用户重算用）。
func (s *ManagerRuleService) ResolveOne(ctx context.Context, u *models.User) (*ManagerResolution, error) {
	resolver, err := s.LoadResolver(ctx)
	if err != nil {
		return nil, err
	}
	res := resolver.Resolve(u)
	return &res, nil
}

// ResolveManagerDN 解析用户经理并返回其 AD DN（DN 格式）。
//
// 语义（排水器 update_manager 分派使用）：
//   - 有启用规则 → 走规则引擎
//   - 无启用规则 → 回退既有的「部门 leader 递归祖先」逻辑
//   - 返回 dn 为空串时，reason 说明为什么没有可写的 manager（跳过，不写 AD）
func (s *ManagerRuleService) ResolveManagerDN(ctx context.Context, u *models.User) (dn string, reason string, err error) {
	hasRules, err := s.HasEnabledRules(ctx)
	if err != nil {
		return "", "", err
	}
	if !hasRules {
		// 回退旧逻辑：部门 leader 递归祖先
		syncSvc := &UserADSyncService{db: s.db}
		if u.DeptID == nil || *u.DeptID == "" {
			return "", "用户无部门，无法回溯部门负责人", nil
		}
		var dept models.Department
		if err := s.db.WithContext(ctx).Table("sys_dept").Where("id = ?", *u.DeptID).First(&dept).Error; err != nil {
			return "", fmt.Sprintf("查询用户部门失败: %v", err), nil
		}
		leaderID, err := syncSvc.resolveLeaderByAncestors(ctx, &dept)
		if err != nil {
			return "", "", fmt.Errorf("回溯部门负责人失败: %w", err)
		}
		if leaderID == "" {
			return "", "部门链均无负责人", nil
		}
		if leaderID == u.ID {
			return "", "负责人为本人（自指），跳过", nil
		}
		managerDN, err := s.lookupAdDn(ctx, leaderID)
		if err != nil {
			return "", "", err
		}
		if managerDN == "" {
			return "", "负责人无 ad_dn", nil
		}
		return managerDN, "", nil
	}

	res, err := s.ResolveOne(ctx, u)
	if err != nil {
		return "", "", err
	}
	if res.Skipped || res.ManagerID == "" {
		r := res.Reason
		if r == "" {
			r = "规则未解析出经理"
		}
		return "", r, nil
	}
	managerDN, err := s.lookupAdDn(ctx, res.ManagerID)
	if err != nil {
		return "", "", err
	}
	if managerDN == "" {
		return "", "经理用户无 ad_dn", nil
	}
	return managerDN, "", nil
}

// ManagerPreviewItem 经理预览项（用户列表「经理」列展示用，不落库）。
type ManagerPreviewItem struct {
	UserID          string   `json:"userId"`
	ManagerID       string   `json:"managerId,omitempty"`
	ManagerName     string   `json:"managerName,omitempty"`     // 经理姓名（nickname）
	ManagerUsername string   `json:"managerUsername,omitempty"` // 经理账号（username）
	RuleName        string   `json:"ruleName,omitempty"`        // 命中的规则名
	Reason          string   `json:"reason,omitempty"`          // 未解析出经理时的原因
	Warnings        []string `json:"warnings,omitempty"`
}

// PreviewManagers 批量预览用户的经理（用户列表展示用）。
//
// 与 SyncManagersToAD 的区别：纯内存计算、不写 AD、不落库；
// 复用 LoadResolver（带进程内 TTL 缓存，规则/机构变更时主动失效），
// 当页 ≤100 用户的解析开销可忽略。
// 无启用规则 → 返回空列表（前端不显示经理列内容），不回退旧 leader 逻辑
// （预览语义 = "配置了规则之后的经理是谁"）。
func (s *ManagerRuleService) PreviewManagers(ctx context.Context, userIDs []string) ([]ManagerPreviewItem, error) {
	if len(userIDs) == 0 {
		return []ManagerPreviewItem{}, nil
	}
	resolver, err := s.LoadResolver(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ManagerPreviewItem, 0, len(userIDs))
	for _, uid := range userIDs {
		u, ok := resolver.users[uid]
		if !ok {
			continue
		}
		res := resolver.Resolve(&u)
		item := ManagerPreviewItem{
			UserID:   uid,
			RuleName: res.RuleName,
			Reason:   res.Reason,
			Warnings: res.Warnings,
		}
		if res.ManagerID != "" {
			item.ManagerID = res.ManagerID
			if m, ok := resolver.users[res.ManagerID]; ok {
				if m.Nickname != nil && *m.Nickname != "" {
					item.ManagerName = *m.Nickname
				}
				item.ManagerUsername = m.Username
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// lookupAdDn 查询用户的 ad_dn（空串表示没有 AD 绑定）。
func (s *ManagerRuleService) lookupAdDn(ctx context.Context, userID string) (string, error) {
	var row struct {
		AdDn *string
	}
	if err := s.db.WithContext(ctx).
		Table("sys_user").
		Select("ad_dn").
		Where("id = ?", userID).
		Scan(&row).Error; err != nil {
		return "", fmt.Errorf("查询用户 ad_dn 失败: %w", err)
	}
	if row.AdDn == nil {
		return "", nil
	}
	return *row.AdDn, nil
}

// ManagerResolver 规则引擎的一次性预加载解析上下文。
//
// 解析过程全部在内存完成，不产生额外 DB 查询。
type ManagerResolver struct {
	rules []models.ManagerRule
	orgs  []models.ManagerOrg

	orgByID   map[string]models.ManagerOrg
	orgByDept map[string]models.ManagerOrg

	depts map[string]models.Department
	users map[string]models.User

	postNames map[string]string
	deptNames map[string]string
	orgNames  map[string]string

	// deptLeaderUserIDs 所有担任部门负责人的用户 id
	deptLeaderUserIDs map[string]bool
	// postHolders post_id → 任职用户 id 列表（user id 升序）
	postHolders map[string][]string
	// userPosts user_id → 岗位 id 列表
	userPosts map[string][]string
}

// addPostHolder 记录一条岗位任职关系（去重）。
func (r *ManagerResolver) addPostHolder(postID, userID string) {
	for _, existing := range r.postHolders[postID] {
		if existing == userID {
			return
		}
	}
	r.postHolders[postID] = append(r.postHolders[postID], userID)
	r.userPosts[userID] = append(r.userPosts[userID], postID)
}

// sortPostHolders 把任职人列表按 user id 升序排序（保证「多人取第一个」的结果稳定）。
func (r *ManagerResolver) sortPostHolders() {
	for postID, holders := range r.postHolders {
		sort.Strings(holders)
		r.postHolders[postID] = holders
	}
	for userID, posts := range r.userPosts {
		sort.Strings(posts)
		r.userPosts[userID] = posts
	}
}

// Resolve 解析单个用户的经理。
func (r *ManagerResolver) Resolve(u *models.User) ManagerResolution {
	res := ManagerResolution{UserID: u.ID}

	org, hasOrg := r.OrgOf(u)
	if !hasOrg {
		res.Skipped = true
		res.Reason = "未命中任何机构定义（sys_manager_org），无法判定经理"
		return res
	}

	for _, rule := range r.rules {
		if !r.matchSubject(rule, u, org) {
			continue
		}
		if !r.matchScope(rule, u, org) {
			continue
		}
		res.RuleID = rule.ID
		res.RuleName = rule.RuleName

		managerID, warns := r.resolveManagerID(rule, u, org)
		res.Warnings = append(res.Warnings, warns...)
		if managerID == "" {
			// 该规则解析不出经理 → 顺延尝试下一条规则（first-match 不短路于解析失败）
			res.Reason = fmt.Sprintf("规则「%s」未解析出经理", rule.RuleName)
			continue
		}
		if managerID == u.ID {
			// 自指（如机构一把手被规则指回自己）→ 顺延尝试下一条规则
			res.Reason = fmt.Sprintf("规则「%s」解析出的经理为本人（自指），跳过", rule.RuleName)
			continue
		}
		res.ManagerID = managerID
		return res
	}

	res.Skipped = true
	if res.Reason == "" {
		res.Reason = "没有命中任何启用的经理同步规则"
	}
	return res
}

// OrgOf 用户所属机构：部门 ancestors 链上最近的、命中 sys_manager_org 的部门节点。
func (r *ManagerResolver) OrgOf(u *models.User) (models.ManagerOrg, bool) {
	if u == nil || u.DeptID == nil || *u.DeptID == "" {
		return models.ManagerOrg{}, false
	}
	if org, ok := r.orgByDept[*u.DeptID]; ok {
		return org, true
	}
	dept, ok := r.depts[*u.DeptID]
	if !ok {
		return models.ManagerOrg{}, false
	}
	// ancestors 为根→父序，反向遍历即父→根（由近到远）
	ancestors := splitAncestorIDs(dept.Ancestors, dept.ID)
	for i := len(ancestors) - 1; i >= 0; i-- {
		if org, ok := r.orgByDept[ancestors[i]]; ok {
			return org, true
		}
	}
	return models.ManagerOrg{}, false
}

// matchSubject 主体匹配。
func (r *ManagerResolver) matchSubject(rule models.ManagerRule, u *models.User, org models.ManagerOrg) bool {
	// 主体机构约束：规则限定了机构时，主体必须属于该机构
	if rule.SubjectOrgID != nil && *rule.SubjectOrgID != "" && org.ID != *rule.SubjectOrgID {
		return false
	}
	switch rule.SubjectType {
	case models.ManagerSubjectEmployee:
		return !r.IsDeptLeader(u.ID) && !r.isOrgHead(u, rule)
	case models.ManagerSubjectDeptLeader:
		return r.IsDeptLeader(u.ID)
	case models.ManagerSubjectOrgHead:
		return r.isOrgHead(u, rule)
	case models.ManagerSubjectPostHolder:
		if rule.SubjectPostID == nil || *rule.SubjectPostID == "" {
			return false
		}
		return r.HasPost(u.ID, *rule.SubjectPostID)
	default:
		return false
	}
}

// matchScope 范围匹配。
func (r *ManagerResolver) matchScope(rule models.ManagerRule, u *models.User, org models.ManagerOrg) bool {
	switch rule.ScopeType {
	case models.ManagerScopeAll:
		return true
	case models.ManagerScopeDeptTree:
		if rule.ScopeID == nil || *rule.ScopeID == "" {
			return false
		}
		return r.DeptInSubtree(*rule.ScopeID, u.DeptID)
	case models.ManagerScopeOrgTree:
		if rule.ScopeID == nil || *rule.ScopeID == "" {
			return false
		}
		return r.OrgInSubtree(*rule.ScopeID, org.ID)
	default:
		return false
	}
}

// resolveManagerID 经理取值（返回经理的 sys_user.id）。
func (r *ManagerResolver) resolveManagerID(rule models.ManagerRule, u *models.User, org models.ManagerOrg) (string, []string) {
	var warns []string
	switch rule.ManagerType {
	case models.ManagerSourceDeptLeader:
		dept, ok := r.depts[ptrValue(u.DeptID)]
		if !ok {
			return "", warns
		}
		return r.resolveDeptLeader(&dept), warns
	case models.ManagerSourceOrgLeader:
		// 机构负责人：取目标机构节点部门（org.DeptID）的 leader。
		// 目标机构遵循 resolveTargetOrg（自身/上级/指定），与 org_head/post_holder 语义一致；
		// 未配置目标机构时即自身机构，与旧行为兼容。
		targetOrgID, terr := r.resolveTargetOrg(rule, org)
		if terr != nil {
			warns = append(warns, terr.Error())
			return "", warns
		}
		targetOrg, ok := r.orgByID[targetOrgID]
		if !ok {
			warns = append(warns, fmt.Sprintf("目标机构 %s 不存在", targetOrgID))
			return "", warns
		}
		dept, ok := r.depts[targetOrg.DeptID]
		if !ok || dept.Leader == nil || *dept.Leader == "" {
			return "", warns
		}
		return *dept.Leader, warns
	case models.ManagerSourcePostHolder, models.ManagerSourceOrgHead:
		if rule.ManagerPostID == nil || *rule.ManagerPostID == "" {
			warns = append(warns, "规则未配置目标岗位")
			return "", warns
		}
		targetOrgID, err := r.resolveTargetOrg(rule, org)
		if err != nil {
			warns = append(warns, err.Error())
			return "", warns
		}
		return r.pickPostHolder(*rule.ManagerPostID, targetOrgID, &warns), warns
	default:
		return "", warns
	}
}

// resolveTargetOrg 按 manager_org_id 解析目标机构。
func (r *ManagerResolver) resolveTargetOrg(rule models.ManagerRule, org models.ManagerOrg) (string, error) {
	if rule.ManagerOrgID == nil || *rule.ManagerOrgID == "" {
		return org.ID, nil
	}
	if *rule.ManagerOrgID == models.ManagerOrgSpecialParent {
		if org.ParentOrgID == nil || *org.ParentOrgID == "" {
			return "", fmt.Errorf("机构「%s」没有上级机构，无法取上级机构的任职人", org.OrgName)
		}
		return *org.ParentOrgID, nil
	}
	return *rule.ManagerOrgID, nil
}

// pickPostHolder 在目标机构内挑选岗位任职人。
//
// 筛选：未软删、status=0 正常、且其所属机构 == 目标机构。
// 多人时：取 ad_dn 非空、user id 升序的第一个，并记 WARN。
func (r *ManagerResolver) pickPostHolder(postID, targetOrgID string, warns *[]string) string {
	holders := r.postHolders[postID]
	var withDN []models.User
	for _, uid := range holders {
		u, ok := r.users[uid]
		if !ok {
			continue
		}
		if u.Status != models.UserStatusEnabled {
			continue
		}
		holderOrg, ok := r.OrgOf(&u)
		if !ok || holderOrg.ID != targetOrgID {
			continue
		}
		if u.AdDn == nil || *u.AdDn == "" {
			continue
		}
		withDN = append(withDN, u)
	}
	if len(withDN) == 0 {
		return ""
	}
	if len(withDN) > 1 {
		postLabel := r.postNames[postID]
		if postLabel == "" {
			postLabel = postID
		}
		*warns = append(*warns, fmt.Sprintf("岗位「%s」在目标机构内存在 %d 名任职人，已按用户 ID 升序取第一个", postLabel, len(withDN)))
	}
	return withDN[0].ID
}

// resolveDeptLeader 部门负责人回溯：当前部门 → 祖先（由近到远），深度上限 maxLeaderDepth。
func (r *ManagerResolver) resolveDeptLeader(dept *models.Department) string {
	if dept == nil {
		return ""
	}
	if dept.Leader != nil && *dept.Leader != "" {
		return *dept.Leader
	}
	ancestors := splitAncestorIDs(dept.Ancestors, dept.ID)
	for i := len(ancestors) - 1; i >= 0; i-- {
		if len(ancestors)-1-i >= maxLeaderDepth {
			break
		}
		if anc, ok := r.depts[ancestors[i]]; ok && anc.Leader != nil && *anc.Leader != "" {
			return *anc.Leader
		}
	}
	return ""
}

// IsDeptLeader 用户是否是至少一个部门的负责人。
func (r *ManagerResolver) IsDeptLeader(userID string) bool {
	return r.deptLeaderUserIDs[userID]
}

// HasPost 用户是否任职于指定岗位。
func (r *ManagerResolver) HasPost(userID, postID string) bool {
	for _, p := range r.userPosts[userID] {
		if p == postID {
			return true
		}
	}
	return false
}

// isOrgHead 用户是否是机构一把手。
//
// 「一把手岗位」取规则的一把手岗位：优先 R.SubjectPostID，为空时取 R.ManagerPostID。
// 判定：用户任职于该岗位 **且** 其所属机构存在（能落到某个机构上才算一把手）。
func (r *ManagerResolver) isOrgHead(u *models.User, rule models.ManagerRule) bool {
	postID := ""
	if rule.SubjectPostID != nil && *rule.SubjectPostID != "" {
		postID = *rule.SubjectPostID
	} else if rule.ManagerPostID != nil && *rule.ManagerPostID != "" {
		postID = *rule.ManagerPostID
	}
	if postID == "" {
		return false
	}
	if !r.HasPost(u.ID, postID) {
		return false
	}
	_, ok := r.OrgOf(u)
	return ok
}

// DeptInSubtree 部门是否在 scopeID 的子树内（含自身）。
func (r *ManagerResolver) DeptInSubtree(scopeID string, deptID *string) bool {
	if deptID == nil || *deptID == "" {
		return false
	}
	if *deptID == scopeID {
		return true
	}
	dept, ok := r.depts[*deptID]
	if !ok {
		return false
	}
	for _, anc := range splitAncestorIDs(dept.Ancestors, dept.ID) {
		if anc == scopeID {
			return true
		}
	}
	return false
}

// OrgInSubtree 机构是否在 scopeOrgID 的子树内（含自身）。
func (r *ManagerResolver) OrgInSubtree(scopeOrgID, orgID string) bool {
	cur := orgID
	for i := 0; i < maxLeaderDepth && cur != ""; i++ {
		if cur == scopeOrgID {
			return true
		}
		org, ok := r.orgByID[cur]
		if !ok || org.ParentOrgID == nil || *org.ParentOrgID == "" {
			return false
		}
		cur = *org.ParentOrgID
	}
	return false
}

// PostName 岗位名称（列表翻译用）。
func (r *ManagerResolver) PostName(postID string) string {
	if postID == "" {
		return ""
	}
	return r.postNames[postID]
}

// DeptName 部门名称（列表翻译用）。
func (r *ManagerResolver) DeptName(deptID string) string {
	if deptID == "" {
		return ""
	}
	return r.deptNames[deptID]
}

// OrgName 机构名称（列表翻译用）。
func (r *ManagerResolver) OrgName(orgID string) string {
	if orgID == "" {
		return ""
	}
	return r.orgNames[orgID]
}

// ptrValue 取指针值（nil → 空串）。
func ptrValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// ============================ 规则 CRUD ============================

// ManagerRuleCreateRequest 新建/修改规则的请求体。
//
// Enabled 用 *bool：nil 表示「不修改」（更新场景），非 nil 表示显式设置。
type ManagerRuleCreateRequest struct {
	RuleName      string  `json:"ruleName" binding:"required"`
	Priority      int     `json:"priority"`
	Enabled       *bool   `json:"enabled"`
	SubjectType   string  `json:"subjectType" binding:"required"`
	SubjectPostID *string `json:"subjectPostId"`
	SubjectOrgID  *string `json:"subjectOrgId"`
	ManagerType   string  `json:"managerType" binding:"required"`
	ManagerPostID *string `json:"managerPostId"`
	ManagerOrgID  *string `json:"managerOrgId"`
	ScopeType     int     `json:"scopeType"`
	ScopeID       *string `json:"scopeId"`
}

// ManagerRuleListItem 规则列表项：UUID 已翻译为可读名称（前端直接展示）。
type ManagerRuleListItem struct {
	models.ManagerRule
	SubjectPostName string `json:"subjectPostName,omitempty"`
	SubjectOrgName  string `json:"subjectOrgName,omitempty"`
	ManagerPostName string `json:"managerPostName,omitempty"`
	ManagerOrgName  string `json:"managerOrgName,omitempty"`
	ScopeName       string `json:"scopeName,omitempty"`
}

// ManagerRulePage 规则分页结果。
type ManagerRulePage struct {
	List     []ManagerRuleListItem `json:"list"`
	Total    int64                 `json:"total"`
	Current  int                   `json:"current"`
	PageSize int                   `json:"pageSize"`
}

// 合法常量集合。
var (
	validManagerSubjectTypes = map[string]bool{
		models.ManagerSubjectEmployee:   true,
		models.ManagerSubjectDeptLeader: true,
		models.ManagerSubjectOrgHead:    true,
		models.ManagerSubjectPostHolder: true,
	}
	validManagerSourceTypes = map[string]bool{
		models.ManagerSourceDeptLeader: true,
		models.ManagerSourcePostHolder: true,
		models.ManagerSourceOrgLeader:  true,
		models.ManagerSourceOrgHead:    true,
	}
)

// validate 规则参数校验。
//
// 校验项：
//   - ruleName 非空
//   - subjectType / managerType 必须是合法常量
//   - manager_type = post_holder|org_head 时 managerPostId 必填
//   - scope_type != 0 时 scopeId 必填
//   - managerOrgId 允许 "parent" 或 UUID
func (r *ManagerRuleCreateRequest) validate() error {
	if strings.TrimSpace(r.RuleName) == "" {
		return fmt.Errorf("规则名称不能为空")
	}
	if !validManagerSubjectTypes[r.SubjectType] {
		return fmt.Errorf("主体类型非法：%s", r.SubjectType)
	}
	if !validManagerSourceTypes[r.ManagerType] {
		return fmt.Errorf("经理来源非法：%s", r.ManagerType)
	}
	if (r.ManagerType == models.ManagerSourcePostHolder || r.ManagerType == models.ManagerSourceOrgHead) &&
		(r.ManagerPostID == nil || strings.TrimSpace(*r.ManagerPostID) == "") {
		return fmt.Errorf("经理来源为岗位任职人时必须指定目标岗位")
	}
	switch r.ScopeType {
	case models.ManagerScopeAll:
	case models.ManagerScopeDeptTree, models.ManagerScopeOrgTree:
		if r.ScopeID == nil || strings.TrimSpace(*r.ScopeID) == "" {
			return fmt.Errorf("作用范围非「全部」时必须指定范围节点")
		}
	default:
		return fmt.Errorf("作用范围类型非法：%d", r.ScopeType)
	}
	if r.ManagerOrgID != nil && *r.ManagerOrgID != "" &&
		*r.ManagerOrgID != models.ManagerOrgSpecialParent && !isUUIDLike(*r.ManagerOrgID) {
		return fmt.Errorf("目标机构非法：只允许 %q 或机构 ID", models.ManagerOrgSpecialParent)
	}
	if r.SubjectOrgID != nil && *r.SubjectOrgID != "" && !isUUIDLike(*r.SubjectOrgID) {
		return fmt.Errorf("主体机构非法：必须是机构 ID")
	}
	return nil
}

// isUUIDLike 粗判是否为 UUID 形态（避免把脏值写进 uuid 列导致 PG 22P02）。
func isUUIDLike(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

// ListRules 查询规则列表（带名称翻译 + 分页）。
func (s *ManagerRuleService) ListRules(ctx context.Context, current, pageSize int, keyword string, enabled *bool) (*ManagerRulePage, error) {
	if current < 1 {
		current = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.WithContext(ctx).Model(&models.ManagerRule{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("rule_name LIKE ?", like)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计经理同步规则失败: %w", err)
	}

	var rules []models.ManagerRule
	if err := query.Order("priority ASC, created_at ASC").
		Offset((current - 1) * pageSize).
		Limit(pageSize).
		Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("查询经理同步规则失败: %w", err)
	}

	resolver, err := s.LoadResolver(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]ManagerRuleListItem, 0, len(rules))
	for _, rule := range rules {
		item := ManagerRuleListItem{ManagerRule: rule}
		if rule.SubjectPostID != nil {
			item.SubjectPostName = resolver.PostName(*rule.SubjectPostID)
		}
		if rule.SubjectOrgID != nil {
			item.SubjectOrgName = resolver.OrgName(*rule.SubjectOrgID)
		}
		if rule.ManagerPostID != nil {
			item.ManagerPostName = resolver.PostName(*rule.ManagerPostID)
		}
		if rule.ManagerOrgID != nil && *rule.ManagerOrgID != "" {
			if *rule.ManagerOrgID == models.ManagerOrgSpecialParent {
				item.ManagerOrgName = "上级机构"
			} else {
				item.ManagerOrgName = resolver.OrgName(*rule.ManagerOrgID)
			}
		}
		if rule.ScopeID != nil && *rule.ScopeID != "" {
			switch rule.ScopeType {
			case models.ManagerScopeDeptTree:
				item.ScopeName = resolver.DeptName(*rule.ScopeID)
			case models.ManagerScopeOrgTree:
				item.ScopeName = resolver.OrgName(*rule.ScopeID)
			}
		}
		items = append(items, item)
	}

	return &ManagerRulePage{List: items, Total: total, Current: current, PageSize: pageSize}, nil
}

// CreateRule 新建规则。
func (s *ManagerRuleService) CreateRule(ctx context.Context, req *ManagerRuleCreateRequest) (*models.ManagerRule, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}
	if err := req.validate(); err != nil {
		return nil, err
	}
	rule := models.ManagerRule{
		RuleName:      strings.TrimSpace(req.RuleName),
		Priority:      req.Priority,
		Enabled:       true,
		SubjectType:   req.SubjectType,
		SubjectPostID: req.SubjectPostID,
		SubjectOrgID:  req.SubjectOrgID,
		ManagerType:   req.ManagerType,
		ManagerPostID: req.ManagerPostID,
		ManagerOrgID:  req.ManagerOrgID,
		ScopeType:     req.ScopeType,
		ScopeID:       req.ScopeID,
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if err := s.db.WithContext(ctx).Create(&rule).Error; err != nil {
		return nil, fmt.Errorf("创建经理同步规则失败: %w", err)
	}
	invalidateResolverCache()
	enqueueAllManagersRecheck(ctx, s.db)
	return &rule, nil
}

// GetRule 查询单条规则。
func (s *ManagerRuleService) GetRule(ctx context.Context, id string) (*models.ManagerRule, error) {
	var rule models.ManagerRule
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("经理同步规则不存在")
		}
		return nil, fmt.Errorf("查询经理同步规则失败: %w", err)
	}
	return &rule, nil
}

// UpdateRule 修改规则。
func (s *ManagerRuleService) UpdateRule(ctx context.Context, id string, req *ManagerRuleCreateRequest) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}
	if err := req.validate(); err != nil {
		return err
	}
	var rule models.ManagerRule
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&rule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("经理同步规则不存在")
		}
		return fmt.Errorf("查询经理同步规则失败: %w", err)
	}
	rule.RuleName = strings.TrimSpace(req.RuleName)
	rule.Priority = req.Priority
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	rule.SubjectType = req.SubjectType
	rule.SubjectPostID = req.SubjectPostID
	rule.SubjectOrgID = req.SubjectOrgID
	rule.ManagerType = req.ManagerType
	rule.ManagerPostID = req.ManagerPostID
	rule.ManagerOrgID = req.ManagerOrgID
	rule.ScopeType = req.ScopeType
	rule.ScopeID = req.ScopeID
	if err := s.db.WithContext(ctx).Save(&rule).Error; err != nil {
		return fmt.Errorf("更新经理同步规则失败: %w", err)
	}
	invalidateResolverCache()
	enqueueAllManagersRecheck(ctx, s.db)
	return nil
}

// DeleteRule 删除规则。
func (s *ManagerRuleService) DeleteRule(ctx context.Context, id string) error {
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&models.ManagerRule{}).Error; err != nil {
		return fmt.Errorf("删除经理同步规则失败: %w", err)
	}
	invalidateResolverCache()
	enqueueAllManagersRecheck(ctx, s.db)
	return nil
}

// ============================ 机构 CRUD ============================

// ManagerOrgCreateRequest 新建/修改机构的请求体。
type ManagerOrgCreateRequest struct {
	OrgName     string  `json:"orgName" binding:"required"`
	DeptID      string  `json:"deptId" binding:"required"`
	ParentOrgID *string `json:"parentOrgId"`
	OrgType     int     `json:"orgType"`
	Enabled     *bool   `json:"enabled"`
}

// ManagerOrgListItem 机构列表项（部门名/上级机构名已翻译）。
type ManagerOrgListItem struct {
	models.ManagerOrg
	DeptName      string `json:"deptName,omitempty"`
	ParentOrgName string `json:"parentOrgName,omitempty"`
}

// ManagerOrgPage 机构分页结果。
type ManagerOrgPage struct {
	List     []ManagerOrgListItem `json:"list"`
	Total    int64                `json:"total"`
	Current  int                  `json:"current"`
	PageSize int                  `json:"pageSize"`
}

// ManagerOrgService 机构维护服务。
type ManagerOrgService struct {
	db *gorm.DB
}

// NewManagerOrgService 创建机构维护服务。
func NewManagerOrgService(db *gorm.DB) *ManagerOrgService {
	return &ManagerOrgService{db: db}
}

// validate 机构参数校验。
func (r *ManagerOrgCreateRequest) validate() error {
	if strings.TrimSpace(r.OrgName) == "" {
		return fmt.Errorf("机构名称不能为空")
	}
	if strings.TrimSpace(r.DeptID) == "" {
		return fmt.Errorf("机构必须绑定一个部门")
	}
	if r.OrgType < 0 || r.OrgType > 2 {
		return fmt.Errorf("机构类型非法：0=分公司 1=中支 2=支公司")
	}
	return nil
}

// ListOrgs 查询机构列表（带部门名翻译 + 分页）。
func (s *ManagerOrgService) ListOrgs(ctx context.Context, current, pageSize int, keyword string, enabled *bool) (*ManagerOrgPage, error) {
	if current < 1 {
		current = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.WithContext(ctx).Model(&models.ManagerOrg{})
	if keyword != "" {
		query = query.Where("org_name LIKE ?", "%"+keyword+"%")
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("统计机构失败: %w", err)
	}

	var orgs []models.ManagerOrg
	if err := query.Order("created_at ASC").
		Offset((current - 1) * pageSize).
		Limit(pageSize).
		Find(&orgs).Error; err != nil {
		return nil, fmt.Errorf("查询机构失败: %w", err)
	}

	deptNames := make(map[string]string)
	var depts []models.Department
	if err := s.db.WithContext(ctx).Table("sys_dept").Select("id, dept_name").Find(&depts).Error; err == nil {
		for _, d := range depts {
			deptNames[d.ID] = d.DeptName
		}
	}
	orgNames := make(map[string]string)
	var allOrgs []models.ManagerOrg
	if err := s.db.WithContext(ctx).Model(&models.ManagerOrg{}).Select("id, org_name").Find(&allOrgs).Error; err == nil {
		for _, o := range allOrgs {
			orgNames[o.ID] = o.OrgName
		}
	}

	items := make([]ManagerOrgListItem, 0, len(orgs))
	for _, org := range orgs {
		item := ManagerOrgListItem{ManagerOrg: org}
		item.DeptName = deptNames[org.DeptID]
		if org.ParentOrgID != nil && *org.ParentOrgID != "" {
			item.ParentOrgName = orgNames[*org.ParentOrgID]
		}
		items = append(items, item)
	}

	return &ManagerOrgPage{List: items, Total: total, Current: current, PageSize: pageSize}, nil
}

// CreateOrg 新建机构。
func (s *ManagerOrgService) CreateOrg(ctx context.Context, req *ManagerOrgCreateRequest) (*models.ManagerOrg, error) {
	if req == nil {
		return nil, fmt.Errorf("请求参数不能为空")
	}
	if err := req.validate(); err != nil {
		return nil, err
	}
	if err := s.checkDeptAvailable(ctx, req.DeptID, ""); err != nil {
		return nil, err
	}
	if err := s.checkParentOrg(ctx, "", req.ParentOrgID); err != nil {
		return nil, err
	}
	org := models.ManagerOrg{
		OrgName:     strings.TrimSpace(req.OrgName),
		DeptID:      strings.TrimSpace(req.DeptID),
		ParentOrgID: req.ParentOrgID,
		OrgType:     req.OrgType,
		Enabled:     true,
	}
	if req.Enabled != nil {
		org.Enabled = *req.Enabled
	}
	if err := s.db.WithContext(ctx).Create(&org).Error; err != nil {
		return nil, fmt.Errorf("创建机构失败: %w", err)
	}
	invalidateResolverCache()
	enqueueAllManagersRecheck(ctx, s.db)
	return &org, nil
}

// GetOrg 查询单条机构。
func (s *ManagerOrgService) GetOrg(ctx context.Context, id string) (*models.ManagerOrg, error) {
	var org models.ManagerOrg
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("机构不存在")
		}
		return nil, fmt.Errorf("查询机构失败: %w", err)
	}
	return &org, nil
}

// UpdateOrg 修改机构。
func (s *ManagerOrgService) UpdateOrg(ctx context.Context, id string, req *ManagerOrgCreateRequest) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}
	if err := req.validate(); err != nil {
		return err
	}
	var org models.ManagerOrg
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("机构不存在")
		}
		return fmt.Errorf("查询机构失败: %w", err)
	}
	if err := s.checkDeptAvailable(ctx, req.DeptID, id); err != nil {
		return err
	}
	if err := s.checkParentOrg(ctx, id, req.ParentOrgID); err != nil {
		return err
	}
	org.OrgName = strings.TrimSpace(req.OrgName)
	org.DeptID = strings.TrimSpace(req.DeptID)
	org.ParentOrgID = req.ParentOrgID
	org.OrgType = req.OrgType
	if req.Enabled != nil {
		org.Enabled = *req.Enabled
	}
	if err := s.db.WithContext(ctx).Save(&org).Error; err != nil {
		return fmt.Errorf("更新机构失败: %w", err)
	}
	invalidateResolverCache()
	enqueueAllManagersRecheck(ctx, s.db)
	return nil
}

// DeleteOrg 删除机构（被规则引用时拒绝）。
func (s *ManagerOrgService) DeleteOrg(ctx context.Context, id string) error {
	var org models.ManagerOrg
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&org).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("机构不存在")
		}
		return fmt.Errorf("查询机构失败: %w", err)
	}
	refCount, err := s.countRuleReferences(ctx, id)
	if err != nil {
		return err
	}
	if refCount > 0 {
		return fmt.Errorf("该机构被 %d 条规则引用，请先调整规则", refCount)
	}
	// 子机构的上级指针需要清理，避免出现指向已删机构的孤儿
	if err := s.db.WithContext(ctx).Model(&models.ManagerOrg{}).
		Where("parent_org_id = ?", id).
		Updates(map[string]interface{}{"parent_org_id": nil}).Error; err != nil {
		return fmt.Errorf("清理子机构上级引用失败: %w", err)
	}
	// 机构映射是配置数据（审计靠 operlog），删除走物理删除：
	// 软删行会在 dept_id 唯一索引上占坑（idx_sys_manager_org_dept_id），
	// 导致同部门无法重建机构（SQLSTATE 23505）。migration_223 已把索引改为
	// WHERE deleted_at IS NULL 的部分唯一索引作为双保险，此处 Unscoped 双管齐下。
	if err := s.db.WithContext(ctx).Unscoped().Where("id = ?", id).Delete(&models.ManagerOrg{}).Error; err != nil {
		return fmt.Errorf("删除机构失败: %w", err)
	}
	invalidateResolverCache()
	enqueueAllManagersRecheck(ctx, s.db)
	return nil
}

// checkDeptAvailable 校验部门存在且未被其它机构占用。
func (s *ManagerOrgService) checkDeptAvailable(ctx context.Context, deptID, excludeOrgID string) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Department{}).
		Where("id = ?", deptID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("校验机构部门失败: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("机构绑定的部门不存在")
	}
	query := s.db.WithContext(ctx).Model(&models.ManagerOrg{}).Where("dept_id = ?", deptID)
	if excludeOrgID != "" {
		query = query.Where("id <> ?", excludeOrgID)
	}
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("校验部门占用失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该部门已被其它机构占用")
	}
	return nil
}

// checkParentOrg 校验上级机构存在且不成环。
func (s *ManagerOrgService) checkParentOrg(ctx context.Context, orgID string, parentOrgID *string) error {
	if parentOrgID == nil || *parentOrgID == "" {
		return nil
	}
	if orgID != "" && *parentOrgID == orgID {
		return fmt.Errorf("上级机构不能是自己")
	}
	var parent models.ManagerOrg
	if err := s.db.WithContext(ctx).Where("id = ?", *parentOrgID).First(&parent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("上级机构不存在")
		}
		return fmt.Errorf("查询上级机构失败: %w", err)
	}
	if orgID == "" {
		return nil
	}
	// 成环检测：从 parent 沿 parent_org_id 向上走，若能走回自己即成环
	cur := parent
	for i := 0; i < maxLeaderDepth; i++ {
		if cur.ID == orgID {
			return fmt.Errorf("上级机构不能是自己的子孙机构")
		}
		if cur.ParentOrgID == nil || *cur.ParentOrgID == "" {
			return nil
		}
		var next models.ManagerOrg
		if err := s.db.WithContext(ctx).Where("id = ?", *cur.ParentOrgID).First(&next).Error; err != nil {
			return nil // 上级链断裂：不再判定成环
		}
		cur = next
	}
	return nil
}

// countRuleReferences 统计引用该机构的规则条数。
func (s *ManagerOrgService) countRuleReferences(ctx context.Context, orgID string) (int64, error) {
	if !s.db.Migrator().HasTable(&models.ManagerRule{}) {
		return 0, nil
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.ManagerRule{}).
		Where("subject_org_id = ?", orgID).
		Or("manager_org_id = ?", orgID).
		Or("scope_type = ? AND scope_id = ?", models.ManagerScopeOrgTree, orgID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计机构规则引用失败: %w", err)
	}
	return count, nil
}

// GenerateFromDeptResult 从部门架构自动生成机构的结果摘要。
type GenerateFromDeptResult struct {
	TotalCandidates int      `json:"totalCandidates"` // is_external_org=1 的部门数
	Created         int      `json:"created"`         // 新建机构数
	SkippedExisting int      `json:"skippedExisting"` // 已存在（含手工创建）跳过数
	OrphanCount     int      `json:"orphanCount"`     // 祖先链上找不到机构父节点的顶层机构数
	Warnings        []string `json:"warnings,omitempty"`
}

// GenerateOrgsFromDept 从部门架构自动生成机构映射（「从部门架构生成」按钮）。
//
// 规则（2026-10 与用户确认的三层架构）：
//  1. 候选 = sys_dept.is_external_org = 1 且未软删的部门（有职场 ≈ 机构）；
//  2. 已存在的机构（按 dept_id 匹配）不覆盖 —— 保护手工修改；
//  3. 新机构的 parent_org_id = 部门 ancestors 链上最近的「同样是机构」的祖先
//     （优先用本表已存在机构，其次本次同批生成的候选）；链上没有 → 顶层机构（nil）；
//  4. org_type 按祖先机构链深度默认（系统三层架构：分公司 → 中支 → 支公司）：
//     深度 0 → 0=分公司，深度 1 → 1=中支，深度 >=2 → 2=支公司；
//     特例（如中支同时含分公司本部）由页面手工调整。
//
// 幂等：重复点击只补增量，不破坏已有数据。
func (s *ManagerOrgService) GenerateOrgsFromDept(ctx context.Context) (*GenerateFromDeptResult, error) {
	if !s.db.Migrator().HasTable(&models.Department{}) {
		return nil, fmt.Errorf("部门表不存在")
	}

	// 1) 取全部机构候选部门（含 ancestors，供推导父机构）。
	type deptRow struct {
		ID        string
		DeptName  string
		ParentID  *string
		Ancestors string
	}
	var depts []deptRow
	if err := s.db.WithContext(ctx).Table("sys_dept").
		Select("id, dept_name, parent_id, ancestors").
		Where("is_external_org = ? AND deleted_at IS NULL", 1).
		Order("length(ancestors) ASC"). // 祖先短的先建，保证父机构先生成
		Find(&depts).Error; err != nil {
		return nil, fmt.Errorf("查询机构候选部门失败: %w", err)
	}

	result := &GenerateFromDeptResult{TotalCandidates: len(depts)}
	if len(depts) == 0 {
		return result, nil
	}

	// 2) 已存在的机构按 dept_id 建索引（不覆盖）。
	var existing []models.ManagerOrg
	if err := s.db.WithContext(ctx).Find(&existing).Error; err != nil {
		return nil, fmt.Errorf("查询已有机构失败: %w", err)
	}
	orgByDept := make(map[string]struct{}, len(existing))
	for _, o := range existing {
		orgByDept[o.DeptID] = struct{}{}
	}

	// deptID → 是否机构（已存在 ∪ 本次候选），用于祖先链判定。
	isOrgDept := func(deptID string) bool {
		if _, ok := orgByDept[deptID]; ok {
			return true
		}
		for _, d := range depts {
			if d.ID == deptID {
				return true
			}
		}
		return false
	}

	// deptID → ManagerOrg.ID（已存在 ∪ 本次新建），供父机构引用。
	orgIDByDept := make(map[string]string, len(existing))
	for _, o := range existing {
		orgIDByDept[o.DeptID] = o.ID
	}
	deptByID := make(map[string]deptRow, len(depts))
	for _, d := range depts {
		deptByID[d.ID] = d
	}

	for _, d := range depts {
		if _, done := orgByDept[d.ID]; done {
			result.SkippedExisting++
			continue
		}

		// 3) 沿 ancestors（逗号分隔，由根到父）向上找最近的机构祖先。
		var parentOrgID *string
		ancestorOrgDepth := 0
		ancestors := splitDeptAncestors(d.Ancestors)
		for _, anc := range ancestors {
			if anc == "" || anc == d.ID || !isOrgDept(anc) {
				continue
			}
			ancestorOrgDepth++
			if parentOrgID == nil {
				if oid, ok := orgIDByDept[anc]; ok {
					pid := oid
					parentOrgID = &pid
				}
				// 祖先是机构但机构行还没建（同为候选且祖先序在前，正常已建；
				// 极端乱序兜底：留空由下一次生成补），记 WARN。
			}
		}
		if parentOrgID == nil && ancestorOrgDepth > 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("部门 %s 存在机构祖先但父机构行尚未就绪，可再次点击生成补齐", d.DeptName))
			result.OrphanCount++
		}

		org := models.ManagerOrg{
			OrgName:     d.DeptName,
			DeptID:      d.ID,
			ParentOrgID: parentOrgID,
			OrgType:     orgTypeByOrgDepth(ancestorOrgDepth),
			Enabled:     true,
		}
		if err := s.db.WithContext(ctx).Create(&org).Error; err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("创建机构 %s 失败: %v", d.DeptName, err))
			continue
		}
		orgIDByDept[d.ID] = org.ID
		orgByDept[d.ID] = struct{}{}
		result.Created++
	}
	if result.Created > 0 {
		invalidateResolverCache()
		enqueueAllManagersRecheck(ctx, s.db)
	}
	return result, nil
}

// orgTypeByOrgDepth 按「机构祖先链深度」推导机构层级（系统三层架构：分公司 → 中支 → 支公司）。
//
// 深度 0（顶层机构）→ 0=分公司；深度 1 → 1=中支；深度 >=2 → 2=支公司。
//
// 只作为自动生成时的默认值，特例（如中支同时含分公司本部）由页面手工调整。
func orgTypeByOrgDepth(orgAncestorDepth int) int {
	switch {
	case orgAncestorDepth <= 0:
		return 0 // 分公司
	case orgAncestorDepth == 1:
		return 1 // 中支
	default:
		return 2 // 支公司
	}
}

// splitDeptAncestors 按逗号切分 sys_dept.ancestors（形如 "0,id1,id2"）。
func splitDeptAncestors(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && p != "0" {
			out = append(out, p)
		}
	}
	return out
}
