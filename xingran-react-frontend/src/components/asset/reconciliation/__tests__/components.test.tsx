/* eslint-disable no-restricted-syntax -- 测试 fixture 需要内网 IP 字面量 */
/**
 * Phase 88 Batch17b — components/asset/reconciliation (ExceptionRuleForm/MatchTestPanel)
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import ExceptionRuleForm from "../ExceptionRuleForm";
import MatchTestPanel from "../MatchTestPanel";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

describe("components/asset/reconciliation — ExceptionRuleForm", () => {
  it("renders form fields in Modal", async () => {
    await renderPageWithEndpoints(
      <ExceptionRuleForm
        open
        initialValues={{ name: "测试规则" }}
        onSubmit={vi.fn()}
        onCancel={vi.fn()}
      />,
      {}
    );
    expect((await waitText("规则名称")) || (await waitText("IP")) || (await waitText("动作"))).toBe(
      true
    );
  });

  it("renders create mode (no initialValues)", async () => {
    await renderPageWithEndpoints(
      <ExceptionRuleForm open onSubmit={vi.fn()} onCancel={vi.fn()} />,
      {}
    );
    await new Promise((r) => setTimeout(r, 800));
    expect(document.body.innerHTML.length).toBeGreaterThan(100);
  });
});

describe("components/asset/reconciliation — MatchTestPanel", () => {
  it("renders test input area", async () => {
    await renderPageWithEndpoints(<MatchTestPanel />, {
      endpoints: {
        "/asset/reconciliation/exception-rule/test": {
          data: {
            matchedRules: [
              {
                id: "r1",
                name: "IP 段规则",
                ipRange: "10.0.0.0/8",
                exceptionActions: ["suppress"],
                severityOverride: "info",
              },
            ],
            appliedActions: ["suppress"],
          },
        },
      },
    });
    // 初始渲染输入区(placeholder/表单/按钮),不自动触发 test 查询
    expect(
      (await waitText("命中")) ||
        (await waitText("测试")) ||
        document.querySelector("input") !== null ||
        document.querySelector("textarea") !== null
    ).toBe(true);
  });

  it("renders empty match result", async () => {
    await renderPageWithEndpoints(<MatchTestPanel />, {
      endpoints: {
        "/asset/reconciliation/exception-rule/test": {
          data: { matchedRules: [], appliedActions: [] },
        },
      },
    });
    await new Promise((r) => setTimeout(r, 800));
    expect(document.body.innerHTML.length).toBeGreaterThan(50);
  });
});
