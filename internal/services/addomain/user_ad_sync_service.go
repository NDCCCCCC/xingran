package addomain

import (
	"context"
	"fmt"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// UserADSyncService 用户AD同步服务
// 负责在用户信息修改时同步变更到AD域控
type UserADSyncService struct {
	db     *gorm.DB
	ldap   *LDAPClient
	mapper *DeptOUmapper
}

// NewUserADSyncService 创建UserADSyncService实例
func NewUserADSyncService(db *gorm.DB, ldapClient *LDAPClient, mapper *DeptOUmapper) *UserADSyncService {
	return &UserADSyncService{
		db:     db,
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

	// 创建LDAP客户端
	ldapClient := NewLDAPClient(&adConfig)
	if err := ldapClient.Connect(); err != nil {
		return fmt.Errorf("连接AD失败: %w", err)
	}
	defer ldapClient.Close()

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

	// 5. 更新同步时间戳
	if err := s.updateSyncTimestamp(ctx, userID); err != nil {
		applogger.Warnf("更新同步时间戳失败: %v", err)
	}

	applogger.Infof("用户 %s 信息同步到AD成功", user.Username)
	return nil
}

// moveUserToNewOU 移动用户到新OU
func (s *UserADSyncService) moveUserToNewOU(ctx context.Context, ldapClient *LDAPClient, userID, newDeptID string) error {
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
func (s *UserADSyncService) syncUserAttributes(ctx context.Context, ldapClient *LDAPClient, user *models.User, updateReq map[string]interface{}) error {
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

	// 更新AD属性
	if err := ldapClient.UpdateUserAttribute(*user.AdDn, attributes); err != nil {
		return fmt.Errorf("更新AD用户属性失败: %w", err)
	}

	applogger.Infof("用户 %s 属性同步到AD: %v", user.Username, attributes)
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

	// 创建LDAP客户端
	ldapClient := NewLDAPClient(&adConfig)
	if err := ldapClient.Connect(); err != nil {
		return fmt.Errorf("连接AD失败: %w", err)
	}
	defer ldapClient.Close()

	// 3. 批量移动用户（分批处理，每批10个）
	batchSize := 10
	successCount := 0
	failedCount := 0

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

	applogger.Infof("批量移动用户完成: 成功=%d, 失败=%d, 目标OU=%s", successCount, failedCount, ouDN)

	if failedCount > 0 {
		return fmt.Errorf("批量移动完成，部分失败: 成功=%d, 失败=%d", successCount, failedCount)
	}
	return nil
}

// moveSingleUserToOU 移动单个用户到指定OU
func (s *UserADSyncService) moveSingleUserToOU(ctx context.Context, ldapClient *LDAPClient, userID, ouDN string) error {
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
