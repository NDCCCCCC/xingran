package db

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"golang.org/x/crypto/argon2"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "modernc.org/sqlite"
	"database/sql"
	_ "github.com/lib/pq"
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
	// 临时只迁移核心表，避免UUID默认值问题
	if err := d.DB.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Menu{},
	); err != nil {
		return err
	}

	log.Println("核心表迁移成功")
	return nil
}

// InitData 初始化基础数据
func (d *Database) InitData() error {
	// 两种数据库都使用相同的初始化逻辑
	return initData(d.DB)
}

// initData 初始化基础数据
func initData(db *gorm.DB) error {
	// 创建默认用户
	if err := createDefaultUser(db); err != nil {
		return fmt.Errorf("创建默认用户失败: %w", err)
	}

	// 创建默认角色
	if err := createDefaultRole(db); err != nil {
		return fmt.Errorf("创建默认角色失败: %w", err)
	}

	// 创建默认菜单
	if err := createDefaultMenu(db); err != nil {
		return fmt.Errorf("创建默认菜单失败: %w", err)
	}

	log.Println("基础数据初始化完成")
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

	// 创建密码管理器
	pwdManager := NewSimplePasswordManager()

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

// createDefaultMenu 创建默认菜单
func createDefaultMenu(db *gorm.DB) error {
	// 1. 首先创建或获取顶级菜单
	topMenus := []models.Menu{
		{
			MenuName:   "系统管理",
			ParentID:   nil, // 顶级菜单
			OrderNum:   1,
			MenuType:   models.MenuType("M"),
			Visible:    models.VisibleShow,
			Status:     models.MenuStatusNormal,
			Icon:       func() *string { s := "system"; return &s }(),
		},
		{
			MenuName:   "系统监控",
			ParentID:   nil, // 顶级菜单
			OrderNum:   2,
			MenuType:   models.MenuType("M"),
			Visible:    models.VisibleShow,
			Status:     models.MenuStatusNormal,
			Icon:       func() *string { s := "monitor"; return &s }(),
		},
	}

	// 存储顶级菜单的ID
	topMenuIDs := make(map[string]*string)

	for _, menu := range topMenus {
		var count int64
		// 检查顶级菜单是否已存在
		db.Model(&models.Menu{}).Where("menu_name = ? AND parent_id IS NULL", menu.MenuName).Count(&count)
		if count > 0 {
			// 获取已存在的菜单ID
			var existingMenu models.Menu
			if err := db.Where("menu_name = ? AND parent_id IS NULL", menu.MenuName).First(&existingMenu).Error; err == nil {
				topMenuIDs[menu.MenuName] = &existingMenu.ID
				log.Printf("顶级菜单 %s 已存在，ID: %s", menu.MenuName, existingMenu.ID)
			}
			continue
		}

		// 创建顶级菜单
		if err := db.Create(&menu).Error; err != nil {
			return fmt.Errorf("创建顶级菜单 %s 失败: %w", menu.MenuName, err)
		}
		topMenuIDs[menu.MenuName] = &menu.ID
		log.Printf("创建顶级菜单 %s 成功，ID: %s", menu.MenuName, menu.ID)
	}

	// 2. 创建子菜单
	if systemID, exists := topMenuIDs["系统管理"]; exists {
		systemSubMenus := []models.Menu{
			{
				MenuName:   "用户管理",
				ParentID:   systemID,
				OrderNum:   1,
				Path:       func() *string { s := "system/user"; return &s }(),
				Component:  func() *string { s := "system/user/index"; return &s }(),
				MenuType:   models.MenuType("C"),
				Visible:    models.VisibleShow,
				Status:     models.MenuStatusNormal,
				Perms:      func() *string { s := "system:user:list"; return &s }(),
				Icon:       func() *string { s := "user"; return &s }(),
			},
			{
				MenuName:   "角色管理",
				ParentID:   systemID,
				OrderNum:   2,
				Path:       func() *string { s := "system/role"; return &s }(),
				Component:  func() *string { s := "system/role/index"; return &s }(),
				MenuType:   models.MenuType("C"),
				Visible:    models.VisibleShow,
				Status:     models.MenuStatusNormal,
				Perms:      func() *string { s := "system:role:list"; return &s }(),
				Icon:       func() *string { s := "peoples"; return &s }(),
			},
			{
				MenuName:   "菜单管理",
				ParentID:   systemID,
				OrderNum:   3,
				Path:       func() *string { s := "system/menu"; return &s }(),
				Component:  func() *string { s := "system/menu/index"; return &s }(),
				MenuType:   models.MenuType("C"),
				Visible:    models.VisibleShow,
				Status:     models.MenuStatusNormal,
				Perms:      func() *string { s := "system:menu:list"; return &s }(),
				Icon:       func() *string { s := "tree-table"; return &s }(),
			},
		}

		for _, menu := range systemSubMenus {
			var count int64
			// 检查子菜单是否已存在
			db.Model(&models.Menu{}).Where("menu_name = ? AND parent_id = ?", menu.MenuName, menu.ParentID).Count(&count)
			if count > 0 {
				log.Printf("子菜单 %s 已存在，跳过创建", menu.MenuName)
				continue
			}

			if err := db.Create(&menu).Error; err != nil {
				return fmt.Errorf("创建子菜单 %s 失败: %w", menu.MenuName, err)
			}
			log.Printf("创建子菜单 %s 成功", menu.MenuName)
		}
	}

	// 3. 创建系统监控的子菜单
	if monitorID, exists := topMenuIDs["系统监控"]; exists {
		monitorSubMenus := []models.Menu{
			{
				MenuName:   "在线用户",
				ParentID:   monitorID,
				OrderNum:   1,
				Path:       func() *string { s := "monitor/online"; return &s }(),
				Component:  func() *string { s := "monitor/online/index"; return &s }(),
				MenuType:   models.MenuType("C"),
				Visible:    models.VisibleShow,
				Status:     models.MenuStatusNormal,
				Perms:      func() *string { s := "monitor:online:list"; return &s }(),
				Icon:       func() *string { s := "online"; return &s }(),
			},
		}

		for _, menu := range monitorSubMenus {
			var count int64
			// 检查子菜单是否已存在
			db.Model(&models.Menu{}).Where("menu_name = ? AND parent_id = ?", menu.MenuName, menu.ParentID).Count(&count)
			if count > 0 {
				log.Printf("子菜单 %s 已存在，跳过创建", menu.MenuName)
				continue
			}

			if err := db.Create(&menu).Error; err != nil {
				return fmt.Errorf("创建子菜单 %s 失败: %w", menu.MenuName, err)
			}
			log.Printf("创建子菜单 %s 成功", menu.MenuName)
		}
	}

	log.Println("默认菜单检查/创建完成")
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

// SimplePasswordManager 简单密码管理器
type SimplePasswordManager struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

// NewSimplePasswordManager 创建简单密码管理器
func NewSimplePasswordManager() *SimplePasswordManager {
	return &SimplePasswordManager{
		time:    3,
		memory:  64 * 1024,
		threads: 4,
		keyLen:  32,
	}
}

// HashPassword 哈希密码
func (pm *SimplePasswordManager) HashPassword(password string) (string, error) {
	// 生成随机盐
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成盐失败: %w", err)
	}

	// 使用Argon2哈希密码
	hash := argon2.IDKey([]byte(password), salt, pm.time, pm.memory, pm.threads, pm.keyLen)

	// 编码为Base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// 组合格式: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	format := "$argon2id=v=%d$m=%d,t=%d,p=%d$%s$%s"
	return fmt.Sprintf(format, argon2.Version, pm.memory, pm.time, pm.threads, b64Salt, b64Hash), nil
}

// VerifyPassword 验证密码
func (pm *SimplePasswordManager) VerifyPassword(password, hash string) (bool, error) {
	// 使用字符串分割解析哈希格式
	// 实际格式: $argon2id=v=19$m=65536,t=3,p=4$<salt>$<hash>
	parts := strings.Split(hash, "$")
	if len(parts) != 5 || parts[0] != "" || !strings.HasPrefix(parts[1], "argon2id=") {
		return false, fmt.Errorf("无效的密码哈希格式")
	}

	// 解析版本部分 "argon2id=v=19"
	var version int
	_, err := fmt.Sscanf(parts[1], "argon2id=v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("解析版本失败: %w", err)
	}

	// 解析参数部分 "m=65536,t=3,p=4"
	paramParts := strings.Split(parts[2], ",")
	if len(paramParts) != 3 {
		return false, fmt.Errorf("无效的参数格式")
	}

	var memory, timeVal, threads int
	_, err = fmt.Sscanf(paramParts[0], "m=%d", &memory)
	if err != nil {
		return false, fmt.Errorf("解析内存失败: %w", err)
	}
	_, err = fmt.Sscanf(paramParts[1], "t=%d", &timeVal)
	if err != nil {
		return false, fmt.Errorf("解析时间失败: %w", err)
	}
	_, err = fmt.Sscanf(paramParts[2], "p=%d", &threads)
	if err != nil {
		return false, fmt.Errorf("解析线程数失败: %w", err)
	}

	saltStr := parts[3]
	hashStr := parts[4]

	salt, err := base64.RawStdEncoding.DecodeString(saltStr)
	if err != nil {
		return false, fmt.Errorf("解码盐失败: %w", err)
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hashStr)
	if err != nil {
		return false, fmt.Errorf("解码哈希失败: %w", err)
	}

	// 使用相同的参数哈希输入密码
	comparisonHash := argon2.IDKey([]byte(password), salt, uint32(timeVal), uint32(memory), uint8(threads), uint32(len(hashBytes)))

	// 安全比较哈希
	return subtle.ConstantTimeCompare(hashBytes, comparisonHash) == 1, nil
}