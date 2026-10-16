/**
 * Phase 88 batch8 — vdi/ad-domain/workorder 剩余子页渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import VDIServerConfig from "../vdi/VDIServerConfig";
import VirtualMachineDetail from "../vdi/VirtualMachineDetail";
import WorkOrderStatisticsPage from "../workorder/statistics";
import WorkOrderCategoryPage from "../workorder/categories";
import ADOuPage from "../ad-domain/ous";
import ADConfigPage from "../ad-domain/configs";
import ADLogsPage from "../ad-domain/logs";
import RpaExecutions from "../operations/rpa/executions";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("batch8 子页渲染", () => {
  it("vdi/VDIServerConfig 页", async () => {
    await expectRenders(<VDIServerConfig />);
  });
  it("workorder/statistics 页", async () => {
    await expectRenders(<WorkOrderStatisticsPage />);
  });
  it("workorder/categories 页", async () => {
    await expectRenders(<WorkOrderCategoryPage />);
  });
  it("ad-domain/configs 页", async () => {
    await expectRenders(<ADConfigPage />);
  });
  it("ad-domain/logs 页", async () => {
    await expectRenders(<ADLogsPage />);
  });
  it("ad-domain/ous 页(重复深测)", async () => {
    await expectRenders(<ADOuPage />);
  });
});
