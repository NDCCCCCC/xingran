/**
 * Phase 156 / NOCODE-04：动作 → 业务语义中文名
 *
 * 这是「零技术术语」约束的**唯一权威转换点**：向导第 2 步的步骤列表、录制结果
 * 的步骤预览、模板实例化后的展示，全部经过这里。任何地方直接把 action.type
 * 或 selector 渲染给用户，都是本文件的漏用。
 *
 * 与 constants.tsx 的 ACTION_TYPE_TEXT_MAP 的关系（归属协议第 1 条）：
 *   - ACTION_TYPE_TEXT_MAP 是**编辑器内部**的动作类型短名（「点击」「输入」），
 *     面向配置动作的人，仍偏技术。
 *   - 本文件是**面向一般员工**的步骤描述生成器：它结合动作类型 + 参数 + 上下文
 *     产出一句完整的话（「在「金额」填入 1280」），并且**只消费** 154 的动作
 *     枚举，不新增动作（协议第 1 条：156 不得作为第三写方）。
 */

import type { Action, ActionType } from "@/types/rpa";
import { ACTION_TYPE_TEXT_MAP } from "../constants";

/**
 * 面向员工的动词表。
 *
 * 与 ACTION_TYPE_TEXT_MAP 的差异是刻意为之：这里追求「读起来像人说的话」，
 * 例如 click → 「点击」而不是「点击元素」，fill → 「填入」而不是「填写表单」。
 *
 * REC-03：类型从 `Record<string, string>` 收紧为 `Record<ActionType, string>`，
 * 键集被**编译期**强制等于共享枚举（types/rpa.ts 的 RPA_ACTION_TYPES，26 个）。
 * 少一个键 → tsc 报错；多一个键（历史上有 Go 侧不存在的 press）→ 同样报错。
 * 这从根上消灭了「前端动词表」与「worker 动作全集」的双源漂移。
 */
export const HUMAN_VERBS: Record<ActionType, string> = {
  navigate: "打开网页",
  click: "点击",
  fill: "填入",
  select: "选择",
  wait: "稍等片刻",
  waitFor: "等待页面出现",
  screenshot: "截图留存",
  extract: "读取页面内容",
  scroll: "滚动页面",
  upload: "上传文件",
  download: "下载文件",
  hover: "鼠标移到",
  doubleClick: "双击",
  rightClick: "右键点击",
  drag: "拖动",
  keyboard: "按键",
  switchTab: "切换页面",
  iframe: "进入内嵌页面",
  assert: "检查",
  close: "关闭页面",
  evaluate: "执行页面操作",
  script: "执行页面操作",
  pause: "暂停等你确认",
  loop: "重复执行",
  condition: "判断条件",
  autologin: "自动登录",
};

/** 不需要选择器/参数说明的动作（纯等待类）。 */
const SIMPLE_ACTIONS = new Set(["wait", "waitFor", "screenshot", "close", "pause"]);

/**
 * describeAction 把一条动作翻译成一句人话。
 *
 * 输出示例：
 *   navigate  → 「打开网页」
 *   fill      → 「在「金额」填入 1280」
 *   click     → 「点击「提交」」
 *   waitFor   → 「等待页面出现」
 *   select    → 「选择「住宿费」」
 *
 * 关键约束：绝不输出 selector / url / JSON / http 方法。
 */
export function describeAction(action: Action): string {
  const verb = HUMAN_VERBS[action.type] ?? ACTION_TYPE_TEXT_MAP[action.type] ?? "执行操作";

  // 优先用动作自带的 description（录制器/模板生成的中文描述最准确）
  if (action.description && isHumanReadable(action.description)) {
    return action.description;
  }

  if (SIMPLE_ACTIONS.has(action.type)) {
    return verb;
  }

  const value = action.params?.value;
  const label = extractHumanLabel(action);
  // REC-03：对共享枚举做**穷尽 switch** —— 直接 switch (action.type)，
  // default 分支用 never 断言兜底。将来共享枚举（types/rpa.ts 的
  // RPA_ACTION_TYPES）新增动作而这里漏了 case 时，action.type 不再是 never，
  // 编译期直接报错 —— 不再依赖「字符串分派 + 运行时回落」这种会静默漏网的做法。
  switch (action.type) {
    case "navigate":
      // 只显示站点名，不显示完整 URL（URL 对员工是噪声）
      return "打开网页";
    case "fill":
      if (label && value !== undefined && value !== null && value !== "") {
        return `在「${label}」填入 ${truncate(String(value), 24)}`;
      }
      if (label) return `在「${label}」填入内容`;
      return "填入内容";
    case "select":
      if (label) return `选择「${label}」`;
      return "选择选项";
    case "click":
    case "doubleClick":
    case "rightClick":
      if (label) return `${verb}「${label}」`;
      return `${verb}按钮`;
    case "upload":
      return label ? `上传文件到「${label}」` : "上传文件";
    case "download":
      return "下载文件";
    case "extract":
      return label ? `读取「${label}」的内容` : "读取页面内容";
    case "assert":
      return label ? `检查「${label}」` : "检查结果";
    case "keyboard": {
      const keys = action.params?.keys;
      if (typeof keys === "string" && keys) return `按下 ${humanKeys(keys)}`;
      // 无具体键位时也要说清「按下某个键」，而不是含糊的「按键」
      return "按下按键";
    }
    case "iframe":
      return action.params?.action === "exit" ? "退出内嵌页面" : "进入内嵌页面";
    case "switchTab":
      return "切换到另一个页面";
    case "scroll":
      return "滚动页面";
    case "hover":
      return label ? `鼠标移到「${label}」` : "鼠标悬停";
    case "drag":
      // 拖拽动作：有 label 时描述为「拖动到「label」」，无 label 时用动词表兜底
      return label ? `拖动到「${label}」` : verb;
    // 以下动作没有独立的句式，统一回落到动词表（等待类已在 SIMPLE_ACTIONS
    // 提前返回；evaluate/script/loop/condition/autologin 本身就是动词本身）。
    case "wait":
    case "waitFor":
    case "screenshot":
    case "close":
    case "pause":
    case "evaluate":
    case "script":
    case "loop":
    case "condition":
    case "autologin":
      return verb;
    default: {
      // 穷尽性断言：共享枚举新增动作而上面漏写 case 时，此处类型不是 never，
      // 赋值即编译报错（漏网动作无法再静默落到「执行操作」）。
      const _exhaustive: never = action.type;
      return _exhaustive;
    }
  }
}

/**
 * extractHumanLabel 从动作里提取「人能认出来的标签」。
 *
 * 来源优先级：
 *   1. params.label（录制器/拾取器写入的可读标签）
 *   2. params.text（可见文本）
 *   3. 从 selector 里提取的语义线索（testid / name / aria-label 的值）
 *
 * 最后一条很重要：即使只有 `[data-testid="submit-btn"]`，也要说成
 * 「submit-btn」这种半可读的标签，而不是把整个 CSS 选择器糊到用户脸上。
 */
export function extractHumanLabel(action: Action): string {
  const p = action.params ?? {};
  for (const key of ["label", "text", "title", "name"]) {
    const v = p[key];
    if (typeof v === "string" && v.trim()) return truncate(v.trim(), 20);
  }

  const sel = action.selector?.value ?? "";
  if (!sel) return "";

  // text= 前缀 → 直接就是可见文本
  if (sel.startsWith("text=")) {
    return truncate(sel.slice(5).trim(), 20);
  }

  // data-testid / aria-label / name 的值
  const attrMatch = sel.match(/\[(?:data-testid|aria-label|name)\s*=\s*["']?([^"'\]]+)/i);
  if (attrMatch) {
    return humanizeToken(attrMatch[1]);
  }

  // #id
  const idMatch = sel.match(/#([A-Za-z0-9_-]+)/);
  if (idMatch) {
    return humanizeToken(idMatch[1]);
  }

  return "";
}

/**
 * humanizeToken 把技术化的标识转成勉强可读的短标签。
 *
 * `submit-btn` → `submit btn` 这种仍不好看，所以只保留「去掉前缀/后缀噪声」的
 * 最小处理：真实产品里这个位置应该由拾取器写入 params.label 覆盖。
 */
function humanizeToken(token: string): string {
  return truncate(
    token
      .replace(/^btn[-_]?/i, "")
      .replace(/[-_](btn|button|input|field|wrapper|container)$/i, "")
      .replace(/[-_]+/g, " ")
      .trim() || token,
    20
  );
}

/** humanKeys 把 "ctrl+shift+s" 说成「Ctrl+Shift+S」。 */
function humanKeys(keys: string): string {
  return keys
    .split("+")
    .map((k) => {
      const t = k.trim();
      if (t.length === 1) return t.toUpperCase();
      return t.charAt(0).toUpperCase() + t.slice(1);
    })
    .join("+");
}

/** isHumanReadable 判定一个 description 是否是「人话」（不含技术符号）。 */
export function isHumanReadable(text: string): boolean {
  if (!text.trim()) return false;
  // 含 CSS/XPath 特征或 JSON 片段 → 不是人话
  if (/[#.[\]{}<>]/.test(text)) return false;
  if (/xpath|css|selector|json|http:|https:/i.test(text)) return false;
  return true;
}

/** truncate 截断长文本（加省略号）。 */
function truncate(text: string, max: number): string {
  const runes = Array.from(text);
  if (runes.length <= max) return text;
  return runes.slice(0, max).join("") + "…";
}

/**
 * describeActions 批量翻译（步骤列表直接用）。
 */
export function describeActions(actions: Action[]): string[] {
  return actions.map(describeAction);
}

/**
 * TECH_TERM_BLOCKLIST 面向员工界面禁止出现的词（测试与自检用）。
 *
 * 这是 NOCODE-04「零技术术语泄漏」的可机械断言清单。
 */
export const TECH_TERM_BLOCKLIST = [
  "selector",
  "xpath",
  "css",
  "json",
  "http://",
  "https://",
  "POST",
  "GET",
  "DOM",
  "nth-child",
  "data-testid",
] as const;

/** containsTechTerm 检查文本是否泄漏技术术语（返回命中的词）。 */
export function containsTechTerm(text: string): string | null {
  const lower = text.toLowerCase();
  for (const term of TECH_TERM_BLOCKLIST) {
    if (lower.includes(term.toLowerCase())) return term;
  }
  return null;
}
