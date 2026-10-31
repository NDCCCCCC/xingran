package browser

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// ====================== TIDY-03 配套：选择器上报明细测试 ======================

// TestSelectorEventCollector_EnvelopeFormat 契约形状：信封行是含 selectorEvents
// 键的单行 JSON，事件含必填 pageUrl/elementId/selector/success。
func TestSelectorEventCollector_EnvelopeFormat(t *testing.T) {
	c := NewSelectorEventCollector()
	c.ObserveNavigation("http://target/login")

	start := time.Now()
	c.Record("#btn-submit", start, nil)
	c.Record("//*[contains(text(),'登录')]", start, errors.New("element not found"))

	line := c.EnvelopeLine()
	if line == "" {
		t.Fatal("有事件时信封行不得为空")
	}
	if strings.ContainsAny(line, "\n\r") {
		t.Fatalf("信封必须是单行 JSON: %q", line)
	}

	// 模拟后端 selectorEventEnvelope 解析（struct 形状与
	// internal/services/rpa/selector_learner.go 对齐）
	var envelope struct {
		SelectorEvents []struct {
			PageURL      string `json:"pageUrl"`
			ElementID    string `json:"elementId"`
			Selector     string `json:"selector"`
			SelectorType string `json:"selectorType,omitempty"`
			Success      bool   `json:"success"`
			DurationMs   int64  `json:"durationMs,omitempty"`
			ErrorType    string `json:"errorType,omitempty"`
			ErrorMessage string `json:"errorMessage,omitempty"`
		} `json:"selectorEvents"`
	}
	if err := json.Unmarshal([]byte(line), &envelope); err != nil {
		t.Fatalf("信封 JSON 解析失败: %v", err)
	}
	if len(envelope.SelectorEvents) != 2 {
		t.Fatalf("期望 2 个事件，got %d", len(envelope.SelectorEvents))
	}
	ok := envelope.SelectorEvents[0]
	if !ok.Success || ok.PageURL != "http://target/login" || ok.ElementID == "" || ok.Selector != "#btn-submit" {
		t.Fatalf("success 事件契约字段不完整: %+v", ok)
	}
	fail := envelope.SelectorEvents[1]
	if fail.Success || fail.ErrorType == "" || fail.ErrorMessage == "" {
		t.Fatalf("failed 事件契约字段不完整: %+v", fail)
	}
}

// TestSelectorEventCollector_EmptyNoEnvelope 验收 #2：无事件不追加（空数组不发）。
func TestSelectorEventCollector_EmptyNoEnvelope(t *testing.T) {
	c := NewSelectorEventCollector()
	if line := c.EnvelopeLine(); line != "" {
		t.Fatalf("无事件时信封行必须为空串，got %q", line)
	}
}

// TestSelectorEventCollector_ErrorClassification 错误归类（timeout/not_found）。
func TestSelectorEventCollector_ErrorClassification(t *testing.T) {
	c := NewSelectorEventCollector()
	c.ObserveNavigation("http://x/")
	c.Record("#slow", time.Now().Add(-10*time.Second), errors.New("timeout waiting for element"))
	c.Record("#missing", time.Now(), errors.New("element not found"))

	line := c.EnvelopeLine()
	if !strings.Contains(line, `"errorType":"timeout"`) {
		t.Fatalf("超时错误应归类 timeout: %s", line)
	}
	if !strings.Contains(line, `"errorType":"not_found"`) {
		t.Fatalf("普通缺失应归类 not_found: %s", line)
	}
}
