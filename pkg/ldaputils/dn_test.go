package ldaputils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractOUDNFromUserDN(t *testing.T) {
	// 空
	assert.Equal(t, "", ExtractOUDNFromUserDN(""))

	// 只有 CN → 空
	assert.Equal(t, "", ExtractOUDNFromUserDN("CN=alice"))

	// 标准: CN + 多个 OU
	got := ExtractOUDNFromUserDN("CN=zhangsan,OU=科技创新部,OU=分公司本部,OU=湖北分公司,DC=company,DC=com")
	assert.Equal(t, "OU=科技创新部,OU=分公司本部,OU=湖北分公司,DC=company,DC=com", got)

	// 无 OU → 去掉 CN 返回 Base
	got = ExtractOUDNFromUserDN("CN=bob,DC=company,DC=com")
	assert.Equal(t, "DC=company,DC=com", got)
}

func TestParseOUDN(t *testing.T) {
	// 空
	assert.Equal(t, []string{}, ParseOUDN(""))

	// 标准 OU + DC 混合
	got := ParseOUDN("OU=基础运维科,OU=科技创新部,DC=PR,DC=cpic,DC=com")
	assert.Equal(t, []string{"OU=基础运维科", "OU=科技创新部"}, got)

	// 仅 DC → 空
	assert.Empty(t, ParseOUDN("DC=PR,DC=com"))
}

func TestExtractParentDN(t *testing.T) {
	assert.Equal(t, "", ExtractParentDN(""))
	assert.Equal(t, "", ExtractParentDN("OU=root"))
	assert.Equal(t, "DC=company,DC=com", ExtractParentDN("OU=dept,DC=company,DC=com"))
}

func TestBuildOUPath(t *testing.T) {
	// ouDN == baseDN → "/"
	assert.Equal(t, "/", BuildOUPath("DC=x,DC=com", "DC=x,DC=com"))

	// 嵌套 OU → 反序拼接
	got := BuildOUPath("OU=研发,OU=分公司,DC=x,DC=com", "DC=x,DC=com")
	assert.Equal(t, "/分公司/研发", got)
}