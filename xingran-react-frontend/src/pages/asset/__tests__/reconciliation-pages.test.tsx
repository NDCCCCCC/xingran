/* eslint-disable no-restricted-syntax -- 测试 fixture 需要内网 IP 字面量 */
/**
 * Phase 88 Batch16 — asset/reconciliation 子页面渲染(exceptions/exception-rules)
 * renderPage 模式: 真实 hooks + QueryClient + @/lib/api 端点 mock
 * 注意: react-query + antd 异步链在 findBy 轮询下 DOM 更新被 act 批处理延迟,
 * 采用 settle 等待 + innerHTML 断言模式(debug 实测可靠)。
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import Exceptions from "../reconciliation/exceptions";
import ExceptionRulesPage from "../reconciliation/exception-rules";

/** 等 react-query 完成 + antd 表格渲染(spin 消失后表格处于稳态) */
async function settleTable(timeoutMs = 6000): Promise<void> {
  const start = Date.now();
  // 第一阶段: 等 spin 出现或直接出数据
  while (Date.now() - start < timeoutMs) {
    if (document.querySelector(".ant-table-row")) return;
    const spin = document.querySelector(".ant-spin-spinning");
    const placeholder = document.querySelector(".ant-table-placeholder");
    if (spin) break; // 进入 loading,等第二阶段
    if (placeholder) break;
    await new Promise((r) => setTimeout(r, 100));
  }
  // 第二阶段: 等 spin 消失(loading 完成)
  while (Date.now() - start < timeoutMs) {
    if (!document.querySelector(".ant-spin-spinning")) return;
    await new Promise((r) => setTimeout(r, 100));
  }
}

const exceptionList = {
  data: {
    list: [
      {
        id: "ex1",
        conflictType: "DEPT_MISMATCH",
        severity: "critical",
        assetCode: "AST-001",
        assetIp: "10.1.1.100",
        assetIpDisplay: "10.1.1.100",
        physicalUsername: "zhangsan",
        responsibleUsername: "lisi",
        detectedAt: "2026-01-01 10:00:00",
        exceptionRuleId: null,
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

const ruleList = {
  data: {
    list: [
      {
        id: "r1",
        name: "高风险部门漂移",
        ipRange: "10.0.0.0/8",
        exceptionActions: [],
        severityOverride: "critical",
        scopeType: "GLOBAL",
        expiresAt: null,
        isActive: true,
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

describe("asset/reconciliation — exceptions", () => {
  it("renders filter form + table rows", async () => {
    await renderPageWithEndpoints(<Exceptions />, {
      endpoints: { "/asset/reconciliation/exception/list": exceptionList },
    });
    await settleTable();
    expect(document.body.innerHTML).toContain("AST-001");
    expect(document.body.innerHTML).toContain("10.1.1.100");
  });

  it("renders empty state when no exceptions", async () => {
    await renderPageWithEndpoints(<Exceptions />, {
      endpoints: {
        "/asset/reconciliation/exception/list": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
      },
    });
    await settleTable();
    expect(document.querySelector(".ant-table-placeholder")).not.toBeNull();
  });

  it("reads initial filters from URL query (?type=)", async () => {
    await renderPageWithEndpoints(<Exceptions />, {
      route: "/?type=DEPT_MISMATCH",
      endpoints: { "/asset/reconciliation/exception/list": exceptionList },
    });
    await settleTable();
    expect(document.body.innerHTML).toContain("AST-001");
  });
});

describe("asset/reconciliation — exception-rules", () => {
  it("renders rules table with row", async () => {
    await renderPageWithEndpoints(<ExceptionRulesPage />, {
      endpoints: { "/asset/reconciliation/exception-rule/list": ruleList },
    });
    await settleTable();
    expect(document.body.innerHTML).toContain("高风险部门漂移");
  });

  it("renders empty rules state", async () => {
    await renderPageWithEndpoints(<ExceptionRulesPage />, {
      endpoints: {
        "/asset/reconciliation/exception-rule/list": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
      },
    });
    await settleTable();
    expect(document.querySelector(".ant-table-placeholder")).not.toBeNull();
  });
});
