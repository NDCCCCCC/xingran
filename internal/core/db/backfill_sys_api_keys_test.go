package db

import (
	"crypto/subtle"
	"encoding/hex"
	"path/filepath"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/config"
)

// 新建 sqlite 测试库(纯 Go glebarez 驱动,与 NewDatabase 生产路径同一入口)
func newBackfillTestDB(t *testing.T) *Database {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "backfill-test.db")
	d, err := NewDatabase(&config.DatabaseConfig{Type: "sqlite", Path: dbPath})
	if err != nil {
		t.Fatalf("创建 sqlite 测试库失败: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// createLegacyAPIKeysTable 建立带旧明文 key 列的 legacy 形态表(SEC-01 迁移前 schema)
func createLegacyAPIKeysTable(t *testing.T, d *Database) {
	t.Helper()
	if err := d.DB.Exec(`CREATE TABLE sys_api_keys (
		id text PRIMARY KEY,
		name text,
		key text NOT NULL
	)`).Error; err != nil {
		t.Fatalf("建 legacy 表失败: %v", err)
	}
}

func insertLegacyKeyRow(t *testing.T, d *Database, id, name, key string) {
	t.Helper()
	if err := d.DB.Exec("INSERT INTO sys_api_keys (id, name, key) VALUES (?,?,?)", id, name, key).Error; err != nil {
		t.Fatalf("插入 legacy 行失败: %v", err)
	}
}

// backfillAPIKeyRow 读取迁移后单行的三列值
type backfillAPIKeyRow struct {
	Salt      string
	KeyHash   string
	KeyPrefix string
}

func queryBackfilledRow(t *testing.T, d *Database, id string) backfillAPIKeyRow {
	t.Helper()
	var row backfillAPIKeyRow
	if err := d.DB.Raw("SELECT salt, key_hash, key_prefix FROM sys_api_keys WHERE id = ?", id).
		Scan(&row).Error; err != nil {
		t.Fatalf("查询回填行失败: %v", err)
	}
	return row
}

// TestBackfillSysAPIKeysKeyHash_LegacyTableBackfillsAndDropsKey 存量表迁移主路径:
// 三列回填值与 apikey_service.go 算法逐字节一致 + 旧明文列删除 + ValidateAPIKey 查找语义成立。
func TestBackfillSysAPIKeysKeyHash_LegacyTableBackfillsAndDropsKey(t *testing.T) {
	d := newBackfillTestDB(t)
	createLegacyAPIKeysTable(t, d)

	// 与 generateKey() 同形态: "rec_" + 64 hex chars(68 字符)
	key1 := "rec_" + "0ab3f7d2c9e15486ab01cf23456789fedcba9876543210ffeeddccbbaa998877"
	key2 := "rec_" + "7e2155dd99bb00ff11aa22bb33cc44dd55ee66ff778899001122334455667788"
	if len(key1) != 68 || len(key2) != 68 {
		t.Fatalf("测试密钥长度非法: %d/%d", len(key1), len(key2))
	}
	insertLegacyKeyRow(t, d, "11111111-1111-1111-1111-111111111111", "key-a", key1)
	insertLegacyKeyRow(t, d, "22222222-2222-2222-2222-222222222222", "key-b", key2)

	if err := d.backfillSysAPIKeysKeyHash(); err != nil {
		t.Fatalf("回填失败: %v", err)
	}

	// 旧列已删,新列就位
	if d.tableColumnExists(d.DB, "sys_api_keys", "key") {
		t.Errorf("旧明文 key 列应已删除")
	}
	for _, col := range []string{"key_hash", "salt", "key_prefix"} {
		if !d.tableColumnExists(d.DB, "sys_api_keys", col) {
			t.Errorf("新列 %s 应已创建", col)
		}
	}

	// 行级断言:格式 + 算法逐字节一致(与 hashAPIKey 相同的 SM3(key‖salt) hex)
	row1 := queryBackfilledRow(t, d, "11111111-1111-1111-1111-111111111111")
	row2 := queryBackfilledRow(t, d, "22222222-2222-2222-2222-222222222222")

	for name, row := range map[string]backfillAPIKeyRow{"row1": row1, "row2": row2} {
		if len(row.Salt) != 32 {
			t.Errorf("%s salt 长度应 32(hex 16 字节),实际 %d", name, len(row.Salt))
		}
		if _, err := hex.DecodeString(row.Salt); err != nil {
			t.Errorf("%s salt 应为合法 hex: %v", name, err)
		}
		if len(row.KeyHash) != 64 {
			t.Errorf("%s key_hash 长度应 64,实际 %d", name, len(row.KeyHash))
		}
	}

	if row1.KeyPrefix != key1[:12] {
		t.Errorf("row1 key_prefix 应为 %q,实际 %q", key1[:12], row1.KeyPrefix)
	}
	if row2.KeyPrefix != key2[:12] {
		t.Errorf("row2 key_prefix 应为 %q,实际 %q", key2[:12], row2.KeyPrefix)
	}
	// per-row 独立随机盐
	if row1.Salt == row2.Salt {
		t.Errorf("两行盐值不应相同(per-key 独立随机盐)")
	}

	// ValidateAPIKey 核心比对语义(prefix 缩窄 + SM3(key+salt) 恒定时间比对)对迁移后行成立
	var candidates []backfillAPIKeyRow
	if err := d.DB.Raw("SELECT salt, key_hash, key_prefix FROM sys_api_keys WHERE key_prefix = ?", key1[:12]).
		Scan(&candidates).Error; err != nil {
		t.Fatalf("prefix 缩窄查询失败: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("prefix 缩窄应命中 1 行,实际 %d", len(candidates))
	}
	computed := backfillAPIKeyHash(key1, candidates[0].Salt)
	if subtle.ConstantTimeCompare([]byte(computed), []byte(candidates[0].KeyHash)) != 1 {
		t.Errorf("SM3(key+salt) 恒定时间比对失败,ValidateAPIKey 将拒绝该存量密钥")
	}
}

// TestBackfillSysAPIKeysKeyHash_IdempotentNoOpAfterMigration 迁移完成后再次启动应 no-op:
// 无旧 key 列且有 key_hash 列 → 不改写任何已回填值。
func TestBackfillSysAPIKeysKeyHash_IdempotentNoOpAfterMigration(t *testing.T) {
	d := newBackfillTestDB(t)
	createLegacyAPIKeysTable(t, d)
	key1 := "rec_" + "0ab3f7d2c9e15486ab01cf23456789fedcba9876543210ffeeddccbbaa998877"
	insertLegacyKeyRow(t, d, "11111111-1111-1111-1111-111111111111", "key-a", key1)

	if err := d.backfillSysAPIKeysKeyHash(); err != nil {
		t.Fatalf("首次回填失败: %v", err)
	}
	first := queryBackfilledRow(t, d, "11111111-1111-1111-1111-111111111111")

	if err := d.backfillSysAPIKeysKeyHash(); err != nil {
		t.Fatalf("二次运行(应 no-op)失败: %v", err)
	}
	second := queryBackfilledRow(t, d, "11111111-1111-1111-1111-111111111111")

	if first != second {
		t.Errorf("二次运行不应改写已迁移行:\nfirst=%+v\nsecond=%+v", first, second)
	}
}

// TestBackfillSysAPIKeysKeyHash_AlreadyMigratedNoOp 全新模型建表(无旧 key 列)应 no-op。
func TestBackfillSysAPIKeysKeyHash_AlreadyMigratedNoOp(t *testing.T) {
	d := newBackfillTestDB(t)
	if err := d.DB.Exec(`CREATE TABLE sys_api_keys (
		id text PRIMARY KEY,
		key_hash text,
		salt text,
		key_prefix text
	)`).Error; err != nil {
		t.Fatalf("建已迁移形态表失败: %v", err)
	}
	if err := d.DB.Exec("INSERT INTO sys_api_keys (id, key_hash, salt, key_prefix) VALUES (?,?,?,?)",
		"33333333-3333-3333-3333-333333333333", "hash-keep", "salt-keep", "prefix-keep").Error; err != nil {
		t.Fatalf("插入行失败: %v", err)
	}

	if err := d.backfillSysAPIKeysKeyHash(); err != nil {
		t.Fatalf("已迁移态运行失败: %v", err)
	}
	row := queryBackfilledRow(t, d, "33333333-3333-3333-3333-333333333333")
	if row.KeyHash != "hash-keep" || row.Salt != "salt-keep" || row.KeyPrefix != "prefix-keep" {
		t.Errorf("已迁移态应 no-op,实际被改写: %+v", row)
	}
}

// TestBackfillSysAPIKeysKeyHash_NoTableNoOp 表不存在(全新部署)应 no-op。
func TestBackfillSysAPIKeysKeyHash_NoTableNoOp(t *testing.T) {
	d := newBackfillTestDB(t)
	if err := d.backfillSysAPIKeysKeyHash(); err != nil {
		t.Errorf("表不存在时应 no-op nil,实际: %v", err)
	}
}

// TestBackfillSysAPIKeysKeyHash_ShortKeyFailsFast 短于 12 字符的存量 key 无法派生
// key_prefix,应 fail-fast 报错而非静默产出错误数据。
func TestBackfillSysAPIKeysKeyHash_ShortKeyFailsFast(t *testing.T) {
	d := newBackfillTestDB(t)
	createLegacyAPIKeysTable(t, d)
	insertLegacyKeyRow(t, d, "44444444-4444-4444-4444-444444444444", "short-key", "rec_short")

	if err := d.backfillSysAPIKeysKeyHash(); err == nil {
		t.Errorf("短 key 行应报错 fail-fast")
	}
}
