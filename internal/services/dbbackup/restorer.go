package dbbackup

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 本文件实现恢复流程。恢复是高危不可逆操作，顺序严格锁定：
//
//	① verifyContainerOnly：解析 zip、校验 manifest sha256（零凭据、防篡改）——
//	  非法上传必须在做任何昂贵动作之前被拒绝；
//	② 恢复前自动全量备份（SourcePreRestore）：失败即中止，无回退退路不下发；
//	③ verifyAndExtract：流式解压 dump 并复算 dump_sha256 / container_sha256；
//	④ restoreDump：PG pg_restore 或 SQLite 原子替换文件。
//
// ①②③④ 的编排由 service.runRestore 驱动，本文件提供 ①/③/④ 的纯函数，
// 不直接写任务表、不写日志到 operlog（那是 handler 层职责）。

// restoreDump 把已校验解压的 dump 产物灌回数据库。
//   - PG：pg_restore --clean --if-exists --single-transaction；
//     **不加 --disable-triggers**（该选项需要 superuser，普通业务库会失败），
//     按依赖顺序恢复由 pg_restore 自身保证（PLAN.md 风险降级条款）。
//   - SQLite：不关闭在线连接池（会破坏运行中的进程），落 .pending_restore 后
//     返回需离线恢复的明确错误，由 scripts/ops/db-backup-restore.go 停服执行。
func restoreDump(ctx context.Context, dumpFile string, cfg DatabaseConfig) error {
	if dumpFile == "" {
		return fmt.Errorf("待恢复 dump 路径为空")
	}
	if _, err := os.Stat(dumpFile); err != nil {
		return fmt.Errorf("待恢复 dump 文件不存在: %w", err)
	}

	if isSQLiteCfg(cfg) {
		return restoreSQLite(ctx, dumpFile, cfg)
	}
	return restorePostgres(ctx, dumpFile, cfg)
}

// pgRestoreArgs 构造 pg_restore 参数。
// --single-transaction 属于 pg_restore（使恢复在单事务内完成，失败整体回滚），
// 不属于 pg_dump（见 dumper.go 的 pgDumpArgs 注释）。口令只经 PGPASSWORD。
func pgRestoreArgs(cfg DatabaseConfig, dumpFile string) []string {
	args := append([]string{}, pgHostArgs(cfg)...)
	return append(args,
		"--clean",
		"--if-exists",
		"--single-transaction",
		"--no-owner",
		"--no-privileges",
		"--exit-on-error",
		dumpFile, // custom 格式归档文件，路径不含任何凭据
	)
}

// restorePostgres 调用外部 pg_restore。
// 密码只经 PGPASSWORD 环境变量传递，绝不进命令行参数；stderr 经脱敏后才进错误文本。
func restorePostgres(ctx context.Context, dumpFile string, cfg DatabaseConfig) error {
	cmd := exec.CommandContext(ctx, resolveToolBinary(cfg.PgRestorePath, "pg_restore"), pgRestoreArgs(cfg, dumpFile)...)
	cmd.Env = pgEnv(cfg)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore 执行失败: %w, stderr=%s", err,
			redactSecrets(stderr.String(), cfg))
	}
	return nil
}

// restoreSQLite 处理 SQLite 备份恢复。
//
// 安全约束（code review S1 修复）：在线服务的 GORM 连接池持有 dbCfg.Path 对应
// 的文件句柄，**绝不能** sqlDB.Close() 后原子替换文件——那会让 runTask 后续的
// succeedTask/failTask 写终态时报 "sql: database is closed"，任务永远卡在
// running，且此后所有模块的 DB 查询全部失败直至重启。
//
// SQLite 的整库文件替换本质是离线操作。在线端点为此明确降级：
//  1. 把已校验的恢复产物复制到目标同目录的 .pending_restore 文件；
//  2. 返回 ErrSQLiteOfflineRestoreRequired，任务被标为 failed（错误信息给出
//     具体离线恢复命令），进程与全局连接池保持完全健康；
//  3. 运维停服后执行 scripts/ops/db-backup-restore.go 完成原子替换再重启。
//
// 这样「导入恢复」对 PostgreSQL 完全在线可用；对 SQLite 给出可执行的明确路径，
// 而不是静默破坏运行中的系统。
var ErrSQLiteOfflineRestoreRequired = errors.New("SQLite 整库恢复需停服后离线执行")

func restoreSQLite(ctx context.Context, dumpFile string, cfg DatabaseConfig) error {
	target := cfg.Path
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("SQLite 恢复需要 DatabaseConfig.Path，但配置为空")
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("解析 SQLite 库绝对路径: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return fmt.Errorf("创建 SQLite 库目录: %w", err)
	}

	// 先把 dump 复制到目标同目录的待恢复文件，保证离线 CLI 可直接 rename
	pending := absTarget + ".pending_restore"
	if err := copyFile(ctx, dumpFile, pending); err != nil {
		return err
	}
	// 校验产物是合法 SQLite 文件（sqlite_master 可查询 + 魔数）
	if err := probeSQLite(ctx, pending); err != nil {
		_ = os.Remove(pending)
		return fmt.Errorf("恢复产物不是合法 SQLite 数据库: %w", err)
	}

	return fmt.Errorf("%w: 已将恢复产物落到 %s；请停服后执行 "+
		"`go run ./scripts/ops/db-backup-restore.go -db %s -pending %s`，再启动服务",
		ErrSQLiteOfflineRestoreRequired, pending, absTarget, pending)
}

// copyFile 流式复制文件（GB 级不进内存），支持 ctx 取消
func copyFile(ctx context.Context, src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开复制源: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建复制目标: %w", err)
	}
	_, copyErr := ioCopyWithContext(ctx, out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("复制恢复产物失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("关闭复制目标失败: %w", closeErr)
	}
	return nil
}

// ioCopyWithContext 带 ctx 取消的流式复制：每轮检查 ctx，避免大文件复制
// 期间恢复任务被超时后仍继续写盘。
func ioCopyWithContext(ctx context.Context, dst interface{ Write([]byte) (int, error) }, src interface{ Read([]byte) (int, error) }) (int64, error) {
	buf := make([]byte, 1<<20) // 1 MiB 缓冲，内存占用恒定
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nw != nr {
				return written, fmt.Errorf("写入字节数不足")
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				break
			}
			return written, er
		}
	}
	return written, nil
}

// ioCopyWithContextLimit 带 ctx 取消 + 累计写出上限的流式复制（zip bomb 防护）。
func ioCopyWithContextLimit(ctx context.Context, dst interface{ Write([]byte) (int, error) }, src interface{ Read([]byte) (int, error) }, maxBytes int64) (int64, error) {
	buf := make([]byte, 1<<20)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			if written+int64(nr) > maxBytes {
				return written, fmt.Errorf("解压数据量超过上限 %d 字节", maxBytes)
			}
			nw, ew := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nw != nr {
				return written, fmt.Errorf("写入字节数不足")
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				break
			}
			return written, er
		}
	}
	return written, nil
}

// probeSQLite 用 database/sql 打开独立连接校验恢复产物是否为合法 SQLite 库：
// 先查 16 字节魔数，再执行 SELECT count(*) FROM sqlite_master 确认文件结构完整
// （仅验魔数会被伪造/半截文件骗过）。连接只读且用完即关，不触碰全局 GORM 池。
func probeSQLite(ctx context.Context, path string) error {
	if err := probeSQLiteHeader(path); err != nil {
		return err
	}
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("打开恢复产物独立连接失败: %w", err)
	}
	defer sqlDB.Close()
	var n int
	if err := sqlDB.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master").Scan(&n); err != nil {
		return fmt.Errorf("恢复产物不是可读的 SQLite 数据库: %w", err)
	}
	return nil
}

// probeSQLiteHeader 校验 SQLite 文件头魔数（16 字节标准头）
func probeSQLiteHeader(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开恢复产物: %w", err)
	}
	defer f.Close()
	header := make([]byte, 16)
	if _, err := f.Read(header); err != nil {
		return fmt.Errorf("读取恢复产物文件头: %w", err)
	}
	if string(header) != "SQLite format 3\x00" {
		return fmt.Errorf("文件头不是 SQLite format 3")
	}
	return nil
}
