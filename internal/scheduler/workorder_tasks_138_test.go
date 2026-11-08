package scheduler

// v1.53 Phase 138 — FIX-05 双路径一致性测试（2026-11-08 开放项盘点 #B8）。
//
// 契约：同一输入下，scheduler cron 路径（assignWorkOrderHandler）与 service
// 路径（workorder.PeriodicService.GenerateWorkOrder）必须产生相同的处理人，
// 且两者都按「member_order ASC 排序的 sys_duty_pool_member 集合 + 分配前
// total_generated 取模」语义选人。
//
// 实现层保证一致性的手段：两路径共用 workorder.LoadRotationMembers 与
// workorder.RotationSelectedIndex（internal/services/workorder/rotation.go），
// 本测试从行为侧锁定该不变量，防止未来任一路径偷偷改语义。

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	workordersvc "github.com/xingran-next/xingran-go-backend/internal/services/workorder"
	"gorm.io/gorm"
)

func ptr138(s string) *string { return &s }

// newRotationDualPathDB 建一个 file-based sqlite（避免 :memory: 连接池拿到不同
// 连接导致表不可见），AutoMigrate 两路径共用的全部模型。
func newRotationDualPathDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "rotation138.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Config{},
		&models.DutyPoolMember{},
		&models.PeriodicWorkOrderTemplate{},
		&models.WorkOrder{},
		&models.PeriodicWorkOrderLog{},
	))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// seedRotation138 写入 1 个提交人 + 3 个成员（member_order 故意乱序插入以验证
// 排序生效）+ 1 个 rotation 模板（total_generated = cursor）。
func seedRotation138(t *testing.T, db *gorm.DB, cursor int) *models.PeriodicWorkOrderTemplate {
	t.Helper()
	// 提交人（template.CreatedBy 命中，避免走 sys_config/admin fallback）
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, password, salt, auth_source) VALUES
		('submitter-138', 'submitter-138', 'x', 'x', 'local')`).Error)
	// 3 个成员用户必须存在于 sys_user：service.GenerateWorkOrder 末尾会校验
	// assigneeID 对应用户存在，不存在则置 nil（两路径一致性前提）。
	require.NoError(t, db.Exec(`INSERT INTO sys_user (id, username, password, salt, auth_source) VALUES
		('rot-u-A', 'rot-u-A', 'x', 'x', 'local'),
		('rot-u-B', 'rot-u-B', 'x', 'x', 'local'),
		('rot-u-C', 'rot-u-C', 'x', 'x', 'local')`).Error)
	// 故意乱序插入：member_order=1 先插，验证 LoadRotationMembers 的 ORDER BY 生效
	require.NoError(t, db.Exec(`INSERT INTO sys_duty_pool_member (id, pool_id, user_id, member_order) VALUES
		('rm-b', 'pool-138', 'rot-u-B', 1),
		('rm-c', 'pool-138', 'rot-u-C', 2),
		('rm-a', 'pool-138', 'rot-u-A', 0)`).Error)

	tpl := &models.PeriodicWorkOrderTemplate{
		BaseModel:      models.BaseModel{ID: "tpl-138", CreatedBy: "submitter-138"},
		TemplateName:   "138 一致性测试模板",
		WorkOrderTitle: "周期性工单-{date}",
		Description:    "双路径一致性",
		CategoryID:     "cat-138",
		Type:           models.WorkOrderType("incident"),
		Priority:       1,
		CronExpression: "0 0 8 * * *",
		AssignType:     models.PeriodicAssignTypeRotation,
		AssignTargetID: ptr138("pool-138"),
		IsEnabled:      true,
		TotalGenerated: cursor,
	}
	require.NoError(t, db.Create(tpl).Error)
	return tpl
}

// TestWot138_RotationDualPathConsistency 表驱动：多个 total_generated 游标下，
// scheduler 路径与 service 路径必须选中同一个成员。
//
// 每个 case 用独立 DB，顺序固定：先 scheduler（只读游标，不改库）→ 再 service
// （读取同一游标后自行 +1）。这样两路径读到的「分配前游标」完全一致。
func TestWot138_RotationDualPathConsistency(t *testing.T) {
	cases := []struct {
		name           string
		cursor         int
		wantAssigneeID string
	}{
		{"cursor=0 → member_order 0", 0, "rot-u-A"},
		{"cursor=1 → member_order 1", 1, "rot-u-B"},
		{"cursor=2 → member_order 2", 2, "rot-u-C"},
		{"cursor=3 → 回绕到 0", 3, "rot-u-A"},
		{"cursor=7 → 7%3=1", 7, "rot-u-B"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newRotationDualPathDB(t)
			tpl := seedRotation138(t, db, tc.cursor)

			// ---- 路径 1：scheduler cron 路径 ----
			woSched := &models.WorkOrder{}
			gotSched, err := assignWorkOrderHandler(db, tpl, woSched)
			require.NoError(t, err)
			require.NotNil(t, gotSched, "scheduler 路径应返回处理人")
			assert.Equal(t, tc.wantAssigneeID, *gotSched, "scheduler 路径选人错误")
			assert.Equal(t, "rotation", woSched.AssignStrategy)

			// ---- 路径 2：service 路径（同一 DB、同一 cursor） ----
			svc := workordersvc.NewPeriodicService(db)
			woSvc, err := svc.GenerateWorkOrder(context.Background(), tpl.ID)
			require.NoError(t, err)
			require.NotNil(t, woSvc)
			require.NotNil(t, woSvc.AssigneeID, "service 路径应分配处理人")
			assert.Equal(t, tc.wantAssigneeID, *woSvc.AssigneeID, "service 路径选人错误")
			assert.Equal(t, "rotation", woSvc.AssignStrategy)

			// ---- 一致性断言：两路径同输入同处理人 ----
			assert.Equal(t, *gotSched, *woSvc.AssigneeID,
				"scheduler 与 service 双路径必须产生相同处理人（cursor=%d）", tc.cursor)
		})
	}
}

// TestWot138_SharedRotationPureFunctions 锁定共享纯函数语义（成员集合排序 +
// 取模索引），这是两路径共用的真相源。
func TestWot138_SharedRotationPureFunctions(t *testing.T) {
	db := newRotationDualPathDB(t)
	seedRotation138(t, db, 0)

	// 成员集合按 member_order ASC：A(0) → B(1) → C(2)（插入顺序是 B,C,A）
	members, err := workordersvc.LoadRotationMembers(context.Background(), db, "pool-138")
	require.NoError(t, err)
	require.Len(t, members, 3)
	assert.Equal(t, "rot-u-A", members[0].UserID)
	assert.Equal(t, "rot-u-B", members[1].UserID)
	assert.Equal(t, "rot-u-C", members[2].UserID)

	// 取模索引：cursor % memberCount，负数归一，0/负成员数返回 -1
	assert.Equal(t, 0, workordersvc.RotationSelectedIndex(0, 3))
	assert.Equal(t, 1, workordersvc.RotationSelectedIndex(1, 3))
	assert.Equal(t, 2, workordersvc.RotationSelectedIndex(2, 3))
	assert.Equal(t, 0, workordersvc.RotationSelectedIndex(3, 3))
	assert.Equal(t, 1, workordersvc.RotationSelectedIndex(7, 3))
	assert.Equal(t, 2, workordersvc.RotationSelectedIndex(-1, 3), "负游标归一到末尾")
	assert.Equal(t, -1, workordersvc.RotationSelectedIndex(0, 0), "无成员返回 -1")
	assert.Equal(t, -1, workordersvc.RotationSelectedIndex(0, -1), "负成员数返回 -1")

	// 空池：LoadRotationMembers 返回空切片（两路径据此降级为未分配）
	empty, err := workordersvc.LoadRotationMembers(context.Background(), db, "pool-not-exist")
	require.NoError(t, err)
	assert.Empty(t, empty)

	// 顺序用例：连续 cursor 应轮转覆盖全部成员
	got := make([]string, 0, 6)
	for c := 0; c < 6; c++ {
		got = append(got, fmt.Sprintf("%d", workordersvc.RotationSelectedIndex(c, 3)))
	}
	assert.Equal(t, []string{"0", "1", "2", "0", "1", "2"}, got)
}
