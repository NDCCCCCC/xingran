/**
 * Phase 88 Batch16b — fix-suggestion + my-notices 渲染
 * 教训(Batch16): mock 字段必须 camelCase 匹配 dataIndex;两阶段 settle 等 spin。
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import FixSuggestion from "../reconciliation/fix-suggestion";
import MyNoticesPage from "@/pages/my-notices";

/** 两阶段 settle: 等 spin 出现→消失(表格稳态) */
async function settleTable(timeoutMs = 6000): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.querySelector(".ant-table-row")) return;
    if (document.querySelector(".ant-spin-spinning")) break;
    if (document.querySelector(".ant-table-placeholder")) break;
    await new Promise((r) => setTimeout(r, 100));
  }
  while (Date.now() - start < timeoutMs) {
    if (!document.querySelector(".ant-spin-spinning")) return;
    await new Promise((r) => setTimeout(r, 100));
  }
}

const fixList = {
  data: {
    list: [
      {
        id: "fs1",
        assetCode: "AST-002",
        currentUserId: "u1",
        suggestedUserId: "u2",
        confidenceScore: 0.92,
        conflictType: "USER_MISMATCH",
        fixStatus: "pending",
        createdAt: "2026-01-02 09:00:00",
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

const fixStats = {
  data: {
    windowDays: 7,
    pending: 3,
    applied: 5,
    rolledBack: 1,
    misFixRate: 0.05,
    rejected: 2,
  },
};

const myNoticesList = {
  data: {
    list: [
      {
        noticeId: "n1",
        noticeTitle: "系统维护通知",
        noticeType: "notice",
        priority: "high",
        publishTime: "2026-01-01 08:00:00",
        isRead: false,
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

describe("asset/reconciliation — fix-suggestion", () => {
  it("renders KPI cards + suggestion table", async () => {
    await renderPageWithEndpoints(<FixSuggestion />, {
      endpoints: {
        "/asset/reconciliation/fix-suggestion/list": fixList,
        "/asset/reconciliation/fix-suggestion/stats": fixStats,
      },
    });
    await settleTable();
    expect(document.body.innerHTML).toContain("AST-002");
    expect(document.body.innerHTML).toContain("USER_MISMATCH");
  });

  it("renders empty state", async () => {
    await renderPageWithEndpoints(<FixSuggestion />, {
      endpoints: {
        "/asset/reconciliation/fix-suggestion/list": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
        "/asset/reconciliation/fix-suggestion/stats": fixStats,
      },
    });
    await settleTable();
    expect(document.querySelector(".ant-table-placeholder")).not.toBeNull();
  });
});

describe("pages/my-notices — index", () => {
  it("renders notice list rows", async () => {
    await renderPageWithEndpoints(<MyNoticesPage />, {
      endpoints: {
        "/system/my-notices": myNoticesList,
      },
    });
    await settleTable();
    expect(document.body.innerHTML).toContain("系统维护通知");
  });

  it("renders empty notice state", async () => {
    await renderPageWithEndpoints(<MyNoticesPage />, {
      endpoints: {
        "/system/my-notices": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
      },
    });
    await settleTable();
    expect(document.querySelector(".ant-table-placeholder")).not.toBeNull();
  });
});
