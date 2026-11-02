/**
 * antd message context 桥接
 *
 * 背景：antd v5 的静态 `message`（`import { message } from "antd"`）在模块加载时
 * 实例化为全局单例，不在 React 树内，无法消费 ConfigProvider 的动态主题 context。
 * 后果：暗色模式下 message 弹框仍是白底、自定义主色不生效，并喷控制台警告
 *   [antd: message] Static function can not consume context like dynamic theme.
 *
 * 官方方案是 `App.useApp().message`。但 axios 拦截器（lib/api.ts）、静态错误处理器
 * （utils/errorHandler.ts）、Excel 工具函数等运行在模块作用域，不能调用 hook。
 *
 * 本模块提供桥接：AntdThemeBridge（位于 antd <App> 子树内）通过 useEffect 把
 * context-aware 的 message 实例写入 module-level ref；这些非组件代码改用
 * getAppMessage() 实时读取，从而共享同一 context-aware 实例。
 *
 * <App> 尚未挂载时（应用初始化极早期）返回 no-op 实例：静默短路、不崩溃，
 * 也不破坏调用方的 Promise reject 链。
 */
import type { App } from "antd";

/** message 实例类型，从 antd App.useApp 推断，避免依赖 antd 内部子路径 */
type MessageInstance = ReturnType<typeof App.useApp>["message"];

let messageRef: MessageInstance | null = null;

/** no-op 实例：<App> 未挂载时返回，所有方法静默短路 */
const noop = () => {};
const noopMessage = {
  success: noop,
  error: noop,
  info: noop,
  warning: noop,
  loading: noop,
  warn: noop,
  open: noop,
  destroy: noop,
} as unknown as MessageInstance;

/**
 * 由 AntdThemeBridge 在 <App> 挂载后调用，注入 context-aware message 实例。
 * 卸载时传 null 重置，避免持有过期实例。
 */
export function setAppMessageInstance(instance: MessageInstance | null): void {
  messageRef = instance;
}

/**
 * 获取 context-aware message 实例。
 * <App> 未挂载时返回 no-op 实例（静默短路，不抛错）。
 */
export function getAppMessage(): MessageInstance {
  return messageRef ?? noopMessage;
}
