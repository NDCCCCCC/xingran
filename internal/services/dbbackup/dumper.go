package dbbackup

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// 数据库类型常量（契约 DatabaseConfig.Type 的取值）
const (
	dbTypePostgres = "postgres"
	dbTypeSQLite   = "sqlite"
)

// excludedPhysicalTables 导出/恢复时一律排除的迁移元数据表与视图/MV 相关表名。
// 参考 PLAN.md 实证约束：按物理库枚举表，排除迁移表、视图/MV、幻影 join 表。
var excludedPhysicalTables = []string{
	"sys_schema_migrations",
	"schema_migrations",
}

// isSQLiteCfg 判断数据库配置是否指向 SQLite（大小写不敏感、容忍 "sqlite3"）
func isSQLiteCfg(cfg DatabaseConfig) bool {
	t := strings.ToLower(strings.TrimSpace(cfg.Type))
	return t == dbTypeSQLite || t == "sqlite3"
}

// productionDumper 生产 DumpExecutor：
//   - PG 调外部 pg_dump / pg_restore（密码走 PGPASSWORD 环境变量）
//   - SQLite 走 GORM 底层 *sql.DB 执行纯 Go VACUUM INTO（不引 CGO）
//
// 持有 gorm.DB 的原因：接口 Dump/Restore/TableNames 的签名只有 dbCfg，
// 没有 *sql.DB 入参；SQLite 的 VACUUM INTO 必须通过 GORM 连接池执行，
// 因此 newService 在 exec=nil 时把注入的 gorm.DB 一并交给本结构。
// 注入 fake DumpExecutor（单测）时不构造本结构，故不需要真实数据库命令。
type productionDumper struct {
	db *gorm.DB
}

// newProductionDumper 构造生产转储执行器
func newProductionDumper(db *gorm.DB) *productionDumper {
	return &productionDumper{db: db}
}

// resolveToolBinary 返回实际要执行的 PostgreSQL 客户端工具路径。
// 服务层 execToolCfg 会先用 resolvePGToolPath 解析并写入 cfg.PgDumpPath /
// PgRestorePath；本函数仅兜底（例如单测直接调 restorePostgres 未走服务层）。
func resolveToolBinary(configured, defaultName string) string {
	if p := strings.TrimSpace(configured); p != "" {
		return p
	}
	return defaultName
}

// redactSecrets 把凭据从外部命令输出中脱敏，确保密码绝不进入错误文本/日志。
func redactSecrets(s string, cfg DatabaseConfig) string {
	for _, secret := range []string{cfg.Password} {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "******")
		}
	}
	return s
}

// pgEnv 构造外部 libpq 命令（pg_dump / pg_restore）的环境变量。
// 密码只经 PGPASSWORD 传递，绝不作为命令行参数，也绝不写入日志。
func pgEnv(cfg DatabaseConfig) []string {
	env := append([]string{}, os.Environ()...)
	env = append(env, "PGPASSWORD="+cfg.Password)
	if strings.TrimSpace(cfg.SSLMode) != "" {
		env = append(env, "PGSSLMODE="+strings.TrimSpace(cfg.SSLMode))
	}
	return env
}

// pgHostArgs 构造 libpq 的连接参数（不含密码）
func pgHostArgs(cfg DatabaseConfig) []string {
	args := []string{"-h", cfg.Host, "-p", strconv.Itoa(cfg.Port), "-U", cfg.User, "-d", cfg.DBName}
	if cfg.Host == "" {
		// 本机 socket 场景：保留 -h "" 由 libpq 走默认，去掉空 host 参数更稳妥
		args = []string{"-p", strconv.Itoa(cfg.Port), "-U", cfg.User, "-d", cfg.DBName}
	}
	if cfg.Port <= 0 {
		args = filterOutPort(args)
	}
	return args
}

// filterOutPort 移除 -p 与其后的端口号（端口无效时 libpq 用默认 5432）
func filterOutPort(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-p" {
			i++ // 跳过其参数值
			continue
		}
		out = append(out, args[i])
	}
	return out
}

// pgDumpArgs 构造 pg_dump 参数。
//
// 注意：--single-transaction 是 pg_restore / psql 的选项，**pg_dump 不支持**
// （PG 18 实机报 "unrecognized option '--single-transaction'"）。custom 格式
// 导出本就由 pg_dump 在单一快照事务内完成，无需该标志。口令只经 PGPASSWORD。
func pgDumpArgs(cfg DatabaseConfig, target string) []string {
	args := append([]string{}, pgHostArgs(cfg)...)
	return append(args,
		"--format=custom",
		"--no-owner",
		"--no-privileges",
		"-f", target,
	)
}

// Dump 把整库导出到 target 文件，返回实际产物路径（== target）。
//
// PG：pg_dump -Fc（custom 格式，pg_restore 可读取，本身即一致快照）。
// SQLite：VACUUM INTO 写临时文件后原子 rename 到 target（见 vacuumInto）。
func (d *productionDumper) Dump(ctx context.Context, target string, cfg DatabaseConfig) (string, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("创建 dump 目录: %w", err)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("解析 dump 目标绝对路径: %w", err)
	}

	if isSQLiteCfg(cfg) {
		if err := d.vacuumInto(ctx, abs, cfg); err != nil {
			return "", err
		}
		return abs, nil
	}

	// PG：用 -f 指定产物路径，口令仍在 PGPASSWORD，绝不进命令行参数。
	cmd := exec.CommandContext(ctx, resolveToolBinary(cfg.PgDumpPath, "pg_dump"), pgDumpArgs(cfg, abs)...)
	cmd.Env = pgEnv(cfg)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pg_dump 执行失败: %w, stderr=%s", err,
			redactSecrets(stderr.String(), cfg))
	}
	return abs, nil
}

// vacuumInto 用 GORM 底层 *sql.DB 执行 SQLite VACUUM INTO。
//
// 关键约束：VACUUM INTO 不能在事务内对同一连接执行；database/sql 的
// ExecContext 不开启显式事务，满足要求。产物先写到独立临时文件，再原子
// rename 到 target：避免备份中途崩溃留下半截文件被后续误当完整备份。
// Windows 下 os.Rename 不能覆盖已存在文件，故 rename 前先删除 target。
func (d *productionDumper) vacuumInto(ctx context.Context, target string, cfg DatabaseConfig) error {
	if d.db == nil {
		return fmt.Errorf("SQLite VACUUM INTO 需要 gorm.DB，但执行器未注入数据库连接")
	}
	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("获取底层 *sql.DB: %w", err)
	}

	// 临时文件与 target 同目录，保证 rename 在同一文件系统内是原子操作
	tmp := target + ".vtmp"
	_ = os.Remove(tmp) // VACUUM INTO 要求目标文件不存在
	_ = os.Remove(target)

	// INTO 后接字符串表达式；路径为内部生成的绝对路径，转义单引号防注入
	escaped := strings.ReplaceAll(tmp, "'", "''")
	stmt := fmt.Sprintf("VACUUM INTO '%s'", escaped)
	if _, err := sqlDB.ExecContext(ctx, stmt); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("SQLite VACUUM INTO 失败: %w", err)
	}

	// 校验临时产物确实生成（某些驱动对 VACUUM 的 RowsAffected 不可靠）
	if info, statErr := os.Stat(tmp); statErr != nil || info.Size() == 0 {
		_ = os.Remove(tmp)
		if statErr != nil {
			return fmt.Errorf("VACUUM INTO 产物缺失: %w", statErr)
		}
		return fmt.Errorf("VACUUM INTO 产物为空文件")
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("原子替换 SQLite 备份文件失败: %w", err)
	}
	return nil
}

// TableNames 枚举物理库中的表（用于 manifest 与完整性校验）。
//   - PG：information_schema.tables 的 BASE TABLE，天然排除 VIEW / 物化视图；
//   - SQLite：sqlite_master 的 type='table'，排除 sqlite_% 内部表与视图；
//   - 两方言都排除 sys_schema_migrations / schema_migrations 迁移元数据表。
func (d *productionDumper) TableNames(ctx context.Context, cfg DatabaseConfig) ([]string, error) {
	if d.db == nil {
		return nil, fmt.Errorf("枚举物理表需要 gorm.DB，但执行器未注入数据库连接")
	}
	var names []string
	if isSQLiteCfg(cfg) {
		query := `SELECT name FROM sqlite_master
			WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
			  AND name NOT IN ('sys_schema_migrations', 'schema_migrations')
			ORDER BY name`
		if err := d.db.WithContext(ctx).Raw(query).Scan(&names).Error; err != nil {
			return nil, fmt.Errorf("枚举 SQLite 物理表失败: %w", err)
		}
	} else {
		query := `SELECT table_name FROM information_schema.tables
			WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
			  AND table_type = 'BASE TABLE'
			  AND table_name NOT IN ('sys_schema_migrations', 'schema_migrations')
			ORDER BY table_name`
		if err := d.db.WithContext(ctx).Raw(query).Scan(&names).Error; err != nil {
			return nil, fmt.Errorf("枚举 PostgreSQL 物理表失败: %w", err)
		}
	}
	if names == nil {
		names = []string{}
	}
	return names, nil
}

// Restore 底层恢复实现委托给 restorer.go 的 restoreDump（PG pg_restore /
// SQLite 落待恢复文件 + 离线恢复指引）。source 为已解压的 dump 产物文件路径。
//
// 说明：DumpExecutor.Restore 契约注释写的是「备份容器」，但完整的「解析 zip →
// 校验 manifest → 恢复前自动备份」编排在 service.runRestore 中完成，本方法只负责
// 把已校验的 dump 产物灌回数据库，避免 GB 级容器被解压两次。
func (d *productionDumper) Restore(ctx context.Context, source string, cfg DatabaseConfig) error {
	return restoreDump(ctx, source, cfg)
}

// assertSQLDB 仅用于编译期确认 *sql.DB 类型可用（VACUUM INTO 走 database/sql）
var _ = (*sql.DB)(nil)
