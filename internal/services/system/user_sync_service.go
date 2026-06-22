package system

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
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

// SyncUserFromAD 从AD同步用户到sys_user表
// 如果用户已存在（按ad_username查找），则更新信息；否则创建新用户
func (s *UserSyncService) SyncUserFromAD(ctx context.Context, adUser *ADUserInfoForSync, defaultRoleID string) (*models.User, error) {
	var user models.User

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 检查用户是否已存在（按ad_username查找，仅活跃用户）
		err := tx.Where("ad_username = ? AND deleted_at IS NULL", adUser.Username).First(&user).Error

		if err == nil {
			// 用户已存在，更新信息
			return s.updateExistingUser(tx, &user, adUser)
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 2. 检查是否存在被软删除的同 username 用户。
			// sys_user.username 是普通唯一索引(非 partial index),软删除的行仍占用 username,
			// 直接 createNewUser 会撞 idx_sys_user_username (SQLSTATE 23505)。
			// 因此先尝试恢复软删除用户,而非插入新行。
			var softDeletedUser models.User
			if findErr := tx.Unscoped().
				Where("username = ? AND deleted_at IS NOT NULL", adUser.Username).
				First(&softDeletedUser).Error; findErr == nil {
				user = softDeletedUser
				return s.restoreSoftDeletedUser(tx, &user, adUser, defaultRoleID)
			}
			// 3. 用户确实不存在，创建新用户
			return s.createNewUser(tx, &user, adUser, defaultRoleID)
		}

		return fmt.Errorf("查询用户失败: %w", err)
	})

	if err != nil {
		return nil, fmt.Errorf("同步AD用户失败: %w", err)
	}

	return &user, nil
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
	user, err := s.SyncUserFromAD(ctx, adUser, defaultRoleID)
	if err != nil {
		applogger.Errorf("[SyncADUser] 同步失败: %v", err)
		return nil, err
	}

	applogger.Infof("[SyncADUser] 同步成功: userID=%s, username=%s", user.ID, user.Username)

	// 2. 自动解析并设置部门（整合的逻辑）
	applogger.Infof("[SyncADUser] 准备解析部门: OUDN=%s, ouMapper=%v", adUserInfo.OUDN, s.ouMapper != nil)
	if adUserInfo.OUDN != "" && s.ouMapper != nil {
		deptID, err := s.resolveDeptFromOU(ctx, adUserInfo.OUDN)
		if err != nil {
			applogger.Warnf("解析部门失败（不影响同步）: %v", err)
			// 部门解析失败，使用默认部门继续
		} else {
			// 更新用户部门
			if err := s.db.WithContext(ctx).Model(&user).Update("dept_id", deptID).Error; err != nil {
				applogger.Warnf("更新用户部门失败: %v", err)
			} else {
				applogger.Infof("[SyncADUser] 成功设置用户部门: userID=%s, deptID=%s, ouDN=%s", user.ID, deptID, adUserInfo.OUDN)
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

// BatchSyncADUsers 批量同步AD用户到sys_user表
// 支持批量操作，返回成功/失败统计和详细错误信息
func (s *UserSyncService) BatchSyncADUsers(
	ctx context.Context,
	users []*ADUserInfoForSync,
	defaultRoleID string,
) (*BatchSyncResult, error) {
	result := &BatchSyncResult{
		Total:   len(users),
		Success: 0,
		Failed:  0,
		Skipped: 0,
		Errors:  []SyncError{},
	}

	if len(users) == 0 {
		return result, nil
	}

	applogger.Infof("[BatchSyncADUsers] 开始批量同步 %d 个AD用户", len(users))

	for _, adUser := range users {
		// 转换为 security.ADUserInfo 以复用 SyncADUser 方法
		adUserInfo := &security.ADUserInfo{
			UserDN:      adUser.UserDN,
			OUDN:        adUser.OuDn, // 重要：传递 OU DN 用于部门解析
			Username:    adUser.Username,
			DisplayName: adUser.DisplayName,
			Email:       adUser.Email,
			Phone:       adUser.Phone,
			Mobile:      adUser.Mobile,
			Title:       adUser.Title,
			Department:  adUser.Department,
		}

		// 调用现有的同步逻辑
		syncedUser, err := s.SyncADUser(ctx, adUserInfo, defaultRoleID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, SyncError{
				Username: adUser.Username,
				Error:    err.Error(),
			})
			applogger.Errorf("[BatchSyncADUsers] 同步用户 %s 失败: %v", adUser.Username, err)
		} else {
			result.Success++
			applogger.Infof("[BatchSyncADUsers] 同步用户 %s 成功, userID=%s", adUser.Username, syncedUser.ID)
		}
	}

	applogger.Infof("[BatchSyncADUsers] 批量同步完成: 成功%d, 失败%d, 跳过%d",
		result.Success, result.Failed, result.Skipped)

	return result, nil
}

// resolveDeptFromOU 解析OU并设置部门
// 1. 先尝试查找现有OU映射
// 2. 未找到映射，自动创建部门及映射
func (s *UserSyncService) resolveDeptFromOU(ctx context.Context, ouDN string) (string, error) {
	// 1. 尝试查找现有映射
	deptID, err := s.ouMapper.FindDeptByOUDN(ctx, ouDN)
	if err == nil {
		applogger.Infof("[resolveDeptFromOU] 找到现有OU映射: ouDN=%s -> deptID=%s", ouDN, deptID)
		return deptID, nil
	}

	// 2. 未找到映射，自动创建部门及映射
	applogger.Infof("[resolveDeptFromOU] 未找到映射，自动创建部门: ouDN=%s", ouDN)
	return s.createDeptFromOUDN(ctx, ouDN)
}

// createDeptFromOUDN 从 AD OU DN 自动创建部门及映射关系
// OU DN 格式：OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn
func (s *UserSyncService) createDeptFromOUDN(ctx context.Context, ouDN string) (string, error) {
	// 1. 获取 AD 配置
	var adConfig models.ADConfig
	if err := s.db.WithContext(ctx).Where("status = 0").First(&adConfig).Error; err != nil {
		return "", fmt.Errorf("获取AD配置失败: %w", err)
	}

	// 2. 解析 OU DN，提取部门层级
	ouParts := addomain.ParseOUDN(ouDN)
	if len(ouParts) == 0 {
		return "", fmt.Errorf("无效的OU DN: %s", ouDN)
	}

	// 3. 从右到左处理部门层级（跳过域名部分）
	var deptNames []string
	for _, part := range ouParts {
		if !strings.Contains(part, "DC=") {
			// 提取 OU= 后面的名称
			name := strings.TrimPrefix(part, "OU=")
			deptNames = append(deptNames, name)
		}
	}

	// 4. 从最底层部门开始，逐层创建部门
	// deptNames 是从上到下的顺序，需要反转从下到上创建
	var parentDeptID *string
	for i := len(deptNames) - 1; i >= 0; i-- {
		deptName := deptNames[i]

		// 检查部门是否已存在
		var dept models.Department
		err := s.db.WithContext(ctx).
			Where("dept_name = ? AND status = 0", deptName).
			First(&dept).Error

		if err == gorm.ErrRecordNotFound {
			// 生成唯一的 DeptCode
			deptCode := s.generateUniqueDeptCode(ctx, deptName)

			// 部门不存在，创建新部门
			dept = models.Department{
				DeptName:  deptName,
				DeptCode:  deptCode,
				ParentID:  parentDeptID,
				Status:    0, // 正常
				Ancestors: s.buildAncestors(ctx, parentDeptID),
			}

			if err := s.db.WithContext(ctx).Create(&dept).Error; err != nil {
				return "", fmt.Errorf("创建部门 %s 失败: %w", deptName, err)
			}
			applogger.Infof("[createDeptFromOUDN] 自动创建部门: %s (dept_code=%s, parent_id=%v)", deptName, deptCode, parentDeptID)
		} else if err != nil {
			return "", fmt.Errorf("查询部门 %s 失败: %w", deptName, err)
		}

		// 当前部门成为下一个部门的父部门
		deptID := dept.ID
		parentDeptID = &deptID

		// 如果是最底层部门（用户所在部门），创建 OU 映射
		if i == 0 {
			now := time.Now()
			mapping := &models.DeptOUMapping{
				DeptID:      dept.ID,
				ADConfigID:  adConfig.ID,
				OUDN:        ouDN,
				OUName:      deptName,
				SyncEnabled: true,
				SyncStatus:  "synced",
				LastSyncAt:  &now,
			}

			if err := s.ouMapper.UpsertMapping(ctx, mapping); err != nil {
				applogger.Warnf("创建OU映射失败: %v", err)
				// 映射创建失败不影响部门使用
			}
		}
	}

	// 返回最底层部门ID
	if parentDeptID == nil {
		return "", fmt.Errorf("无法确定部门ID")
	}

	return *parentDeptID, nil
}

// buildAncestors 构建部门祖先路径
func (s *UserSyncService) buildAncestors(ctx context.Context, parentID *string) string {
	if parentID == nil {
		return ""
	}

	// 查询父部门的 ancestors
	var parentDept models.Department
	if err := s.db.WithContext(ctx).
		Select("ancestors").
		Where("id = ?", *parentID).
		First(&parentDept).Error; err != nil {
		return ""
	}

	// 构建当前部门的 ancestors
	if parentDept.Ancestors == "" {
		return *parentID
	}
	return fmt.Sprintf("%s,%s", parentDept.Ancestors, *parentID)
}

// generateUniqueDeptCode 为部门生成唯一的编码
// 优先使用部门名称作为编码，如果名称重复则添加序号后缀
func (s *UserSyncService) generateUniqueDeptCode(ctx context.Context, deptName string) string {
	// 尝试使用部门名称作为编码
	var count int64
	err := s.db.WithContext(ctx).
		Model(&models.Department{}).
		Where("dept_code = ?", deptName).
		Count(&count).Error

	if err != nil || count == 0 {
		// 部门名称可用作编码
		return deptName
	}

	// 部门名称已被占用，尝试添加序号后缀
	for i := 1; i <= 100; i++ {
		candidateCode := fmt.Sprintf("%s-%d", deptName, i)
		err := s.db.WithContext(ctx).
			Model(&models.Department{}).
			Where("dept_code = ?", candidateCode).
			Count(&count).Error

		if err != nil || count == 0 {
			return candidateCode
		}
	}

	// 如果序号方法也失败，使用时间戳生成唯一编码
	return fmt.Sprintf("%s-%d", deptName, time.Now().Unix())
}
