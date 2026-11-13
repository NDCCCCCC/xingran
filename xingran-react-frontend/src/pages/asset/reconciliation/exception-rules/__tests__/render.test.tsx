/**
 * Phase 88 Batch422 — pages/asset/reconciliation/exception-rules 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import ExceptionRulesPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/assetApi", () => ({
  exceptionRuleApi: {
    list: vi.fn(async () => ({ list: [], total: 0 })),
    create: vi.fn(async () => ({})),
    update: vi.fn(async () => ({})),
    delete: vi.fn(async () => ({})),
  },
  // 页面实际消费 reconciliationApi —— 造一条带例外动作的规则,验证动作列显示字典标签
  reconciliationApi: {
    exceptionRule: {
      list: vi.fn(async () => ({
        list: [
          {
            id: "r1",
            name: "测试规则",
            ipRange: "10.0.0.0/8",
            exceptionActions: ["silence"],
            severityOverride: "high",
            scopeType: "global",
            scopeId: null,
            reason: "测试环境豁免规则",
            isActive: 0,
            expiresAt: null,
          },
        ],
        total: 1,
      })),
      create: vi.fn(async () => ({})),
      update: vi.fn(async () => ({})),
      delete: vi.fn(async () => ({})),
    },
    baseline: {
      snapshot: vi.fn(async () => ({})),
    },
  },
}));

import { createApiMock } from "@/test/utils/createApiMock";

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/asset/reconciliation/exception-rules"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("exception-rules page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<ExceptionRulesPage />, { wrapper: Wrap })).not.toThrow();
  });

  it("例外动作 / 严重度覆盖显示字典标签(字典优先)", async () => {
    createApiMock("/system/dicts/data/list").endpoint.mockResolvedValue({
      data: {
        list: [
          { id: "d1", dictLabel: "静默期", dictValue: "silence", listClass: "default" },
          { id: "d2", dictLabel: "高", dictValue: "high", listClass: "error" },
        ],
        total: 2,
      },
    });
    render(<ExceptionRulesPage />, { wrapper: Wrap });
    expect(await screen.findByText("静默期")).not.toBeNull();
    expect(await screen.findByText("高")).not.toBeNull();
  });
});
