package addomain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractRDN(t *testing.T) {
	tests := []struct {
		name     string
		dn       string
		expected string
	}{
		{
			name:     "标准用户DN",
			dn:       "CN=zhangsan,OU=TestDept,DC=company,DC=com",
			expected: "CN=zhangsan",
		},
		{
			name:     "多层OU",
			dn:       "CN=lisi,OU=SubDept,OU=ParentDept,DC=company,DC=com",
			expected: "CN=lisi",
		},
		{
			name:     "单个部分",
			dn:       "CN=zhangsan",
			expected: "CN=zhangsan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRDNFromDN(tt.dn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractOUDNFromUserDN(t *testing.T) {
	tests := []struct {
		name     string
		userDN   string
		expected string
	}{
		{
			name:     "标准用户DN",
			userDN:   "CN=zhangsan,OU=TestDept,DC=company,DC=com",
			expected: "OU=TestDept,DC=company,DC=com",
		},
		{
			name:     "多层OU",
			userDN:   "CN=lisi,OU=SubDept,OU=ParentDept,DC=company,DC=com",
			expected: "OU=SubDept,OU=ParentDept,DC=company,DC=com",
		},
		{
			name:     "空DN",
			userDN:   "",
			expected: "",
		},
		{
			name:     "无OU",
			userDN:   "CN=zhangsan,DC=company,DC=com",
			expected: "DC=company,DC=com",
		},
		{
			name:     "CN在后",
			userDN:   "OU=TestDept,DC=company,DC=com,CN=zhangsan",
			expected: "OU=TestDept,DC=company,DC=com,CN=zhangsan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractOUDNFromUserDN(tt.userDN)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractRDN_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		dn       string
		expected string
	}{
		{
			name:     "空字符串",
			dn:       "",
			expected: "",
		},
		{
			name:     "只有RDN",
			dn:       "CN=zhangsan",
			expected: "CN=zhangsan",
		},
		{
			name:     "特殊字符",
			dn:       "CN=zhang san,OU=Test Dept,DC=company,DC=com",
			expected: "CN=zhang san",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRDNFromDN(tt.dn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractOUDNFromUserDN_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		userDN   string
		expected string
	}{
		{
			name:     "只有CN和DC",
			userDN:   "CN=zhangsan,DC=company,DC=com",
			expected: "DC=company,DC=com",
		},
		{
			name:     "OU在开头",
			userDN:   "OU=TestDept,DC=company,DC=com,CN=zhangsan",
			expected: "OU=TestDept,DC=company,DC=com,CN=zhangsan",
		},
		{
			name:     "多个OU",
			userDN:   "CN=user,OU=Level3,OU=Level2,OU=Level1,DC=company,DC=com",
			expected: "OU=Level3,OU=Level2,OU=Level1,DC=company,DC=com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractOUDNFromUserDN(tt.userDN)
			assert.Equal(t, tt.expected, result)
		})
	}
}
