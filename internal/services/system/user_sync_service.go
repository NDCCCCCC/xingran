package system

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/addomain"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// ErrADIdentityConflict AD 身份冲突（classifyADUser 规则 2b/3b）：存量行已绑定其它
// AD 身份或其它域的配置。策略性跳过，绝不 INSERT / 改写，调用方按 WARN 记录。
var ErrADIdentityConflict = errors.New("AD 身份冲突，拒绝同步")

// ADUserInfoForSync AD用户信息（用于同步，由认证器提供）
type ADUserInfoForSync struct {
	UserDN      string // 用户DN
	OuDn        string // OU DN
	Username    string // sAMAccountName
	DisplayName string // 显示名称
	Email       string // 邮箱
	Phone       string // 电话
	Mobile      string // 手机
	Title       string // 职位
	Department  string // 部门
	ConfigID    string // 来源 AD 配置 ID（多域隔离用；登录路径由认证器经 context 传入，批量路径由 handler 直接填）
}

// ============================================================================
// AD → sys_user 字段同步策略（单一事实源）
// ============================================================================
//
// 四个写入函数（createNewUser / updateExistingUser / bindLocalUserToAD /
// applyRestoreSoftDeleted）必须共用本策略，禁止各自散落"改哪些列"的判断。
// 策略表是文档也是实现约束：新增字段时先在此登记，再改统一的 helper。
//
// adPolicyADAuthoritative —— AD 权威覆盖：AD 有值就写，AD 为空也写（清空本地旧值）。
//   适用：email / phone / mobile / title / ad_dn / ad_ou_dn / ad_config_id
//   理由：联系与任职属性以域控为准；ad_dn/ad_ou_dn 是身份定位的唯一权威来源。
// adPolicyFillIfLocalEmpty —— 仅当本地为空时用 AD 值填充。
//   适用：nickname
//   理由：本地昵称常是中文姓名，AD displayName 多为英文/拼音，覆盖会丢中文名。
// adPolicyNeverOverwrite —— 永不覆盖（本地语义字段）。
//   适用：password / employee_no / dept_name / 角色关联
//   理由：这些字段承载本地管理语义，AD 侧不存在对应权威来源。
//
// 例外：dept_id 同样是"AD 权威"，但受 GUARD-01 保护 —— 本地正在往 AD 推送的
// 变更（sys_ad_push_queue 里的 pending/failed update_user 行）不能被 AD 回写覆盖，
// 因此 dept_id 不在本策略表内单独成列，而是走 shouldSkipDeptUpdate 守卫。
type adSyncPolicyType int

const (
	adPolicyADAuthoritative adSyncPolicyType = iota // AD 权威覆盖（含空值覆盖清空）
	adPolicyFillIfLocalEmpty                         // 仅本地为空时填充
	adPolicyNeverOverwrite                           // 永不覆盖
)

// adFieldPolicy 单字段同步策略
type adFieldPolicy struct {
	Column string          // 数据库列名
	Policy adSyncPolicyType // 覆盖策略
	Reason string          // 策略理由（写进代码即文档）
}

// adFieldPolicies AD → sys_user 字段同步策略表。
// 四个写入函数共用；dept_id 见上方说明（受 GUARD-01 保护，故不在表内）。
var adFieldPolicies = []adFieldPolicy{
	{Column: "email", Policy: adPolicyADAuthoritative, Reason: "联系属性以域控为准"},
	{Column: "phone", Policy: adPolicyADAuthoritative, Reason: "联系属性以域控为准"},
	{Column: "mobile", Policy: adPolicyADAuthoritative, Reason: "联系属性以域控为准（AD mobile）"},
	{Column: "title", Policy: adPolicyADAuthoritative, Reason: "任职属性以域控为准（AD title）"},
	{Column: "ad_dn", Policy: adPolicyADAuthoritative, Reason: "身份定位（AD DN 全局唯一，跨域判据）"},
	{Column: "ad_ou_dn", Policy: adPolicyADAuthoritative, Reason: "身份定位（用户 OU 路径，支持移动场景）"},
	{Column: "ad_config_id", Policy: adPolicyADAuthoritative, Reason: "多域隔离：记录该用户归属的 AD 配置"},
	{Column: "nickname", Policy: adPolicyFillIfLocalEmpty, Reason: "本地昵称常是中文姓名，AD displayName 多为英文/拼音，覆盖会丢中文名"},
	{Column: "password", Policy: adPolicyNeverOverwrite, Reason: "本地语义；AD 账号本地密码为随机不可知值"},
	{Column: "employee_no", Policy: adPolicyNeverOverwrite, Reason: "本地语义（工号由本地维护）"},
	{Column: "dept_name", Policy: adPolicyNeverOverwrite, Reason: "本地语义（部门名由 sys_dept 维护）"},
}

// adAuthoritativeUpdates 返回「AD 权威覆盖」字段的更新集合。
//
// 语义：AD 有值就写，AD 为空**也写**（覆盖清空本地旧值）——联系与任职属性以域控为准。
// 值一律以指针传入，避免 GORM 把空串当作"零值"跳过。
func adAuthoritativeUpdates(adUser *ADUserInfoForSync) map[string]interface{} {
	email := adUser.Email
	phone := adUser.Phone
	mobile := adUser.Mobile
	title := adUser.Title
	return map[string]interface{}{
		"email":  &email,
		"phone":  &phone,
		"mobile": &mobile,
		"title":  &title,
	}
}

// adIdentityUpdates 返回「身份定位」字段的更新集合（ad_dn / ad_ou_dn / ad_config_id）。
//
// ad_config_id 只在已知来源配置 ID 时写入（ConfigID 为空表示未知，保留原值，
// 避免把已有归属清成 NULL 造成多域隔离失效）。
func adIdentityUpdates(adUser *ADUserInfoForSync) map[string]interface{} {
	adDN := adUser.UserDN
	ouDN := adUser.OuDn
	updates := map[string]interface{}{
		"ad_dn":    &adDN,
		"ad_ou_dn": &ouDN,
	}
	if adUser.ConfigID != "" {
		configID := adUser.ConfigID
		updates["ad_config_id"] = &configID
	}
	return updates
}

// adNicknameUpdate 在本地昵称为空时回填 AD displayName（尊重本地已有昵称）。
// 返回 nil 表示本次不回填。
func adNicknameUpdate(user *models.User, adUser *ADUserInfoForSync) interface{} {
	if user.Nickname != nil && *user.Nickname != "" {
		return nil
	}
	displayName := adUser.DisplayName
	return &displayName
}

// mergeADUpdates 合并多个更新集合（后者覆盖前者的同名列）。
func mergeADUpdates(maps ...map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

// generateRandomPassword 生成密码学随机的初始密码。
//
// AD 来源账号的本地密码必须是"不可知"的：写死 "123456" 会让任何拿到 AD 密码的人
// 都能直接登录本地账号（本地认证对 auth_source='ad' 的账号直接失败，两边配套生效）。
// 随机值不写日志、不返回给调用方，只用于哈希。
func generateRandomPassword() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成随机密码失败: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// UserSyncService 用户同步服务
// 负责将AD用户信息同步到sys_user表，并自动解析部门
type UserSyncService struct {
	db         *gorm.DB
	pwdManager PasswordManager
	ouMapper   *addomain.DeptOUmapper // OU映射解析器
	cache      CacheProvider          // 缓存提供者（可选，L-02 用于角色分配后失效 user-scoped 菜单缓存）
}

// UserSyncOption UserSyncService 可选注入项（functional options，避免构造函数签名连锁改动）
type UserSyncOption func(*UserSyncService)

// WithCacheProvider 注入缓存提供者（可选）。
// 注入后，写 sys_user_role 成功路径会失效 user-scoped 菜单缓存（menu:user:*）。
// 未注入或传 nil 时跳过失效逻辑，保持旧行为（测试/无缓存场景兼容）。
func WithCacheProvider(cache CacheProvider) UserSyncOption {
	return func(s *UserSyncService) {
		s.cache = cache
	}
}

// NewUserSyncService 创建用户同步服务
func NewUserSyncService(db *gorm.DB, pwdManager PasswordManager, ouMapper *addomain.DeptOUmapper, opts ...UserSyncOption) *UserSyncService {
	s := &UserSyncService{
		db:         db,
		pwdManager: pwdManager,
		ouMapper:   ouMapper,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// invalidateUserMenuCache 失效 user-scoped 菜单缓存（L-02）。
// sys_user_role 写入会改变用户可见菜单，需清 menu:user:* 前缀；
// 未注入 CacheProvider 时跳过（nil-safe）。批量同步只需在整体成功后调用一次，
// 不放入逐用户循环（InvalidateUserMenuCacheByProvider 清全部 menu:user:* 前缀）。
func (s *UserSyncService) invalidateUserMenuCache(ctx context.Context) {
	if s.cache == nil {
		return
	}
	InvalidateUserMenuCacheByProvider(ctx, s.cache)
}

// SyncUserFromAD 从AD同步单个用户到sys_user表（首次AD登录路径使用）。
// 按 4 类分类处理，返回同步后的用户及其分类（供调用方决定部门处理等后续动作）。
func (s *UserSyncService) SyncUserFromAD(ctx context.Context, adUser *ADUserInfoForSync, defaultRoleID string) (*models.User, userCategory, error) {
	var user models.User
	var category userCategory

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		idx, err := s.indexExistingUsers(tx, []string{adUser.Username}, []string{adUser.UserDN})
		if err != nil {
			return err
		}
		category, matched := classifyADUser(adUser.Username, adUser.UserDN, adUser.ConfigID,
			idx.byAdDN, idx.byADUsername, idx.byLocalName, idx.softByName)
		if matched != nil {
			user = *matched
		}
		switch category {
		case catActiveAD:
			return s.updateExistingUser(tx, &user, adUser)
		case catLocalSameName:
			return s.bindLocalUserToAD(tx, &user, adUser)
		case catSoftDeleted:
			return s.restoreSoftDeletedUser(tx, &user, adUser, defaultRoleID)
		case catADConflict:
			// 冲突是策略性跳过不是故障：绝不 INSERT（会撞 username 唯一索引回滚整个事务），
			// 也绝不改写冲突行的 ad_username/ad_dn。ErrSoftDeleteIdentityMismatch 同级 WARN。
			return fmt.Errorf("%w: username=%s", ErrADIdentityConflict, adUser.Username)
		default: // catNew
			return s.createNewUser(tx, &user, adUser, defaultRoleID)
		}
	})

	if err != nil {
		if errors.Is(err, ErrADIdentityConflict) {
			applogger.Warnf("[SECURITY] AD 身份冲突，跳过同步: %v", err)
			return nil, catADConflict, err
		}
		return nil, catNew, fmt.Errorf("同步AD用户失败: %w", err)
	}

	// L-02: 新建/恢复路径在事务内已写 sys_user_role（分配默认角色），
	// 失效 user-scoped 菜单缓存；其余分类未写角色关联，无需失效。
	if defaultRoleID != "" && (category == catNew || category == catSoftDeleted) {
		s.invalidateUserMenuCache(ctx)
	}

	return &user, category, nil
}

// userCategory 用户同步分类
//
// 判定规则（连续编号，classifyADUser 按此顺序判定）：
//
//	规则 1  ad_dn 命中                → catActiveAD      更新已有 AD 绑定
//	规则 2  ad_username 命中          → catActiveAD / catADConflict
//	规则 3  username 命中（活跃行）    → catLocalSameName / catADConflict
//	规则 4  username 命中（软删行）    → catSoftDeleted   恢复（仍走身份校验）
//	规则 5  无任何命中                → catNew
//
// ⚠ 数值不可重排：catActiveAD=0 / catLocalSameName=1 / catSoftDeleted=2 / catNew=3
// 已被既有测试锁定；新增分类一律追加在末尾（catADConflict=4）。
type userCategory int

const (
	catActiveAD      userCategory = iota // 规则 1 命中：活跃 AD 绑定用户
	catLocalSameName                     // 规则 3a 命中：活跃 local 同名用户（绑定到 AD）
	catSoftDeleted                       // 规则 4 命中：软删除同名用户（恢复）
	catNew                               // 规则 5 命中：全新用户
	catADConflict                        // 规则 2b/3b 命中：AD 身份冲突（跳过，绝不 INSERT）
)

// classifyADUser 按规则 1~5 判定 AD 用户的同步方式（纯函数，不依赖 DB）。
//
// 参数：
//   - adUsername   AD 侧 sAMAccountName
//   - adDN         AD 侧用户 DN（原始格式，内部归一化后比较）
//   - adConfigID   当前 AD 配置 ID（多域隔离用；空表示未知）
//   - byAdDN       活跃行索引：归一化 ad_dn → 行
//   - byADUsername 活跃行索引：ad_username → 行
//   - byLocalName  活跃行索引：username → 行（**包含 ad_username 非空的行**）
//   - softByName   软删行索引：username → 行
//
// 返回命中的现有用户；catNew 时返回 nil，catADConflict 时返回冲突行（供调用方写日志）。
func classifyADUser(
	adUsername, adDN, adConfigID string,
	byAdDN, byADUsername, byLocalName, softByName map[string]*models.User,
) (userCategory, *models.User) {
	// 规则 1：ad_dn 命中 —— 同一 DN 即同一 AD 身份，最可靠的跨域判据。
	if u, ok := byAdDN[normalizeDNKey(adDN)]; ok {
		return catActiveAD, u
	}

	// 规则 2：ad_username 命中（ad_dn 未命中）。
	if u, ok := byADUsername[adUsername]; ok {
		// 2a：本行 ad_dn 为空（历史数据未记录 DN）。
		//
		//     选择归入规则 1（catActiveAD）而非冲突，理由：
		//     ad_username 精确命中说明这行就是该 AD 用户的本地记录；ad_dn 为空属于
		//     历史缺录，补齐 AD 返回的 DN 是安全的（不会覆盖任何非空 DN）。
		//     若判为冲突，这类历史脏数据将永远无法被同步修复，用户永远进不了系统。
		//
		//     例外（多域隔离优先）：本行已归属其它 AD 配置（ad_config_id 不同）→ 冲突。
		if isEmptyDN(u.AdDn) {
			if isSameOrUnknownConfig(u.AdConfigID, adConfigID) {
				return catActiveAD, u
			}
			return catADConflict, u
		}
		// 2b：本行 ad_dn 非空且与本次 AD DN 不同 → 跨域同名 / 历史脏数据。
		//     绝不覆盖对方的 ad_username / ad_dn，也绝不 INSERT（避免撞 username 唯一索引）。
		return catADConflict, u
	}

	// 规则 3：username 命中活跃行。
	if u, ok := byLocalName[adUsername]; ok {
		// 3a：ad_username 为空 → 纯 local 同名用户 → 绑定到 AD。
		if u.ADUsername == nil || *u.ADUsername == "" {
			return catLocalSameName, u
		}
		// 3b：ad_username 非空（且不等于本次 AD 用户名，否则规则 2 已命中）
		//     → 历史脏数据：username 已被绑定到别的 AD 身份。
		//     判为冲突：绝不 INSERT（INSERT 会撞 username 唯一索引并回滚整个事务）。
		return catADConflict, u
	}

	// 规则 4：username 命中软删行 → 恢复（身份校验由 checkSoftDeletedRestore 负责）。
	if u, ok := softByName[adUsername]; ok {
		return catSoftDeleted, u
	}

	// 规则 5：全新用户。
	return catNew, nil
}

// classifyUser 历史兼容入口（保留旧签名：只按 ad_username / 同名 local / 同名软删三类判定）。
//
// 保留原因：既有测试直接调用本函数锁定分类语义，且该签名不含 DN 与 AD 配置 ID，
// 无法承载多域判据。新代码一律使用 classifyADUser。
//
// ⚠ 本函数不参与实际同步链路（SyncUserFromAD / BatchSyncADUsers 都调用 classifyADUser）。
func classifyUser(
	adUsername string,
	byADUsername, activeLocalByName, softDeletedByName map[string]*models.User,
) (userCategory, *models.User) {
	if u, ok := byADUsername[adUsername]; ok {
		return catActiveAD, u
	}
	if u, ok := activeLocalByName[adUsername]; ok {
		return catLocalSameName, u
	}
	if u, ok := softDeletedByName[adUsername]; ok {
		return catSoftDeleted, u
	}
	return catNew, nil
}

// isEmptyDN 判断 DN 指针是否为空（NULL / 空串 / 纯空白）。
func isEmptyDN(dn *string) bool {
	return dn == nil || strings.TrimSpace(*dn) == ""
}

// isSameOrUnknownConfig 判断存量行的 ad_config_id 与当前 AD 配置是否兼容。
// 任一侧未知（空）即视为兼容 —— 未知时不做多域判断，避免误伤历史数据。
func isSameOrUnknownConfig(stored *string, current string) bool {
	if stored == nil || *stored == "" || current == "" {
		return true
	}
	return *stored == current
}

// derefString 解引用 *string（nil 返回空串）。
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// normalizeDNKey 计算与 SQL 表达式 REPLACE(LOWER(ad_dn), ' ', '') 等价的匹配键。
//
// 为什么在 NormalizeDN 之后还要"去掉全部空格"：AD DN 常见写法是
// "CN=张, OU=IT , DC=Corp"（逗号后带空格），而库里存的可能是另一种空格/大小写组合。
// NormalizeDN 已统一段间空格与大小写，但仍保留属性值内部的空格；再统一去掉全部空格后，
// Go 侧与 SQL 侧结果完全一致，跨库（sqlite 测试 / PostgreSQL 生产）行为相同。
func normalizeDNKey(dn string) string {
	return strings.ReplaceAll(addomain.NormalizeDN(dn), " ", "")
}

// userIndex 分类索引（由 indexExistingUsers 一次性预查询构建）
type userIndex struct {
	byAdDN       map[string]*models.User // 归一化 ad_dn → 活跃行
	byADUsername map[string]*models.User // ad_username → 活跃行
	byLocalName  map[string]*models.User // username → 活跃行（含 ad_username 非空的行）
	softByName   map[string]*models.User // username → 软删行
}

// indexExistingUsers 一次性预查询 sys_user，构建 4 个分类索引：
//   - byAdDN:       活跃且 ad_dn 非空的用户（key = 归一化 DN，与 SQL 表达式对齐）
//   - byADUsername: 活跃且 ad_username 非空的用户（key = ad_username）
//   - byLocalName:  活跃用户（key = username）——**不限 ad_username 是否为空**
//   - softByName:   软删除的同名用户（key = username）
//
// ⚠ byLocalName 必须包含 ad_username 非空的行：历史脏数据
//   （username='bob' 且 ad_username='bob@corp'）若不进入索引，AD 侧 'bob' 会落 catNew
//   并 INSERT username='bob' → 撞 sys_user.username 唯一索引 → 整个事务回滚 →
//   AD 密码正确也进不了系统。
//
// ⚠ ad_dn 必须进入查找条件：DN 全局唯一，是最可靠的跨域判据。
//   只按 ad_username 查会让多域同名用户互相串味（域 A 的 alice 改写域 B 的 alice 的 ad_dn）。
//
// 批量同步用此函数把 N 次逐用户查询压缩为 2 次（活跃+软删除），消除 N+1。
// db 可传 tx（事务内）或 s.db.WithContext(ctx)（批量只读预查询）。
func (s *UserSyncService) indexExistingUsers(
	db *gorm.DB,
	adUsernames []string,
	adDNs []string,
) (*userIndex, error) {
	idx := &userIndex{
		byAdDN:       make(map[string]*models.User),
		byADUsername: make(map[string]*models.User),
		byLocalName:  make(map[string]*models.User),
		softByName:   make(map[string]*models.User),
	}
	if len(adUsernames) == 0 && len(adDNs) == 0 {
		return idx, nil
	}

	// 归一化 DN 匹配键（与 SQL 侧 REPLACE(LOWER(ad_dn),' ','') 语义一致）
	dnKeys := make([]string, 0, len(adDNs))
	for _, dn := range adDNs {
		if k := normalizeDNKey(dn); k != "" {
			dnKeys = append(dnKeys, k)
		}
	}

	// 活跃用户（GORM 默认过滤软删除）。一次查询覆盖 ad_username / username / ad_dn 三个匹配键。
	// ⚠ 三个条件是 OR 关系，必须拼在同一个 Where 里（GORM 的链式 Where 是 AND）。
	conds := make([]string, 0, 3)
	args := make([]interface{}, 0, 6)
	if len(adUsernames) > 0 {
		conds = append(conds, "(ad_username IN ? OR username IN ?)")
		args = append(args, adUsernames, adUsernames)
	}
	if len(dnKeys) > 0 {
		conds = append(conds, "REPLACE(LOWER(ad_dn), ' ', '') IN ?")
		args = append(args, dnKeys)
	}
	if len(conds) > 0 {
		var active []models.User
		if err := db.Where(strings.Join(conds, " OR "), args...).Find(&active).Error; err != nil {
			return nil, fmt.Errorf("预查询活跃用户失败: %w", err)
		}
		for i := range active {
			u := &active[i]
			idx.byLocalName[u.Username] = u
			if u.ADUsername != nil && *u.ADUsername != "" {
				idx.byADUsername[*u.ADUsername] = u
			}
			if key := normalizeDNKey(derefString(u.AdDn)); key != "" {
				idx.byAdDN[key] = u
			}
		}
	}

	// 软删除的同名用户（Unscoped + 手动条件）
	if len(adUsernames) > 0 {
		var deleted []models.User
		if err := db.Unscoped().
			Where("username IN ? AND deleted_at IS NOT NULL", adUsernames).
			Find(&deleted).Error; err != nil {
			return nil, fmt.Errorf("预查询软删除用户失败: %w", err)
		}
		for i := range deleted {
			idx.softByName[deleted[i].Username] = &deleted[i]
		}
	}
	return idx, nil
}

// createNewUser 创建新用户（AD用户首次登录）
func (s *UserSyncService) createNewUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, defaultRoleID string) error {
	adUsername := adUser.Username
	adDN := adUser.UserDN
	displayName := adUser.DisplayName

	user.Username = adUser.Username
	user.Nickname = &displayName
	user.Status = models.UserStatusEnabled
	user.AuthSource = "ad"
	user.ADUsername = &adUsername
	user.AdDn = &adDN
	user.InitFlag = true // 标记为初始密码，需要修改
	user.PwdExpireDays = 90

	// 统一字段策略（见文件顶部 adFieldPolicies）：
	//   - email/phone/mobile/title：AD 权威覆盖（AD 为空也写 → 清空本地旧值）
	//   - ad_dn/ad_ou_dn/ad_config_id：AD 权威覆盖（身份定位 / 多域隔离）
	//   - nickname：新建时本地为空，直接用 AD displayName
	//   - password/employee_no/dept_name/角色：本地语义，不写
	applyADUpdatesToUser(user, mergeADUpdates(adAuthoritativeUpdates(adUser), adIdentityUpdates(adUser)))

	// 初始密码：AD 来源账号的本地密码必须是随机不可知值（见 generateRandomPassword）
	randomPassword, err := generateRandomPassword()
	if err != nil {
		return fmt.Errorf("生成初始密码失败: %w", err)
	}
	hashedPassword, err := s.pwdManager.HashPassword(randomPassword)
	if err != nil {
		return fmt.Errorf("生成初始密码哈希失败: %w", err)
	}
	user.Password = hashedPassword

	// 设置默认部门（如果配置了）
	if defaultDeptID := s.getDefaultDeptID(tx); defaultDeptID != "" {
		user.DeptID = &defaultDeptID
	}

	// 创建用户
	if err := tx.Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}

	// 分配默认角色
	if defaultRoleID != "" {
		if err := s.assignRole(tx, user.ID, defaultRoleID); err != nil {
			return fmt.Errorf("分配角色失败: %w", err)
		}
	}

	return nil
}

// applyADUpdatesToUser 把策略 helper 产出的更新集合落到 models.User 的对应列上。
//
// 新建路径没有行可 Update（还没有主键），只能直接赋字段；这里用"列名 → setter"映射
// 保证与 Updates 路径使用**同一份策略产出的列集合**（同一策略，两个入口）。
func applyADUpdatesToUser(user *models.User, updates map[string]interface{}) {
	for column, value := range updates {
		str, ok := value.(*string)
		if !ok {
			continue // 策略 helper 只产出 *string，其它类型不落到具体列
		}
		switch column {
		case "email":
			user.Email = str
		case "phone":
			user.Phone = str
		case "mobile":
			user.Mobile = str
		case "title":
			user.Title = str
		case "ad_dn":
			user.AdDn = str
		case "ad_ou_dn":
			user.AdOuDn = str
		case "ad_config_id":
			user.AdConfigID = str
		}
	}
}

// updateExistingUser 更新已存在的用户信息
func (s *UserSyncService) updateExistingUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync) error {
	// 统一字段策略（见文件顶部 adFieldPolicies）：
	//   - email/phone/mobile/title：AD 权威覆盖（AD 为空也写 → 清空本地旧值）
	//   - ad_dn/ad_ou_dn/ad_config_id：AD 权威覆盖
	//   - nickname：仅当本地为空时才回填（尊重本地中文昵称）
	//   - password/employee_no/dept_name/角色：永不覆盖
	updates := mergeADUpdates(adAuthoritativeUpdates(adUser), adIdentityUpdates(adUser))

	// 只有当nickname为空时才更新（尊重本地修改）
	if nickname := adNicknameUpdate(user, adUser); nickname != nil {
		updates["nickname"] = nickname
	}

	if err := tx.Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新用户信息失败: %w", err)
	}

	return nil
}

// checkSoftDeletedRestore 软删用户恢复的身份校验共享入口（SAFE-01 / SAFE-02）。
// 返回 true 表示允许恢复；false 表示拒绝，且本函数已写出唯一一条安全审计 WARN。
//
// 判定（D-01 / D-02）：
//   - 存量 ad_dn 归一化后为空（NULL / 空串 / 纯空白）→ 放行；
//   - 与 AD 当前 DN 归一化后相等 → 放行；
//   - 其余 → 拒绝。
//
// 安全不变量（D-03）：本函数只读，绝不改写存量 ad_dn，也绝不解除软删状态。
// 登录路径与批量路径都必须经由本函数判定，禁止在调用点自行实现 DN 比较。
func (s *UserSyncService) checkSoftDeletedRestore(user *models.User, currentUserDN, path string) bool {
	storedDN := ""
	if user.AdDn != nil {
		storedDN = *user.AdDn
	}
	normalizedStored := addomain.NormalizeDN(storedDN)

	// D-01: 存量 DN 为空 → 无身份可比对，放行（不写日志）。
	if normalizedStored == "" {
		return true
	}

	// D-02: 归一化比较（属性名与属性值都大小写不敏感）。
	normalizedCurrent := addomain.NormalizeDN(currentUserDN)
	if normalizedStored == normalizedCurrent {
		return true
	}

	// D-05: 两路径共用的唯一审计日志产出点。用户可见文案不含 DN；
	// 完整 DN 细节只进 WARN。
	applogger.Warnf(
		"[SECURITY] 软删用户恢复被拒绝: AD 身份不一致, path=%s, userID=%s, username=%s, storedAdDN=%q, currentAdDN=%q, normalizedStored=%q, normalizedCurrent=%q",
		path, user.ID, user.Username, storedDN, currentUserDN, normalizedStored, normalizedCurrent,
	)
	return false
}

// restoreSoftDeletedUser 恢复被软删除的用户并补充/重置AD信息。
// 用于处理"用户曾被软删除(deleted_at IS NOT NULL),再次从AD同步"的场景:
// 由于 sys_user.username 是普通唯一索引(软删除行仍占用 username),
// 不能直接创建新行,必须恢复已有行。
//
// 恢复动作:
//  1. 清除 deleted_at（恢复为活跃用户）
//  2. 重置为初始密码(对齐 createNewUser: InitFlag=true, PwdExpireDays=90)
//  3. 更新 AD 信息(ad_username/ad_dn/ad_ou_dn/ad_synced_at/email/phone)
//  4. 重新分配默认角色(软删除期间角色关联可能已被清理)
func (s *UserSyncService) restoreSoftDeletedUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, defaultRoleID string) error {
	// SAFE-01/SAFE-02: 在密码哈希与任何写库之前做身份校验。
	// 拒绝时返回 sentinel；事务内无写入 → 回滚后行保持软删、存量 ad_dn 原值保留（D-03）。
	if !s.checkSoftDeletedRestore(user, adUser.UserDN, "login") {
		return fmt.Errorf("%w: username=%s", security.ErrSoftDeleteIdentityMismatch, adUser.Username)
	}

	// 重置初始密码（随机不可知；InitFlag=true 保持"未设置过可用本地密码"语义）
	randomPassword, err := generateRandomPassword()
	if err != nil {
		return fmt.Errorf("生成初始密码失败: %w", err)
	}
	hashedPassword, err := s.pwdManager.HashPassword(randomPassword)
	if err != nil {
		return fmt.Errorf("生成初始密码哈希失败: %w", err)
	}
	if err := s.applyRestoreSoftDeleted(tx, user, adUser, hashedPassword); err != nil {
		return err
	}

	// 重新分配默认角色（软删除期间关联可能丢失）
	if defaultRoleID != "" {
		if err := s.assignRole(tx, user.ID, defaultRoleID); err != nil {
			// 角色分配失败不阻断恢复，仅记录
			applogger.Warnf("[SyncADUser] 恢复用户 %s 分配角色失败(不影响恢复): %v", adUser.Username, err)
		}
	}

	applogger.Infof("[SyncADUser] 恢复软删除用户成功: username=%s, userID=%s", adUser.Username, user.ID)
	return nil
}

// applyRestoreSoftDeleted 应用软删除恢复操作（不含角色分配，角色由调用方处理）。
// hashedPassword 为已计算好的初始密码哈希（随机不可知）；批量路径预哈希后传入以避免重复计算。
func (s *UserSyncService) applyRestoreSoftDeleted(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, hashedPassword string) error {
	adUsername := adUser.Username

	// 恢复骨架（本地语义）：解除软删、重置初始密码、启用、切到 ad 来源
	updates := map[string]interface{}{
		"deleted_at":      nil,
		"password":        hashedPassword,
		"init_flag":       true,
		"pwd_expire_days": 90,
		"status":          models.UserStatusEnabled,
		"auth_source":     "ad",
		"ad_username":     &adUsername,
	}

	// 统一字段策略（见文件顶部 adFieldPolicies）：email/phone/mobile/title 与
	// ad_dn/ad_ou_dn/ad_config_id 均为 AD 权威覆盖；nickname 仅本地为空时回填。
	updates = mergeADUpdates(updates, adAuthoritativeUpdates(adUser), adIdentityUpdates(adUser))

	// 仅当 nickname 为空时才回填（尊重本地修改）
	if nickname := adNicknameUpdate(user, adUser); nickname != nil {
		updates["nickname"] = nickname
	}

	// 恢复 + 更新（Unscoped 绕过软删除过滤）
	if err := tx.Unscoped().Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("恢复软删除用户失败: %w", err)
	}
	return nil
}

// bindLocalUserToAD 将现有活跃 local 用户关联绑定到 AD 账号。
// 场景: sys_user 已存在同名 local 用户(auth_source=local, ad_username 为空, 活跃),
// 现从 AD 同步该用户。补充 AD 字段并将 auth_source 改为 'ad'。
// 保留: password / dept_id / employee_no / 角色 —— 不重置密码、不覆盖部门。
func (s *UserSyncService) bindLocalUserToAD(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync) error {
	adUsername := adUser.Username

	// 统一字段策略（见文件顶部 adFieldPolicies）：
	//   - auth_source/ad_username：绑定动作本身（AD 权威）
	//   - email/phone/mobile/title、ad_dn/ad_ou_dn/ad_config_id：AD 权威覆盖
	//   - nickname：仅本地为空时回填
	//   - password/dept_id/employee_no/角色：保留本地值，永不覆盖
	updates := mergeADUpdates(
		map[string]interface{}{
			"auth_source": "ad",
			"ad_username": &adUsername,
		},
		adAuthoritativeUpdates(adUser),
		adIdentityUpdates(adUser),
	)
	// 仅当 nickname 为空时才回填（尊重本地已有昵称）
	if nickname := adNicknameUpdate(user, adUser); nickname != nil {
		updates["nickname"] = nickname
	}

	if err := tx.Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("关联绑定AD失败: %w", err)
	}
	applogger.Infof("[SyncADUser] 关联绑定 local 用户: username=%s, userID=%s", adUser.Username, user.ID)
	return nil
}

// assignRole 分配角色给用户
func (s *UserSyncService) assignRole(tx *gorm.DB, userID string, roleID string) error {
	// 使用原生SQL插入角色关联（ON CONFLICT避免重复）。
	// created_at 由 Go 侧传参而非 SQL 方言函数（datetime('now')/NOW()），
	// 保证 sqlite（测试）与 PostgreSQL（生产）行为一致。
	sql := `INSERT INTO sys_user_role (user_id, role_id, created_at)
			VALUES (?, ?, ?)
			ON CONFLICT (user_id, role_id) DO NOTHING`

	if err := tx.Exec(sql, userID, roleID, time.Now()).Error; err != nil {
		return fmt.Errorf("分配角色失败: %w", err)
	}

	return nil
}

// getDefaultDeptID 获取默认部门ID
//
// P2-11：必须过滤 deleted_at —— 软删除的参数配置不应再参与默认部门解析，
// 否则管理员删掉的配置项仍在生效。
func (s *UserSyncService) getDefaultDeptID(tx *gorm.DB) string {
	var config models.Config
	err := tx.Where("config_key = ? AND deleted_at IS NULL", "sys.auth.ad.default_dept_id").First(&config).Error
	if err != nil {
		return "" // 没有配置默认部门
	}
	return config.ConfigValue
}

// SyncADUser 实现 security.UserSyncer 接口
// 将 security.ADUserInfo 转换为内部类型并调用同步逻辑
// 完整处理：同步用户信息 + 解析部门
func (s *UserSyncService) SyncADUser(ctx context.Context, adUserInfo *security.ADUserInfo, defaultRoleID string) (*security.SyncedUser, error) {
	// 添加调试日志
	applogger.Infof("[SyncADUser] 开始同步 AD 用户: username=%s, userDN=%s", adUserInfo.Username, adUserInfo.UserDN)

	// 转换 ADUserInfo -> ADUserInfoForSync
	adUser := &ADUserInfoForSync{
		UserDN:      adUserInfo.UserDN,
		OuDn:        adUserInfo.OUDN,
		Username:    adUserInfo.Username,
		DisplayName: adUserInfo.DisplayName,
		Email:       adUserInfo.Email,
		Phone:       adUserInfo.Phone,
		Mobile:      adUserInfo.Mobile,
		Title:       adUserInfo.Title,
		Department:  adUserInfo.Department,
		// AD 配置 ID（多域隔离）：登录路径由认证器经 context 传入
		// （见 security.WithADConfigID / security.ADConfigIDFromContext）。
		ConfigID: security.ADConfigIDFromContext(ctx),
	}

	// 1. 转换并同步用户基本信息
	user, category, err := s.SyncUserFromAD(ctx, adUser, defaultRoleID)
	if err != nil {
		// 安全策略拒绝是预期行为（WARN），其余是基础设施/数据故障（ERROR）。
		if errors.Is(err, security.ErrSoftDeleteIdentityMismatch) {
			applogger.Warnf("[SECURITY] 软删恢复被安全策略拒绝: username=%s, err=%v", adUser.Username, err)
		} else {
			applogger.Errorf("[SyncADUser] 同步失败: %v", err)
		}
		return nil, err
	}

	applogger.Infof("[SyncADUser] 同步成功: userID=%s, username=%s, category=%d", user.ID, user.Username, category)

	// 2. 自动解析并设置部门
	// 关联绑定的 local 用户保留其原有部门(已配置合理)，不按 AD OU 覆盖；
	// 其余分类(新建/恢复/活跃AD更新)按 AD OU 解析部门。
	if category == catLocalSameName {
		applogger.Infof("[SyncADUser] 关联绑定用户保留原部门: userID=%s, deptID=%v", user.ID, user.DeptID)
	} else if adUserInfo.OUDN != "" && s.ouMapper != nil {
		deptID, err := s.resolveDeptFromOU(ctx, adUserInfo.OUDN)
		if err != nil {
			applogger.Warnf("解析部门失败（不影响同步）: %v", err)
		} else if deptID == "" {
			// 无 OU 映射且不自动创建部门 → 保留用户原部门（不置空）
			applogger.Infof("[SyncADUser] 无OU映射，保留原部门: userID=%s, deptID=%v", user.ID, user.DeptID)
		} else {
			// GUARD-01 (Phase 59 D-04): 活跃队列守卫只包住唯一一次 dept_id 回写。
			// 命中 pending/failed 的 update_user 行 → 管理员本地编辑仍在队列中，跳过覆盖。
			// 查询失败 → 保守跳过（D-07），绝不回滚本地编辑。
			// 无活跃行 → 走与 Phase 58 完全相同的 Update 调用（D-08，无平行分支）。
			//
			// P1-4：守卫逻辑已抽成公共函数 shouldSkipDeptUpdate，登录路径与批量路径
			// （catActiveAD / catSoftDeleted）共用同一份语义。
			skip, reason := s.shouldSkipDeptUpdate(s.db, ctx, user.ID, user.Username, deptValue(user.DeptID), deptID)
			if skip {
				applogger.Infof("[SyncADUser] 跳过部门回写: userID=%s, username=%s, deptID=%s, reason=%s",
					user.ID, user.Username, deptID, reason)
			} else if err := s.db.WithContext(ctx).Model(user).Update("dept_id", deptID).Error; err != nil {
				applogger.Warnf("更新用户部门失败: %v", err)
			} else {
				applogger.Infof("[SyncADUser] 成功设置用户部门: userID=%s, deptID=%s", user.ID, deptID)
			}
		}
	} else {
		applogger.Warnf("[SyncADUser] 跳过部门解析: OUDN为空=%v, ouMapper为nil=%v", adUserInfo.OUDN == "", s.ouMapper == nil)
	}

	// 3. 转换 models.User -> security.SyncedUser
	result := &security.SyncedUser{
		ID:       user.ID,
		Username: user.Username,
		Nickname: user.Nickname,
		Email:    user.Email,
		Phone:    user.Phone,
		Status:   int(user.Status),
		DeptID:   user.DeptID,
		Roles:    user.Roles,
	}

	return result, nil
}

// SyncError 同步错误信息
type SyncError struct {
	Username string `json:"username"`
	Error    string `json:"error"`
}

// BatchSyncResult 批量同步结果
type BatchSyncResult struct {
	Total   int         `json:"total"`
	Success int         `json:"success"`
	Failed  int         `json:"failed"`
	Skipped int         `json:"skipped"`
	Errors  []SyncError `json:"errors,omitempty"`
}

// batchHashConcurrency 批量密码哈希并发度。
// SM3-PBKDF2 60万次迭代为 CPU 密集型，限制并发避免单实例过载；可按 CPU 核数调整。
const batchHashConcurrency = 8

// batchWriteSize 批量写入分批大小（对齐 sync.go 的 500，兼顾 PostgreSQL 参数上限）
const batchWriteSize = 500

// batchPlanItem 批量同步计划项（一个 AD 用户的分类结果与命中的现有用户）
type batchPlanItem struct {
	adUser   *ADUserInfoForSync
	category userCategory
	user     *models.User // 命中的现有用户（catNew 时为 nil）
}

// BatchSyncADUsers 批量同步AD用户到sys_user表（批量流水线实现）。
//
// 相比逐用户同步（每用户独立事务 + 串行 60万次哈希 + 部门 N+1），本实现通过：
//  1. 一次性预查询 sys_user 并内存分类 4 类用户（消除 N+1，正确处理同名 local/软删除）
//  2. errgroup worker pool 并发计算密码哈希（catNew + catSoftDeleted）
//  3. 部门解析 ouCache 缓存（同 OU 只解析一次）
//  4. catNew 用 CreateInBatches 批量 INSERT；UPDATE 类按小批次事务，失败降级逐个
//
// 保留逐用户错误隔离：单个用户失败计入 failed 并返回明细，不中断整批。
func (s *UserSyncService) BatchSyncADUsers(
	ctx context.Context,
	users []*ADUserInfoForSync,
	defaultRoleID string,
) (*BatchSyncResult, error) {
	result := &BatchSyncResult{Total: len(users), Errors: []SyncError{}}
	if len(users) == 0 {
		return result, nil
	}
	totalStart := time.Now()
	applogger.Infof("[BatchSyncADUsers] 开始批量同步 %d 个AD用户", len(users))

	// 阶段1: 一次性预查询 sys_user，构建分类索引（消除 N+1）
	usernames := make([]string, 0, len(users))
	dns := make([]string, 0, len(users))
	for _, u := range users {
		usernames = append(usernames, u.Username)
		if u.UserDN != "" {
			dns = append(dns, u.UserDN)
		}
	}
	t0 := time.Now()
	idx, err := s.indexExistingUsers(s.db.WithContext(ctx), usernames, dns)
	if err != nil {
		return nil, err
	}

	// 阶段2: 内存分类（DN 优先 + 多域隔离，classifyADUser 规则 1~5）
	plansByCat := make(map[userCategory][]*batchPlanItem)
	for i := range users {
		cat, matched := classifyADUser(users[i].Username, users[i].UserDN, users[i].ConfigID,
			idx.byAdDN, idx.byADUsername, idx.byLocalName, idx.softByName)
		plansByCat[cat] = append(plansByCat[cat], &batchPlanItem{adUser: users[i], category: cat, user: matched})
	}
	applogger.Infof("[BatchSyncADUsers] 分类(预查询耗时%.2fs): 全新=%d, 活跃AD更新=%d, 软删除恢复=%d, local关联=%d, 冲突=%d",
		time.Since(t0).Seconds(),
		len(plansByCat[catNew]), len(plansByCat[catActiveAD]),
		len(plansByCat[catSoftDeleted]), len(plansByCat[catLocalSameName]), len(plansByCat[catADConflict]))

	// catADConflict（规则 2b/3b）：策略性跳过，绝不 INSERT / 改写（撞唯一索引会回滚整批）。
	for _, it := range plansByCat[catADConflict] {
		result.Skipped++
		result.Errors = append(result.Errors, SyncError{
			Username: it.adUser.Username,
			Error:    "AD 身份冲突（该 username/ad_dn 已绑定其它 AD 身份），拒绝同步",
		})
	}
	delete(plansByCat, catADConflict)

	// SAFE-01/SAFE-02 (D-06/D-07): 在阶段 3 哈希与阶段 5c 写库之前过滤被拒项。
	// 校验入口与登录路径共用 checkSoftDeletedRestore；拒绝是策略行为不是故障，
	// 因此短路跳过并计入 Skipped，禁止返回 error（否则 runInBatchedTx 会整批降级
	// 逐个重试并把策略拒绝记为 Failed）。响应中的错误文本不含 DN（DN 细节只进 WARN）。
	if items := plansByCat[catSoftDeleted]; len(items) > 0 {
		allowed := make([]*batchPlanItem, 0, len(items))
		for _, it := range items {
			if !s.checkSoftDeletedRestore(it.user, it.adUser.UserDN, "batch") {
				result.Skipped++
				result.Errors = append(result.Errors, SyncError{
					Username: it.adUser.Username,
					Error:    "AD 身份不一致，拒绝恢复（需管理员核对 ad_dn）",
				})
				continue
			}
			allowed = append(allowed, it)
		}
		plansByCat[catSoftDeleted] = allowed
	}

	// 阶段3: 并发哈希密码（catNew + catSoftDeleted 需重置默认密码）
	t0 = time.Now()
	var needHash []string
	for _, it := range plansByCat[catNew] {
		needHash = append(needHash, it.adUser.Username)
	}
	for _, it := range plansByCat[catSoftDeleted] {
		needHash = append(needHash, it.adUser.Username)
	}
	hashMap, err := s.hashPasswordsConcurrent(ctx, needHash)
	if err != nil {
		return nil, err
	}
	applogger.Infof("[BatchSyncADUsers] 并发哈希完成(耗时%.2fs, %d个)", time.Since(t0).Seconds(), len(needHash))

	// 阶段4: 部门预解析（catNew + catSoftDeleted + catActiveAD 按需；catLocalSameName 保留原部门）
	t0 = time.Now()
	ouCache := make(map[string]string) // ouDN -> deptID（"" 表示解析失败/无）
	resolveDept := func(ouDN string) string {
		if ouDN == "" {
			return ""
		}
		if id, ok := ouCache[ouDN]; ok {
			return id
		}
		id, derr := s.resolveDeptFromOU(ctx, ouDN)
		if derr != nil {
			applogger.Warnf("[BatchSyncADUsers] 解析部门失败 ouDN=%s: %v", ouDN, derr)
			ouCache[ouDN] = ""
			return ""
		}
		ouCache[ouDN] = id
		return id
	}
	ouSet := make(map[string]struct{})
	for _, cat := range []userCategory{catNew, catSoftDeleted, catActiveAD} {
		for _, it := range plansByCat[cat] {
			if it.adUser.OuDn != "" {
				ouSet[it.adUser.OuDn] = struct{}{}
			}
		}
	}
	for ouDN := range ouSet {
		resolveDept(ouDN)
	}
	applogger.Infof("[BatchSyncADUsers] 部门预解析完成(耗时%.2fs, %d个唯一OU)", time.Since(t0).Seconds(), len(ouSet))

	// 阶段5a: catNew 批量 INSERT（CreateInBatches）
	var newCreatedIDs []string
	if news := plansByCat[catNew]; len(news) > 0 {
		t0 = time.Now()
		defaultDeptID := s.getDefaultDeptID(s.db.WithContext(ctx))
		newUsers := make([]models.User, 0, len(news))
		for _, it := range news {
			adUsername := it.adUser.Username
			adDN := it.adUser.UserDN
			displayName := it.adUser.DisplayName
			email := it.adUser.Email
			phone := it.adUser.Phone
			u := models.User{
				Username:      it.adUser.Username,
				Nickname:      &displayName,
				Email:         &email,
				Phone:         &phone,
				Status:        models.UserStatusEnabled,
				AuthSource:    "ad",
				ADUsername:    &adUsername,
				AdDn:          &adDN,
				AdOuDn:        &it.adUser.OuDn,
				InitFlag:      true,
				PwdExpireDays: 90,
				Password:      hashMap[it.adUser.Username],
			}
			if deptID := ouCache[it.adUser.OuDn]; deptID != "" {
				u.DeptID = &deptID
			} else if defaultDeptID != "" {
				u.DeptID = &defaultDeptID
			}
			newUsers = append(newUsers, u)
		}
		if err := s.db.WithContext(ctx).CreateInBatches(&newUsers, batchWriteSize).Error; err != nil {
			// 批量插入失败，降级逐个独立事务
			applogger.Warnf("[BatchSyncADUsers] catNew 批量插入失败，降级逐个: %v", err)
			for i := range newUsers {
				u := newUsers[i]
				if oneErr := s.db.WithContext(ctx).Create(&u).Error; oneErr != nil {
					result.Failed++
					result.Errors = append(result.Errors, SyncError{Username: u.Username, Error: oneErr.Error()})
				} else {
					result.Success++
					newCreatedIDs = append(newCreatedIDs, u.ID)
				}
			}
		} else {
			for i := range newUsers {
				newCreatedIDs = append(newCreatedIDs, newUsers[i].ID)
			}
			result.Success += len(newUsers)
		}
		applogger.Infof("[BatchSyncADUsers] catNew 创建完成(耗时%.2fs, %d个)", time.Since(t0).Seconds(), len(news))
	}

	// 阶段5b: catActiveAD 批量更新（小批次事务，失败降级逐个）
	if items := plansByCat[catActiveAD]; len(items) > 0 {
		s.runInBatchedTx(ctx, items, result, func(tx *gorm.DB, it *batchPlanItem) error {
			if err := s.updateExistingUser(tx, it.user, it.adUser); err != nil {
				return err
			}
			if deptID := ouCache[it.adUser.OuDn]; deptID != "" {
				if err := tx.Model(it.user).Update("dept_id", deptID).Error; err != nil {
					applogger.Warnf("[BatchSyncADUsers] 更新部门失败 %s: %v", it.adUser.Username, err)
				}
			}
			return nil
		})
	}

	// 阶段5c: catSoftDeleted 批量恢复（小批次事务，用预哈希密码；角色随恢复同事务分配）
	if items := plansByCat[catSoftDeleted]; len(items) > 0 {
		s.runInBatchedTx(ctx, items, result, func(tx *gorm.DB, it *batchPlanItem) error {
			if err := s.applyRestoreSoftDeleted(tx, it.user, it.adUser, hashMap[it.adUser.Username]); err != nil {
				return err
			}
			if deptID := ouCache[it.adUser.OuDn]; deptID != "" {
				if err := tx.Model(it.user).Update("dept_id", deptID).Error; err != nil {
					applogger.Warnf("[BatchSyncADUsers] 恢复后更新部门失败 %s: %v", it.adUser.Username, err)
				}
			}
			if defaultRoleID != "" {
				if err := s.assignRole(tx, it.user.ID, defaultRoleID); err != nil {
					applogger.Warnf("[BatchSyncADUsers] 恢复用户 %s 分配角色失败: %v", it.adUser.Username, err)
				}
			}
			return nil
		})
	}

	// 阶段5d: catLocalSameName 关联绑定（保留原部门，不改密码/角色）
	if items := plansByCat[catLocalSameName]; len(items) > 0 {
		s.runInBatchedTx(ctx, items, result, func(tx *gorm.DB, it *batchPlanItem) error {
			return s.bindLocalUserToAD(tx, it.user, it.adUser)
		})
	}

	// 阶段6: 批量角色分配（catNew 新建用户，CreateInBatches 已回填 ID）
	if defaultRoleID != "" && len(newCreatedIDs) > 0 {
		if err := s.assignRolesBatch(s.db.WithContext(ctx), newCreatedIDs, defaultRoleID); err != nil {
			applogger.Warnf("[BatchSyncADUsers] 批量角色分配失败(不影响用户同步): %v", err)
		}
	}

	// L-02: 本批已写 sys_user_role（阶段5c软删除恢复分配 + 阶段6新建批量分配），
	// 整批只需失效一次 user-scoped 菜单缓存（全量清 menu:user:* 前缀）。
	if defaultRoleID != "" && (len(newCreatedIDs) > 0 || len(plansByCat[catSoftDeleted]) > 0) {
		s.invalidateUserMenuCache(ctx)
	}

	applogger.Infof("[BatchSyncADUsers] 完成: 成功=%d, 失败=%d, 跳过=%d, 总耗时=%.2fs",
		result.Success, result.Failed, result.Skipped, time.Since(totalStart).Seconds())
	return result, nil
}

// hashPasswordsConcurrent 并发计算默认密码 "123456" 的哈希（每用户名一次，线程安全）。
// SM3-PBKDF2 为 CPU 密集型，并发可大幅缩短耗时。
func (s *UserSyncService) hashPasswordsConcurrent(ctx context.Context, usernames []string) (map[string]string, error) {
	hashes := make(map[string]string, len(usernames))
	if len(usernames) == 0 {
		return hashes, nil
	}
	var mu sync.Mutex
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(batchHashConcurrency)
	for _, name := range usernames {
		name := name
		g.Go(func() error {
			h, err := s.pwdManager.HashPassword("123456")
			if err != nil {
				return fmt.Errorf("哈希用户 %s 密码失败: %w", name, err)
			}
			mu.Lock()
			hashes[name] = h
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return hashes, nil
}

// runInBatchedTx 对一组计划项按 batchWriteSize 分批写入，每批一个事务。
// 某批事务失败则降级为该批逐个独立事务（保留逐用户错误隔离），失败项计入 result.Failed。
func (s *UserSyncService) runInBatchedTx(
	ctx context.Context,
	items []*batchPlanItem,
	result *BatchSyncResult,
	handleOne func(tx *gorm.DB, it *batchPlanItem) error,
) {
	for i := 0; i < len(items); i += batchWriteSize {
		end := i + batchWriteSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]
		batchErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			for _, it := range batch {
				if err := handleOne(tx, it); err != nil {
					return err
				}
			}
			return nil
		})
		if batchErr == nil {
			result.Success += len(batch)
			continue
		}
		// 降级：逐个独立事务
		applogger.Warnf("[BatchSyncADUsers] 批次事务失败(%d个)，降级逐个处理: %v", len(batch), batchErr)
		for _, it := range batch {
			oneErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				return handleOne(tx, it)
			})
			if oneErr != nil {
				result.Failed++
				result.Errors = append(result.Errors, SyncError{Username: it.adUser.Username, Error: oneErr.Error()})
			} else {
				result.Success++
			}
		}
	}
}

// assignRolesBatch 批量分配同一角色给多个用户（多值 INSERT ON CONFLICT DO NOTHING）。
// 相比逐个 assignRole，把 N 次 SQL 往返压成 ceil(N/500) 次。
// created_at 由 Go 侧传参（同 assignRole），方言中立：sqlite / PostgreSQL 通用。
func (s *UserSyncService) assignRolesBatch(db *gorm.DB, userIDs []string, roleID string) error {
	if len(userIDs) == 0 || roleID == "" {
		return nil
	}
	now := time.Now()
	for i := 0; i < len(userIDs); i += batchWriteSize {
		end := i + batchWriteSize
		if end > len(userIDs) {
			end = len(userIDs)
		}
		placeholders := make([]string, 0, end-i)
		args := make([]interface{}, 0, (end-i)*3)
		for _, uid := range userIDs[i:end] {
			placeholders = append(placeholders, "(?, ?, ?)")
			args = append(args, uid, roleID, now)
		}
		sql := fmt.Sprintf(
			`INSERT INTO sys_user_role (user_id, role_id, created_at) VALUES %s ON CONFLICT (user_id, role_id) DO NOTHING`,
			strings.Join(placeholders, ","),
		)
		if err := db.Exec(sql, args...).Error; err != nil {
			return fmt.Errorf("批量分配角色失败: %w", err)
		}
	}
	return nil
}

// resolveDeptFromOU 解析 OU 对应的部门。
// AD 同步仅同步用户，不自动创建部门结构：
//   - 若存在 OU→部门 映射，返回对应 deptID
//   - 若无映射，返回空字符串，由调用方使用默认部门（sys.auth.ad.default_dept_id）
//
// 部门结构应由管理员在系统中预先维护，避免从域控 OU 自动推导出错误/重复部门。
func (s *UserSyncService) resolveDeptFromOU(ctx context.Context, ouDN string) (string, error) {
	deptID, err := s.ouMapper.FindDeptByOUDN(ctx, ouDN)
	if err == nil {
		applogger.Infof("[resolveDeptFromOU] 找到现有OU映射: ouDN=%s -> deptID=%s", ouDN, deptID)
		return deptID, nil
	}
	// 无 OU 映射 → 不创建部门，返回空，调用方使用默认部门
	applogger.Infof("[resolveDeptFromOU] 无OU映射，不创建部门(使用默认部门): ouDN=%s", ouDN)
	return "", nil
}

// findActiveUserUpdateQueue 查询该用户是否存在活跃的 update_user 队列行（GUARD-01）。
//
// 谓词直接拼接 models 包导出的活跃行谓词常量 —— 该常量是 migration_215 部分唯一索引
// DDL 与 adPushOnConflictClause 的单一事实源（Phase 57 D-02 红线：不得改写、不得加表名限定）。
// 只取单行（Take）即可：守卫需要的是布尔信号（Phase 59 D-03）。
//
// 返回 (nil, gorm.ErrRecordNotFound) 表示无活跃行；其余 error 由调用方按 D-07 保守降级。
//
// db 参数说明：调用方必须传**当前事务的 tx**（批量路径）或 s.db.WithContext(ctx)（登录路径）。
// 绝不能在写事务内用另一条连接查询 —— sqlite（测试）会因库锁报错，PostgreSQL 会多占一个连接。
func (s *UserSyncService) findActiveUserUpdateQueue(db *gorm.DB, ctx context.Context, userID string) (*models.ADPushQueue, error) {
	var row models.ADPushQueue
	err := db.WithContext(ctx).
		Where("entity_id = ? AND entity_type = ? AND action_type = ?",
			userID, models.ADPushEntityUser, models.ADPushActionUpdateUser).
		Where(models.ADPushQueueActivePredicate).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// shouldSkipDeptUpdate 判断本次 AD 部门回写是否必须跳过（GUARD-01 公共入口，P1-4）。
//
// 登录路径（SyncADUser）与批量路径（BatchSyncADUsers 的 catActiveAD / catSoftDeleted）
// **必须**共用本函数，保证守卫语义完全一致：
//   - 命中 sys_ad_push_queue 的 pending/failed update_user 行 → 跳过
//     （管理员的本地部门变更仍在推送队列中，不能被 AD 回写覆盖）
//   - 守卫查询失败（DB 报错，非 ErrRecordNotFound）→ 保守跳过
//     （宁可不同步部门，也绝不覆盖本地编辑）
//   - 查不到活跃行 → 不跳过，正常回写
//
// 返回 (是否跳过, 原因)。原因同时用于：批量路径的计数口径（跳过不能记 Success）
// 与用户可读的提示文案。
//
// db 参数说明：登录路径传 s.db.WithContext(ctx)（此处在事务之外）；
// 批量路径传当前事务的 tx（避免写事务内跨连接查询导致 sqlite 库锁）。
func (s *UserSyncService) shouldSkipDeptUpdate(db *gorm.DB, ctx context.Context, userID, username, localDeptID, adDeptID string) (bool, string) {
	qrow, qerr := s.findActiveUserUpdateQueue(db, ctx, userID)
	switch {
	case qerr != nil && !errors.Is(qerr, gorm.ErrRecordNotFound):
		applogger.Warnf("部门回写守卫查询失败，跳过 dept_id 更新: userID=%s, username=%s, localDeptID=%s, adDeptID=%s, err=%v",
			userID, username, localDeptID, adDeptID, qerr)
		return true, fmt.Sprintf("部门回写守卫查询失败，已跳过部门更新: %v", qerr)
	case qerr == nil:
		applogger.Warnf("部门回写被活跃队列守卫拦截: userID=%s, username=%s, localDeptID=%s, adDeptID=%s, queueID=%s, actionType=%s, attempts=%d",
			userID, username, localDeptID, adDeptID, qrow.ID, qrow.ActionType, qrow.Attempts)
		return true, "本地部门变更正在同步到 AD，暂不覆盖部门"
	default:
		return false, ""
	}
}

// deptValue 将 *string 部门 ID 渲染为日志可读文本（Phase 59 Pitfall 7：
// %v 直接打印 *string 会输出指针地址，目标是让运维看到本地 dept UUID）。
func deptValue(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}
