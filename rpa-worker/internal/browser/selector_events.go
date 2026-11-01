package browser

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Phase 122 / TIDY-03 配套：选择器明细上报（契约 docs/RPA-选择器上报明细契约.md）。
//
// worker 侧职责（契约 §4）：执行过程中每次选择器定位（成功/失败）累积一个事件；
// 执行收尾把事件数组序列化为「一行 JSON 信封」追加进执行记录日志上报体。
// 零新协议：信封作为普通日志行走既有 Progress 通道，后端
// SelectorLearner.LearnFromExecution 按 logs 逐行解析消费。

// SelectorEvent 单次选择器定位事件（契约 §2 JSON 结构）。
type SelectorEvent struct {
	PageURL      string `json:"pageUrl"`                // 必填：页面 URL
	ElementID    string `json:"elementId"`              // 必填：元素标识
	Selector     string `json:"selector"`               // 必填：实际使用的选择器
	SelectorType string `json:"selectorType,omitempty"` // 可选：css|xpath|text|aria|data-testid
	Success      bool   `json:"success"`                // 必填：本次命中结果
	DurationMs   int64  `json:"durationMs,omitempty"`   // success 事件：查找耗时（毫秒）
	ErrorType    string `json:"errorType,omitempty"`    // failed：timeout|not_found|multiple_matches
	ErrorMessage string `json:"errorMessage,omitempty"` // failed：原始错误
}

// SelectorEventCollector 选择器事件累积器（单执行单 goroutine，零锁设计；
// mutex 仅防御 worker 复用 page 的边缘场景）。
type SelectorEventCollector struct {
	mu        sync.Mutex
	events    []SelectorEvent
	lastURL   string
	elementSeq int
}

// NewSelectorEventCollector 构造。
func NewSelectorEventCollector() *SelectorEventCollector {
	return &SelectorEventCollector{}
}

// ObserveNavigation 页面导航时更新 pageUrl 来源。
func (c *SelectorEventCollector) ObserveNavigation(url string) {
	if c == nil || url == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastURL = url
}

// Record 记录一次选择器定位结果。
//
// elementID 从 selector 派生（契约 elementId 语义=元素标识；worker 侧无独立
// 元素 ID 概念，取 selector 归一串，去重由后端 upsert 承担）。
func (c *SelectorEventCollector) Record(selector string, start time.Time, err error) {
	if c == nil || selector == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.elementSeq++
	ev := SelectorEvent{
		PageURL:      c.lastURL,
		ElementID:    elementIDFromSelector(selector, c.elementSeq),
		Selector:     selector,
		SelectorType: classifySelector(selector),
	}
	if err != nil {
		ev.Success = false
		ev.DurationMs = time.Since(start).Milliseconds()
		ev.ErrorType = classifySelectorError(err, start)
		ev.ErrorMessage = err.Error()
	} else {
		ev.Success = true
		ev.DurationMs = time.Since(start).Milliseconds()
	}
	c.events = append(c.events, ev)
}

// EnvelopeLine 契约信封行：无事件返回 ""（验收 #2：空数组不发）。
func (c *SelectorEventCollector) EnvelopeLine() string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.events) == 0 {
		return ""
	}
	data, err := json.Marshal(map[string]interface{}{"selectorEvents": c.events})
	if err != nil {
		return ""
	}
	return string(data)
}

// elementIDFromSelector 从选择器派生稳定元素标识（#id 优先；否则类型+序号）。
func elementIDFromSelector(selector string, seq int) string {
	s := strings.TrimSpace(selector)
	if strings.HasPrefix(s, "#") && len(s) > 1 {
		return strings.SplitN(s[1:], " ", 2)[0]
	}
	// data-testid="x" / 其他：取尾部标识 + 序号保证唯一
	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '"' || r == '\'' {
			return -1
		}
		return r
	}, s)
	if len(compact) > 40 {
		compact = compact[:40]
	}
	return compact + "#" + strconv.Itoa(seq)
}

// classifySelector 粗分类选择器类型（契约 selectorType 可选字段）。
func classifySelector(selector string) string {
	switch {
	case strings.HasPrefix(selector, "/") || strings.HasPrefix(selector, "("):
		return "xpath"
	case strings.HasPrefix(selector, "#"):
		return "css"
	case strings.HasPrefix(selector, "text="):
		return "text"
	case strings.Contains(selector, "data-testid"):
		return "data-testid"
	case strings.Contains(selector, "[aria"):
		return "aria"
	default:
		return "css"
	}
}

// classifySelectorError 错误归类（契约 errorType：timeout|not_found|multiple_matches）。
func classifySelectorError(err error, start time.Time) string {
	msg := ""
	if err != nil {
		msg = strings.ToLower(err.Error())
	}
	switch {
	case strings.Contains(msg, "multiple") || strings.Contains(msg, "resolve") && strings.Contains(msg, "than one"):
		return "multiple_matches"
	case strings.Contains(msg, "timeout") || time.Since(start) > 5*time.Second:
		return "timeout"
	default:
		return "not_found"
	}
}
