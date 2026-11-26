/**
 * Phase 156 / NOCODE-04：开箱模板
 *
 * 产品定位（261123-v156-research §4）：UiPath StudioX / Power Automate Desktop 的
 * 共同抽象是「录制 → 可视化步骤列表（自然语言描述）→ 少量参数填写 → 一键运行」。
 * 模板就是「用户什么都不用录」的起点 —— 选一个最像自己工作的模板，改几个参数
 * 就能跑，这是零门槛的最后一块。
 *
 * 硬约束（NOCODE-04）：
 *   - 步骤名称必须是**业务语义中文名**（「打开网页」/「填入内容」/「点击按钮」），
 *     不得出现 http 方法、selector、JSON 这类术语。
 *   - 每个模板带中文说明与示例数据（用户能看懂「这个模板帮我做什么」）。
 *   - 参数是「人能填的空」：金额、账号、部门、日期，而不是 CSS 选择器。
 *
 * 模板里的 selector 是**技术细节**，只在内部使用（回放/拾取时会替换成
 * 用户环境里真实的选择器），绝不在步骤列表里展示给用户。
 */

import type { Action, ActionType } from "@/types/rpa";

/** 模板参数：用户要填的「空」。 */
export interface TemplateParam {
  /** 参数键（与步骤里的 {{key}} 占位符对应） */
  key: string;
  /** 中文标签（展示给用户） */
  label: string;
  /** 输入提示（示例数据） */
  placeholder: string;
  /** 示例值（一键填入示例时用） */
  example: string;
  /** 是否必填 */
  required: boolean;
  /** 敏感参数（密码/令牌）→ 输入框掩码 + 不落明文 */
  sensitive?: boolean;
}

/** 模板步骤（中文语义 + 底层动作）。 */
export interface TemplateStep {
  /** 业务语义中文名（NOCODE-04 要求） */
  title: string;
  /** 用户能看懂的一句话说明（可选） */
  hint?: string;
  /** 底层动作（回放用） */
  action: Omit<Action, "id">;
}

/** 开箱模板。 */
export interface TaskTemplate {
  /** 模板 ID（稳定，用于埋点与「最近使用」） */
  id: string;
  /** 中文名 */
  name: string;
  /** 一句话说明「这个模板帮你做什么」 */
  summary: string;
  /** 适用场景（帮助用户对号入座） */
  scenario: string;
  /** 分类（登录类 / 填表类 / 查询导出类） */
  category: "login" | "form" | "query";
  /** 图标名（antd icon） */
  icon: string;
  /** 参数列表 */
  params: TemplateParam[];
  /** 步骤列表 */
  steps: TemplateStep[];
  /** 预计耗时（人话：「约 2 分钟」） */
  estimatedTime: string;
}

// ==================== 步骤构造辅助 ====================

/**
 * mkStep 构造一个模板步骤。
 *
 * 刻意把「中文名」放在必填第一位：强迫模板作者先想「用户会怎么读这一步」，
 * 而不是先想技术动作。这是 NOCODE-04 零术语约束的结构性保障。
 */
function mkStep(
  title: string,
  type: ActionType,
  opts: {
    hint?: string;
    selector?: string;
    params?: Record<string, unknown>;
  } = {}
): TemplateStep {
  return {
    title,
    hint: opts.hint,
    action: {
      type,
      description: title,
      selector: opts.selector ? { type: "css", value: opts.selector } : undefined,
      params: opts.params,
      timeout: 30000,
    },
  };
}

// ==================== 模板 1：登录类 ====================

const LOGIN_TEMPLATE: TaskTemplate = {
  id: "tpl-login",
  name: "登录系统取数",
  summary: "自动登录业务系统，登录成功后打开你需要的数据页面。",
  scenario: "每天上班第一件事就是登录系统、点进某个菜单 —— 这个模板帮你自动完成。",
  category: "login",
  icon: "LoginOutlined",
  estimatedTime: "约 1 分钟",
  params: [
    {
      key: "loginUrl",
      label: "系统登录地址",
      placeholder: "例如：https://oa.example.com/login",
      example: "https://oa.example.com/login",
      required: true,
    },
    {
      key: "username",
      label: "登录账号",
      placeholder: "你的工号或用户名",
      example: "zhangsan",
      required: true,
    },
    {
      key: "password",
      label: "登录密码",
      placeholder: "建议在「凭证管理」里保存，避免每次填写",
      example: "",
      required: true,
      sensitive: true,
    },
    {
      key: "homeUrl",
      label: "登录后要去的页面",
      placeholder: "例如：https://oa.example.com/report",
      example: "https://oa.example.com/report",
      required: false,
    },
  ],
  steps: [
    mkStep("打开网页", "navigate", {
      hint: "打开系统登录页面",
      params: { url: "{{loginUrl}}" },
    }),
    mkStep("填入内容", "fill", {
      hint: "在账号框填入你的账号",
      selector: "#username",
      params: { value: "{{username}}" },
    }),
    mkStep("填入密码", "fill", {
      hint: "在密码框填入密码（不会显示明文）",
      selector: "#password",
      params: { value: "{{password}}" },
    }),
    mkStep("点击按钮", "click", {
      hint: "点击登录按钮",
      selector: "#login-submit",
    }),
    mkStep("稍等页面出现", "wait", {
      hint: "等登录完成、首页加载出来",
      selector: "#main-content",
      params: { duration: 3000 },
    }),
    mkStep("打开网页", "navigate", {
      hint: "打开你需要的数据页面",
      params: { url: "{{homeUrl}}" },
    }),
  ],
};

// ==================== 模板 2：填表类 ====================

const FORM_TEMPLATE: TaskTemplate = {
  id: "tpl-form",
  name: "批量填报单据",
  summary: "打开表单页面，把准备好的内容逐项填进去，最后提交。",
  scenario: "报销单、请假单、采购申请这类「每次都要填一遍」的单据。",
  category: "form",
  icon: "FormOutlined",
  estimatedTime: "约 2 分钟",
  params: [
    {
      key: "formUrl",
      label: "表单页面地址",
      placeholder: "例如：https://oa.example.com/expense/new",
      example: "https://oa.example.com/expense/new",
      required: true,
    },
    {
      key: "amount",
      label: "金额",
      placeholder: "例如：1280",
      example: "1280",
      required: true,
    },
    {
      key: "reason",
      label: "事由说明",
      placeholder: "例如：10 月出差交通费",
      example: "10 月出差交通费",
      required: true,
    },
    {
      key: "applicant",
      label: "申请人",
      placeholder: "你的姓名",
      example: "张三",
      required: true,
    },
  ],
  steps: [
    mkStep("打开网页", "navigate", {
      hint: "打开表单填写页面",
      params: { url: "{{formUrl}}" },
    }),
    mkStep("填入内容", "fill", {
      hint: "在「申请人」填入姓名",
      selector: "#applicant",
      params: { value: "{{applicant}}" },
    }),
    mkStep("填入内容", "fill", {
      hint: "在「金额」填入报销金额",
      selector: "#amount",
      params: { value: "{{amount}}" },
    }),
    mkStep("填入内容", "fill", {
      hint: "在「事由」填入说明",
      selector: "#reason",
      params: { value: "{{reason}}" },
    }),
    mkStep("点击按钮", "click", {
      hint: "点击提交",
      selector: "#submit-btn",
    }),
    mkStep("稍等页面出现", "wait", {
      hint: "等「提交成功」的提示出现",
      selector: ".success-tip",
      params: { duration: 3000 },
    }),
  ],
};

// ==================== 模板 3：查询导出类 ====================

const EXPORT_TEMPLATE: TaskTemplate = {
  id: "tpl-export",
  name: "查询并导出报表",
  summary: "按条件查询报表，然后把结果下载成文件保存到本地。",
  scenario: "每天/每周要把某个报表导出来存档或发给同事。",
  category: "query",
  icon: "DownloadOutlined",
  estimatedTime: "约 2 分钟",
  params: [
    {
      key: "reportUrl",
      label: "报表页面地址",
      placeholder: "例如：https://oa.example.com/report/sales",
      example: "https://oa.example.com/report/sales",
      required: true,
    },
    {
      key: "startDate",
      label: "开始日期",
      placeholder: "例如：2026-11-01",
      example: "2026-11-01",
      required: true,
    },
    {
      key: "endDate",
      label: "结束日期",
      placeholder: "例如：2026-11-30",
      example: "2026-11-30",
      required: true,
    },
  ],
  steps: [
    mkStep("打开网页", "navigate", {
      hint: "打开报表查询页面",
      params: { url: "{{reportUrl}}" },
    }),
    mkStep("填入内容", "fill", {
      hint: "填入开始日期",
      selector: "#start-date",
      params: { value: "{{startDate}}" },
    }),
    mkStep("填入内容", "fill", {
      hint: "填入结束日期",
      selector: "#end-date",
      params: { value: "{{endDate}}" },
    }),
    mkStep("点击按钮", "click", {
      hint: "点击查询",
      selector: "#search-btn",
    }),
    mkStep("稍等页面出现", "wait", {
      hint: "等查询结果表格出来",
      selector: ".result-table",
      params: { duration: 3000 },
    }),
    mkStep("点击按钮", "click", {
      hint: "点击导出，把结果下载下来",
      selector: "#export-btn",
    }),
  ],
};

/** 全部开箱模板（NOCODE-04 要求 ≥3 个：登录类 / 填表类 / 查询导出类）。 */
export const TASK_TEMPLATES: TaskTemplate[] = [LOGIN_TEMPLATE, FORM_TEMPLATE, EXPORT_TEMPLATE];

/** 按 ID 查模板。 */
export function getTemplateById(id: string): TaskTemplate | undefined {
  return TASK_TEMPLATES.find((t) => t.id === id);
}

/** 模板分类的中文名（向导第 1 步分组展示用）。 */
export const TEMPLATE_CATEGORY_LABELS: Record<TaskTemplate["category"], string> = {
  login: "登录类",
  form: "填表类",
  query: "查询导出类",
};

// ==================== 模板 → 可执行动作 ====================

/**
 * instantiateTemplate 把模板 + 用户填的参数实例化成可执行动作。
 *
 * 占位符语法 {{key}} 与 Worker 的变量替换（ENGINE-04）同源 —— 这样模板产出的
 * 动作可以直接交给引擎，不需要额外的中间格式。
 *
 * 未填的可选参数保留占位符（由引擎的变量作用域在运行时补），必填项由向导层
 * 校验（validateTemplateParams），本函数不做业务校验。
 */
export function instantiateTemplate(
  template: TaskTemplate,
  values: Record<string, string>
): Action[] {
  return template.steps.map((step, index) => {
    const params = { ...(step.action.params ?? {}) };
    // 替换 params 里的占位符
    for (const key of Object.keys(params)) {
      const v = params[key];
      if (typeof v === "string") {
        params[key] = substitutePlaceholders(v, values);
      }
    }
    // 选择器里也可能有占位符（少见的动态 id 场景）
    const selector = step.action.selector
      ? {
          ...step.action.selector,
          value: substitutePlaceholders(step.action.selector.value, values),
        }
      : undefined;

    return {
      ...step.action,
      id: `tpl_${template.id}_${index + 1}`,
      description: substitutePlaceholders(step.title, values),
      params,
      selector,
    } as Action;
  });
}

/** substitutePlaceholders 把 {{key}} 替换成用户填的值（未填则原样保留）。 */
export function substitutePlaceholders(text: string, values: Record<string, string>): string {
  return text.replace(/\{\{\s*([\w.-]+)\s*\}\}/g, (whole, key: string) => {
    const v = values[key];
    return v === undefined || v === null || v === "" ? whole : v;
  });
}

/** 模板参数校验结果。 */
export interface TemplateParamValidation {
  ok: boolean;
  /** 缺失的必填参数（中文标签，直接给用户看） */
  missing: string[];
}

/**
 * validateTemplateParams 校验必填参数。
 *
 * 错误信息用**中文标签**而不是 key（用户看到的是「登录账号」不是 "username"）。
 */
export function validateTemplateParams(
  template: TaskTemplate,
  values: Record<string, string>
): TemplateParamValidation {
  const missing: string[] = [];
  for (const p of template.params) {
    if (!p.required) continue;
    const v = values[p.key];
    if (v === undefined || v === null || String(v).trim() === "") {
      missing.push(p.label);
    }
  }
  return { ok: missing.length === 0, missing };
}
