// rpa_orphan_job_prune：回收 sys_job 里「业务表已无对应调度」的孤儿 RPA job。
//
// 用途（W7.2 / 任务 A-3）：生产库实证有 10 条孤儿 job —— 对应的
// sys_rpa_schedules 行已被软删或物理删，而 job 仍 deleted_at IS NULL /
// status=0。调度器启动时（internal/scheduler/cron.go 的 Scheduler.Start）
// 用 `Where("status = ?", JobStatusNormal).Find(&jobs)` 加载**全部**启用 job
// （不区分 job_group），于是这 10 条被挂进 cron，到点调用
// rpa_schedule:<已删 id> → handler 里 TriggerNow 查不到 schedule → 报错刷日志。
//
// 根因：Ensure/Remove 只做「按 id 从业务表推向执行表」，从不对账执行表里
// 「业务表已经没有」的行 —— 方向性缺口，孤儿因此永久存在。
//
// 本工具**不自带判定逻辑**：清理走与后端启动时完全相同的代码路径
// （rpav1.ScheduleService.SyncEnabledJobs → scheduler.RPAScheduleJobSyncer
// 的同一个 prune 实现），保证「工具清掉的」与「后端启动会清的」是同一批。
// 工具只负责：连接 → 报告 → 调用 → 验证。
//
// 输出（机器可读，便于 SUMMARY 引用）：
//
//	清理前：RPA 分组未删 job 数 / 调度器将加载的任务数 / 启用态 schedule 数
//	逐条清单：job 名 | job id | schedule id | 判定与原因
//	清理后：同上三项
//	验证：RPA 组未删 job 数 == 启用态 schedule 数（一致则退出码 0）
//
// 用法：
//
//	go run ./scripts/ops/rpa_orphan_job_prune -dry-run     # 只报告，不改数据
//	go run ./scripts/ops/rpa_orphan_job_prune              # 真清理
//	go run ./scripts/ops/rpa_orphan_job_prune -dsn "postgres://..."
//
// 连接参数默认读仓库根的环境文件（DB_HOST/DB_USER/DB_PASSWORD/DB_NAME/DB_PORT），
// 与 scripts/ops/rpa_action_type_census 同款连接方式（PG only）。
//
// 退出码：0 = 清理完成且验证一致；1 = 失败或验证不一致；2 = 参数/连接错误。
package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/scheduler"
	rpav1 "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
)

const (
	exitOK       = 0
	exitFailed   = 1
	exitBadInput = 2
)

// readEnvFileKey 从仓库根的环境文件里读一个键（找不到回落进程环境变量）。
func readEnvFileKey(key string) string {
	data, err := os.ReadFile(".env")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, key+"=") {
				return strings.Trim(strings.TrimPrefix(line, key+"="), `"'`)
			}
		}
	}
	return os.Getenv(key)
}

// buildDSN 用环境变量拼 PG DSN。
func buildDSN() (string, error) {
	host := readEnvFileKey("DB_HOST")
	user := readEnvFileKey("DB_USER")
	port := readEnvFileKey("DB_PORT")
	name := readEnvFileKey("DB_NAME")
	pw := readEnvFileKey("DB_PASSWORD")
	if host == "" || pw == "" {
		return "", fmt.Errorf("DB_HOST / DB_PASSWORD 缺失（读环境文件或设置环境变量）")
	}
	if port == "" {
		port = "5432"
	}
	if name == "" {
		name = "ruoyi"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&connect_timeout=20",
		user, url.QueryEscape(pw), host, port, name), nil
}

// counts 三个关键计数（同一口径贯穿清理前后）。
type counts struct {
	// rpaJobs RPA 分组下未软删的 job 数（= 派生镜像现状）。
	rpaJobs int64
	// loadableJobs 调度器启动时会加载的任务数（status=0 且未软删，不分分组）。
	loadableJobs int64
	// enabledSchedules 启用态（status=0）且未软删的调度数（= 真相源现状）。
	enabledSchedules int64
}

// snapshot 取当前三个计数。
func snapshot(db *gorm.DB) (counts, error) {
	var c counts
	if err := db.Table("sys_job").
		Where("job_group = ?", "RPA").
		Where("deleted_at IS NULL").
		Count(&c.rpaJobs).Error; err != nil {
		return c, fmt.Errorf("统计 RPA job 数失败: %w", err)
	}
	if err := db.Table("sys_job").
		Where("status = ?", 0).
		Where("deleted_at IS NULL").
		Count(&c.loadableJobs).Error; err != nil {
		return c, fmt.Errorf("统计调度器加载数失败: %w", err)
	}
	if err := db.Table("sys_rpa_schedules").
		Where("status = ?", 0).
		Where("deleted_at IS NULL").
		Count(&c.enabledSchedules).Error; err != nil {
		return c, fmt.Errorf("统计启用调度数失败: %w", err)
	}
	return c, nil
}

func main() {
	dsn := flag.String("dsn", "", "直接给 DSN（覆盖环境变量拼装）")
	dryRun := flag.Bool("dry-run", false, "只报告不清理")
	flag.Parse()

	dsnValue := *dsn
	if dsnValue == "" {
		built, err := buildDSN()
		if err != nil {
			fmt.Fprintln(os.Stderr, "参数错误:", err)
			os.Exit(exitBadInput)
		}
		dsnValue = built
	}

	db, err := gorm.Open(postgres.Open(dsnValue), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "连接失败:", err)
		os.Exit(exitBadInput)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 真同步器（与后端启动时同一个实现）+ 真调度服务（正向 ensure + 反向 prune）。
	sched := scheduler.NewScheduler(db)
	syncer := scheduler.NewRPAScheduleJobSyncer(sched, db)
	svc := rpav1.NewScheduleService(db, nil, syncer)

	before, err := snapshot(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "统计失败:", err)
		os.Exit(exitFailed)
	}
	fmt.Printf("=== 清理前 ===\n")
	fmt.Printf("RPA 分组未删 job 数 : %d\n", before.rpaJobs)
	fmt.Printf("调度器将加载任务数 : %d\n", before.loadableJobs)
	fmt.Printf("启用态 schedule 数 : %d\n", before.enabledSchedules)

	// 逐条清单（只读判定，与真实清理同一份实现）
	orphans, err := syncer.FindOrphanJobs(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "扫描孤儿 job 失败:", err)
		os.Exit(exitFailed)
	}
	fmt.Printf("--- 孤儿清单（%d 条）---\n", len(orphans))
	for i, item := range orphans {
		fmt.Printf("[%d] %s | job=%s | schedule=%s | %s\n",
			i+1, item.JobName, item.JobID, item.ScheduleID, item.Reason)
	}

	if *dryRun {
		fmt.Printf("\nDRY-RUN：未做任何修改。\n")
		fmt.Printf("ORPHAN_COUNT=%d\n", len(orphans))
		os.Exit(exitOK)
	}

	// 真清理：走与后端启动时完全相同的代码路径（正向 ensure + 反向 prune）。
	if err := svc.SyncEnabledJobs(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "同步/对账失败:", err)
		os.Exit(exitFailed)
	}

	after, err := snapshot(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "统计失败:", err)
		os.Exit(exitFailed)
	}
	fmt.Printf("\n=== 清理后 ===\n")
	fmt.Printf("RPA 分组未删 job 数 : %d\n", after.rpaJobs)
	fmt.Printf("调度器将加载任务数 : %d\n", after.loadableJobs)
	fmt.Printf("启用态 schedule 数 : %d\n", after.enabledSchedules)

	// 验证：派生镜像与真相源对齐
	fmt.Printf("\n=== 验证 ===\n")
	consistent := after.rpaJobs == after.enabledSchedules
	if consistent {
		fmt.Printf("一致：RPA 组未删 job 数(%d) == 启用态 schedule 数(%d)\n",
			after.rpaJobs, after.enabledSchedules)
	} else {
		fmt.Printf("不一致：RPA 组未删 job 数(%d) != 启用态 schedule 数(%d)\n",
			after.rpaJobs, after.enabledSchedules)
	}

	fmt.Printf("\nBEFORE_RPA_JOBS=%d\n", before.rpaJobs)
	fmt.Printf("AFTER_RPA_JOBS=%d\n", after.rpaJobs)
	fmt.Printf("BEFORE_LOADABLE_JOBS=%d\n", before.loadableJobs)
	fmt.Printf("AFTER_LOADABLE_JOBS=%d\n", after.loadableJobs)
	fmt.Printf("ENABLED_SCHEDULES=%d\n", after.enabledSchedules)
	fmt.Printf("ORPHAN_COUNT=%d\n", len(orphans))
	fmt.Printf("CONSISTENT=%t\n", consistent)

	if !consistent {
		os.Exit(exitFailed)
	}
}
