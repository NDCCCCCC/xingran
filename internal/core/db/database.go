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
	); err != nil {
		return err
	}

	log.Println("所有表迁移成功")
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

	// 创建默认菜单
	if err := createDefaultMenu(db); err != nil {
		return fmt.Errorf("创建默认菜单失败: %w", err)
	}

	// 创建角色菜单关联
	if err := createRoleMenuRelations(db); err != nil {
		return fmt.Errorf("创建角色菜单关联失败: %w", err)
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
			{
				MenuName:   "部门管理",
				ParentID:   systemID,
				OrderNum:   4,
				Path:       func() *string { s := "system/dept"; return &s }(),
				Component:  func() *string { s := "system/dept/index"; return &s }(),
				MenuType:   models.MenuType("C"),
				Visible:    models.VisibleShow,
				Status:     models.MenuStatusNormal,
				Perms:      func() *string { s := "system:dept:list"; return &s }(),
				Icon:       func() *string { s := "tree"; return &s }(),
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


// createRoleMenuRelations 创建角色菜单关联
func createRoleMenuRelations(db *gorm.DB) error {
	// 获取admin角色
	var adminRole models.Role
	if err := db.Where("role_key = ?", "admin").First(&adminRole).Error; err != nil {
		log.Printf("未找到admin角色: %v", err)
		return nil // 如果角色不存在，跳过
	}

	// 检查是否已有角色菜单关联
	var count int64
	db.Table("sys_role_menu").Where("role_id = ?", adminRole.ID).Count(&count)
	if count > 0 {
		log.Println("角色菜单关联已存在，跳过创建")
		return nil
	}

	// 获取所有菜单ID
	var menus []models.Menu
	if err := db.Select("id").Find(&menus).Error; err != nil {
		return fmt.Errorf("获取菜单列表失败: %w", err)
	}

	// 为admin角色创建所有菜单的权限关联
	for _, menu := range menus {
		roleMenu := models.RoleMenu{
			RoleID: adminRole.ID,
			MenuID: menu.ID,
		}
		if err := db.Create(&roleMenu).Error; err != nil {
			log.Printf("创建角色菜单关联失败 (角色: %s, 菜单: %s): %v", adminRole.ID, menu.ID, err)
			continue
		}
	}

	log.Printf("为admin角色创建了 %d 个菜单权限", len(menus))
	return nil
}
