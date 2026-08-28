/**
 * Phase 88 batch6 — dashboard settings 子组件渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import DashboardScopeSelector from "../DashboardScopeSelector";
import DataSourceForm from "../DataSourceForm";
import EndpointSelector from "../EndpointSelector";
import ParamsEditor from "../ParamsEditor";
import RefreshIntervalSelector from "../RefreshIntervalSelector";
import WidgetSelector from "../WidgetSelector";
import DashboardSettings from "../DashboardSettings";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("dashboard settings 组件渲染", () => {
  it("DashboardScopeSelector", async () => {
    await expectRenders(<DashboardScopeSelector onChange={vi.fn()} />);
  });
  it("DataSourceForm", async () => {
    await expectRenders(<DataSourceForm />);
  });
  it("EndpointSelector", async () => {
    await expectRenders(<EndpointSelector value="/workorder/statistics" onChange={vi.fn()} />);
  });
  it("ParamsEditor", async () => {
    await expectRenders(<ParamsEditor />);
  });
  it("RefreshIntervalSelector", async () => {
    await expectRenders(<RefreshIntervalSelector value={60} onChange={vi.fn()} />);
  });
  it("WidgetSelector(closed)", async () => {
    await expectRenders(<WidgetSelector visible={false} onClose={vi.fn()} onSelect={vi.fn()} />);
  });
  it("DashboardSettings(closed)", async () => {
    await expectRenders(<DashboardSettings visible={false} onClose={vi.fn()} />);
  });
});
