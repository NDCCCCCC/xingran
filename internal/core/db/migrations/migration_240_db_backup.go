package migrations

import (
	"fmt"
	"log"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)

// Migrate240DBBackup —— quick 261110-tbq：全库备份与恢复菜单/权限/配置 seed + 两表 DDL。
//
// 背景：migration_207 有「系统管理根已存在就整体跳过 seed」快速路径，
// menu_catalog_seed.sql 只对全新空库生效；存量库升级必须由此独立迁移补种。
//
// 幂等：全部按固定 UUID / perms / config_key 先查后建，重复执行无副作用；
// 已存在则跳过（绝不覆盖管理员在参数管理页改过的值）。
// 失败非阻断（调用方 WARN），下次启动重试。
//
// 不 INSERT sys_role_menu（与 169/195/228/229 一致：admin 超管旁路立即可见，
// 非 admin 角色由管理员手动授权）。
func Migrate240DBBackup(db *gorm.DB) error {
	log.Println("Running migration 240: 全库备份菜单/权限/配置 seed")

	// ---------- 1. C 型菜单 + 3 个 F 型按钮（固定 UUID，与 seed.sql 完全一致） ----------
	const (
		menuID       = "a1b2c3d4-0001-4000-8000-000000000240"
		btnListID    = "a1b2c3d4-0002-4000-8000-000000000240"
		btnExportID  = "a1b2c3d4-0003-4000-8000-000000000240"
		btnRestoreID = "a1b2c3d4-0004-4000-8000-000000000240"
		component    = "system/backup/index"
		pagePerms    = "system:backup"
	)

	// F_TYPE_SEED_BEGIN
	// 全库备份 F 型按钮权限（供 scripts/check-permission-codes.mjs 识别）：
	//   "system:backup:list"
	//   "system:backup:export"
	//   "system:backup:restore"
	// F_TYPE_SEED_END

	var menuCount int64
	if err := db.Model(&models.Menu{}).Where("id = ?", menuID).Count(&menuCount).Error; err != nil {
		return fmt.Errorf("migration 240: 查询 C 菜单失败: %w", err)
	}
	if menuCount == 0 {
		path, comp, perms, icon := "backup", component, pagePerms, "CloudDownloadOutlined"
		parentID := canonicalSystemRootID
		remark := "数据库全量备份导出/导入恢复（migration_240）"
		menu := models.Menu{
			MenuName:  "数据备份与恢复",
			ParentID:  &parentID,
			OrderNum:  12,
			Path:      &path,
			Component: &comp,
			MenuType:  models.MenuTypeMenu,
			Visible:   models.VisibleShow,
			Status:    models.MenuStatusNormal,
			Perms:     &perms,
			Icon:      &icon,
			Remark:    remark,
		}
		menu.ID = menuID
		if err := db.Create(&menu).Error; err != nil {
			return fmt.Errorf("migration 240: 创建 C 菜单失败: %w", err)
		}
		applogger.Infof("[迁移] sys_menu seed: 数据备份与恢复 (perms=%s, parent=%s)", perms, parentID)
	}

	buttons := []struct {
		id, name, perms string
		order           int
	}{
		{btnListID, "备份列表查询", "system:backup:list", 1},
		{btnExportID, "备份导出下载", "system:backup:export", 2},
		{btnRestoreID, "备份恢复", "system:backup:restore", 3},
	}
	for _, b := range buttons {
		var cnt int64
		if err := db.Model(&models.Menu{}).Where("id = ?", b.id).Count(&cnt).Error; err != nil {
			return fmt.Errorf("migration 240: 查询 F 按钮 %s 失败: %w", b.perms, err)
		}
		if cnt > 0 {
			continue
		}
		empty, icon, perms := "", "#", b.perms
		parentID := menuID
		btn := models.Menu{
			MenuName: b.name,
			ParentID: &parentID,
			OrderNum: b.order,
			Path:     &empty,
			MenuType: models.MenuTypeButton,
			Visible:  models.VisibleHidden,
			Status:   models.MenuStatusNormal,
			Perms:    &perms,
			Icon:     &icon,
		}
		btn.ID = b.id
		if err := db.Create(&btn).Error; err != nil {
			return fmt.Errorf("migration 240: 创建 F 按钮 %s 失败: %w", b.perms, err)
		}
	}

	// ---------- 2. sys_config 种子（ConfigTypeNo + IsSystemNo → 参数管理页可编辑） ----------
	configs := []struct{ key, value, name, remark string }{
		{"sys.database.backup.enabled", "false", "全库备份-启用定期备份", "是否启用 cron 自动全库备份；true/false"},
		{"sys.database.backup.cron", "0 0 2 * * *", "全库备份-周期表达式", "6 字段含秒(秒 分 时 日 月 周)，Asia/Shanghai；修改后下一个 tick 内生效，无需重启"},
		{"sys.database.backup.retention", "7", "全库备份-保留份数", "本地与远程保留的最新备份数量，超出自动清理"},
		{"sys.database.backup.storage", "local", "全库备份-存储后端", "local=程序部署服务器本地磁盘；oss/ftp/smb=远程存储"},
		{"sys.database.backup.oss.endpoint", "", "全库备份-OSS Endpoint", "S3 兼容对象存储 endpoint（不含协议头）"},
		{"sys.database.backup.oss.bucket", "", "全库备份-OSS Bucket", "对象存储桶名"},
		{"sys.database.backup.oss.accessKey", "", "全库备份-OSS AccessKey", "对象存储访问密钥（敏感，页面脱敏展示）"},
		{"sys.database.backup.oss.secretKey", "", "全库备份-OSS SecretKey", "对象存储私密密钥（敏感，页面脱敏展示）"},
		{"sys.database.backup.oss.useSSL", "true", "全库备份-OSS 使用 SSL", "true/false"},
		{"sys.database.backup.ftp.host", "", "全库备份-FTP 主机", "FTP 服务器地址"},
		{"sys.database.backup.ftp.port", "21", "全库备份-FTP 端口", "FTP 服务端口"},
		{"sys.database.backup.ftp.user", "", "全库备份-FTP 用户", "FTP 登录用户"},
		{"sys.database.backup.ftp.password", "", "全库备份-FTP 密码", "FTP 登录密码（敏感，页面脱敏展示）"},
		{"sys.database.backup.ftp.remoteDir", "/", "全库备份-FTP 远程目录", "FTP 上的备份存放目录"},
		{"sys.database.backup.smb.path", "", "全库备份-SMB 本地挂载路径", "部署机 OS 已挂载 SMB 共享后的本地目录绝对路径；Go 端当本地磁盘使用"},
		{"sys.database.backup.pg_dump_path", "", "全库备份-pg_dump 可执行路径", "部署机未把 PostgreSQL 客户端加入 PATH 时，填 pg_dump 绝对路径；留空按 PATH 查找；修改后无需重启"},
		{"sys.database.backup.pg_restore_path", "", "全库备份-pg_restore 可执行路径", "部署机未把 PostgreSQL 客户端加入 PATH 时，填 pg_restore 绝对路径；留空按 PATH 查找；修改后无需重启"},
	}
	for _, c := range configs {
		var cnt int64
		if err := db.Model(&models.Config{}).Where("config_key = ?", c.key).Count(&cnt).Error; err != nil {
			return fmt.Errorf("migration 240: 查询配置 %s 失败: %w", c.key, err)
		}
		if cnt > 0 { // 幂等：绝不覆盖管理员已改值
			continue
		}
		cfg := &models.Config{
			ConfigName:  c.name,
			ConfigKey:   c.key,
			ConfigValue: c.value,
			ConfigType:  models.ConfigTypeNo,
			IsSystem:    models.ConfigIsSystemNo,
			Remark:      c.remark,
		}
		if err := db.Create(cfg).Error; err != nil {
			return fmt.Errorf("migration 240: 创建配置 %s 失败: %w", c.key, err)
		}
	}

	// ---------- 3. PG 分支建表（sqlite 由 MigrateModelList 的 AutoMigrate 建表） ----------
	if isPostgreSQL(db) {
		const ddl = `
CREATE TABLE IF NOT EXISTS sys_db_backup (
    id uuid PRIMARY KEY,
    created_at timestamptz,
    updated_at timestamptz,
    deleted_at timestamptz,
    created_by varchar(64),
    updated_by varchar(64),
    version integer DEFAULT 0,
    file_name varchar(255) NOT NULL,
    file_path varchar(1024) NOT NULL,
    storage_type varchar(20) NOT NULL,
    remote_path varchar(1024),
    db_type varchar(20) NOT NULL,
    file_size bigint DEFAULT 0,
    sha256 varchar(64),
    schema_version varchar(20),
    table_count integer DEFAULT 0,
    source varchar(20) NOT NULL,
    status varchar(20) NOT NULL,
    triggered_by varchar(64),
    remark varchar(500)
);
CREATE INDEX IF NOT EXISTS idx_sys_db_backup_deleted_at ON sys_db_backup (deleted_at);
CREATE INDEX IF NOT EXISTS idx_sys_db_backup_source ON sys_db_backup (source);
CREATE INDEX IF NOT EXISTS idx_sys_db_backup_status ON sys_db_backup (status);
CREATE INDEX IF NOT EXISTS idx_sys_db_backup_storage_type ON sys_db_backup (storage_type);
CREATE TABLE IF NOT EXISTS sys_db_backup_task (
    id uuid PRIMARY KEY,
    task_type varchar(20) NOT NULL,
    backup_id uuid,
    status varchar(20) NOT NULL,
    progress integer DEFAULT 0,
    stage varchar(64),
    error_message text,
    started_at timestamptz,
    completed_at timestamptz,
    created_at timestamptz,
    updated_at timestamptz,
    created_by varchar(64)
);
CREATE INDEX IF NOT EXISTS idx_sys_db_backup_task_status ON sys_db_backup_task (status);
CREATE INDEX IF NOT EXISTS idx_sys_db_backup_task_type ON sys_db_backup_task (task_type);
CREATE INDEX IF NOT EXISTS idx_sys_db_backup_task_backup_id ON sys_db_backup_task (backup_id);
`
		if err := db.Exec(ddl).Error; err != nil {
			return fmt.Errorf("migration 240: 建表失败: %w", err)
		}
	}

	applogger.Infof("Migration 240: 全库备份菜单/权限/配置/表 DDL 完成")
	return nil
}
