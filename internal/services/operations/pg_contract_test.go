// Package operations: PG-gated 类型契约集成测试。
//
// 根因(2026-10-15,ops 三列表 500,详见 .planning/debug/resolved/ops-list-join-uuid-500.md):
// ops 表 FK 列在 PG 里是混合类型(多数 varchar(64),但 sys_workstation.building_id 与
// ops_floors.plan_image_id 是 uuid)。手写 JOIN SQL 一旦按"FK 全 varchar"统一写成
// CAST(pk AS TEXT) = fk,对 uuid 列即产生 text=uuid / varchar=uuid → SQLSTATE 42883。
// SQLite 动态类型对两种写法都放行 → "SQLite 单测全绿 + PG 42883 炸"类回归 SQLite 测不出,
// 本测试是唯一能真实守护这类类型契约的回归防线。
//
// 启用方式(默认跳过,无 TEST_PG_DSN 时 t.Skip,CI 不红):
//
//	TEST_PG_DSN="postgres://user:pass@host/db?sslmode=disable" \
//	  go test -v -run TestPGContract ./internal/services/operations/
//
// 只读保证:本测试对 DB 仅执行三服务 List/Statistics 内部的 SELECT(JOIN 无条件生效,
// 正是被守护路径);禁止任何 Create/Update/Delete/Raw 写 SQL —— TEST_PG_DSN 可能指向生产库。
package operations

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/api/v1/operations/requests"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openPGForContractTest 按 TEST_PG_DSN 打开真实 PG 连接(共享 helper)。
// 空 DSN → skip;连不通 → skip。返回 *gorm.DB,测试结束后自动关闭底层连接池。
func openPGForContractTest(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set; PG contract test skipped")
	}
	// pgx v5 禁写 keepalives 类 DSN 参数(进 RuntimeParams → FATAL 42704),
	// connect_timeout 是安全参数;防坏主机挂死。
	// URL 形式(postgres://...?k=v)用 & 追加,keyword 形式(host=... dbname=...)用空格;
	// 对 URL 用空格会把 sslmode 的值污染成 "disable connect_timeout=5" → TLS 配置解析失败。
	if !strings.Contains(dsn, "connect_timeout") {
		if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
			sep := "?"
			if strings.Contains(dsn, "?") {
				sep = "&"
			}
			dsn += sep + "connect_timeout=5"
		} else {
			dsn += " connect_timeout=5"
		}
	}

	// DisableAutomaticPing:Open 阶段不探活(等价 lazy connect;GORM v1.30.5 无 OpenLazy 选项),
	// 由下方 PingContext 统一探活并控制超时
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Skipf("TEST_PG_DSN unreachable (gorm open): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Skipf("TEST_PG_DSN unreachable (sql.DB): %v", err)
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		t.Skipf("TEST_PG_DSN unreachable: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// requireNoPGErr 断言 err==nil;42883(operator does not exist)给出显式类型契约失败信息。
func requireNoPGErr(t *testing.T, stage string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "42883") {
		t.Fatalf("%s: PG type contract broken — FK comparison direction mismatch (uuid vs text/varchar): %v", stage, err)
	}
	t.Fatalf("%s: %v", stage, err)
}

// TestPGContract_OpsListJoins 三服务 List+Statistics 真实打 PG:
// 任一 JOIN 比较方向写反(uuid 列被 CAST 成 text 比较)都会在真实 PG 上报 42883 并在此失败。
func TestPGContract_OpsListJoins(t *testing.T) {
	db := openPGForContractTest(t)
	ctx := context.Background()

	t.Run("floor", func(t *testing.T) {
		floorSvc := NewFloorService(db)
		page, err := floorSvc.List(ctx, nil)
		requireNoPGErr(t, "floor List", err)
		if page == nil {
			t.Fatal("floor List: result is nil")
		}
		requireNoPGErr(t, "floor Statistics", func() error {
			_, err := floorSvc.Statistics(ctx)
			return err
		}())
	})

	t.Run("workstation", func(t *testing.T) {
		wsSvc := NewWorkstationService(db)
		page, err := wsSvc.List(ctx, requests.WorkstationListRequest{})
		requireNoPGErr(t, "workstation List", err)
		if page == nil {
			t.Fatal("workstation List: result is nil")
		}
		requireNoPGErr(t, "workstation Statistics", func() error {
			_, err := wsSvc.Statistics(ctx, nil)
			return err
		}())
	})

	t.Run("infopoint", func(t *testing.T) {
		ipSvc := NewInfoPointService(db)
		page, err := ipSvc.List(ctx, requests.InfoPointListRequest{})
		requireNoPGErr(t, "infopoint List", err)
		if page == nil {
			t.Fatal("infopoint List: result is nil")
		}
		requireNoPGErr(t, "infopoint Statistics", func() error {
			_, err := ipSvc.Statistics(ctx)
			return err
		}())
	})
}
