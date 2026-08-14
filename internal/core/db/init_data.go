package db

import (
	"errors"
	"fmt"
	"os"

	"github.com/xingran-next/xingran-go-backend/internal/core/security"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// initData 初始化基础数据
func initData(db *gorm.DB) error {
	// 创建默认部门
	if err := createDefaultDept(db); err != nil {
		return fmt.Errorf("创建默认部门失败: %w", err)
	}

	// 创建默认用户
	if err := createDefaultUser(db); err != nil {
		return fmt.Errorf("创建默认用户失败: %w", err)
	}

	// 创建默认角色
	if err := createDefaultRole(db); err != nil {
		return fmt.Errorf("创建默认角色失败: %w", err)
	}

	// 创建用户角色关联
	if err := createUserRoleRelations(db); err != nil {
		return fmt.Errorf("创建用户角色关联失败: %w", err)
	}

	// 创建网络设备系统参数
	if err := createNetworkDeviceSystemParams(db); err != nil {
		return fmt.Errorf("创建网络设备系统参数失败: %w", err)
	}

	// 创建网络设备定时任务
	if err := createNetworkDeviceScheduledJobs(db); err != nil {
		return fmt.Errorf("创建网络设备定时任务失败: %w", err)
	}

	// 创建验证码背景图系统参数
	if err := createCaptchaBackgroundSystemParams(db); err != nil {
		return fmt.Errorf("创建验证码背景图系统参数失败: %w", err)
	}

	// 创建运维管理菜单
	if err := createOperationsManagementMenus(db); err != nil {
		return fmt.Errorf("创建运维管理菜单失败: %w", err)
	}

	// 创建请求加密开关配置参数
	if err := createRequestEncryptionToggleConfig(db); err != nil {
		return fmt.Errorf("创建请求加密开关配置失败: %w", err)
	}
	// 创建AD认证配置参数
	if err := createADAuthConfig(db); err != nil {
		return fmt.Errorf("创建AD认证配置参数失败: %w", err)
	}

	applogger.Infof("基础数据初始化完成")
	return nil
}

// ensureDept 按 dept_name + parent_id 语义查询,已存在则把已存在行的 ID
// 写回 dept.ID 并返回 nil;不存在则 db.Create。Count 查询的真实错误不再吞,
// 用 fmt.Errorf 包装上抛。
//
// C5 修复:细粒度幂等的核心 helper——首次启动中途失败后,下次启动可逐棵子树
// 补齐缺失部门,不再因 "count > 0 整体跳过" 永久遗留半成品。
//
// parentID 为 nil 时查询 parent_id IS NULL(顶级部门);
// 非 nil 时查询 parent_id = ?。
func ensureDept(db *gorm.DB, dept *models.Department, parentID *string) error {
	var existing models.Department
	q := db.Where("dept_name = ?", dept.DeptName)
	if parentID == nil {
		q = q.Where("parent_id IS NULL")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}

	if err := q.First(&existing).Error; err == nil {
		// 已存在:把已存在行的 ID 写回(调用方需要 ID 建子树)
		dept.ID = existing.ID
		applogger.Infof("部门 %s 已存在，跳过创建", dept.DeptName)
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询部门 %s 失败: %w", dept.DeptName, err)
	}

	if err := db.Create(dept).Error; err != nil {
		return fmt.Errorf("创建部门 %s 失败: %w", dept.DeptName, err)
	}
	applogger.Infof("创建部门 %s 成功", dept.DeptName)
	return nil
}

// createDefaultDept 创建默认部门
//
// C5 修复要点:消除原 "count > 0 整体跳过" 的粗粒度幂等——一旦任意部门
// 已存在就永久跳过,导致首次启动中途失败后无法补齐缺失子树。
// 改为逐棵子树独立 ensureDept:已存在的子树跳过,缺失的子树补齐。
//
// DeptCode 字段(模型 uniqueIndex;not null)在种子中填充唯一编码,使整个
// 种子树在空库中能够完成首装(原代码留空会触发 UNIQUE 冲突)。
func createDefaultDept(db *gorm.DB) error {
	// 1. 顶级部门
	topDept := &models.Department{
		DeptName: "若依科技有限公司",
		DeptCode: "ROOT",
		OrderNum: 1,
		Leader:   func() *string { s := "若依"; return &s }(),
		Phone:    func() *string { s := "15888888888"; return &s }(),
		Email:    func() *string { s := "xingran@qq.com"; return &s }(),
		Status:   models.DeptStatusNormal,
	}
	if err := ensureDept(db, topDept, nil); err != nil {
		return err
	}

	// 2. 一级子公司(深圳/长沙)
	subDepts := []*models.Department{
		{
			DeptName:  "深圳总公司",
			DeptCode:  "SHENZHEN",
			Ancestors: topDept.ID,
			OrderNum:  1,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "xingran@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
		},
		{
			DeptName:  "长沙分公司",
			DeptCode:  "CHANGSHA",
			Ancestors: topDept.ID,
			OrderNum:  2,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "xingran@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
		},
	}
	for _, dept := range subDepts {
		dept.ParentID = &topDept.ID
		if err := ensureDept(db, dept, &topDept.ID); err != nil {
			return err
		}
	}

	// 3. 深圳总公司的子部门(研发/市场/测试)
	shenzhenSubDepts := []*models.Department{
		{
			DeptName:  "研发部门",
			DeptCode:  "RD",
			OrderNum:  1,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "xingran@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
		},
		{
			DeptName:  "市场部门",
			DeptCode:  "MARKET",
			OrderNum:  2,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "xingran@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
		},
		{
			DeptName:  "测试部门",
			DeptCode:  "TEST",
			OrderNum:  3,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "xingran@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
		},
	}

	// 找到"深圳总公司"作为父级(ensureDept 已写回其 ID)
	var shenzhen *models.Department
	for _, dept := range subDepts {
		if dept.DeptName == "深圳总公司" {
			shenzhen = dept
			break
		}
	}
	if shenzhen == nil {
		return fmt.Errorf("内部错误:未找到深圳总公司种子节点")
	}

	for _, dept := range shenzhenSubDepts {
		dept.ParentID = &shenzhen.ID
		dept.Ancestors = topDept.ID + "," + shenzhen.ID
		if err := ensureDept(db, dept, &shenzhen.ID); err != nil {
			return err
		}
	}

	applogger.Infof("默认部门检查/创建完成")
	return nil
}

// createDefaultUser 创建默认管理员用户
//
// C2 修复要点:
//  1. 初始密码优先读取环境变量 SYS_ADMIN_BOOTSTRAP_PASSWORD(运维可控入口)
//  2. 未设置时回退到 admin123,并在写入成功后输出 applogger.Warnf 大声告警,
//     文案明确"立即登录修改 / 重建时设置 SYS_ADMIN_BOOTSTRAP_PASSWORD"
//  3. Salt 字段置空串而非字面量 "default"——User.Salt 是遗留死字段,
//     PasswordManager 的随机盐已嵌入 $sm3$iterations$salt$hash 哈希串,
//     VerifyPassword 从哈希串解析盐,不读本字段
func createDefaultUser(db *gorm.DB) error {
	var count int64

	// 检查是否已存在管理员用户
	db.Model(&models.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return nil // 已存在，跳过
	}

	// 使用新的SM3密码管理器
	pwdManager := security.NewPasswordManager(nil)

	// 确定初始密码:env 覆盖优先,否则回退到 admin123
	bootstrapPassword := os.Getenv("SYS_ADMIN_BOOTSTRAP_PASSWORD")
	usingDefaultPassword := false
	if bootstrapPassword == "" {
		bootstrapPassword = "admin123"
		usingDefaultPassword = true
	}

	// 生成密码哈希
	passwordHash, err := pwdManager.HashPassword(bootstrapPassword)
	if err != nil {
		return err
	}

	// 创建默认管理员用户(Salt 留空:该字段是死字段,
	// 真实盐在 PasswordManager 哈希串内,见函数顶部注释)
	user := models.User{
		Username: "admin",
		Password: passwordHash,
		Salt:     "",
		Nickname: func() *string { s := "超级管理员"; return &s }(),
		Email:    func() *string { s := "admin@xingran.com"; return &s }(),
		Gender:   models.GenderMale,
		Status:   models.UserStatusEnabled,
		DeptName: func() *string { s := "总公司"; return &s }(),
		Roles:    []string{"admin"},
	}

	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("创建默认用户失败: %w", err)
	}

	if usingDefaultPassword {
		// 大声告警:使用了出厂默认密码,运维必须立刻处理
		// (密码值本身绝不出现在日志中)
		applogger.Warnf("=========================================================")
		applogger.Warnf("[安全告警] 管理员账户已使用出厂默认密码 admin123")
		applogger.Warnf("  1) 请立即登录并修改管理员密码")
		applogger.Warnf("  2) 或重建实例前设置环境变量 SYS_ADMIN_BOOTSTRAP_PASSWORD=<强密码>")
		applogger.Warnf("  3) 完整方案(首登强制改密)已 deferred,需登录链路改动")
		applogger.Warnf("=========================================================")
	} else {
		applogger.Infof("默认管理员密码已从 SYS_ADMIN_BOOTSTRAP_PASSWORD 环境变量读取")
	}

	applogger.Infof("创建默认管理员用户成功")
	return nil
}

// createDefaultRole 创建默认角色
func createDefaultRole(db *gorm.DB) error {
	roles := []models.Role{
		{
			RoleName:          "超级管理员",
			RoleKey:           "admin",
			RoleSort:          1,
			DataScope:         models.DataScopeAll,
			MenuCheckStrictly: true,
			DeptCheckStrictly: true,
			Status:            models.RoleStatusEnabled,
			Remark:            "超级管理员",
		},
		{
			RoleName:          "普通用户",
			RoleKey:           "user",
			RoleSort:          2,
			DataScope:         models.DataScopeSelf,
			MenuCheckStrictly: true,
			DeptCheckStrictly: true,
			Status:            models.RoleStatusEnabled,
			Remark:            "普通用户",
		},
	}

	for _, role := range roles {
		var count int64
		// 检查角色是否已存在（通过role_key）
		db.Model(&models.Role{}).Where("role_key = ?", role.RoleKey).Count(&count)
		if count > 0 {
			applogger.Infof("角色 %s (role_key: %s) 已存在，跳过创建", role.RoleName, role.RoleKey)
			continue
		}

		if err := db.Create(&role).Error; err != nil {
			return fmt.Errorf("创建角色 %s 失败: %w", role.RoleName, err)
		}
		applogger.Infof("创建角色 %s 成功", role.RoleName)
	}

	applogger.Infof("默认角色检查/创建完成")
	return nil
}

// createUserRoleRelations 创建用户角色关联
//
// CDX-M-USERROLE 修复:用 db.Create(&models.UserRole{...}) 取代硬编码表名
// 的原生 SQL Exec(老代码用 db.Exec 拼字符串 + 硬编码 sys_user_role 表名,
// 一旦 UserRole.TableName 改名就静默断裂)。UserRole 无 ID 字段,
// BeforeCreate 钩子不影响,db.Create 行为等价。好处:消除表名漂移风险
// (models.UserRole.TableName 改名后仍正确),接入 GORM hooks / 自动 UUID typing。
func createUserRoleRelations(db *gorm.DB) error {
	// 获取默认用户
	var adminUser models.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		applogger.Warnf("未找到管理员用户: %v", err)
		return nil // 如果用户不存在，跳过
	}

	// 获取管理员角色
	var adminRole models.Role
	if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
		applogger.Warnf("未找到管理员角色: %v", err)
		return nil // 如果角色不存在，跳过
	}

	// 检查关联是否已存在
	var count int64
	db.Table("sys_user_role").Where("user_id = ? AND role_id = ?", adminUser.ID, adminRole.ID).Count(&count)
	if count > 0 {
		applogger.Infof("用户角色关联已存在，跳过创建")
		return nil
	}

	// 创建用户角色关联(走 GORM Create,不再硬编码表名)
	if err := db.Create(&models.UserRole{
		UserID: adminUser.ID,
		RoleID: adminRole.ID,
	}).Error; err != nil {
		return fmt.Errorf("创建用户角色关联失败: %w", err)
	}

	applogger.Infof("创建用户角色关联成功")
	return nil
}

// createNetworkDeviceSystemParams 创建网络设备系统参数
func createNetworkDeviceSystemParams(db *gorm.DB) error {
	params := []models.Config{
		{
			ConfigName:  "配置备份文件大小阈值",
			ConfigKey:   "network.config.backup.threshold",
			ConfigValue: "100",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "配置备份文件大小阈值（单位：KB），小于此值的配置存储在数据库，大于此值存储在文件系统",
		},
		{
			ConfigName:  "设备连接超时时间",
			ConfigKey:   "network.device.connect.timeout",
			ConfigValue: "30",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "设备连接超时时间（单位：秒）",
		},
		{
			ConfigName:  "命令执行超时时间",
			ConfigKey:   "network.command.execute.timeout",
			ConfigValue: "300",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "命令执行超时时间（单位：秒）",
		},
		{
			ConfigName:  "批量命令并发数",
			ConfigKey:   "network.command.batch.concurrency",
			ConfigValue: "10",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "批量命令执行时的最大并发设备数量",
		},
		// 新增：网络设备并发配置参数
		{
			ConfigName:  "网络设备监控并发数",
			ConfigKey:   "network.device.monitor.concurrent",
			ConfigValue: "10",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "设备状态检查和信息更新的最大并发数，默认10",
		},
		{
			ConfigName:  "端口采集并发数",
			ConfigKey:   "network.port.collection.concurrent",
			ConfigValue: "10",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "端口状态采集的最大并发数，默认10",
		},
		{
			ConfigName:  "MAC地址采集并发数",
			ConfigKey:   "network.mac.collection.concurrent",
			ConfigValue: "10",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "MAC地址表采集的最大并发数，默认10",
		},
		{
			ConfigName:  "配置备份并发数",
			ConfigKey:   "network.config.backup.concurrent",
			ConfigValue: "5",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "配置备份的最大并发数，默认5（配置备份较耗时，建议较低并发）",
		},
		{
			ConfigName:  "设备操作超时时间",
			ConfigKey:   "network.device.timeout",
			ConfigValue: "30",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "单个设备连接和操作的超时时间（秒），默认30秒",
		},
	}

	for _, param := range params {
		var count int64
		db.Model(&models.Config{}).Where("config_key = ?", param.ConfigKey).Count(&count)
		if count > 0 {
			applogger.Infof("系统参数 %s 已存在，跳过创建", param.ConfigName)
			continue
		}

		if err := db.Create(&param).Error; err != nil {
			return fmt.Errorf("创建系统参数 %s 失败: %w", param.ConfigName, err)
		}
		applogger.Infof("创建系统参数 %s 成功", param.ConfigName)
	}

	applogger.Infof("网络设备系统参数创建完成")
	return nil
}

// createNetworkDeviceScheduledJobs 创建网络设备定时任务
func createNetworkDeviceScheduledJobs(db *gorm.DB) error {
	remark := func(s string) *string { return &s }

	jobs := []models.Job{
		{
			JobName:        "设备状态检查",
			JobGroup:       "NETWORK",
			InvokeTarget:   "device_status_check",
			CronExpression: "0 */5 * * * ?",       // 每5分钟执行一次
			Status:         models.JobStatusPause, // 默认暂停，由用户手动启动
			Concurrent:     false,                 // 禁止并发
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("通过SNMP定时检查所有网络设备的在线/离线状态"),
		},
		{
			JobName:        "设备信息更新",
			JobGroup:       "NETWORK",
			InvokeTarget:   "device_info_update",
			CronExpression: "0 0 * * * ?", // 每小时执行一次
			Status:         models.JobStatusPause,
			Concurrent:     false,
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("通过SSH采集网络设备的详细信息（型号、版本、序列号等）"),
		},
		{
			JobName:        "端口状态采集",
			JobGroup:       "NETWORK",
			InvokeTarget:   "port_collection",
			CronExpression: "0 0 * * * ?", // 每小时执行一次
			Status:         models.JobStatusPause,
			Concurrent:     false,
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("采集所有在线网络设备的端口状态信息（启用/禁用、802.1X、端口安全等）"),
		},
		{
			JobName:        "MAC地址采集",
			JobGroup:       "NETWORK",
			InvokeTarget:   "mac_collection",
			CronExpression: "0 0 * * * ?", // 每小时执行一次
			Status:         models.JobStatusPause,
			Concurrent:     false,
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("采集网络设备的MAC地址表信息"),
		},
		{
			JobName:        "配置备份",
			JobGroup:       "NETWORK",
			InvokeTarget:   "config_backup",
			CronExpression: "0 0 2 * * ?", // 每天凌晨2点执行
			Status:         models.JobStatusPause,
			Concurrent:     false,
			MisfirePolicy:  models.MisfirePolicyImmediately,
			Remark:         remark("自动备份网络设备配置文件"),
		},
	}

	for _, job := range jobs {
		var count int64
		db.Model(&models.Job{}).Where("invoke_target = ?", job.InvokeTarget).Count(&count)
		if count > 0 {
			applogger.Infof("定时任务 %s 已存在，跳过创建", job.JobName)
			continue
		}

		if err := db.Create(&job).Error; err != nil {
			return fmt.Errorf("创建定时任务 %s 失败: %w", job.JobName, err)
		}
		applogger.Infof("创建定时任务 %s 成功", job.JobName)
	}

	applogger.Infof("网络设备定时任务创建完成")
	return nil
}

// createCaptchaBackgroundSystemParams 创建验证码背景图系统参数
func createCaptchaBackgroundSystemParams(db *gorm.DB) error {
	params := []models.Config{
		{
			ConfigName:  "验证码背景图模式",
			ConfigKey:   "sys.account.captchaBackgroundMode",
			ConfigValue: "mixed",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "背景图模式: auto=自动生成 custom=仅自定义图片 mixed=混合模式",
		},
		{
			ConfigName:  "验证码默认拼图形状",
			ConfigKey:   "sys.account.captchaPieceShape",
			ConfigValue: "circle",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "默认拼图形状: circle=圆形 square=方形 star=星形 heart=心形",
		},
		{
			ConfigName:  "验证码默认难度",
			ConfigKey:   "sys.account.captchaDifficulty",
			ConfigValue: "1",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "难度级别: 1=简单 2=中等 3=困难",
		},
		{
			ConfigName:  "验证码缓存池大小",
			ConfigKey:   "sys.account.captchaCachePoolSize",
			ConfigValue: "50",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "每种形状和难度预生成的验证码数量",
		},
		{
			ConfigName:  "验证码图片存储路径",
			ConfigKey:   "sys.account.captchaStoragePath",
			ConfigValue: "./uploads/captcha/backgrounds",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "背景图存储路径",
		},
		{
			ConfigName:  "验证码图片最大大小",
			ConfigKey:   "sys.account.captchaMaxFileSize",
			ConfigValue: "2097152",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "单张图片最大大小(字节)，默认2MB",
		},
		{
			ConfigName:  "验证码允许的图片格式",
			ConfigKey:   "sys.account.captchaAllowedFormats",
			ConfigValue: "jpg,jpeg,png",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "允许的图片格式，逗号分隔",
		},
	}

	for _, param := range params {
		var count int64
		db.Model(&models.Config{}).Where("config_key = ?", param.ConfigKey).Count(&count)
		if count > 0 {
			applogger.Infof("系统参数 %s 已存在，跳过创建", param.ConfigName)
			continue
		}

		if err := db.Create(&param).Error; err != nil {
			return fmt.Errorf("创建系统参数 %s 失败: %w", param.ConfigName, err)
		}
		applogger.Infof("创建系统参数 %s 成功", param.ConfigName)
	}

	applogger.Infof("验证码背景图系统参数创建完成")
	return nil
}

// createCaptchaBackgroundMenus 创建验证码背景图管理菜单
// TODO: 此函数未使用，如需要启用验证码背景图功能，请取消注释并调用此函数
/*
func createCaptchaBackgroundMenus(db *gorm.DB) error {
	// 先查找系统管理菜单的ID
	var systemMenu models.Menu
	if err := db.Where("menu_name = ? AND menu_type = ?", "系统管理", "M").First(&systemMenu).Error; err != nil {
		log.Printf("未找到系统管理菜单，跳过创建验证码背景图子菜单: %v", err)
		return nil
	}

	// 检查验证码背景图菜单是否已存在
	var existingCount int64
	db.Model(&models.Menu{}).Where("menu_name = ?", "验证码背景图").Count(&existingCount)
	if existingCount > 0 {
		log.Println("验证码背景图菜单已存在，跳过创建")
		return nil
	}

	menus := []models.Menu{
		{
			MenuName:  "验证码背景图",
			ParentID:  &systemMenu.ID,
			OrderNum:  10,
			Path:      func() *string { s := "captcha-background"; return &s }(),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeDir,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     NULL_STRING_PTR(""),
			Icon:      func() *string { s := "picture"; return &s }(),
			Remark:    "验证码背景图管理",
		},
		{
			MenuName:  "背景图查询",
			ParentID:  nil, // 稍后更新
			OrderNum:  1,
			Path:      func() *string { s := "background"; return &s }(),
			Component: func() *string { s := "system/captcha-background/index"; return &s }(),
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := "system:captchaBackground:list"; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    "验证码背景图查询菜单",
		},
		{
			MenuName:  "背景图新增",
			ParentID:  nil, // 稍后更新
			OrderNum:  2,
			Path:      NULL_STRING_PTR(""),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := "system:captchaBackground:add"; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    "验证码背景图新增按钮",
		},
		{
			MenuName:  "背景图修改",
			ParentID:  nil, // 稍后更新
			OrderNum:  3,
			Path:      NULL_STRING_PTR(""),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := "system:captchaBackground:edit"; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    "验证码背景图修改按钮",
		},
		{
			MenuName:  "背景图删除",
			ParentID:  nil, // 稍后更新
			OrderNum:  4,
			Path:      NULL_STRING_PTR(""),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := "system:captchaBackground:remove"; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    "验证码背景图删除按钮",
		},
	}

	// 首先创建目录菜单
	if err := db.Create(&menus[0]).Error; err != nil {
		return fmt.Errorf("创建验证码背景图目录菜单失败: %w", err)
	}
	log.Printf("创建菜单 %s 成功", menus[0].MenuName)

	// 更新子菜单的ParentID为目录菜单的ID
	catalogID := menus[0].ID
	for i := 1; i < len(menus); i++ {
		menus[i].ParentID = &catalogID
		if err := db.Create(&menus[i]).Error; err != nil {
			return fmt.Errorf("创建菜单 %s 失败: %w", menus[i].MenuName, err)
		}
		log.Printf("创建菜单 %s 成功", menus[i].MenuName)
	}

	log.Println("验证码背景图管理菜单创建完成")
	return nil
}
*/

// NULL_STRING_PTR 返回字符串指针的辅助函数
func NULL_STRING_PTR(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// createDutyManagementMenus 创建值班管理菜单
//
// 已废弃：请使用 migrations/018_unify_duty_menus.sql 迁移文件
//
// 此函数保留仅为向后兼容，不会在菜单已存在时重复创建
// 所有值班管理菜单（值班池管理、排班管理、节假日管理、值班配置、我的值班）
// 现在通过 SQL 迁移文件统一创建
//
// createOperationsManagementMenus 创建运维管理菜单
func createOperationsManagementMenus(db *gorm.DB) error {
	// 检查运维管理菜单是否已存在
	var opsMenu models.Menu
	err := db.Where("menu_name = ? AND menu_type = ?", "运维管理", "M").First(&opsMenu).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("查询运维管理菜单失败: %w", err)
	}

	// 如果运维管理菜单不存在，创建它
	if err == gorm.ErrRecordNotFound {
		opsMenu = models.Menu{
			MenuName:  "运维管理",
			OrderNum:  4,
			Path:      func() *string { s := "ops"; return &s }(),
			Component: NULL_STRING_PTR("Layout"),
			MenuType:  models.MenuTypeDir,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Icon:      func() *string { s := "Control"; return &s }(),
			Remark:    "运维管理目录",
		}
		if err := db.Create(&opsMenu).Error; err != nil {
			return fmt.Errorf("创建运维管理菜单失败: %w", err)
		}
		applogger.Infof("创建菜单 %s 成功", opsMenu.MenuName)
	} else {
		applogger.Infof("运维管理菜单已存在，跳过创建")
	}

	// 定义需要创建的页面菜单（检查是否存在，不存在才创建）
	pageMenus := []struct {
		name      string
		path      string
		component string
		perms     string
		remark    string
		orderNum  int
		icon      string
	}{
		{"楼宇管理", "buildings", "operations/buildings/index", "ops:building:list", "楼宇管理菜单", 1, "BuildOutlined"},
		{"楼层管理", "floors", "operations/floors/index", "ops:floor:list", "楼层管理菜单", 2, "ApartmentOutlined"},
		{"工位管理", "workstations", "operations/workstations/index", "ops:workstation:list", "工位管理菜单", 3, "DesktopOutlined"},
		{"信息点管理", "info-points", "operations/info-points/index", "ops:infopoint:list", "信息点管理菜单", 4, "DotChartOutlined"},
		{"机房管理", "server-rooms", "operations/server-rooms/index", "ops:serverroom:list", "机房管理菜单", 5, "CloudServerOutlined"},
		{"专线管理", "dedicated-lines", "operations/dedicated-lines/index", "ops:dedicatedline:list", "专线管理菜单", 6, "LineChartOutlined"},
		{"机房设备管理", "room-devices", "operations/room-devices/index", "ops:roomdevice:list", "机房设备管理菜单", 7, "AppstoreOutlined"},
	}

	// 存储页面菜单的ID，用于后续创建按钮权限
	menuIDs := make(map[string]string)

	for _, pm := range pageMenus {
		// 检查菜单是否已存在
		var existingMenu models.Menu
		err := db.Where("menu_name = ? AND parent_id = ?", pm.name, opsMenu.ID).First(&existingMenu).Error

		if err == nil {
			// 菜单已存在，使用现有菜单ID，跳过创建
			menuIDs[pm.name] = existingMenu.ID
			applogger.Infof("菜单 %s 已存在，跳过创建", pm.name)
			continue
		}
		// OC-M-MENUSEED 修复:err != nil 时必须显式区分 ErrRecordNotFound,
		// 真实 DB 错误直接 return,不再 fallthrough 到 Create 制造重复菜单。
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询菜单 %s 失败: %w", pm.name, err)
		}

		// 菜单不存在，创建新菜单
		menu := models.Menu{
			MenuName:  pm.name,
			ParentID:  &opsMenu.ID,
			OrderNum:  pm.orderNum,
			Path:      func() *string { s := pm.path; return &s }(),
			Component: func() *string { s := pm.component; return &s }(),
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := pm.perms; return &s }(),
			Icon:      func() *string { s := pm.icon; return &s }(),
			Remark:    pm.remark,
		}
		if err := db.Create(&menu).Error; err != nil {
			return fmt.Errorf("创建菜单 %s 失败: %w", pm.name, err)
		}
		menuIDs[pm.name] = menu.ID
		applogger.Infof("创建菜单 %s 成功", pm.name)
	}

	// 定义需要创建的按钮权限菜单
	buttonMenus := []struct {
		parent   string
		name     string
		perms    string
		remark   string
		orderNum int
	}{
		// 楼宇管理的按钮
		{"楼宇管理", "楼宇查询", "ops:building:query", "楼宇查询", 1},
		{"楼宇管理", "楼宇新增", "ops:building:add", "楼宇新增", 2},
		{"楼宇管理", "楼宇修改", "ops:building:edit", "楼宇修改", 3},
		{"楼宇管理", "楼宇删除", "ops:building:delete", "楼宇删除", 4},
		// 楼层管理的按钮
		{"楼层管理", "楼层查询", "ops:floor:query", "楼层查询", 1},
		{"楼层管理", "楼层新增", "ops:floor:add", "楼层新增", 2},
		{"楼层管理", "楼层修改", "ops:floor:edit", "楼层修改", 3},
		{"楼层管理", "楼层删除", "ops:floor:delete", "楼层删除", 4},
		// 工位管理的按钮
		{"工位管理", "工位查询", "ops:workstation:query", "工位查询", 1},
		{"工位管理", "工位新增", "ops:workstation:add", "工位新增", 2},
		{"工位管理", "工位修改", "ops:workstation:edit", "工位修改", 3},
		{"工位管理", "工位删除", "ops:workstation:delete", "工位删除", 4},
		// 信息点管理的按钮
		{"信息点管理", "信息点查询", "ops:infopoint:query", "信息点查询", 1},
		{"信息点管理", "信息点新增", "ops:infopoint:add", "信息点新增", 2},
		{"信息点管理", "信息点修改", "ops:infopoint:edit", "信息点修改", 3},
		{"信息点管理", "信息点删除", "ops:infopoint:delete", "信息点删除", 4},
		// 机房管理的按钮
		{"机房管理", "机房查询", "ops:serverroom:query", "机房查询", 1},
		{"机房管理", "机房新增", "ops:serverroom:add", "机房新增", 2},
		{"机房管理", "机房修改", "ops:serverroom:edit", "机房修改", 3},
		{"机房管理", "机房删除", "ops:serverroom:delete", "机房删除", 4},
		// 专线管理的按钮
		{"专线管理", "专线查询", "ops:dedicatedline:query", "专线查询", 1},
		{"专线管理", "专线新增", "ops:dedicatedline:add", "专线新增", 2},
		{"专线管理", "专线修改", "ops:dedicatedline:edit", "专线修改", 3},
		{"专线管理", "专线删除", "ops:dedicatedline:delete", "专线删除", 4},
		// 机房设备管理的按钮
		{"机房设备管理", "设备查询", "ops:roomdevice:query", "设备查询", 1},
		{"机房设备管理", "设备新增", "ops:roomdevice:add", "设备新增", 2},
		{"机房设备管理", "设备修改", "ops:roomdevice:edit", "设备修改", 3},
		{"机房设备管理", "设备删除", "ops:roomdevice:delete", "设备删除", 4},
	}

	for _, bm := range buttonMenus {
		// 检查按钮菜单是否已存在
		var existingButton models.Menu
		parentMenuID := menuIDs[bm.parent]
		// OC-M-MENUSEED 修复:父菜单 ID 缺失(上层页面菜单未创建成功)
		// 时直接报错,不再用空 parent_id 触发无效 UUID 插入。
		if parentMenuID == "" {
			return fmt.Errorf("按钮菜单 %s 的父菜单 %s 不存在", bm.name, bm.parent)
		}
		err := db.Where("menu_name = ? AND parent_id = ?", bm.name, parentMenuID).First(&existingButton).Error

		if err == nil {
			// 按钮已存在，跳过创建
			applogger.Infof("按钮菜单 %s 已存在，跳过创建", bm.name)
			continue
		}
		// OC-M-MENUSEED 修复:与页面循环一致,真实 DB 错误不再 fallthrough。
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询按钮菜单 %s 失败: %w", bm.name, err)
		}

		// 按钮不存在，创建新按钮
		menu := models.Menu{
			MenuName:  bm.name,
			ParentID:  &parentMenuID,
			OrderNum:  bm.orderNum,
			Path:      NULL_STRING_PTR(""),
			Component: NULL_STRING_PTR(""),
			MenuType:  models.MenuTypeButton,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     func() *string { s := bm.perms; return &s }(),
			Icon:      NULL_STRING_PTR(""),
			Remark:    bm.remark,
		}
		if err := db.Create(&menu).Error; err != nil {
			return fmt.Errorf("创建按钮菜单 %s 失败: %w", bm.name, err)
		}
		applogger.Infof("创建按钮菜单 %s 成功", bm.name)
	}

	applogger.Infof("运维管理菜单创建完成")
	return nil
}

// createRequestEncryptionToggleConfig 创建请求加密开关配置参数
func createRequestEncryptionToggleConfig(db *gorm.DB) error {
	// 检查配置是否已存在
	var count int64
	err := db.Table("sys_config").
		Where("config_key = ?", "sys.request.encryption.enabled").
		Count(&count).Error

	if err != nil {
		applogger.Warnf("查询请求加密开关配置失败: %v", err)
		return err
	}

	if count > 0 {
		applogger.Infof("请求加密开关配置已存在，跳过初始化")
		return nil
	}

	// 插入默认配置：true (启用)
	config := models.Config{
		ConfigName:  "请求加密开关",
		ConfigKey:   "sys.request.encryption.enabled",
		ConfigValue: "true",
		ConfigType:  models.ConfigTypeYes,
		IsSystem:    models.ConfigIsSystemYes,
		Remark:      "控制请求体加密功能的启停（true=启用，false=停用），修改后立即生效",
	}

	applogger.Infof("尝试插入请求加密开关配置...")
	if err := db.Create(&config).Error; err != nil {
		applogger.Warnf("创建请求加密开关配置失败: %v", err)
		return err
	}

	applogger.Infof("请求加密开关配置已创建（默认启用）")
	return nil
}

// createADAuthConfig 创建AD认证配置参数
func createADAuthConfig(db *gorm.DB) error {
	authConfigs := []models.Config{
		{
			ConfigName:  "AD认证启用",
			ConfigKey:   "sys.auth.ad.enabled",
			ConfigValue: "false",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "是否启用AD域控认证（true/false）",
		},
		{
			ConfigName:  "默认认证模式",
			ConfigKey:   "sys.auth.default.mode",
			ConfigValue: "local",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemYes,
			Remark:      "默认认证模式：local=本地, ad=AD, hybrid=混合",
		},
		{
			ConfigName:  "AD配置ID",
			ConfigKey:   "sys.auth.ad.config_id",
			ConfigValue: "",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      "AD域控配置ID（为空则使用第一个启用的配置）",
		},
		{
			ConfigName:  "AD用户默认角色",
			ConfigKey:   "sys.auth.ad.default_role_id",
			ConfigValue: "",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      "AD用户首次登录时分配的默认角色ID",
		},
		{
			ConfigName:  "AD用户默认部门",
			ConfigKey:   "sys.auth.ad.default_dept_id",
			ConfigValue: "",
			ConfigType:  models.ConfigTypeYes,
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      "AD用户首次登录时分配的默认部门ID",
		},
	}

	for _, cfg := range authConfigs {
		var count int64
		err := db.Table("sys_config").
			Where("config_key = ?", cfg.ConfigKey).
			Count(&count).Error

		if err != nil {
			applogger.Warnf("查询AD认证配置 %s 失败: %v", cfg.ConfigKey, err)
			return err
		}

		if count > 0 {
			applogger.Infof("AD认证配置 %s 已存在，跳过", cfg.ConfigKey)
			continue
		}

		if err := db.Create(&cfg).Error; err != nil {
			applogger.Warnf("创建AD认证配置 %s 失败: %v", cfg.ConfigKey, err)
			return err
		}

		applogger.Infof("AD认证配置 %s 已创建", cfg.ConfigKey)
	}

	return nil
}
