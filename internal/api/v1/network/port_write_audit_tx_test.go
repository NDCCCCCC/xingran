package network

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ====================== DEBT-02 批量审计事务语义测试 ======================

// TestCreateAuditRowsInTx_AllOrNothing 事务原子性：任一行失败 → 全部回滚。
func TestCreateAuditRowsInTx_AllOrNothing(t *testing.T) {
	db := newAuditTestDB(t)

	// 第二行主键与第一行相同（BeforeCreate 只在 ID 空时生成 → 手工指定即冲突）
	// → 主键冲突让事务整体回滚，第一行不得落库
	dupPK := "dup-pk-0001"
	rows := []*models.PortWriteAudit{
		{ID: dupPK, DeviceID: "dev-1", PortID: "p1", Action: "shutdown", Status: "succeeded", Operator: "alice"},
		{ID: dupPK, DeviceID: "dev-1", PortID: "p2", Action: "shutdown", Status: "failed", Operator: "alice"},
	}
	ids, err := createAuditRowsInTx(db, rows)
	require.Error(t, err, "主键冲突必须让事务失败")
	assert.Nil(t, ids)

	var count int64
	require.NoError(t, db.Model(&models.PortWriteAudit{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "事务回滚后不得残留任何审计行（无孤儿）")
}

// TestCreateAuditRowsInTx_Success 全部成功 → 顺序返回全部 IDs。
func TestCreateAuditRowsInTx_Success(t *testing.T) {
	db := newAuditTestDB(t)
	rows := []*models.PortWriteAudit{
		{DeviceID: "dev-1", PortID: "p1", Action: "shutdown", Status: "succeeded", Operator: "bob"},
		{DeviceID: "dev-1", PortID: "p2", Action: "shutdown", Status: "failed", Operator: "bob"},
		{DeviceID: "dev-1", PortID: "p3", Action: "set_access_vlan", Status: "succeeded", Operator: "bob"},
	}
	ids, err := createAuditRowsInTx(db, rows)
	require.NoError(t, err)
	require.Len(t, ids, 3)

	var count int64
	require.NoError(t, db.Model(&models.PortWriteAudit{}).Count(&count).Error)
	assert.Equal(t, int64(3), count)

	// IDs 与行序一致
	for i, id := range ids {
		assert.Equal(t, rows[i].ID, id)
	}
}

// TestCreateAuditRowsInTx_Empty 空行集 → 空 IDs 无错误。
func TestCreateAuditRowsInTx_Empty(t *testing.T) {
	db := newAuditTestDB(t)
	ids, err := createAuditRowsInTx(db, nil)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

// newAuditTestDB 内存 sqlite + audit 表（独立于 network 包既有 helpers，避免耦合）。
func newAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.PortWriteAudit{}))
	return db
}
