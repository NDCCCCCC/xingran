package addomain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
	"gorm.io/gorm"
)

type UserOUService struct {
	db     *gorm.DB
	mapper *DeptOUmapper
}

func NewUserOUService(db *gorm.DB, mapper *DeptOUmapper) *UserOUService {
	return &UserOUService{
		db:     db,
		mapper: mapper,
	}
}

// HandleUserLoginAD 处理用户AD登录后的部门设置
// 新增功能：当OU不存在时，自动创建部门及映射关系
func (s *UserOUService) HandleUserLoginAD(ctx context.Context, username, userDN, ouDN string) error {
	// 1. 通过用户名查找系统用户
	var user models.User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 用户不存在，跳过部门设置（由注册流程处理）
			applogger.Infof("用户 %s 不存在于系统，跳过部门设置", username)
			return nil
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 2. 通过AD OU DN查找对应的部门
	deptID, err := s.mapper.FindDeptByOUDN(ctx, ouDN)
	if err != nil {
		// 未找到映射部门，尝试自动创建部门
		applogger.Infof("用户 %s 的AD OU %s 未找到映射，尝试自动创建部门", username, ouDN)
		deptID, err = s.createDeptFromOUDN(ctx, ouDN, userDN)
		if err != nil {
			applogger.Warnf("自动创建部门失败 %s: %v", username, err)
			// 不阻断登录流程，用户会被分配到默认部门
			return nil
		}
		applogger.Infof("成功为用户 %s 自动创建部门: dept_id=%s", username, deptID)
	}

	// 3. 更新用户的部门和AD信息
	if err := s.updateUserDeptAndADInfo(ctx, user.ID, deptID, userDN, ouDN); err != nil {
		applogger.Errorf("更新用户 %s 部门信息失败: %v", username, err)
		// 不阻断登录流程，只记录错误
		return nil
	}

	applogger.Infof("用户 %s 登录时自动设置部门: dept_id=%s, ou_dn=%s", username, deptID, ouDN)
	return nil
}

// updateUserDeptAndADInfo 更新用户部门和AD信息
func (s *UserOUService) updateUserDeptAndADInfo(ctx context.Context, userID, deptID, userDN, ouDN string) error {
	updates := map[string]interface{}{
		"dept_id":      deptID,
		"ad_dn":        userDN,
		"ad_ou_dn":     ouDN,
		"ad_synced_at": time.Now(),
	}

	return s.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		Updates(updates).Error
}

// GetUserDeptByADOU 获取用户AD OU对应的部门信息（辅助方法）
func (s *UserOUService) GetUserDeptByADOU(ctx context.Context, username string) (*models.Department, error) {
	var user models.User
	if err := s.db.WithContext(ctx).
		Preload("Dept").
		Where("username = ?", username).
		First(&user).Error; err != nil {
		return nil, err
	}

	if user.Dept == nil {
		return nil, fmt.Errorf("用户 %s 未设置部门", username)
	}

	return user.Dept, nil
}

// createDeptFromOUDN 从 AD OU DN 自动创建部门及映射关系
// OU DN 格式：OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn
func (s *UserOUService) createDeptFromOUDN(ctx context.Context, ouDN, userDN string) (string, error) {
	// 1. 获取 AD 配置
	var adConfig models.ADConfig
	if err := s.db.WithContext(ctx).Where("status = 0").First(&adConfig).Error; err != nil {
		return "", fmt.Errorf("获取AD配置失败: %w", err)
	}

	// 2. 解析 OU DN，提取部门层级
	// 示例：["OU=基础运维科", "OU=科技创新部", "OU=分公司本部", "OU=湖北分公司"]
	ouParts := ParseOUDN(ouDN)
	if len(ouParts) == 0 {
		return "", fmt.Errorf("无效的OU DN: %s", ouDN)
	}

	// 3. 从右到左处理部门层级（跳过域名部分）
	// 域名部分通常包含 DC=，需要过滤掉
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
			// 部门不存在，创建新部门
			dept = models.Department{
				DeptName:  deptName,
				ParentID:  parentDeptID,
				Status:    0, // 正常
				Ancestors: s.buildAncestors(ctx, parentDeptID),
			}

			if err := s.db.WithContext(ctx).Create(&dept).Error; err != nil {
				return "", fmt.Errorf("创建部门 %s 失败: %w", deptName, err)
			}
			applogger.Infof("自动创建部门: %s (parent_id=%v)", deptName, parentDeptID)
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

			if err := s.mapper.UpsertMapping(ctx, mapping); err != nil {
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

// buildAncestors 构建部门祖先路径（用于 GORM tree 等功能）
func (s *UserOUService) buildAncestors(ctx context.Context, parentID *string) string {
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
