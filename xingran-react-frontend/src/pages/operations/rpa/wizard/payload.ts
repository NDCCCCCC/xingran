/**
 * Phase 156 / NOCODE-04：向导草稿 → 后端请求体
 *
 * 独立成文件的原因：
 *   1. 纯函数便于单测（不依赖 React / 路由 / antd）
 *   2. 避免 index.tsx 出现「组件文件同时导出工具函数」触发
 *      react-refresh/only-export-components 警告
 *
 * 字段契约（与 internal/services/rpa/task_service.go Create 对齐）：
 *   name / description / script / status / priority / timeout / triggerType
 *
 * script 的映射与 tasks/modals/EditModal.tsx 的 handleOk 保持同一套形状，
 * 避免「向导保存」与「编辑器保存」产出不同结构（那会让 Worker 端出现
 * 两种解析分支）。
 */

import type { Action } from "@/types/rpa";

/** 向导第 3 步的运行方式。 */
export type WizardRunMode = "immediate" | "schedule" | "trigger";

/** 向导第 1 步的三种起点。 */
export type WizardSource = "describe" | "record" | "template";

/** 向导产出的任务草稿。 */
export interface WizardDraft {
  name: string;
  description: string;
  actions: Action[];
  runMode: WizardRunMode;
  scheduleCron?: string;
  triggerEvent?: string;
  source: WizardSource;
  templateId?: string;
}

/** 后端 ScriptAction 形状（仅用于断言，运行时不引用）。 */
export interface ScriptActionPayload {
  type: string;
  selector: string;
  value: string;
  attributes: Record<string, unknown>;
  timeout: number;
  retry: number;
}

/**
 * actionsToScript 把前端 Action 映射成后端 ScriptAction[]。
 *
 * 与 EditModal.handleOk 同源（同样的字段、同样的 attributes 展开规则）。
 */
export function actionsToScript(actions: Action[]): ScriptActionPayload[] {
  return actions.map((action) => ({
    type: action.type,
    selector: action.selector?.value || "",
    value: (action.params?.value as string) || "",
    attributes: {
      description: action.description,
      ...(action.params?.duration !== undefined && { duration: action.params.duration }),
      ...(action.params?.keys !== undefined && { keys: action.params.keys }),
      ...(action.params?.url !== undefined && { url: action.params.url }),
    },
    timeout: action.timeout ?? 30000,
    retry: action.retry ?? 0,
  }));
}

/**
 * toTaskPayload 把向导草稿翻译成 POST /rpa/tasks 的请求体。
 */
export function toTaskPayload(draft: WizardDraft): Record<string, unknown> {
  return {
    name: draft.name,
    description: draft.description,
    script: actionsToScript(draft.actions),
    // CONTRACT-03：status 是任务态数字 0=启用 / 1=停用
    status: 0,
    priority: 50,
    timeout: 300,
    // 触发器类型与 constants.tsx TRIGGER_TYPE_OPTIONS 同源
    triggerType:
      draft.runMode === "schedule" ? "schedule" : draft.runMode === "trigger" ? "event" : "manual",
    ...(draft.runMode === "schedule" && draft.scheduleCron
      ? { scheduleConfig: { cron: draft.scheduleCron } }
      : {}),
    ...(draft.runMode === "trigger" && draft.triggerEvent
      ? { triggerConfig: { event: draft.triggerEvent } }
      : {}),
  };
}

/**
 * runModeLabel 运行方式的中文名（第 3 步确认卡片与测试断言共用）。
 */
export function runModeLabel(
  draft: Pick<WizardDraft, "runMode" | "scheduleCron" | "triggerEvent">
): string {
  switch (draft.runMode) {
    case "immediate":
      return "现在运行一次";
    case "schedule":
      return `按时间自动跑（${draft.scheduleCron || "0 9 * * *"}）`;
    case "trigger":
      return `事件触发（${draft.triggerEvent || "新工单"}）`;
    default:
      return "现在运行一次";
  }
}
