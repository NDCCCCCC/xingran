package system

import (
	"context"
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
}

// UserSyncService 用户同步服务
// 负责将AD用户信息同步到sys_user表，并自动解析部门
type UserSyncService struct {
	db         *gorm.DB
	pwdManager PasswordManager
	ouMapper   *addomain.DeptOUmapper // OU映射解析器
}

// NewUserSyncService 创建用户同步服务
func NewUserSyncService(db *gorm.DB, pwdManager PasswordManager, ouMapper *addomain.DeptOUmapper) *UserSyncService {
	return &UserSyncService{
		db:         db,
		pwdManager: pwdManager,
		ouMapper:   ouMapper,
	}
}

// SyncUserFromAD 从AD同步单个用户到sys_user表（首次AD登录路径使用）。
// 按 4 类分类处理，返回同步后的用户及其分类（供调用方决定部门处理等后续动作）。
func (s *UserSyncService) SyncUserFromAD(ctx context.Context, adUser *ADUserInfoForSync, defaultRoleID string) (*models.User, userCategory, error) {
	var user models.User
	var category userCategory

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		byAD, byLocal, bySoft, err := s.indexExistingUsers(tx, []string{adUser.Username})
		if err != nil {
			return err
		}
		category, matched := classifyUser(adUser.Username, byAD, byLocal, bySoft)
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
		default: // catNew
			return s.createNewUser(tx, &user, adUser, defaultRoleID)
		}
	})

	if err != nil {
		return nil, catNew, fmt.Errorf("同步AD用户失败: %w", err)
	}

	return &user, category, nil
}

// userCategory 用户同步分类（按处理优先级排序）
type userCategory int

const (
	catActiveAD      userCategory = iota // ① 活跃AD用户(ad_username 精确命中)
	catLocalSameName                     // ③ 活跃local同名用户(username 同名, ad_username 为空)
	catSoftDeleted                       // ② 软删除用户(username 同名, deleted_at 非空)
	catNew                               // ④ 全新用户(无任何命中)
)

// classifyUser 按优先级 ①>③>②>④ 判定 AD 用户应如何同步。
// 三个索引均由 indexExistingUsers 预查询构建。返回命中的现有用户（catNew 时为 nil）。
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

// indexExistingUsers 一次性预查询 sys_user，构建 3 个分类索引：
//   - byADUsername: 活跃且 ad_username 非空的用户（key=ad_username）
//   - activeLocalByName: 活跃且 ad_username 为空的 local 用户（key=username）
//   - softDeletedByName: 软删除的同名用户（key=username）
//
// 批量同步用此函数把 N 次逐用户查询压缩为 2 次（活跃+软删除），消除 N+1。
// db 可传 tx（事务内）或 s.db.WithContext(ctx)（批量只读预查询）。
func (s *UserSyncService) indexExistingUsers(
	db *gorm.DB,
	adUsernames []string,
) (byADUsername, activeLocalByName, softDeletedByName map[string]*models.User, err error) {
	byADUsername = make(map[string]*models.User)
	activeLocalByName = make(map[string]*models.User)
	softDeletedByName = make(map[string]*models.User)
	if len(adUsernames) == 0 {
		return
	}

	// 活跃用户（GORM 默认过滤软删除）。一次查询覆盖 ad_username 与 username 两个匹配键。
	var active []models.User
	if err = db.
		Where("ad_username IN ? OR username IN ?", adUsernames, adUsernames).
		Find(&active).Error; err != nil {
		err = fmt.Errorf("预查询活跃用户失败: %w", err)
		return
	}
	for i := range active {
		u := &active[i]
		if u.ADUsername != nil && *u.ADUsername != "" {
			byADUsername[*u.ADUsername] = u
		} else {
			activeLocalByName[u.Username] = u
		}
	}

	// 软删除的同名用户（Unscoped + 手动条件）
	var deleted []models.User
	if err = db.Unscoped().
		Where("username IN ? AND deleted_at IS NOT NULL", adUsernames).
		Find(&deleted).Error; err != nil {
		err = fmt.Errorf("预查询软删除用户失败: %w", err)
		return
	}
	for i := range deleted {
		softDeletedByName[deleted[i].Username] = &deleted[i]
	}
	return
}

// createNewUser 创建新用户（AD用户首次登录）
func (s *UserSyncService) createNewUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, defaultRoleID string) error {
	adUsername := adUser.Username
	adDN := adUser.UserDN
	displayName := adUser.DisplayName
	email := adUser.Email
	phone := adUser.Phone

	user.Username = adUser.Username
	user.Nickname = &displayName
	user.Email = &email
	user.Phone = &phone
	user.Status = models.UserStatusEnabled
	user.AuthSource = "ad"
	user.ADUsername = &adUsername
	user.AdDn = &adDN
	user.InitFlag = true    // 标记为初始密码，需要修改
	user.PwdExpireDays = 90

	// 设置默认密码（首次登录后要求修改）
	hashedPassword, err := s.pwdManager.HashPassword("123456")
	if err != nil {
		return fmt.Errorf("生成默认密码失败: %w", err)
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

// updateExistingUser 更新已存在的用户信息
func (s *UserSyncService) updateExistingUser(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync) error {
	// 使用 GORM 的 Updates 方法，传入数据库列名
	updates := map[string]interface{}{
		"email":  &adUser.Email,
		"phone":  &adUser.Phone,
		"ad_dn": &adUser.UserDN, // 使用数据库列名 ad_dn
		"ad_ou_dn":  &adUser.OuDn, // 更新用户OU路径（支持用户移动场景）
	}

	// 只有当nickname为空时才更新（尊重本地修改）
	if user.Nickname == nil || *user.Nickname == "" {
		updates["nickname"] = &adUser.DisplayName
	}

	if err := tx.Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新用户信息失败: %w", err)
	}

	return nil
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
	// 重置默认密码（首次登录后要求修改，对齐 createNewUser）
	hashedPassword, err := s.pwdManager.HashPassword("123456")
	if err != nil {
		return fmt.Errorf("生成默认密码失败: %w", err)
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
// hashedPassword 为已计算好的默认密码哈希；批量路径预哈希后传入以避免重复计算。
func (s *UserSyncService) applyRestoreSoftDeleted(tx *gorm.DB, user *models.User, adUser *ADUserInfoForSync, hashedPassword string) error {
	adUsername := adUser.Username
	adDN := adUser.UserDN
	displayName := adUser.DisplayName
	email := adUser.Email
	phone := adUser.Phone

	updates := map[string]interface{}{
		"deleted_at":      nil,
		"password":        hashedPassword,
		"init_flag":       true,
		"pwd_expire_days": 90,
		"status":          models.UserStatusEnabled,
		"auth_source":     "ad",
		"ad_username":     &adUsername,
		"ad_dn":           &adDN,
		"ad_ou_dn":        &adUser.OuDn,
		"email":           &email,
		"phone":           &phone,
	}

	// 仅当 nickname 为空时才回填（尊重本地修改）
	if user.Nickname == nil || *user.Nickname == "" {
		updates["nickname"] = &displayName
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
	adDN := adUser.UserDN
	displayName := adUser.DisplayName
	email := adUser.Email
	phone := adUser.Phone

	updates := map[string]interface{}{
		"auth_source": "ad",
		"ad_username": &adUsername,
		"ad_dn":       &adDN,
		"ad_ou_dn":    &adUser.OuDn,
		"email":       &email,
		"phone":       &phone,
	}
	// 仅当 nickname 为空时才回填（尊重本地已有昵称）
	if user.Nickname == nil || *user.Nickname == "" {
		updates["nickname"] = &displayName
	}

	if err := tx.Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("关联绑定AD失败: %w", err)
	}
	applogger.Infof("[SyncADUser] 关联绑定 local 用户: username=%s, userID=%s", adUser.Username, user.ID)
	return nil
}

// assignRole 分配角色给用户
func (s *UserSyncService) assignRole(tx *gorm.DB, userID string, roleID string) error {
	// 使用原生SQL插入角色关联（ON CONFLICT避免重复）
	sql := `INSERT INTO sys_user_role (user_id, role_id, created_at)
			VALUES (?, ?, NOW())
			ON CONFLICT (user_id, role_id) DO NOTHING`

	if err := tx.Exec(sql, userID, roleID).Error; err != nil {
		return fmt.Errorf("分配角色失败: %w", err)
	}

	return nil
}

// getDefaultDeptID 获取默认部门ID
func (s *UserSyncService) getDefaultDeptID(tx *gorm.DB) string {
	var config models.Config
	err := tx.Where("config_key = ?", "sys.auth.ad.default_dept_id").First(&config).Error
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
	}

	// 1. 转换并同步用户基本信息
	user, category, err := s.SyncUserFromAD(ctx, adUser, defaultRoleID)
	if err != nil {
		applogger.Errorf("[SyncADUser] 同步失败: %v", err)
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
		} else if err := s.db.WithContext(ctx).Model(user).Update("dept_id", deptID).Error; err != nil {
			applogger.Warnf("更新用户部门失败: %v", err)
		} else {
			applogger.Infof("[SyncADUser] 成功设置用户部门: userID=%s, deptID=%s", user.ID, deptID)
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
	for _, u := range users {
		usernames = append(usernames, u.Username)
	}
	t0 := time.Now()
	byAD, byLocal, bySoft, err := s.indexExistingUsers(s.db.WithContext(ctx), usernames)
	if err != nil {
		return nil, err
	}

	// 阶段2: 内存分类 4 组
	plansByCat := make(map[userCategory][]*batchPlanItem)
	for i := range users {
		cat, matched := classifyUser(users[i].Username, byAD, byLocal, bySoft)
		plansByCat[cat] = append(plansByCat[cat], &batchPlanItem{adUser: users[i], category: cat, user: matched})
	}
	applogger.Infof("[BatchSyncADUsers] 分类(预查询耗时%.2fs): 全新=%d, 活跃AD更新=%d, 软删除恢复=%d, local关联=%d",
		time.Since(t0).Seconds(),
		len(plansByCat[catNew]), len(plansByCat[catActiveAD]),
		len(plansByCat[catSoftDeleted]), len(plansByCat[catLocalSameName]))

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
func (s *UserSyncService) assignRolesBatch(db *gorm.DB, userIDs []string, roleID string) error {
	if len(userIDs) == 0 || roleID == "" {
		return nil
	}
	for i := 0; i < len(userIDs); i += batchWriteSize {
		end := i + batchWriteSize
		if end > len(userIDs) {
			end = len(userIDs)
		}
		placeholders := make([]string, 0, end-i)
		args := make([]interface{}, 0, (end-i)*2)
		for _, uid := range userIDs[i:end] {
			placeholders = append(placeholders, "(?, ?, NOW())")
			args = append(args, uid, roleID)
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

