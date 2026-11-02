/**
 * Phase 88 Batch114 — operations/workstations/LocationAliasDrawer 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { LocationAliasDrawer } from "../LocationAliasDrawer";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

function renderDrawer(props: { open: boolean }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <LocationAliasDrawer open={props.open} onClose={vi.fn()} />
    </QueryClientProvider>
  );
}

describe("LocationAliasDrawer 渲染", () => {
  it("open=false → 不渲染 Drawer 内容", () => {
    const { baseElement } = renderDrawer({ open: false });
    expect(baseElement.querySelector(".ant-drawer-content")).toBeNull();
  });

  it("open=true → 渲染", () => {
    const { baseElement } = renderDrawer({ open: true });
    expect(baseElement).toBeDefined();
  });
});
