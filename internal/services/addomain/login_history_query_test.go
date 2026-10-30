package addomain

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestListComputerLoginHistory_SQLiteGuardIsNoop 验证查询服务在非 PG 方言
// (sqlite 内存库)下的 isPostgres 守卫:不建表、不查询,直接返回空列表 + 0,
// 且不报错(quick 261016 风险 8:sqlite 全链路跳过,单测/dev 不因缺表报错)。
//
// 关键点:sqlite 内存库故意不建 sys_ad_login_history 表 —— 若守卫失效,
// 查询会因 "no such table" 报错,测试即失败。
func TestListComputerLoginHistory_SQLiteGuardIsNoop(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	svc := NewADLoginHistoryService(db)
	require.NotNil(t, svc)

	list, total, err := svc.ListComputerLoginHistory(context.Background(), &ADLoginHistoryQuery{
		ComputerID: "some-uuid",
		Current:    1,
		PageSize:   10,
	})
	require.NoError(t, err, "sqlite 守卫应直接返回空,不触达 DB")
	assert.Equal(t, int64(0), total)
	assert.NotNil(t, list)
	assert.Empty(t, list, "应返回空切片而非 nil")
}

// TestListComputerLoginHistory_SQLiteGuardClampsPagination 验证分页钳制
// 在守卫短路之前生效(请求参数被规范化回写,handler response.Page 用得到)。
func TestListComputerLoginHistory_SQLiteGuardClampsPagination(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	svc := NewADLoginHistoryService(db)

	req := &ADLoginHistoryQuery{ComputerID: "some-uuid", Current: -1, PageSize: 9999}
	_, _, err = svc.ListComputerLoginHistory(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, req.Current, "Current<=0 应回退默认 1")
	assert.Equal(t, 100, req.PageSize, "PageSize>100 应钳制到 MaxListPageSize")
}

// TestListComputerLoginHistory_EmptyComputerIDRejected binding 层 required 由
// handler ShouldBindJSON 兜底;service 层对空 computerId 也保持空结果语义
// (守卫短路,不 panic)。
func TestListComputerLoginHistory_EmptyComputerIDRejected(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	svc := NewADLoginHistoryService(db)

	list, total, err := svc.ListComputerLoginHistory(context.Background(), &ADLoginHistoryQuery{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, list)
}

// TestADLoginHistoryModel_TableName 锁定表名与事件常量(公共 API 回归)
func TestADLoginHistoryModel_TableName(t *testing.T) {
	assert.Equal(t, "sys_ad_login_history", models.ADLoginHistory{}.TableName())
	assert.Equal(t, models.ADLoginEventType("initial"), models.ADLoginEventInitial)
	assert.Equal(t, models.ADLoginEventType("login"), models.ADLoginEventLogin)
}
