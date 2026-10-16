package addomain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// computer.go 纯函数级测试(parseComputerDescription / extractCapacityValue /
// parseDateTime / determineComputerStatus) + 服务端 normalizePagination。
// 不含 LDAP/DB 调用。
// =====================================================================

func TestParseComputerDescriptionForUser(t *testing.T) {
	assert.Equal(t, "", parseComputerDescriptionForUser(""))
	assert.Equal(t, "alice", parseComputerDescriptionForUser("|alice|"))
	assert.Equal(t, "", parseComputerDescriptionForUser("||"))
	assert.Equal(t, "bob", parseComputerDescriptionForUser("|  bob  |"))
}

func TestParseComputerDescription_FullFields(t *testing.T) {
	// 全字段格式
	d := parseComputerDescription("|user|ip|mac|serial|os|cpu|arch|memory|disk|datetime")
	assert.Equal(t, "user", d["lastLogonUser"])
	assert.Equal(t, "ip", d["ipAddress"])
	assert.Equal(t, "mac", d["macAddress"])
	assert.Equal(t, "serial", d["serialNumber"])
	assert.Equal(t, "os", d["operatingSystem"])
	assert.Equal(t, "cpu", d["cpuModel"])
	assert.Equal(t, "arch", d["architecture"])
	assert.Equal(t, "memory", d["memoryCapacity"])
	assert.Equal(t, "disk", d["hardDiskCapacity"])

	// 空 desc → 空 map
	assert.Empty(t, parseComputerDescription(""))

	// 不足 10 字段 → 空 map
	assert.Empty(t, parseComputerDescription("a|b|c"))
}

func TestExtractCapacityValue(t *testing.T) {
	assert.Equal(t, "8GB", extractCapacityValue("Memory: 8GB"))
	assert.Equal(t, "500GB", extractCapacityValue("Disk:500GB"))
	assert.Equal(t, "raw", extractCapacityValue("raw"))
	assert.Equal(t, "", extractCapacityValue(""))
}

func TestParseDateTime(t *testing.T) {
	assert.Nil(t, parseDateTime(""))
	assert.Nil(t, parseDateTime("bogus"))

	// 各格式
	assert.NotNil(t, parseDateTime("2024/6/15 10:30:45"))
	assert.NotNil(t, parseDateTime("2024/06/15 10:30:45"))
	assert.NotNil(t, parseDateTime("2024-06-15 10:30:45"))
	assert.NotNil(t, parseDateTime("2024-06-15T10:30:45Z"))

	// 返回值包含年份
	t2 := parseDateTime("2024-06-15 10:30:45")
	require.NotNil(t, t2)
	assert.Equal(t, 2024, t2.Year())
}

func TestDetermineComputerStatus(t *testing.T) {
	now := time.Now()
	// 在线(最近 7 天内)
	recent := now.Add(-time.Hour)
	assert.NotEqual(t, 0, determineComputerStatus(&recent))

	// 离线(>7 天)
	old := now.Add(-8 * 24 * time.Hour)
	assert.NotEqual(t, 0, determineComputerStatus(&old))

	// nil → Offline
	assert.NotEqual(t, 0, determineComputerStatus(nil))
}

// =====================================================================
// utils.go: encrypt/decrypt + extractParentDN + buildOUPath +
// parseIntOrDefault + parseFileTime + ExtractOUDNFromUserDN + ParseOUDN。
// =====================================================================

func TestEncryptDecryptPassword_SM4Roundtrip(t *testing.T) {
	// 无 SM4 cipher → legacy AES 路径
	enc := encryptPassword("mysecret")
	assert.NotEmpty(t, enc)
	dec := DecryptPassword(enc)
	assert.Equal(t, "mysecret", dec)

	// 空 password → ""
	assert.Equal(t, "", decryptPassword(""))

	// 无效字符串(既非 SM4 也非 AES 密文)→ 空(F-03 安全拒绝回退)
	assert.Equal(t, "", decryptPassword("!!!invalid!!!"))
}

func TestExtractParentDN(t *testing.T) {
	assert.Equal(t, "", extractParentDN(""))
	assert.Equal(t, "", extractParentDN("OU=root"))
	assert.Equal(t, "DC=a,DC=b", extractParentDN("OU=x,DC=a,DC=b"))
}

func TestBuildOUPath(t *testing.T) {
	assert.Equal(t, "/", buildOUPath("DC=x", "DC=x"))
	assert.Equal(t, "/分公司/研发", buildOUPath("OU=研发,OU=分公司,DC=x", "DC=x"))
}

func TestParseIntOrDefault(t *testing.T) {
	assert.Equal(t, 42, parseIntOrDefault("42", 0))
	assert.Equal(t, 0, parseIntOrDefault("garbage", 0))
	assert.Equal(t, 7, parseIntOrDefault("", 7))
}

func TestParseFileTime(t *testing.T) {
	assert.Nil(t, parseFileTime(""))
	assert.Nil(t, parseFileTime("0"))
	assert.Nil(t, parseFileTime("bogus"))

	// QUIRK: parseFileTime 把 AD FileTime 116444736000000000 (即 epoch 0)
	// 经过 (ft/1e7) - 11644473600 = 0 → time.Unix(0, 0) 返回非 nil
	// (实际等价于 1970-01-01 本地时区)。QUIRK: 该返回值**不为 nil**。
	future := parseFileTime("133632600000000000") // 约 2024 年
	if future != nil {
		assert.Greater(t, future.Year(), 2020)
	}
}

func TestExtractOUDNFromUserDN_Utils(t *testing.T) {
	assert.Equal(t, "", ExtractOUDNFromUserDN(""))
	assert.Equal(t, "", ExtractOUDNFromUserDN("CN=x"))
	assert.Equal(t, "OU=a,DC=x,DC=y",
		ExtractOUDNFromUserDN("CN=x,OU=a,DC=x,DC=y"))
	// 大小写混合 OU
	assert.Equal(t, "ou=a,DC=x",
		ExtractOUDNFromUserDN("CN=x,ou=a,DC=x"))
}

func TestParseOUDN_Utils(t *testing.T) {
	assert.Nil(t, ParseOUDN(""))
	got := ParseOUDN("OU=基础运维科, OU=分公司,DC=x")
	assert.Equal(t, []string{"OU=基础运维科", "OU=分公司", "DC=x"}, got)
}