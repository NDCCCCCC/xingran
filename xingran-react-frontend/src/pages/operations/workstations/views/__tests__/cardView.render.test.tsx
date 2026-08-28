/**
 * Phase 88 batch14 — workstations/views/CardView + FloorPlanView 渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { WorkstationCardView } from "../CardView";

const mockWs: any = (id: string, name: string) => ({
  id,
  name,
  code: name,
  status: 0,
  workstationType: 1,
  positionX: 100,
  positionY: 200,
  width: 80,
  height: 60,
});

async function expectRenders(ui: React.ReactElement) {
  const { rendered } = renderPageWithEndpoints(ui, {});
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("workstations views 渲染", () => {
  it("CardView(空)", async () => {
    await expectRenders(
      <WorkstationCardView workstations={[]} onEdit={vi.fn()} onDelete={vi.fn()} />
    );
  });
  it("CardView(3 工位)", async () => {
    await expectRenders(
      <WorkstationCardView
        workstations={[mockWs("1", "A01"), mockWs("2", "A02"), mockWs("3", "A03")]}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
      />
    );
  });
});
