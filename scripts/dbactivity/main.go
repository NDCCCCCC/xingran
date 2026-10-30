// dbactivity: 诊断远端 dev DB 的连接活动状态 (debug session backend-hang-on-automigrate)。
//
// 用途: 当后端启动长时间无日志输出时, 从服务端 pg_stat_activity 视图确认后端连接
// 到底是 active(在跑慢查询) / idle(未在操作 DB, 可能卡在 Go 侧) / idle in transaction
// (事务中阻塞) / 以及是否有 lock 等待。这是判断"慢 vs 卡"的权威方式。
//
// 连接复用 config.GetDSN() (带 keepalive)。只读, 不改任何数据。
//
// 用法 (项目根目录):
//
//	go run ./scripts/dbactivity
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/xingran-next/xingran-go-backend/internal/config"
)

// activityRow 一行 pg_stat_activity (仅本库、排除自身 probe 连接)。
type activityRow struct {
	PID           int     `gorm:"column:pid"`
	State         string  `gorm:"column:state"`
	WaitEventType *string `gorm:"column:wait_event_type"`
	WaitEvent     *string `gorm:"column:wait_event"`
	QDurSec       *float64 `gorm:"column:q_dur_s"`
	StDurSec      *float64 `gorm:"column:st_dur_s"`
	Query         string  `gorm:"column:q"`
}

// lockRow 一行 pg_locks + pg_stat_activity (仅未授予的锁等待)。
type lockRow struct {
	PID     int     `gorm:"column:pid"`
	Mode    string  `gorm:"column:mode"`
	Granted bool    `gorm:"column:granted"`
	QDurSec *float64 `gorm:"column:q_dur_s"`
	Query   string  `gorm:"column:q"`
}

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load(context.Background())
	if err != nil {
		die("加载配置失败: %v", err)
	}
	gdb, err := gorm.Open(postgres.Open(cfg.Database.GetDSN()), &gorm.Config{
		Logger:      gormlogger.Default.LogMode(gormlogger.Silent),
		PrepareStmt: false,
	})
	if err != nil {
		die("连接 PG 失败: %v", err)
	}
	sqlDB, _ := gdb.DB()
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	fmt.Printf("connected (host=%s, db=%s)\n\n", cfg.Database.Host, cfg.Database.DBName)

	// 1. pg_stat_activity (本库, 排除自身)
	fmt.Println("=== pg_stat_activity (本库, 排除本 probe) ===")
	var rows []activityRow
	q := `
SELECT pid, state, wait_event_type, wait_event,
       EXTRACT(EPOCH FROM now()-query_start)     AS q_dur_s,
       EXTRACT(EPOCH FROM now()-state_change)    AS st_dur_s,
       COALESCE(LEFT(query,120),'(no query)')    AS q
FROM pg_stat_activity
WHERE datname = current_database() AND pid <> pg_backend_pid()
ORDER BY query_start DESC NULLS LAST;`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = gdb.WithContext(ctx).Raw(q).Scan(&rows).Error
	cancel()
	if err != nil {
		fmt.Printf("!! 查询 pg_stat_activity 失败: %v\n", err)
	} else if len(rows) == 0 {
		fmt.Println("(无其它连接 — 后端未持有任何数据库连接)")
	} else {
		fmt.Printf("%-8s %-22s %-10s %-22s %10s %10s  %s\n", "PID", "STATE", "WAITTYPE", "WAITEVENT", "QDUR(s)", "STDUR(s)", "QUERY")
		for _, r := range rows {
			wt := "-"
			we := "-"
			if r.WaitEventType != nil && *r.WaitEventType != "" {
				wt = *r.WaitEventType
			}
			if r.WaitEvent != nil && *r.WaitEvent != "" {
				we = *r.WaitEvent
			}
			qd := "-"
			if r.QDurSec != nil {
				qd = fmt.Sprintf("%.1f", *r.QDurSec)
			}
			sd := "-"
			if r.StDurSec != nil {
				sd = fmt.Sprintf("%.1f", *r.StDurSec)
			}
			fmt.Printf("%-8d %-22s %-10s %-22s %10s %10s  %s\n", r.PID, r.State, wt, we, qd, sd, oneLine(r.Query))
		}
	}

	// 2. 未授予的锁等待 (被阻塞的查询)
	fmt.Println("\n=== 未授予的锁等待 (blocked) ===")
	var locks []lockRow
	lq := `
SELECT a.pid, l.mode, l.granted,
       EXTRACT(EPOCH FROM now()-a.query_start) AS q_dur_s,
       COALESCE(LEFT(a.query,80),'(no query)') AS q
FROM pg_locks l JOIN pg_stat_activity a USING (pid)
WHERE l.database IS NOT NULL
ORDER BY l.granted, q_dur_s DESC NULLS LAST;`
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	err = gdb.WithContext(ctx2).Raw(lq).Scan(&locks).Error
	cancel2()
	if err != nil {
		fmt.Printf("!! 查询 pg_locks 失败: %v\n", err)
	} else {
		blocked := 0
		for _, l := range locks {
			if !l.Granted {
				blocked++
			}
		}
		fmt.Printf("总锁条目 %d, 其中未授予(被阻塞) %d\n", len(locks), blocked)
		for _, l := range locks {
			if !l.Granted {
				qd := "-"
				if l.QDurSec != nil {
					qd = fmt.Sprintf("%.1f", *l.QDurSec)
				}
				fmt.Printf("  BLOCKED pid=%d mode=%s qdur=%ss q=%s\n", l.PID, l.Mode, qd, oneLine(l.Query))
			}
		}
	}

	// 3. 解读提示
	fmt.Println("\n=== 解读 ===")
	fmt.Println("state=active            → 正在执行查询 (慢但活着; 看 q_dur)")
	fmt.Println("state=idle              → 连接空闲, 后端当前未在操作 DB (可能卡在 Go 侧或思考中)")
	fmt.Println("state=idle in transaction → 事务中阻塞 (持有事务未提交)")
	fmt.Println("wait_event=ClientRead   → server 等客户端发字节 (网络 stall; keepalive 应在 ~25s 判死)")
	fmt.Println("BLOCKED 行              → 该查询在等另一个连接持有的锁")
}

func oneLine(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\n' || c == '\r' || c == '\t' {
			c = ' '
		}
		out = append(out, c)
	}
	return string(out)
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
