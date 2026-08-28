/* eslint-disable no-restricted-syntax -- 测试 fixture 需要内网 IP 字面量 */
/**
 * Phase 88 Batch19b — vdi VirtualMachineList 列表渲染
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import VirtualMachineList from "../index";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

const vmList = {
  data: {
    list: [
      {
        vm_id: "vm-001",
        name: "admin-desktop",
        power_state: "poweredOn",
        ip_address: "10.0.1.50",
        bound_user_name: "zhangsan",
        last_sync_at: "2026-01-04 10:00:00",
      },
    ],
    total: 1,
    current: 1,
    pageSize: 10,
  },
};

describe("pages/vdi — VirtualMachineList", () => {
  it("renders VM rows", async () => {
    await renderPageWithEndpoints(<VirtualMachineList />, {
      endpoints: {
        "/vdi/vms/list": vmList,
        "/vdi/vms/resource-groups": { data: [] },
      },
    });
    expect(await waitText("admin-desktop")).toBe(true);
  });

  it("renders empty VM state", async () => {
    await renderPageWithEndpoints(<VirtualMachineList />, {
      endpoints: {
        "/vdi/vms/list": {
          data: { list: [], total: 0, current: 1, pageSize: 10 },
        },
        "/vdi/vms/resource-groups": { data: [] },
      },
    });
    const ok = await waitText("No data");
    expect(ok || document.querySelector(".ant-table-placeholder")).toBeTruthy();
  });
});
