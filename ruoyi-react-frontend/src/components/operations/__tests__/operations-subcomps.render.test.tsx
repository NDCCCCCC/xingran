/**
 * Phase 88 batch13 — components/operations 子组件渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { DeptSidebar } from "../DeptSidebar";
import { StatisticsCards } from "../StatisticsCards";
import { WorkstationDeviceTable } from "../WorkstationDeviceTable";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("components/operations 子组件渲染", () => {
  it("DeptSidebar", async () => {
    await expectRenders(<DeptSidebar selectedDeptId="d1" onSelect={vi.fn()} />, {
      "/system/departments/tree": { data: [] },
    });
  });
  it("StatisticsCards(空数据)", async () => {
    await expectRenders(
      <StatisticsCards
        items={[]}
        statistics={{ totalWorkstations: 0, occupied: 0, vacant: 0, occupancyRate: 0 } as any}
      />
    );
  });
  it("StatisticsCards(有数据)", async () => {
    await expectRenders(
      <StatisticsCards items={[{ title: "总工位", value: 100, icon: "DesktopOutlined" }]} />
    );
  });
  it("WorkstationDeviceTable(空)", async () => {
    await expectRenders(<WorkstationDeviceTable workstations={[]} loading={false} />);
  });
  it("WorkstationDeviceTable(有数据+loading)", async () => {
    await expectRenders(
      <WorkstationDeviceTable workstations={[{ id: "w1", name: "A01" } as any]} loading={true} />
    );
  });
});
