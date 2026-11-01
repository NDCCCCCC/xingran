package migrations

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// TestMigrate231OperatorIndex_Idempotent 双方言幂等（验收 #2；sqlite 方言，
// PG 同语句 IF NOT EXISTS 语义一致，实机执行留 DBA 窗口）。
func TestMigrate231OperatorIndex_Idempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.PortWriteAudit{}))

	// 首次执行：建索引
	require.NoError(t, Migrate231PortWriteAuditOperatorIndex(db))
	assert.True(t, db.Migrator().HasIndex("sys_port_write_audit", "idx_port_write_audit_operator_created"))

	// 二次执行：幂等跳过不报错
	require.NoError(t, Migrate231PortWriteAuditOperatorIndex(db))
}

// TestMigrate232BatchIdColumn_Idempotent batch_id 列双方言幂等（sqlite 方言）。
func TestMigrate232BatchIdColumn_Idempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.PortWriteAudit{}))

	require.NoError(t, Migrate232PortWriteAuditBatchID(db))
	assert.True(t, db.Migrator().HasColumn(&models.PortWriteAudit{}, "batch_id"))
	// 二次执行幂等
	require.NoError(t, Migrate232PortWriteAuditBatchID(db))
}
