package addomain

import (
	"testing"
)

// TestParseOUDN tests the ParseOUDN function
func TestParseOUDN(t *testing.T) {
	tests := []struct {
		name     string
		ouDN     string
		expected int
	}{
		{
			name:     "Multi-level OU",
			ouDN:     "OU=临时账号,OU=自建账号",
			expected: 2,
		},
		{
			name:     "Single level OU",
			ouDN:     "OU=临时账号",
			expected: 1,
		},
		{
			name:     "Empty string",
			ouDN:     "",
			expected: 0,
		},
		{
			name:     "Three level OU",
			ouDN:     "OU=临时账号,OU=自建账号,OU=湖北分公司",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseOUDN(tt.ouDN)
			if len(result) != tt.expected {
				t.Errorf("ParseOUDN(%q) returned %d parts; want %d", tt.ouDN, len(result), tt.expected)
			}
		})
	}
}

// TestParseOUDNExtractDeptNames tests parsing OU DN and extracting department names
func TestParseOUDNExtractDeptNames(t *testing.T) {
	ouDN := "OU=临时账号,OU=自建账号"

	ouParts := ParseOUDN(ouDN)
	if len(ouParts) != 2 {
		t.Fatalf("ParseOUDN() returned %d parts; want 2", len(ouParts))
	}

	// Extract department names
	deptNames := []string{}
	for _, part := range ouParts {
		if len(part) > 3 && part[:3] == "OU=" {
			name := part[3:]
			deptNames = append(deptNames, name)
		}
	}

	if len(deptNames) != 2 {
		t.Fatalf("Extracted %d department names; want 2", len(deptNames))
	}
	if deptNames[0] != "临时账号" {
		t.Errorf("deptNames[0] = %q; want %q", deptNames[0], "临时账号")
	}
	if deptNames[1] != "自建账号" {
		t.Errorf("deptNames[1] = %q; want %q", deptNames[1], "自建账号")
	}
}
