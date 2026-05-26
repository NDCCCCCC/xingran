package templates

import (
	"testing"
)

func TestDot1xTemplateParsing(t *testing.T) {
	template := `Value Required INTERFACE ([\w\.\/\-]+)
Value DOT1X_ENABLED (Enabled|Disabled)
Value CURRENT_USERS (\d+)

Start
  ^<\w+>.*$$ -> Start
  ^\s*$$ -> Start
  ^Max users:.*$$ -> Start
  ^Current users:.*$$ -> Start
  ^Global default domain.*$$ -> Start
  ^Dot1x abnormal-track.*$$ -> Start
  ^Quiet function is.*$$ -> Start
  ^Mc-trigger port-up-send.*$$ -> Start
  ^Parameter set:.*$$ -> Start
  ^Dot1x URL:.*$$ -> Start
  ^${INTERFACE}\s+status:.*802\.1x.*$$ -> InterfaceInfo
  ^Return.*$$ -> Start
  ^. -> Error

InterfaceInfo
  ^\s+Current users:\s+${CURRENT_USERS}$$ -> Record
  ^. -> Continue
  ^\s*$$ -> Start`

	fsm, err := ParseTemplateString(template)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	testLine := "GigabitEthernet0/0/1 status: UP  802.1x protocol is Enabled"

	// Find the rule that should match (the one with INTERFACE and 802.1x)
	var targetRule *Rule
	for _, rule := range fsm.States["Start"].Rules {
		if len(rule.VarNames) > 0 && rule.VarNames[0] == "INTERFACE" {
			targetRule = rule
			break
		}
	}

	if targetRule == nil {
		t.Fatal("Could not find the INTERFACE rule in Start state")
	}

	if !targetRule.Regex.MatchString(testLine) {
		t.Errorf("INTERFACE rule did not match test line")
		t.Errorf("  Rule pattern: %s", targetRule.RegexPattern)
		t.Errorf("  Rule compiled regex: %s", targetRule.Regex.String())
		t.Errorf("  Test line: %s", testLine)
	}

	// Verify the compiled regex has the correct \. (not \\\.) for 802.1x
	expectedPattern := `^([\w\.\/\-]+)\s+status:.*802\.1x.*$`
	if targetRule.Regex.String() != expectedPattern {
		t.Errorf("Compiled regex mismatch:\n  got:  %s\n  want: %s", targetRule.Regex.String(), expectedPattern)
	}

	// Verify capture groups extract the correct interface name
	matches := targetRule.Regex.FindStringSubmatch(testLine)
	if len(matches) < 2 || matches[1] != "GigabitEthernet0/0/1" {
		t.Errorf("INTERFACE capture failed: matches=%v", matches)
	}
}

func TestDot1xCatchAllRule(t *testing.T) {
	// Test that ^. -> Continue compiles to ^. (wildcard, not ^\. literal dot)
	template := `Value Required INTERFACE ([\w\.\/\-]+)
Value CURRENT_USERS (\d+)

Start
  ^${INTERFACE}\s+status:.*$$ -> InterfaceInfo
  ^. -> Error

InterfaceInfo
  ^. -> Continue
  ^\s*$$ -> Start`

	fsm, err := ParseTemplateString(template)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	// The ^. -> Error rule in Start should match any single character
	for _, rule := range fsm.States["Start"].Rules {
		if rule.NextState == "Error" {
			// Should be "^." not "^\\."
			if rule.Regex.String() != "^." {
				t.Errorf("Start catch-all rule: got %q, want %q", rule.Regex.String(), "^.")
			}
			// Should match "X", "a", etc.
			if !rule.Regex.MatchString("X") {
				t.Errorf("Start catch-all rule should match 'X'")
			}
			break
		}
	}

	// The ^. -> Continue rule in InterfaceInfo should match any line
	for _, rule := range fsm.States["InterfaceInfo"].Rules {
		if rule.NextState == "Continue" {
			if rule.Regex.String() != "^." {
				t.Errorf("InterfaceInfo catch-all rule: got %q, want %q", rule.Regex.String(), "^.")
			}
			// Should match "anything here"
			if !rule.Regex.MatchString("anything here") {
				t.Errorf("InterfaceInfo catch-all rule should match 'anything here'")
			}
			break
		}
	}
}

func TestEscapeRegexLiteralMetaCharEscapes(t *testing.T) {
	// Test that escapeRegexLiteral correctly handles TextFSM template patterns.
	// In TextFSM templates, bare . is a regex wildcard (not escaped),
	// and \. is an escaped literal dot (preserved as-is).
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"escaped dot", `802\.1x`, `802\.1x`},
		{"escaped paren", `\(test\)`, `\(test\)`},
		{"escaped bracket", `\[test\]`, `\[test\]`},
		{"escaped backslash", `\\test`, `\\test`},
		{"escaped pipe", `a\|b`, `a\|b`},
		{"escaped dollar", `\$`, `\$`},
		{"escaped caret", `\^`, `\^`},
		{"escaped asterisk", `\*`, `\*`},
		{"escaped plus", `\+`, `\+`},
		{"escaped question", `\?`, `\?`},
		{"escaped curly", `\{n\}`, `\{n\}`},
		{"char class s", `\s+`, `\s+`},
		{"char class d", `\d+`, `\d+`},
		{"char class w", `\w+`, `\w+`},
		{"dot star wildcard", `.*`, `.*`},
		{"dot plus wildcard", `.+`, `.+`},
		{"bare dot is wildcard", `end.`, `end.`},
		{"single dot", `.`, `.`},
		{"literal colon", `:`, `:`},
		{"literal space", `hello world`, `hello world`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeRegexLiteral(tt.input)
			if result != tt.expected {
				t.Errorf("escapeRegexLiteral(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
