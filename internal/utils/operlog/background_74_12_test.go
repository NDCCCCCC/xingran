package operlog

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// 74-11 escalation gap-closure: RecordBackground(cron 路径操作日志)。
// 用本地 mock Recorder(实现 RecordAsync)避免 import cycle
// (services → operlog)。
// =====================================================================

type bgMockRecorder struct {
	calls    int
	lastArgs []interface{}
}

func (m *bgMockRecorder) RecordAsync(db *gorm.DB, title string, businessType int, method, requestMethod, operUrl string,
	operatorName, operatorNickname, deptName *string, operIP *string, operParam, jsonResult, errorMsg *string, status int, costTime int64) {
	m.calls++
	m.lastArgs = []interface{}{title, businessType, requestMethod, operatorName, operParam}
}

func newBgOperLogDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bg_oper.db")), &gorm.Config{})
	require.NoError(t, err)
	if sqlDB, err := db.DB(); err == nil {
		t.Cleanup(func() { _ = sqlDB.Close() })
	}
	require.NoError(t, db.Migrator().CreateTable(&models.OperLog{}))
	return db
}

func TestRecordBackground_NilGuards(t *testing.T) {
	db := newBgOperLogDB(t)
	mock := &bgMockRecorder{}

	// nil service / nil db → no-op 不 panic
	assert.NotPanics(t, func() {
		RecordBackground(nil, db, "资产管理", 1, "system-cron", nil)
		RecordBackground(nil, nil, "资产管理", 1, "system-cron", map[string]interface{}{"k": "v"})
	})
	assert.Equal(t, 0, mock.calls, "nil 守卫下不产生调用")

	var cnt int64
	require.NoError(t, db.Model(&models.OperLog{}).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt)
}

func TestRecordBackground_WithRecorder(t *testing.T) {
	db := newBgOperLogDB(t)
	mock := &bgMockRecorder{}

	assert.NotPanics(t, func() {
		RecordBackground(mock, db, "资产管理", 2, "system-cron", map[string]interface{}{"id": "a1"})
		RecordBackground(mock, db, "资产管理", 3, "system-cron", nil)
	})
	assert.Equal(t, 2, mock.calls)

	// 第一次调用:module + operType + CRON + system-cron + JSON params
	args := mock.lastArgs
	assert.Equal(t, "资产管理", args[0])
	assert.Equal(t, 3, args[1])
	assert.Equal(t, "CRON", args[2])
	opName, _ := args[3].(*string)
	require.NotNil(t, opName)
	assert.Equal(t, "system-cron", *opName)
	// nil params → operParam 为 nil
	assert.Nil(t, args[4])
}
