/**
 * Phase 88 batch7 — operations 子模块渲染(modals/views/rpa/bs)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { WorkstationCardView } from "../workstations/views/CardView";
import ExecutionManagement from "../rpa/executions";
import BuildingCard from "../building-spaces/components/BuildingCard";
import FloorStack from "../building-spaces/components/FloorStack";
import WorkstationView from "../building-spaces/components/WorkstationView";
import BuildingModal from "../building-spaces/components/BuildingModal";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("operations 子模块渲染", () => {
  it("workstations/views CardView(空列表)", async () => {
    await expectRenders(
      <WorkstationCardView workstations={[]} onEdit={vi.fn()} onDelete={vi.fn()} />
    );
  });
  it("rpa/executions 页", async () => {
    await expectRenders(<ExecutionManagement />);
  });
  it("building-spaces BuildingCard", async () => {
    await expectRenders(
      <BuildingCard building={{ id: "b1", name: "一号楼", status: 0 } as any} onClick={vi.fn()} />
    );
  });
  it("building-spaces FloorStack", async () => {
    await expectRenders((<FloorStack floors={[]} />) as any);
  });
  it("building-spaces WorkstationView", async () => {
    await expectRenders((<WorkstationView floor={{ id: "f1", name: "F1" } as any} />) as any);
  });
  it("building-spaces BuildingModal(closed)", async () => {
    await expectRenders(
      (
        <BuildingModal
          building={{ id: "b1", name: "一号楼" } as any}
          open={false}
          onClose={vi.fn()}
          onSave={vi.fn()}
        />
      ) as any
    );
  });
});
