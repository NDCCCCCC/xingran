/**
 * Phase 88 Batch22 — components/network/port-write SetAccessVlanModal + PortBindingModal
 */
import { describe, it, expect, vi } from "vitest";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import SetAccessVlanModal from "../SetAccessVlanModal";
import PortBindingModal from "../PortBindingModal";

async function waitText(text: string, timeoutMs = 6000): Promise<boolean> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (document.body.innerHTML.includes(text)) return true;
    await new Promise((r) => setTimeout(r, 150));
  }
  return false;
}

const portRec = {
  id: "p1",
  portName: "Eth1/1",
  deviceName: "核心交换机-A",
  accessVlan: 100,
  status: "up",
  boundUser: "zhangsan",
  boundMac: "AA-BB-CC-DD-EE-FF",
};

describe("components/network/port-write — SetAccessVlanModal", () => {
  it("renders closed without crash", () => {
    renderPageWithEndpoints(
      <SetAccessVlanModal open={false} portRecord={null} onClose={vi.fn()} onSuccess={vi.fn()} />,
      {}
    );
    expect(document.body.innerHTML).toBeDefined();
  });

  it("renders form with port record", async () => {
    await renderPageWithEndpoints(
      <SetAccessVlanModal open portRecord={portRec as any} onClose={vi.fn()} onSuccess={vi.fn()} />,
      {}
    );
    expect(
      (await waitText("VLAN")) ||
        (await waitText("端口")) ||
        document.querySelector("input") !== null
    ).toBe(true);
  });
});

describe("components/network/port-write — PortBindingModal", () => {
  it("renders closed without crash", () => {
    renderPageWithEndpoints(
      <PortBindingModal open={false} portRecord={null} onClose={vi.fn()} onSuccess={vi.fn()} />,
      {}
    );
    expect(document.body.innerHTML).toBeDefined();
  });

  it("renders form with port record", async () => {
    await renderPageWithEndpoints(
      <PortBindingModal open portRecord={portRec as any} onClose={vi.fn()} onSuccess={vi.fn()} />,
      {}
    );
    expect(
      (await waitText("MAC")) ||
        (await waitText("绑定")) ||
        document.querySelector("input") !== null
    ).toBe(true);
  });
});
