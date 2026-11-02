/**
 * Phase 88 batch6 — reconciliation 组件渲染(ExceptionMatchList/Timeline/Drawer)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { ExceptionMatchList } from "../ExceptionMatchList";
import { ReconciliationTimeline } from "../ReconciliationTimeline";
import { ReconciliationDrawer } from "../ReconciliationDrawer";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("reconciliation 组件渲染", () => {
  it("ExceptionMatchList(空 rules + loading)", async () => {
    await expectRenders(<ExceptionMatchList rules={[]} loading={true} />);
  });
  it("ExceptionMatchList(rules)", async () => {
    await expectRenders(<ExceptionMatchList rules={[]} loading={false} assetIp="1.2.3.4" />);
  });
  it("ReconciliationTimeline(空 records)", async () => {
    await expectRenders(<ReconciliationTimeline records={[]} loading={false} />);
  });
  it("ReconciliationTimeline(loading)", async () => {
    await expectRenders(<ReconciliationTimeline records={[]} loading={true} />);
  });
  it("ReconciliationDrawer(closed)", async () => {
    await expectRenders(
      <ReconciliationDrawer
        open={false}
        onClose={vi.fn()}
        selectedAssetId={null}
        workstationId={null}
        activeTab="timeline"
        as
        any
        onTabChange={vi.fn()}
      />
    );
  });
});
