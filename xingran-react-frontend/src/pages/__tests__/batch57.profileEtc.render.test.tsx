/**
 * Phase 88 Batch57 — profile/my-notices/settings 等小页面渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import ProfilePage from "../profile";
import MyNoticesPage from "../my-notices";
import NoticeDetailPage from "../my-notices/detail";
import SettingsPage from "../settings";

async function renderAndAssert(
  page: React.ReactElement,
  endpoints: Record<string, unknown> = {}
): Promise<void> {
  const { rendered } = renderPageWithEndpoints(page, { endpoints });
  await vi.waitFor(
    () => {
      expect(rendered.container.firstChild).not.toBeNull();
    },
    { timeout: 10000 }
  );
  await new Promise((r) => setTimeout(r, 300));
}

describe("batch57 小页面渲染", () => {
  it("ProfilePage 渲染", async () => {
    await renderAndAssert(<ProfilePage />);
  }, 15000);

  it("MyNoticesPage 渲染", async () => {
    await renderAndAssert(<MyNoticesPage />, {
      "/system/notices/list": { data: { list: [], total: 0 } },
    });
  }, 15000);

  it("NoticeDetailPage 渲染(route 参数)", async () => {
    await renderAndAssert(<NoticeDetailPage />, {
      route: "/my-notices/detail/n1",
      endpoints: { "/system/notices/n1": { data: { id: "n1", title: "test" } } },
    });
  }, 15000);

  it("SettingsPage 渲染", async () => {
    await renderAndAssert(<SettingsPage />);
  }, 15000);
});
