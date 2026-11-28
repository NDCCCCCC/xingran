package browser

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Page 浏览器页面接口（Phase 154 / ENGINE-01 契约冻结点）。
//
// 语义要点：
//   - Close() 只关闭当前 page（标签页），**绝不**关闭 Chrome 进程；
//     浏览器生命周期由 Manager 独占，见 manager.go。
//   - RawPage() 暴露底层 *rod.Page 出口，Phase 155 LIVEVIEW 的
//     Page.startScreencast / ACK 节流直接依赖它。
//   - Browser() 暴露共享的 *rod.Browser；同一次执行拿到的 Browser 指针相同，
//     证明「1 Chrome + N page」而非「N Chrome」。
type Page interface {
	Goto(url string) error
	Close() error
	Screenshot() ([]byte, error)
	Click(selector string) error
	Fill(selector, value string) error
	Select(selector, value string) error
	WaitFor(selector string, timeout time.Duration) error
	Evaluate(script string) (interface{}, error)
	GetText(selector string) (string, error)
	GetValue(selector string) (string, error)
	GetHTML(selector string) (string, error)
	GetAttribute(selector, attrName string) (string, error)
	ScrollDown(pixels int) error
	ScrollUp(pixels int) error
	ScrollToElement(selector string) error

	// ===== Phase 154 / ENGINE-01：承载契约 =====
	// RawPage 返回当前活动页签的底层 rod 页面句柄（screencast / 高级 CDP
	// 操作的唯一出口；switchTab 后会跟随到新页签）。
	RawPage() *rod.Page
	// ContextPage 返回当前操作上下文：iframe 内为 frame 句柄，否则等同
	// RawPage。元素定位/交互必须走它才能作用于 frame 内部。
	ContextPage() *rod.Page
	// Browser 返回该 page 所属的共享 rod 浏览器实例（1 Chrome 共享给 N page）。
	Browser() *rod.Browser
	// Closed 报告 page 是否已被关闭（Manager 回收时据此丢弃死 page，避免复用）。
	Closed() bool
	// Reset 清理一次执行残留的瞬态状态（退出 iframe、清 frame 句柄），
	// 使 Pool 复用的 page 从主文档重新开始。不得关闭 page 或 browser。
	Reset()

	// ===== Phase 154 / ENGINE-02：动作原语 =====
	Upload(selector string, files []string) error
	Hover(selector string) error
	// ClickWith 指定鼠标键（left/right）与点击次数（1=单击，2=双击）。
	ClickWith(selector string, button proto.InputMouseButton, count int) error
	Drag(fromSelector, toSelector string) error
	// PressKeys 接受 "ctrl+shift+a" 形式的组合键。
	PressKeys(keys string) error
	// SwitchTab 按页签序号切换（0 为主页签）。
	SwitchTab(index int) error
	// EnterFrame 进入 iframe 上下文；ExitFrame 返回主文档。
	EnterFrame(selector string) error
	ExitFrame() error
	// ScrollInElement 在元素内部滚动（不滚动整页）。
	ScrollInElement(selector string, deltaX, deltaY int) error
	// Download 触发一次下载并把产物归档到 dir，返回最终文件路径。
	Download(ctx context.Context, dir string, trigger func() error) (string, error)
	// CloseCurrentTab 关闭当前页签：还有其它页签则切过去，否则新建空白页
	// 继续承载本次执行（"最后一页保护"，避免 close 动作把执行打断）。
	CloseCurrentTab() error
}
