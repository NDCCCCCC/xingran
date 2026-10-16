/**
 * Phase 88 Batch19 — vdi VDIServerConfig 列表渲染
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import VDIServerConfigPage from "../index";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

const serversList = {
  data: {
    list: [
      {
        id: "s1",
        name: "生产 VDI 集群",
        endpoint: "https://vdi.corp.local",
        username: "svc-vdi",
        tenant_id: "t1",
        status: 0,
        token_expiry: "2026-12-31",
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

describe("pages/vdi — VDIServerConfig", () => {
  it("renders server rows", async () => {
    await renderPageWithEndpoints(<VDIServerConfigPage />, {
      endpoints: {
        "/vdi/servers/list": serversList,
      },
    });
    expect(await waitText("生产 VDI 集群")).toBe(true);
  });

  it("renders empty servers state", async () => {
    await renderPageWithEndpoints(<VDIServerConfigPage />, {
      endpoints: {
        "/vdi/servers/list": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
      },
    });
    const ok = await waitText("No data");
    expect(ok || document.querySelector(".ant-table-placeholder")).toBeTruthy();
  });
});
