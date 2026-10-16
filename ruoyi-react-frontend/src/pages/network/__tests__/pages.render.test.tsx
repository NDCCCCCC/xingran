/**
 * Phase 88 — network 页面批量渲染测试(真实 hooks + mock API)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import DeviceManagement from "../devices";
import MACAddressPage from "../mac";
import TemplateManagement from "../templates";
import CredentialManagement from "../credentials";

describe("network 页面渲染(真实 hooks)", () => {
  it("devices 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<DeviceManagement />, {
      endpoints: {
        "/network/devices/list": {
          data: { list: [{ id: "d1", deviceName: "核心交换机", ipAddress: "1.1.1.1" }], total: 1 },
        },
      },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  });

  it("mac 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<MACAddressPage />, {
      endpoints: {
        "/network/mac/list": { data: { list: [], total: 0 } },
      },
    });
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  });

  it("templates 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<TemplateManagement />, {});
    await vi.waitFor(() => {
      expect(rendered.container.querySelector(".ant-table, .ant-card")).not.toBeNull();
    });
  });

  it("credentials 页渲染", async () => {
    const { rendered } = renderPageWithEndpoints(<CredentialManagement />, {});
    await vi.waitFor(() => {
      expect(rendered.container.firstChild).not.toBeNull();
    });
  });
});
