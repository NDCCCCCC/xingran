package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/core/security"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
)

// Database 数据库管理器
type Database struct {
	DB *gorm.DB
	Type string
}

// NewDatabase 创建数据库连接
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	// 确定数据库类型
	dbType := "sqlite"
	if cfg.Host != "" && cfg.Port > 0 {
		dbType = "postgres"
	}

	// 根据类型创建连接
	var db *gorm.DB
	var err error

	switch dbType {
	case "sqlite":
		db, err = createSQLiteConnection(cfg)
	case "postgres":
		db, err = createPostgresConnection(cfg)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}

	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 配置GORM
	if err := configureGORM(db); err != nil {
		return nil, fmt.Errorf("配置GORM失败: %w", err)
	}

	return &Database{
		DB:   db,
		Type: dbType,
	}, nil
}

// createSQLiteConnection 创建SQLite连接
func createSQLiteConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 确保数据目录存在
	dbPath := cfg.Host
	if dbPath == "" {
		dbPath = "./data/ruoyi.db"
	}

	// 配置SQLite选项
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	db, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接SQLite失败: %w", err)
	}

	log.Printf("SQLite连接成功: %s", dbPath)
	return db, nil
}

// createPostgresConnection 创建PostgreSQL连接
func createPostgresConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 首先连接到postgres数据库创建目标数据库
	adminDSN := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode)

	// 尝试创建数据库
	if err := createDatabaseIfNotExists(adminDSN, cfg.DBName); err != nil {
		log.Printf("创建数据库失败: %v", err)
	}

	dsn := cfg.GetDSN()

	// 配置GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接PostgreSQL失败: %w", err)
	}

	// 获取底层的sql.DB对象进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	log.Println("PostgreSQL连接成功")
	return db, nil
}

// configureGORM 配置GORM
func configureGORM(db *gorm.DB) error {
	// 配置表名前缀
	// 对于SQLite，不需要额外配置
	return nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	if d.DB != nil {
		sqlDB, err := d.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// GetDB 获取数据库实例
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

// AutoMigrate 自动迁移所有模型
func (d *Database) AutoMigrate() error {
	// 迁移所有表
	if err := d.DB.AutoMigrate(
		&models.User{},
		&models.UserRole{},
		&models.Department{},
		&models.Role{},
		&models.Menu{},
		&models.Post{},
		&models.RoleMenu{},
		&models.RoleDept{},
		&models.UserPost{},
		&models.Job{},
		&models.JobLog{},
		// 网络设备相关模型
		&models.AuthCredential{},
		&models.NetworkDevice{},
		&models.ConfigTemplate{},
		&models.ConfigExecution{},
		&models.ConfigExecutionDetail{},
		&models.ConfigBackup{},
		&models.DeviceMACAddress{},
		&models.DevicePortStatus{},
		&models.DeviceDiscovery{},
		&models.DeviceEnrichmentTask{},
	); err != nil {
		return err
	}

	log.Println("所有表迁移成功")

	// 执行凭证模型重新设计迁移
	if err := d.migrateCredentialModel(); err != nil {
		log.Printf("凭证模型迁移失败: %v", err)
		// 迁移失败不中断启动，只记录日志
	}

	return nil
}

// migrateCredentialModel 迁移凭证模型（从 credential_type 到 protocol_type）
// 这个迁移将凭证从 SSH/Telnet/SNMP 分类改为统一的凭证模型
func (d *Database) migrateCredentialModel() error {
	// 只对 PostgreSQL 执行 SQL 迁移
	if d.Type != "postgres" {
		log.Println("非 PostgreSQL 数据库，跳过凭证模型 SQL 迁移")
		return nil
	}

	// 检查迁移是否已经执行过（通过检查 protocol_type 列是否存在）
	var columnExists bool
	err := d.DB.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'sys_auth_credential' AND column_name = 'protocol_type'
		)
	`).Scan(&columnExists).Error

	if err != nil {
		return fmt.Errorf("检查迁移状态失败: %w", err)
	}

	// 如果 protocol_type 列已存在，说明迁移已执行过
	if columnExists {
		log.Println("凭证模型迁移已执行，跳过")
		return nil
	}

	log.Println("开始执行凭证模型迁移...")

	// 开始事务
	tx := d.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开启事务失败: %w", tx.Error)
	}

	// 步骤1：添加新列
	if err := tx.Exec("ALTER TABLE sys_auth_credential ADD COLUMN IF NOT EXISTS protocol_type VARCHAR(10)").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("添加 protocol_type 列失败: %w", err)
	}

	if err := tx.Exec("ALTER TABLE sys_auth_credential ADD COLUMN IF NOT EXISTS snmp_communities TEXT[]").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("添加 snmp_communities 列失败: %w", err)
	}

	// 步骤2：根据旧的 credential_type 设置 protocol_type
	if err := tx.Exec("UPDATE sys_auth_credential SET protocol_type = 'ssh' WHERE credential_type = 'ssh'").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("迁移 SSH 凭证失败: %w", err)
	}

	if err := tx.Exec("UPDATE sys_auth_credential SET protocol_type = 'telnet' WHERE credential_type = 'telnet'").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("迁移 Telnet 凭证失败: %w", err)
	}

	if err := tx.Exec("UPDATE sys_auth_credential SET protocol_type = 'ssh' WHERE credential_type = 'snmp' AND protocol_type IS NULL").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("迁移 SNMP 凭证失败: %w", err)
	}

	// 步骤3：将旧的 snmp_community 转换为数组
	if err := tx.Exec("UPDATE sys_auth_credential SET snmp_communities = ARRAY[snmp_community] WHERE snmp_community IS NOT NULL AND snmp_community != '' AND snmp_community != ' '").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("转换 SNMP community 失败: %w", err)
	}

	// 步骤4：初始化空数组
	if err := tx.Exec("UPDATE sys_auth_credential SET snmp_communities = ARRAY[]::TEXT[] WHERE snmp_communities IS NULL").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("初始化 SNMP communities 数组失败: %w", err)
	}

	// 步骤5：设置默认 protocol_type
	if err := tx.Exec("UPDATE sys_auth_credential SET protocol_type = 'ssh' WHERE protocol_type IS NULL").Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("设置默认 protocol_type 失败: %w", err)
	}

	// 步骤6：删除旧列（先注释掉，等用户确认后再执行）
	// if err := tx.Exec("ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS credential_type").Error; err != nil {
	// 	tx.Rollback()
	// 	return fmt.Errorf("删除 credential_type 列失败: %w", err)
	// }
	//
	// if err := tx.Exec("ALTER TABLE sys_auth_credential DROP COLUMN IF EXISTS snmp_community").Error; err != nil {
	// 	tx.Rollback()
	// 	return fmt.Errorf("删除 snmp_community 列失败: %w", err)
	// }

	// 步骤7：添加约束
	if err := tx.Exec("ALTER TABLE sys_auth_credential ADD CONSTRAINT chk_protocol_type CHECK (protocol_type IN ('ssh', 'telnet'))").Error; err != nil {
		// 约束可能已存在，忽略错误
		log.Printf("添加 protocol_type 约束失败（可能已存在）: %v", err)
	}

	if err := tx.Exec("ALTER TABLE sys_auth_credential ADD CONSTRAINT chk_snmp_version CHECK (snmp_version IN ('v1', 'v2c', 'v3'))").Error; err != nil {
		// 约束可能已存在，忽略错误
		log.Printf("添加 snmp_version 约束失败（可能已存在）: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交迁移事务失败: %w", err)
	}

	log.Println("凭证模型迁移完成（旧列保留，请手动验证后删除）")
	log.Println("验证通过后请手动执行：")
	log.Println("  ALTER TABLE sys_auth_credential DROP COLUMN credential_type;")
	log.Println("  ALTER TABLE sys_auth_credential DROP COLUMN snmp_community;")

	return nil
}

// InitData 初始化基础数据
func (d *Database) InitData() error {
	// 两种数据库都使用相同的初始化逻辑
	return initData(d.DB)
}

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

	log.Println("基础数据初始化完成")
	return nil
}

// createDefaultDept 创建默认部门
func createDefaultDept(db *gorm.DB) error {
	// 检查是否已有部门数据
	var count int64
	db.Model(&models.Department{}).Count(&count)
	if count > 0 {
		log.Println("部门数据已存在，跳过初始化")
		return nil
	}

	// 创建顶级部门
	topDept := models.Department{
		DeptName:  "若依科技有限公司",
		OrderNum:  1,
		Leader:    func() *string { s := "若依"; return &s }(),
		Phone:     func() *string { s := "15888888888"; return &s }(),
		Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
		Status:    models.DeptStatusNormal,
		Remark:    "",
	}

	if err := db.Create(&topDept).Error; err != nil {
		return fmt.Errorf("创建顶级部门失败: %w", err)
	}

	// 创建子部门
	subDepts := []models.Department{
		{
			DeptName:  "深圳总公司",
			ParentID:  &topDept.ID,
			Ancestors: topDept.ID,
			OrderNum:  1,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
		{
			DeptName:  "长沙分公司",
			ParentID:  &topDept.ID,
			Ancestors: topDept.ID,
			OrderNum:  2,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
	}

	var shenzhenDeptID string
	for _, dept := range subDepts {
		if err := db.Create(&dept).Error; err != nil {
			return fmt.Errorf("创建部门 %s 失败: %w", dept.DeptName, err)
		}
		if dept.DeptName == "深圳总公司" {
			shenzhenDeptID = dept.ID
		}
		log.Printf("创建部门 %s 成功", dept.DeptName)
	}

	// 创建深圳总公司的子部门
	shenzhenSubDepts := []models.Department{
		{
			DeptName:  "研发部门",
			ParentID:  &shenzhenDeptID,
			Ancestors: topDept.ID + "," + shenzhenDeptID,
			OrderNum:  1,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
		{
			DeptName:  "市场部门",
			ParentID:  &shenzhenDeptID,
			Ancestors: topDept.ID + "," + shenzhenDeptID,
			OrderNum:  2,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
		{
			DeptName:  "测试部门",
			ParentID:  &shenzhenDeptID,
			Ancestors: topDept.ID + "," + shenzhenDeptID,
			OrderNum:  3,
			Leader:    func() *string { s := "若依"; return &s }(),
			Phone:     func() *string { s := "15888888888"; return &s }(),
			Email:     func() *string { s := "ruoyi@qq.com"; return &s }(),
			Status:    models.DeptStatusNormal,
			Remark:    "",
		},
	}

	for _, dept := range shenzhenSubDepts {
		if err := db.Create(&dept).Error; err != nil {
			return fmt.Errorf("创建部门 %s 失败: %w", dept.DeptName, err)
		}
		log.Printf("创建部门 %s 成功", dept.DeptName)
	}

	log.Println("默认部门创建完成")
	return nil
}

// createDefaultUser 创建默认管理员用户
func createDefaultUser(db *gorm.DB) error {
	var count int64

	// 检查是否已存在管理员用户
	db.Model(&models.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		return nil // 已存在，跳过
	}

	// 使用新的SM3密码管理器
	pwdManager := security.NewPasswordManager(nil)

	// 生成密码哈希
	passwordHash, err := pwdManager.HashPassword("admin123")
	if err != nil {
		return err
	}

	// 创建默认管理员用户
	user := models.User{
		Username: "admin",
		Password: passwordHash,
		Salt:     "default",
		Nickname: func() *string { s := "超级管理员"; return &s }(),
		Email:    func() *string { s := "admin@ruoyi.com"; return &s }(),
		Gender:   models.GenderMale,
		Status:   models.UserStatusEnabled,
		DeptName: func() *string { s := "总公司"; return &s }(),
		Roles:    []string{"admin"},
	}

	if err := db.Create(&user).Error; err != nil {
		return fmt.Errorf("创建默认用户失败: %w", err)
	}

	log.Println("创建默认管理员用户成功")
	return nil
}

// createDefaultRole 创建默认角色
func createDefaultRole(db *gorm.DB) error {
	roles := []models.Role{
		{
			RoleName:           "超级管理员",
			RoleKey:            "admin",
			RoleSort:           1,
			DataScope:          models.DataScopeAll,
			MenuCheckStrictly:   true,
			DeptCheckStrictly:   true,
			Status:             models.RoleStatusEnabled,
			Remark:             "超级管理员",
		},
		{
			RoleName:           "普通用户",
			RoleKey:            "user",
			RoleSort:           2,
			DataScope:          models.DataScopeSelf,
			MenuCheckStrictly:   true,
			DeptCheckStrictly:   true,
			Status:             models.RoleStatusEnabled,
			Remark:             "普通用户",
		},
	}

	for _, role := range roles {
		var count int64
		// 检查角色是否已存在（通过role_key）
		db.Model(&models.Role{}).Where("role_key = ?", role.RoleKey).Count(&count)
		if count > 0 {
			log.Printf("角色 %s (role_key: %s) 已存在，跳过创建", role.RoleName, role.RoleKey)
			continue
		}

		if err := db.Create(&role).Error; err != nil {
			return fmt.Errorf("创建角色 %s 失败: %w", role.RoleName, err)
		}
		log.Printf("创建角色 %s 成功", role.RoleName)
	}

	log.Println("默认角色检查/创建完成")
	return nil
}

// createUserRoleRelations 创建用户角色关联
func createUserRoleRelations(db *gorm.DB) error {
	// 获取默认用户
	var adminUser models.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		log.Printf("未找到管理员用户: %v", err)
		return nil // 如果用户不存在，跳过
	}

	// 获取管理员角色
	var adminRole models.Role
	if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
		log.Printf("未找到管理员角色: %v", err)
		return nil // 如果角色不存在，跳过
	}

	// 检查关联是否已存在
	var count int64
	db.Table("sys_user_role").Where("user_id = ? AND role_id = ?", adminUser.ID, adminRole.ID).Count(&count)
	if count > 0 {
		log.Println("用户角色关联已存在，跳过创建")
		return nil
	}

	// 创建用户角色关联
	if err := db.Exec("INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?)",
		adminUser.ID, adminRole.ID).Error; err != nil {
		return fmt.Errorf("创建用户角色关联失败: %w", err)
	}

	log.Println("创建用户角色关联成功")
	return nil
}

// createDatabaseIfNotExists 如果数据库不存在则创建
func createDatabaseIfNotExists(adminDSN, dbName string) error {
	db, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return fmt.Errorf("连接管理员数据库失败: %w", err)
	}
	defer db.Close()

	// 检查数据库是否存在
	var exists bool
	query := "SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)"
	err = db.QueryRow(query, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("检查数据库是否存在失败: %w", err)
	}

	// 如果数据库不存在，则创建
	if !exists {
		createQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
		_, err = db.Exec(createQuery)
		if err != nil {
			return fmt.Errorf("创建数据库失败: %w", err)
		}
		log.Printf("数据库 %s 创建成功", dbName)
	} else {
		log.Printf("数据库 %s 已存在", dbName)
	}

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
	}

	for _, param := range params {
		var count int64
		db.Model(&models.Config{}).Where("config_key = ?", param.ConfigKey).Count(&count)
		if count > 0 {
			log.Printf("系统参数 %s 已存在，跳过创建", param.ConfigName)
			continue
		}

		if err := db.Create(&param).Error; err != nil {
			return fmt.Errorf("创建系统参数 %s 失败: %w", param.ConfigName, err)
		}
		log.Printf("创建系统参数 %s 成功", param.ConfigName)
	}

	log.Println("网络设备系统参数创建完成")
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
			CronExpression: "0 */5 * * * ?", // 每5分钟执行一次
			Status:         models.JobStatusPause, // 默认暂停，由用户手动启动
			Concurrent:     false, // 禁止并发
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
			log.Printf("定时任务 %s 已存在，跳过创建", job.JobName)
			continue
		}

		if err := db.Create(&job).Error; err != nil {
			return fmt.Errorf("创建定时任务 %s 失败: %w", job.JobName, err)
		}
		log.Printf("创建定时任务 %s 成功", job.JobName)
	}

	log.Println("网络设备定时任务创建完成")
	return nil
}
