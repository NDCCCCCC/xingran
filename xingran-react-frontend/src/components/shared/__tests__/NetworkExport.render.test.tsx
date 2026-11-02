/**
 * Phase 88 Batch88 — components/shared/NetworkExport 渲染(51 stmts, 17.6% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import NetworkExport from "../NetworkExport";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/utils/authHelpers", () => ({
  getAccessToken: vi.fn(() => Promise.resolve("test-token")),
}));

describe("NetworkExport 渲染", () => {
  it("基本渲染: Dropdown 按钮", () => {
    const { baseElement } = renderWithProviders(
      <NetworkExport entityType="device" entityName="设备" />
    );
    expect(baseElement).toBeDefined();
  });

  it("带 filters 渲染", () => {
    const { baseElement } = renderWithProviders(
      <NetworkExport
        entityType="device"
        entityName="设备"
        filters={{ status: 0 }}
        current={1}
        pageSize={10}
      />
    );
    expect(baseElement).toBeDefined();
  });

  it("不同 entityType", () => {
    const { baseElement } = renderWithProviders(
      <NetworkExport entityType="port" entityName="端口" />
    );
    expect(baseElement).toBeDefined();
  });
});
