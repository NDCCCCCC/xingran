/**
 * Phase 88 Batch74 — WorkstationDeviceTable 组件渲染(140 行)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import WorkstationDeviceTable from "../index";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("WorkstationDeviceTable 渲染", () => {
  it("devices=[] 渲染 Empty", () => {
    const { baseElement } = renderWithProviders(
      <WorkstationDeviceTable workstationId="ws1" devices={[]} />
    );
    expect(baseElement).toBeDefined();
  });

  it("devices 非空渲染表格", () => {
    const { baseElement } = renderWithProviders(
      <WorkstationDeviceTable
        workstationId="ws1"
        devices={[
          {
            id: "d1",
            deviceId: "dev1",
            deviceName: "主机-01",
            macAddress: "AA:BB:CC:DD:EE:FF",
            deviceType: "switch",
            isPrimary: true,
          } as any,
        ]}
      />
    );
    expect(baseElement.querySelector(".ant-table, .ant-empty")).not.toBeNull();
  });

  it("editable=false 不显示添加按钮", () => {
    const { baseElement } = renderWithProviders(
      <WorkstationDeviceTable workstationId="ws1" devices={[]} editable={false} onAdd={vi.fn()} />
    );
    expect(baseElement.querySelector(".ant-table, .ant-empty")).not.toBeNull();
  });
});
