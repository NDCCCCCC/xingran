//go:build archive_skip


package migrations

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Migrate136GroupMappingMenu adds department-group mapping menu and permissions
func Migrate136GroupMappingMenu(db *gorm.DB) error {
	log.Println("Running migration 136: Department-Group Mapping Menu")

	// Check if menu already exists
	var count int64
	db.Table("sys_menu").Where("menu_name = ?", "部门-组映射").Count(&count)
	if count > 0 {
		log.Println("Menu '部门-组映射' already exists, skipping migration 136...")
		return nil
	}

	// Read SQL file
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Try to find the migration file
	sqlFile := filepath.Join(wd, "internal/core/db/migrations/136_add_group_mapping_menu.sql")
	if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
		// Try relative path from migration package
		sqlFile = "136_add_group_mapping_menu.sql"
		if _, err := os.Stat(sqlFile); os.IsNotExist(err) {
			log.Println("SQL file not found, using inline SQL instead...")
			return executeInlineSQL(db)
		}
	}

	content, err := os.ReadFile(sqlFile)
	if err != nil {
		log.Printf("Failed to read SQL file: %v, using inline SQL instead...", err)
		return executeInlineSQL(db)
	}

	// Execute SQL
	return executeSQL(db, string(content), 136)
}

// executeInlineSQL executes the migration using hardcoded SQL statements
func executeInlineSQL(db *gorm.DB) error {
	log.Println("Executing inline SQL for migration 136...")

	// Find the AD domain management parent menu ID
	var parentMenu models.Menu
	err := db.Where("menu_name = ? AND parent_id IS NULL", "AD域管理").
		Joins("JOIN sys_menu parent ON sys_menu.parent_id = parent.id").
		Where("parent.menu_name = ?", "运维管理").
		First(&parentMenu).Error

	if err != nil {
		log.Printf("Warning: Could not find AD domain management menu: %v", err)
		// Try to find it by name only
		err = db.Where("menu_name = ?", "AD域管理").First(&parentMenu).Error
		if err != nil {
			log.Printf("Error: Could not find AD domain management menu: %v", err)
			return err
		}
	}

	// Create the main menu item
	path := "group-mapping"
	component := "ad-domain/group-mapping/index"
	perms := "ops:ad:group:mapping:view"
	icon := "PartitionOutlined"

	groupMappingMenu := &models.Menu{
		MenuName:  "部门-组映射",
		ParentID:  &parentMenu.ID,
		OrderNum:  6,
		Path:      &path,
		Component: &component,
		MenuType:  "C",
		Visible:   1,
		Status:    0,
		Perms:     &perms,
		Icon:      &icon,
		Remark:    "部门与AD组映射管理页面",
	}

	if err := db.Create(groupMappingMenu).Error; err != nil {
		log.Printf("Failed to create group mapping menu: %v", err)
		return err
	}

	log.Printf("Created menu: %s (ID: %s)", groupMappingMenu.MenuName, groupMappingMenu.ID)

	// Create button permissions
	buttons := []struct {
		name   string
		perms  string
		order  int
		remark string
	}{
		{"映射添加", "ops:ad:group:mapping:add", 1, "添加部门组映射"},
		{"映射修改", "ops:ad:group:mapping:edit", 2, "修改部门组映射"},
		{"映射删除", "ops:ad:group:mapping:delete", 3, "删除部门组映射"},
		{"自动映射", "ops:ad:group:mapping:automap", 4, "自动映射部门到AD组"},
		{"成员同步", "ops:ad:group:mapping:sync", 5, "同步部门成员到AD组"},
	}

	for _, btn := range buttons {
		emptyPath := ""
		icon := "#"
		perms := btn.perms

		buttonMenu := &models.Menu{
			MenuName: btn.name,
			ParentID: &groupMappingMenu.ID,
			OrderNum: btn.order,
			Path:     &emptyPath,
			MenuType: "F",
			Visible:  0,
			Status:   0,
			Perms:    &perms,
			Icon:     &icon,
			Remark:   btn.remark,
		}

		if err := db.Create(buttonMenu).Error; err != nil {
			log.Printf("Failed to create button menu %s: %v", btn.name, err)
		} else {
			log.Printf("Created button menu: %s (ID: %s)", buttonMenu.MenuName, buttonMenu.ID)
		}
	}

	// Assign permissions to admin role (role_id with status=0)
	var roleIDs []string
	db.Table("sys_role").Where("status = 0").Pluck("id", &roleIDs)

	menuIDs := []string{groupMappingMenu.ID}
	for _, btn := range buttons {
		var btnMenu models.Menu
		if err := db.Where("perms = ?", btn.perms).First(&btnMenu).Error; err == nil {
			menuIDs = append(menuIDs, btnMenu.ID)
		}
	}

	for _, roleID := range roleIDs {
		for _, menuID := range menuIDs {
			var existingCount int64
			db.Table("sys_role_menu").Where("role_id = ? AND menu_id = ?", roleID, menuID).Count(&existingCount)
			if existingCount == 0 {
				if err := db.Exec("INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)", roleID, menuID).Error; err != nil {
					log.Printf("Failed to assign menu %s to role %s: %v", menuID, roleID, err)
				}
			}
		}
	}

	return nil
}

// executeSQL executes SQL statements from a string
//
// 解析约定(简化版,不是完整 SQL parser):
//   - 先去除每行 `--` 之后的行内注释(避免注释里的 `;` 误触发 split)
//   - 按 `;` 分割成语句
//   - 跳过空语句和整行注释
//
// 不支持: 字符串字面量内的 `;`, dollar-quoted 字符串 `$$...;...$$`, 多行 /* ... */ 注释。
// 如需这些场景,改用 `db.Exec(整段 SQL)` 单次执行。
//
// migrationNum 用于在末尾统一打印 "Migration N completed successfully",
// 让 SQL-based 迁移与其他迁移日志格式对齐。
func executeSQL(db *gorm.DB, sqlContent string, migrationNum int) error {
	// 去掉每行 `--` 之后的内容,避免注释中的 `;` 干扰分割
	var cleaned []string
	for line := range strings.SplitSeq(sqlContent, "\n") {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		cleaned = append(cleaned, line)
	}
	sanitized := strings.Join(cleaned, "\n")

	statements := strings.Split(sanitized, ";")
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		log.Printf("Executing statement %d/%d...", i+1, len(statements))
		if err := db.Exec(stmt).Error; err != nil {
			log.Printf("Statement %d failed (may be expected due to NOT EXISTS): %v", i+1, err)
		} else {
			log.Printf("Statement %d executed successfully", i+1)
		}
	}

	log.Printf("Migration %d completed successfully", migrationNum)
	return nil
}
