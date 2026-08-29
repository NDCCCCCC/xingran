/**
 * Phase 88 Batch75 — ad-domain computers + system/settings 4 小页面渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import ADComputerPage from "../ad-domain/computers";
import EmailConfigPage from "../system/settings/email-config";
import APIConfigPage from "../system/settings/api-config";
import CaptchaBackgroundSettingsPage from "../system/settings/captcha-background";

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

describe("batch75 小页面渲染", () => {
  it("ADComputerPage 渲染", async () => {
    await renderAndAssert(<ADComputerPage />, {
      "/ad-domain/computers/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("EmailConfigPage 渲染", async () => {
    await renderAndAssert(<EmailConfigPage />, {
      "/system/configs/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("APIConfigPage 渲染", async () => {
    await renderAndAssert(<APIConfigPage />, {
      "/system/configs/list": { data: { list: [], total: 0 } },
    });
  }, 20000);

  it("CaptchaBackgroundSettingsPage 渲染", async () => {
    await renderAndAssert(<CaptchaBackgroundSettingsPage />, {
      "/system/configs/list": { data: { list: [], total: 0 } },
    });
  }, 20000);
});
