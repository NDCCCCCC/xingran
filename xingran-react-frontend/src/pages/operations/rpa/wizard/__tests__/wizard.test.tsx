/**
 * Phase 156 / NOCODE-04：三步向导 + 模板 + 中文步骤名 测试
 *
 * 验收口径（任务书）：
 *   ① 说要做什么（自然语言/录制/模板三选一）
 *   ② 确认步骤（中文步骤列表，可拖动排序、可改参数）
 *   ③ 选运行方式（立即/定时/触发）+ 保存
 *   步骤列表用业务语义中文名，**不用** http 方法 / selector / JSON 这类术语
 *   开箱模板 ≥3 个（登录类/填表类/查询导出类）
 *
 * 本文件分两组：
 *   A. 纯函数测试（模板实例化 / 占位符 / 校验 / 中文描述 / 零术语断言）
 *   B. 组件测试（三步流转 / 排序 / 改参数 / 保存载荷）
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";
import type { Action } from "@/types/rpa";

import {
  TASK_TEMPLATES,
  TEMPLATE_CATEGORY_LABELS,
  instantiateTemplate,
  validateTemplateParams,
  substitutePlaceholders,
  getTemplateById,
} from "../templates";
import {
  describeAction,
  describeActions,
  extractHumanLabel,
  containsTechTerm,
  TECH_TERM_BLOCKLIST,
  isHumanReadable,
} from "../stepLabels";
import { toTaskPayload, actionsToScript, runModeLabel } from "../payload";
import { TaskWizard } from "../TaskWizard";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

// ==================== A. 纯函数：模板 ====================

describe("开箱模板（NOCODE-04：≥3 个，覆盖登录/填表/查询导出）", () => {
  it("至少 3 个模板，且三类场景齐全", () => {
    expect(TASK_TEMPLATES.length).toBeGreaterThanOrEqual(3);
    const categories = new Set(TASK_TEMPLATES.map((t) => t.category));
    expect(categories.has("login")).toBe(true);
    expect(categories.has("form")).toBe(true);
    expect(categories.has("query")).toBe(true);
    // 分类中文名必须齐全（不允许 undefined 出现在 UI）
    for (const c of categories) {
      expect(TEMPLATE_CATEGORY_LABELS[c]).toBeTruthy();
    }
  });

  it("每个模板都有中文说明、示例数据与预计耗时", () => {
    for (const tpl of TASK_TEMPLATES) {
      expect(tpl.name.trim()).not.toBe("");
      expect(tpl.summary.trim()).not.toBe("");
      expect(tpl.scenario.trim()).not.toBe("");
      expect(tpl.estimatedTime).toMatch(/分钟|秒/);
      expect(tpl.steps.length).toBeGreaterThan(0);
      // 每个参数都要有中文标签与示例（示例可空但必须有 placeholder 指引）
      for (const p of tpl.params) {
        expect(p.label.trim()).not.toBe("");
        expect(p.placeholder.trim()).not.toBe("");
      }
    }
  });

  it("模板步骤名全是业务语义中文，零技术术语", () => {
    for (const tpl of TASK_TEMPLATES) {
      for (const step of tpl.steps) {
        expect(step.title.trim()).not.toBe("");
        const term = containsTechTerm(step.title);
        expect(term, `模板「${tpl.name}」的步骤「${step.title}」泄漏技术术语 ${term}`).toBeNull();
        // 不得含 CSS/XPath 特征字符
        expect(step.title).not.toMatch(/[#.[\]]/);
      }
    }
  });

  it("模板 ID 唯一（避免「最近使用」与埋点串号）", () => {
    const ids = TASK_TEMPLATES.map((t) => t.id);
    expect(new Set(ids).size).toBe(ids.length);
    for (const id of ids) {
      expect(getTemplateById(id)).toBeTruthy();
    }
  });

  it("敏感参数被标记（密码不落明文）", () => {
    const login = getTemplateById("tpl-login")!;
    const pwd = login.params.find((p) => p.key === "password");
    expect(pwd).toBeTruthy();
    expect(pwd?.sensitive).toBe(true);
  });
});

describe("模板实例化与占位符替换", () => {
  it("{{key}} 被替换成用户填的值", () => {
    expect(substitutePlaceholders("打开 {{url}}", { url: "https://x" })).toBe("打开 https://x");
    expect(substitutePlaceholders("{{ a }} 和 {{b}}", { a: "1", b: "2" })).toBe("1 和 2");
  });

  it("未填的占位符原样保留（交给运行时变量作用域）", () => {
    expect(substitutePlaceholders("打开 {{url}}", {})).toBe("打开 {{url}}");
    expect(substitutePlaceholders("打开 {{url}}", { url: "" })).toBe("打开 {{url}}");
  });

  it("instantiateTemplate 产出可执行动作且 id 连续", () => {
    const tpl = getTemplateById("tpl-form")!;
    const actions = instantiateTemplate(tpl, {
      formUrl: "https://oa/expense/new",
      amount: "1280",
      reason: "10 月差旅",
      applicant: "张三",
    });
    expect(actions.length).toBe(tpl.steps.length);
    actions.forEach((a, i) => {
      expect(a.id).toBe(`tpl_${tpl.id}_${i + 1}`);
      expect(a.type).toBeTruthy();
      // 步骤描述必须是中文人话
      expect(containsTechTerm(a.description ?? "")).toBeNull();
    });
    // 参数真的被替换进动作
    const fillAmount = actions.find((a) => a.params?.value === "1280");
    expect(fillAmount).toBeTruthy();
    const nav = actions.find((a) => a.type === "navigate");
    expect(nav?.params?.url).toBe("https://oa/expense/new");
  });

  it("validateTemplateParams 用中文标签报告缺失项", () => {
    const tpl = getTemplateById("tpl-login")!;
    const r = validateTemplateParams(tpl, {});
    expect(r.ok).toBe(false);
    // 报的是「登录账号」而不是 "username"
    expect(r.missing).toContain("登录账号");
    expect(r.missing).not.toContain("username");

    const ok = validateTemplateParams(tpl, {
      loginUrl: "https://x",
      username: "u",
      password: "p",
    });
    expect(ok.ok).toBe(true);
  });
});

// ==================== A. 纯函数：中文步骤名 ====================

describe("describeAction 业务语义中文名", () => {
  const mk = (partial: Partial<Action> & Pick<Action, "type">): Action => ({
    id: "a1",
    timeout: 30000,
    ...partial,
  });

  it("常见动作产出人话", () => {
    expect(describeAction(mk({ type: "navigate" }))).toBe("打开网页");
    expect(
      describeAction(
        mk({
          type: "fill",
          selector: { type: "css", value: "#amount" },
          params: { value: "1280", label: "金额" },
        })
      )
    ).toBe("在「金额」填入 1280");
    expect(
      describeAction(mk({ type: "click", params: { label: "提交" } }))
    ).toBe("点击「提交」");
  });

  it("优先使用动作自带的中文 description", () => {
    expect(
      describeAction(
        mk({ type: "click", description: "点击按钮「登录」", params: { label: "x" } })
      )
    ).toBe("点击按钮「登录」");
  });

  it("自带 description 含技术词时**不采信**，重新生成", () => {
    const a = mk({
      type: "click",
      description: "#submit-btn",
      selector: { type: "css", value: "#submit-btn" },
    });
    const text = describeAction(a);
    expect(text).not.toBe("#submit-btn");
    expect(containsTechTerm(text)).toBeNull();
  });

  it("从 selector 提取可读标签（不让 CSS 糊到用户脸上）", () => {
    expect(
      describeAction(
        mk({ type: "click", selector: { type: "css", value: "[data-testid=\"submit-btn\"]" } })
      )
    ).not.toContain("data-testid");
    expect(
      describeAction(mk({ type: "click", selector: { type: "css", value: "text=提交" } }))
    ).toBe("点击「提交」");
  });

  it("154 扩充的动作（doubleClick/keyboard/iframe）也有中文名", () => {
    // 前端 ActionType 是 13 值子集，154 的动作要用字符串断言（见 stepLabels 注释）
    const cases: Array<[string, string]> = [
      ["doubleClick", "双击"],
      ["rightClick", "右键点击"],
      ["keyboard", "按下"],
      ["iframe", "内嵌页面"],
      ["switchTab", "切换"],
    ];
    for (const [type, want] of cases) {
      const text = describeAction(mk({ type: type as Action["type"] }));
      expect(text).toContain(want);
    }
  });

  it("批量描述与单个描述一致", () => {
    const actions = [mk({ type: "navigate" }), mk({ type: "scroll" })];
    expect(describeActions(actions)).toEqual(actions.map(describeAction));
  });

  it("所有模板步骤实例化后都不泄漏技术术语", () => {
    for (const tpl of TASK_TEMPLATES) {
      const values: Record<string, string> = {};
      for (const p of tpl.params) values[p.key] = p.example || "示例值";
      const actions = instantiateTemplate(tpl, values);
      for (const a of actions) {
        const text = describeAction(a);
        const term = containsTechTerm(text);
        expect(term, `模板「${tpl.name}」的步骤「${text}」泄漏 ${term}`).toBeNull();
      }
    }
  });

  it("extractHumanLabel 不返回 CSS 结构", () => {
    const label = extractHumanLabel({
      id: "x",
      type: "click",
      selector: { type: "css", value: "div > div:nth-child(2) > span" },
    } as Action);
    expect(label).toBe("");
  });

  it("TECH_TERM_BLOCKLIST 覆盖验收要求的三类术语", () => {
    expect(TECH_TERM_BLOCKLIST).toContain("selector");
    expect(TECH_TERM_BLOCKLIST).toContain("json");
    expect(TECH_TERM_BLOCKLIST).toContain("http://");
  });

  it("isHumanReadable 判定", () => {
    expect(isHumanReadable("打开网页")).toBe(true);
    expect(isHumanReadable("#submit")).toBe(false);
    expect(isHumanReadable("点击 .btn")).toBe(false);
    expect(isHumanReadable("")).toBe(false);
  });
});

// ==================== A. 纯函数：保存载荷 ====================

describe("toTaskPayload 载荷契约", () => {
  const actions: Action[] = [
    {
      id: "a1",
      type: "navigate",
      description: "打开网页",
      params: { url: "https://x" },
      timeout: 30000,
    },
    {
      id: "a2",
      type: "fill",
      description: "填入金额",
      selector: { type: "css", value: "#amount" },
      params: { value: "1280" },
    },
  ];

  it("script 映射与 EditModal 同形状", () => {
    const script = actionsToScript(actions);
    expect(script[0]).toMatchObject({
      type: "navigate",
      selector: "",
      value: "",
      timeout: 30000,
      retry: 0,
    });
    expect(script[0].attributes.url).toBe("https://x");
    expect(script[1].selector).toBe("#amount");
    expect(script[1].value).toBe("1280");
    expect(script[1].attributes.description).toBe("填入金额");
  });

  it("status 是任务态数字 0（CONTRACT-03）", () => {
    const p = toTaskPayload({
      name: "t",
      description: "",
      actions,
      runMode: "immediate",
      source: "template",
    });
    expect(p.status).toBe(0);
    expect(p.triggerType).toBe("manual");
  });

  it("三种运行方式映射到正确的 triggerType", () => {
    const base = { name: "t", description: "", actions, source: "template" as const };
    expect(toTaskPayload({ ...base, runMode: "immediate" }).triggerType).toBe("manual");
    const sched = toTaskPayload({ ...base, runMode: "schedule", scheduleCron: "0 9 * * *" });
    expect(sched.triggerType).toBe("schedule");
    expect(sched.scheduleConfig).toEqual({ cron: "0 9 * * *" });
    const trig = toTaskPayload({ ...base, runMode: "trigger", triggerEvent: "new-ticket" });
    expect(trig.triggerType).toBe("event");
    expect(trig.triggerConfig).toEqual({ event: "new-ticket" });
  });

  it("runModeLabel 中文表述", () => {
    expect(runModeLabel({ runMode: "immediate" })).toBe("现在运行一次");
    expect(runModeLabel({ runMode: "schedule", scheduleCron: "0 9 * * *" })).toContain("0 9 * * *");
    expect(runModeLabel({ runMode: "trigger", triggerEvent: "新工单" })).toContain("新工单");
  });
});

// ==================== B. 组件：三步流转 ====================

describe("TaskWizard 三步向导", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("默认停在第 1 步，展示三种起点", async () => {
    render(<TaskWizard onSave={vi.fn()} onCancel={vi.fn()} />, { wrapper });
    expect(await screen.findByTestId("wizard-step-1")).toBeTruthy();
    expect(screen.getByText("用中文说")).toBeTruthy();
    expect(screen.getByText("录一段操作")).toBeTruthy();
    expect(screen.getByText("用现成模板")).toBeTruthy();
  });

  it("选模板 → 填参数 → 第 2 步看到中文步骤", async () => {
    render(<TaskWizard onSave={vi.fn()} onCancel={vi.fn()} />, { wrapper });

    // 选「用现成模板」
    fireEvent.click(screen.getByTestId("wizard-source-template"));
    // 选登录模板
    fireEvent.click(await screen.findByTestId("wizard-template-tpl-login"));
    // 参数区出现
    expect(await screen.findByTestId("wizard-template-params")).toBeTruthy();

    // 密码是敏感字段，示例值为空（绝不预填假密码）→ 必须自己填才能继续。
    // 这正是「必填校验」在真实路径上生效的证据。
    fireEvent.change(screen.getByTestId("wizard-param-password"), {
      target: { value: "Passw0rd!" },
    });
    // 诊断断言：确认参数区确实渲染了全部必填项（缺失项会被校验拦住）
    for (const key of ["loginUrl", "username", "password"]) {
      expect(screen.queryByTestId(`wizard-param-${key}`)).toBeTruthy();
    }
    // 诊断：示例值应已预填到 loginUrl（证明 handlePickTemplate 真的跑过）
    expect((screen.getByTestId("wizard-param-loginUrl") as HTMLInputElement).value).toBe(
      "https://oa.example.com/login"
    );

    // 直接验证「校验 + 实例化」这条纯逻辑在同样入参下能产出动作，
    // 把「组件状态没走到」与「业务逻辑算错」两类失败区分开。
    const loginTpl = getTemplateById("tpl-login")!;
    const probe = validateTemplateParams(loginTpl, {
      loginUrl: "https://oa.example.com/login",
      username: "zhangsan",
      password: "Passw0rd!",
    });
    expect(probe.ok).toBe(true);
    expect(instantiateTemplate(loginTpl, { password: "x" }).length).toBe(loginTpl.steps.length);

    fireEvent.click(screen.getByTestId("wizard-next"));

    const step2 = await screen.findByTestId("wizard-step-2");
    expect(step2).toBeTruthy();

    // 步骤行必须是中文人话
    const label0 = screen.getByTestId("wizard-step-row-label-0").textContent ?? "";
    expect(label0).toBe("打开网页");
    expect(containsTechTerm(label0)).toBeNull();

    // 所有步骤都不泄漏技术术语
    const login = getTemplateById("tpl-login")!;
    for (let i = 0; i < login.steps.length; i++) {
      const el = screen.queryByTestId(`wizard-step-row-label-${i}`);
      if (!el) continue;
      expect(containsTechTerm(el.textContent ?? "")).toBeNull();
    }
  });

  it("第 2 步：上移/下移按钮改变顺序（拖拽的降级通道）", async () => {
    const actions: Action[] = [
      { id: "s1", type: "navigate", description: "打开网页", params: { url: "https://x" } },
      { id: "s2", type: "fill", description: "填入金额", params: { value: "1" } },
    ];
    render(<TaskWizard onSave={vi.fn()} onCancel={vi.fn()} initialActions={actions} />, {
      wrapper,
    });

    // initialActions 非空 → 来源判定为录制，可直接下一步
    fireEvent.click(await screen.findByTestId("wizard-next"));
    await screen.findByTestId("wizard-step-2");

    expect(screen.getByTestId("wizard-step-row-label-0").textContent).toBe("打开网页");
    expect(screen.getByTestId("wizard-step-row-label-1").textContent).toBe("填入金额");

    // 第 2 步下移（把「打开网页」移下去）
    fireEvent.click(screen.getByLabelText("第 1 步下移"));

    await waitFor(() => {
      expect(screen.getByTestId("wizard-step-row-label-0").textContent).toBe("填入金额");
    });
    expect(screen.getByTestId("wizard-step-row-label-1").textContent).toBe("打开网页");
  });

  it("第 2 步：可改步骤里的内容（参数）", async () => {
    const actions: Action[] = [
      { id: "s1", type: "fill", description: "填入金额", params: { value: "1" } },
    ];
    render(<TaskWizard onSave={vi.fn()} onCancel={vi.fn()} initialActions={actions} />, {
      wrapper,
    });
    fireEvent.click(await screen.findByTestId("wizard-next"));
    await screen.findByTestId("wizard-step-2");

    const input = screen.getByLabelText("第 1 步的内容") as HTMLInputElement;
    expect(input.value).toBe("1");
    fireEvent.change(input, { target: { value: "999" } });
    await waitFor(() => {
      expect((screen.getByLabelText("第 1 步的内容") as HTMLInputElement).value).toBe("999");
    });
  });

  it("第 2 步：可删除步骤", async () => {
    const actions: Action[] = [
      { id: "s1", type: "navigate", description: "打开网页" },
      { id: "s2", type: "scroll", description: "滚动页面" },
    ];
    render(<TaskWizard onSave={vi.fn()} onCancel={vi.fn()} initialActions={actions} />, {
      wrapper,
    });
    fireEvent.click(await screen.findByTestId("wizard-next"));
    await screen.findByTestId("wizard-step-2");
    fireEvent.click(screen.getByLabelText("删除第 2 步"));
    await waitFor(() => {
      expect(screen.queryByTestId("wizard-step-row-1")).toBeNull();
    });
  });

  it("三步走完并保存：载荷含中文步骤映射后的 script", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);
    render(<TaskWizard onSave={onSave} onCancel={vi.fn()} />, { wrapper });

    // 第 1 步：选填表模板
    fireEvent.click(screen.getByTestId("wizard-source-template"));
    fireEvent.click(await screen.findByTestId("wizard-template-tpl-form"));
    fireEvent.click(screen.getByTestId("wizard-next"));

    // 第 2 步：改任务名
    await screen.findByTestId("wizard-step-2");
    fireEvent.change(screen.getByTestId("wizard-task-name"), {
      target: { value: "每月报销填报" },
    });
    fireEvent.click(screen.getByTestId("wizard-next"));

    // 第 3 步：选定时运行
    await screen.findByTestId("wizard-step-3");
    fireEvent.click(screen.getByText("按固定时间自动跑"));
    await waitFor(() => {
      expect(screen.getByTestId("wizard-schedule-cron")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("wizard-save"));

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledTimes(1);
    });
    const draft = onSave.mock.calls[0][0] as {
      name: string;
      actions: Action[];
      runMode: string;
      source: string;
      templateId: string;
    };
    expect(draft.name).toBe("每月报销填报");
    expect(draft.runMode).toBe("schedule");
    expect(draft.source).toBe("template");
    expect(draft.templateId).toBe("tpl-form");
    expect(draft.actions.length).toBeGreaterThan(0);
    // 所有动作都有中文描述
    for (const a of draft.actions) {
      expect(a.description?.trim()).toBeTruthy();
      expect(containsTechTerm(a.description ?? "")).toBeNull();
    }
  });

  it("必填参数未填时拦住并提示中文标签", async () => {
    render(<TaskWizard onSave={vi.fn()} onCancel={vi.fn()} />, { wrapper });
    fireEvent.click(screen.getByTestId("wizard-source-template"));
    fireEvent.click(await screen.findByTestId("wizard-template-tpl-login"));

    // 清空必填的账号（示例值被清掉）
    const userInput = screen.getByTestId("wizard-param-username") as HTMLInputElement;
    fireEvent.change(userInput, { target: { value: "" } });
    fireEvent.click(screen.getByTestId("wizard-next"));

    // 仍停在第 1 步（没有进入第 2 步）
    await waitFor(() => {
      expect(screen.getByTestId("wizard-step-1")).toBeTruthy();
    });
    expect(screen.queryByTestId("wizard-step-2")).toBeNull();
  });

  it("步骤列表为空时不能进入下一步", async () => {
    render(<TaskWizard onSave={vi.fn()} onCancel={vi.fn()} />, { wrapper });
    fireEvent.click(screen.getByTestId("wizard-source-record"));
    fireEvent.click(screen.getByTestId("wizard-next"));
    await waitFor(() => {
      expect(screen.getByTestId("wizard-step-1")).toBeTruthy();
    });
  });

  it("录制来源但无动作时调用 onOpenRecorder（不留死路）", async () => {
    const onOpenRecorder = vi.fn();
    render(
      <TaskWizard onSave={vi.fn()} onCancel={vi.fn()} onOpenRecorder={onOpenRecorder} />,
      { wrapper }
    );
    fireEvent.click(screen.getByTestId("wizard-source-record"));
    fireEvent.click(screen.getByTestId("wizard-next"));
    await waitFor(() => {
      expect(onOpenRecorder).toHaveBeenCalledTimes(1);
    });
  });

  it("描述来源但无动作时调用 onOpenAIGenerator", async () => {
    const onOpenAIGenerator = vi.fn();
    render(
      <TaskWizard onSave={vi.fn()} onCancel={vi.fn()} onOpenAIGenerator={onOpenAIGenerator} />,
      { wrapper }
    );
    fireEvent.click(screen.getByTestId("wizard-source-describe"));
    fireEvent.click(screen.getByTestId("wizard-next"));
    await waitFor(() => {
      expect(onOpenAIGenerator).toHaveBeenCalledTimes(1);
    });
  });

  it("第 3 步确认卡片显示中文运行方式", async () => {
    const actions: Action[] = [{ id: "s1", type: "navigate", description: "打开网页" }];
    render(<TaskWizard onSave={vi.fn()} onCancel={vi.fn()} initialActions={actions} />, {
      wrapper,
    });
    fireEvent.click(await screen.findByTestId("wizard-next"));
    await screen.findByTestId("wizard-step-2");
    fireEvent.click(screen.getByTestId("wizard-next"));

    const summary = await screen.findByTestId("wizard-summary");
    expect(summary.textContent).toContain("个步骤");
    expect(summary.textContent).toContain("现在运行一次");
  });

  it("每一步的可见文案都不泄漏技术术语", async () => {
    const actions: Action[] = [
      {
        id: "s1",
        type: "click",
        description: "点击按钮「登录」",
        selector: { type: "css", value: "[data-testid=\"login\"]" },
      },
    ];
    const { container } = render(
      <TaskWizard onSave={vi.fn()} onCancel={vi.fn()} initialActions={actions} />,
      { wrapper }
    );
    fireEvent.click(await screen.findByTestId("wizard-next"));
    await screen.findByTestId("wizard-step-2");

    // 抓取整个向导的可见文本，检查是否有技术术语泄漏
    const text = container.textContent ?? "";
    for (const term of TECH_TERM_BLOCKLIST) {
      // 注意：'http://'/'https://' 只可能出现在用户自己填的参数里，这里步骤区不该有
      expect(text.toLowerCase()).not.toContain(term.toLowerCase());
    }
  });
});
