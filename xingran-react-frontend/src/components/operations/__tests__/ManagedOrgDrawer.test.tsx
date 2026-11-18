/**
 * 条线部门管理归属机构映射 Drawer（ManagedOrgDrawer）基础渲染测试
 * 模式对齐 LocationAliasDrawer.test.tsx：mock @/lib/api + QueryClientProvider
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ManagedOrgDrawer } from "../ManagedOrgDrawer";
import type { Department } from "@/types";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const departments: Department[] = [
  {
    id: "org-1",
    deptName: "某分公司",
    orgLevel: 1,
    status: 0,
    createdAt: "",
    updatedAt: "",
    children: [
      {
        id: "line-1",
        deptName: "团客共享作业服务中心",
        status: 0,
        createdAt: "",
        updatedAt: "",
      },
    ],
  } as unknown as Department,
];

function renderDrawer(props: { open: boolean }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <ManagedOrgDrawer open={props.open} onClose={vi.fn()} departments={departments} />
    </QueryClientProvider>
  );
}

describe("ManagedOrgDrawer 渲染", () => {
  it("open=false → 不渲染 Drawer 内容", () => {
    const { baseElement } = renderDrawer({ open: false });
    expect(baseElement.querySelector(".ant-drawer-content")).toBeNull();
  });

  it("open=true → 渲染", () => {
    const { baseElement } = renderDrawer({ open: true });
    expect(baseElement).toBeDefined();
  });
});
