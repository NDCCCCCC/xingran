/**
 * Phase 88 batch9 — port-write 三 Modal 渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import PortBindingModal from "../PortBindingModal";
import PortWriteModal from "../PortWriteModal";
import SetAccessVlanModal from "../SetAccessVlanModal";

const portRecord: any = {
  id: "port-1",
  portName: "GigabitEthernet0/1",
  deviceName: "核心交换机",
  status: 1,
  vlanId: 100,
};

async function expectRenders(ui: React.ReactElement) {
  const { rendered } = renderPageWithEndpoints(ui, {});
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("port-write Modal 渲染", () => {
  it("PortBindingModal(closed)", async () => {
    await expectRenders(
      <PortBindingModal open={false} portRecord={null} onClose={vi.fn()} onSuccess={vi.fn()} />
    );
  });
  it("PortBindingModal(open + record)", async () => {
    await expectRenders(
      <PortBindingModal open={true} portRecord={portRecord} onClose={vi.fn()} onSuccess={vi.fn()} />
    );
  });
  it("PortWriteModal(closed)", async () => {
    await expectRenders(
      <PortWriteModal
        open={false}
        action={"shutdown" as any}
        portRecord={null}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />
    );
  });
  it("PortWriteModal(open + record)", async () => {
    await expectRenders(
      <PortWriteModal
        open={true}
        action={"shutdown" as any}
        portRecord={portRecord}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />
    );
  });
  it("SetAccessVlanModal(closed)", async () => {
    await expectRenders(
      <SetAccessVlanModal open={false} portRecord={null} onClose={vi.fn()} onSuccess={vi.fn()} />
    );
  });
  it("SetAccessVlanModal(open + record)", async () => {
    await expectRenders(
      <SetAccessVlanModal
        open={true}
        portRecord={portRecord}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />
    );
  });
});
