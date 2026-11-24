package browser

import (
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// ============================================================================
// Phase 155 / LIVEVIEW-02：输入注入（注入同一 page）
//
// 复用 rod 已有的 `proto.Input.*`，零新增依赖：
//   - 鼠标 move/down/up/wheel → `Input.dispatchMouseEvent`
//   - 键盘 down/up/text       → `Input.dispatchKeyEvent` / `Input.dispatchTouchEvent` 不用
//   - 粘贴式文本（含中文/IME）→ `Input.insertText`（dispatchKeyEvent 的 text 字段
//     对 CJK 不可靠，rod 的 Keyboard.Type 内部也是 insertText 路径）
//
// 坐标系：前端已经把「画布像素」换算成 **viewport CSS 像素**，Worker 直接透传，
// 不再做二次换算。这是 LIVEVIEW-02「坐标系缩放正确」的唯一约定点：
//
//	canvasX / canvasWidth  * deviceWidth  = cdpX
//	canvasY / canvasHeight * deviceHeight = cdpY
//
// 修饰键位掩码与 CDP 一致：Alt=1, Ctrl=2, Meta=4, Shift=8。
// ============================================================================

// CDP 修饰键位掩码（与 proto.InputDispatch* 的 Modifiers 字段同值）。
const (
	ModifierAlt     = 1
	ModifierControl = 2
	ModifierMeta    = 4
	ModifierShift   = 8
)

// DispatchLiveInput 把一条前端输入指令注入到 page（必须是 RawPage，当前活动页签）。
//
// 返回错误时调用方（Worker 的 liveview hub）应把错误记审计并通知前端，
// 绝不静默吞掉 —— 否则用户点不到东西却看不到任何提示。
func DispatchLiveInput(page *rod.Page, cmd *types.LiveInputCommand) error {
	if page == nil {
		return fmt.Errorf("输入注入失败: page 为 nil")
	}
	if cmd == nil {
		return fmt.Errorf("输入注入失败: 指令为 nil")
	}
	if !types.IsValidLiveInputKind(cmd.Kind) {
		return fmt.Errorf("输入注入失败: 未知输入类型 %q", cmd.Kind)
	}

	switch cmd.Kind {
	case types.LiveInputMouseMove:
		return dispatchMouseMove(page, cmd)
	case types.LiveInputMouseDown:
		return dispatchMouseDown(page, cmd)
	case types.LiveInputMouseUp:
		return dispatchMouseUp(page, cmd)
	case types.LiveInputMouseWheel:
		return dispatchMouseWheel(page, cmd)
	case types.LiveInputKeyDown:
		return dispatchKeyDown(page, cmd)
	case types.LiveInputKeyUp:
		return dispatchKeyUp(page, cmd)
	case types.LiveInputText:
		return dispatchText(page, cmd)
	}
	return fmt.Errorf("输入注入失败: 未处理的类型 %q", cmd.Kind)
}

// buttonsMask CDP 的 buttons 位：Left=1, Right=2, Middle=4。
func buttonsMask(button string) int {
	switch strings.ToLower(button) {
	case "left":
		return 1
	case "right":
		return 2
	case "middle":
		return 4
	}
	return 0
}

// mouseButton 归一 CDP 鼠标键名（非法值回落 none —— 即「只移动不按」）。
func mouseButton(button string) proto.InputMouseButton {
	switch strings.ToLower(button) {
	case "left":
		return proto.InputMouseButtonLeft
	case "right":
		return proto.InputMouseButtonRight
	case "middle":
		return proto.InputMouseButtonMiddle
	}
	return proto.InputMouseButtonNone
}

// clampClickCount 点击次数合法化（1 或 2，其余按 1）。
func clampClickCount(n int) int {
	if n == 2 {
		return 2
	}
	return 1
}

func dispatchMouseMove(page *rod.Page, cmd *types.LiveInputCommand) error {
	// 移动事件携带「当前按住的键」位掩码：按住拖拽时前端会持续带 buttons，
	// 否则页面上的 drag 处理收不到 buttons 而判定为 hover。
	var pressed *int
	if cmd.Button != "" {
		mask := buttonsMask(cmd.Button)
		pressed = &mask
	}
	return (proto.InputDispatchMouseEvent{
		Type:        proto.InputDispatchMouseEventTypeMouseMoved,
		X:           cmd.X,
		Y:           cmd.Y,
		Button:      mouseButton(cmd.Button),
		Buttons:     pressed,
		Modifiers:   cmd.Modifiers,
		PointerType: proto.InputDispatchMouseEventPointerTypeMouse,
	}).Call(page)
}

func dispatchMouseDown(page *rod.Page, cmd *types.LiveInputCommand) error {
	count := clampClickCount(cmd.ClickCount)
	mask := buttonsMask(cmd.Button)
	return (proto.InputDispatchMouseEvent{
		Type:        proto.InputDispatchMouseEventTypeMousePressed,
		X:           cmd.X,
		Y:           cmd.Y,
		Button:      mouseButton(cmd.Button),
		Buttons:     &mask,
		ClickCount:  count,
		Modifiers:   cmd.Modifiers,
		PointerType: proto.InputDispatchMouseEventPointerTypeMouse,
	}).Call(page)
}

func dispatchMouseUp(page *rod.Page, cmd *types.LiveInputCommand) error {
	// 抬起后 buttons 归零（CDP 要求：release 时 buttons=0）。
	zero := 0
	return (proto.InputDispatchMouseEvent{
		Type:        proto.InputDispatchMouseEventTypeMouseReleased,
		X:           cmd.X,
		Y:           cmd.Y,
		Button:      mouseButton(cmd.Button),
		Buttons:     &zero,
		ClickCount:  clampClickCount(cmd.ClickCount),
		Modifiers:   cmd.Modifiers,
		PointerType: proto.InputDispatchMouseEventPointerTypeMouse,
	}).Call(page)
}

func dispatchMouseWheel(page *rod.Page, cmd *types.LiveInputCommand) error {
	return (proto.InputDispatchMouseEvent{
		Type:        proto.InputDispatchMouseEventTypeMouseWheel,
		X:           cmd.X,
		Y:           cmd.Y,
		DeltaX:      cmd.DeltaX,
		DeltaY:      cmd.DeltaY,
		Modifiers:   cmd.Modifiers,
		PointerType: proto.InputDispatchMouseEventPointerTypeMouse,
	}).Call(page)
}

func dispatchKeyUp(page *rod.Page, cmd *types.LiveInputCommand) error {
	return (proto.InputDispatchKeyEvent{
		Type:                  proto.InputDispatchKeyEventTypeKeyUp,
		Modifiers:             cmd.Modifiers,
		Key:                   cmd.Key,
		Code:                  cmd.Code,
		WindowsVirtualKeyCode: cmd.VirtualKey,
		NativeVirtualKeyCode:  cmd.VirtualKey,
	}).Call(page)
}

func dispatchText(page *rod.Page, cmd *types.LiveInputCommand) error {
	if cmd.Text == "" {
		return nil
	}
	// insertText 是 IME/粘贴式输入的权威路径，对 CJK 与 emoji 都可靠。
	return (proto.InputInsertText{Text: cmd.Text}).Call(page)
}

// dispatchKeyDown 键盘按下。
//
// 关键细节（LIVEVIEW-02 验收要求「组合键」可用）：
//   - 可打印单字符必须同时带 `text`（否则 input 的 value 不会变化 —— 很多页面
//     依赖 input 事件而不是 keydown）；
//   - 非可打印键（Enter/Tab/方向键/F1-F12）必须是 `keyDown` 而非 `rawKeyDown`，
//     否则部分页面的 keypress 语义缺失；
//   - `VirtualKey` 由前端从 KeyboardEvent.keyCode 带来，缺省时按 key 名回落。
func dispatchKeyDown(page *rod.Page, cmd *types.LiveInputCommand) error {
	ev := proto.InputDispatchKeyEvent{
		Type:                  proto.InputDispatchKeyEventTypeKeyDown,
		Modifiers:             cmd.Modifiers,
		Key:                   cmd.Key,
		Code:                  cmd.Code,
		WindowsVirtualKeyCode: cmd.VirtualKey,
		NativeVirtualKeyCode:  cmd.VirtualKey,
	}
	if text, ok := printableText(cmd.Key, cmd.Modifiers); ok {
		ev.Text = text
		ev.UnmodifiedText = text
	}
	if ev.WindowsVirtualKeyCode == 0 {
		ev.WindowsVirtualKeyCode = fallbackVirtualKey(cmd.Key)
		ev.NativeVirtualKeyCode = ev.WindowsVirtualKeyCode
	}
	return ev.Call(page)
}

// printableText 判定该键是否产出可打印字符（同时处理 Shift 大写与 Ctrl 快捷键）。
//
// Ctrl/Meta 组合是快捷键，不能带 text（否则 Chrome 会把 "a" 打进输入框，
// 而不是执行全选）。
func printableText(key string, modifiers int) (string, bool) {
	if len([]rune(key)) != 1 {
		return "", false
	}
	if modifiers&(ModifierControl|ModifierMeta|ModifierAlt) != 0 {
		return "", false
	}
	r := []rune(key)[0]
	if r < 0x20 {
		return "", false
	}
	return key, true
}

// fallbackVirtualKey 常见非打印键的 Windows 虚拟键码兜底（前端未带时用）。
func fallbackVirtualKey(key string) int {
	switch strings.ToLower(key) {
	case "enter", "\r", "\n":
		return 13
	case "tab":
		return 9
	case "escape", "esc":
		return 27
	case "backspace":
		return 8
	case "delete":
		return 46
	case "arrowleft", "left":
		return 37
	case "arrowup", "up":
		return 38
	case "arrowright", "right":
		return 39
	case "arrowdown", "down":
		return 40
	case "home":
		return 36
	case "end":
		return 35
	case "pageup":
		return 33
	case "pagedown":
		return 34
	case " ":
		return 32
	case "f1":
		return 112
	case "f2":
		return 113
	case "f3":
		return 114
	case "f4":
		return 115
	case "f5":
		return 116
	case "f6":
		return 117
	case "f7":
		return 118
	case "f8":
		return 119
	case "f9":
		return 120
	case "f10":
		return 121
	case "f11":
		return 122
	case "f12":
		return 123
	}
	return 0
}

// DispatchLiveKeyCombo 以 "ctrl+shift+a" 形式注入组合键。
//
// 该路径复用 154 已冻结的 `parseKeyCombo`（ENGINE-02 keyboard 动作同一套解析），
// 供「动作级快捷键」场景与测试使用；前端逐键注入走 DispatchLiveInput。
func DispatchLiveKeyCombo(page *rod.Page, keys string) error {
	if page == nil {
		return fmt.Errorf("组合键注入失败: page 为 nil")
	}
	parsed, err := parseKeyCombo(keys)
	if err != nil {
		return err
	}
	ka := page.KeyActions()
	for _, k := range parsed {
		ka.Press(k)
	}
	if err := ka.Do(); err != nil {
		return fmt.Errorf("组合键注入失败: %w", err)
	}
	return nil
}

// KeyComboToKeys 把组合键解析成 rod 键序列（供单元测试断言解析结果）。
func KeyComboToKeys(keys string) ([]input.Key, error) { return parseKeyCombo(keys) }
