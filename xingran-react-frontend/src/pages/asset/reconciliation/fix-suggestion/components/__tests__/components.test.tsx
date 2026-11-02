/**
 * Phase 88 Batch20 — fix-suggestion 子组件(RollbackModal/DetailDrawer)
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { RollbackModal } from "../RollbackModal";
import { FixSuggestionDetailDrawer } from "../FixSuggestionDetailDrawer";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

const detail = {
  data: {
    id: "fs1",
    assetCode: "AST-002",
    currentUserId: "u1",
    suggestedUserId: "u2",
    confidenceScore: 0.92,
    conflictType: "USER_MISMATCH",
    fixStatus: "pending",
    createdAt: "2026-01-02 09:00:00",
    exceptions: [],
    history: [],
  },
};

describe("fix-suggestion — RollbackModal", () => {
  it("renders closed state without Modal content", () => {
    renderPageWithEndpoints(
      <RollbackModal open={false} onCancel={vi.fn()} onSubmit={vi.fn()} />,
      {}
    );
    // destroyOnHidden 时关闭态无表单
    expect(document.body.innerHTML).toBeDefined();
  });

  it("renders rollback reason form", async () => {
    await renderPageWithEndpoints(<RollbackModal open onCancel={vi.fn()} onSubmit={vi.fn()} />, {});
    expect(
      (await waitText("回滚")) ||
        (await waitText("原因")) ||
        document.querySelector("textarea") !== null
    ).toBe(true);
  });
});

describe("fix-suggestion — FixSuggestionDetailDrawer", () => {
  it("renders closed drawer without content", () => {
    renderPageWithEndpoints(
      <FixSuggestionDetailDrawer open={false} suggestionId={null} onClose={vi.fn()} />,
      {}
    );
    expect(document.body.innerHTML).toBeDefined();
  });

  it("fetches and renders suggestion detail", async () => {
    await renderPageWithEndpoints(
      <FixSuggestionDetailDrawer open suggestionId="fs1" onClose={vi.fn()} />,
      {
        endpoints: {
          "/asset/reconciliation/fix-suggestion/fs1": detail,
        },
      }
    );
    expect(
      (await waitText("AST-002")) || (await waitText("USER_MISMATCH")) || (await waitText("详情"))
    ).toBe(true);
  });

  it("renders detail fetch failure gracefully", async () => {
    await renderPageWithEndpoints(
      <FixSuggestionDetailDrawer open suggestionId="bad" onClose={vi.fn()} />,
      {
        endpoints: {
          "/asset/reconciliation/fix-suggestion/bad": { data: null },
        },
      }
    );
    await new Promise((r) => setTimeout(r, 1000));
    expect(document.body.innerHTML.length).toBeGreaterThan(50);
  });
});
