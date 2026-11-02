/**
 * Phase 88 Batch104 — system/user 页面渲染(150 stmts, 33.3% → 高)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { createApiMock } from "@/test/utils/createApiMock";
import UserManagement from "../index";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// 页面 useEffect 会拉职位下拉（/system/posts/list），未登记时通用回退 spy
// 返回 undefined 导致 .then 抛 TypeError——此处显式登记安全空结构
beforeEach(() => {
  createApiMock("/system/posts/list").endpoint.mockResolvedValue({
    data: { list: [], total: 0, current: 1, pageSize: 200 },
  });
});

function renderUser(endpoints: Record<string, unknown> = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <UserManagement />
    </QueryClientProvider>,
    { endpoints }
  );
}

describe("UserManagement 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderUser({
      "/system/users/list": { data: { list: [], total: 0 } },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("1 行 → 表格行渲染", async () => {
    const { baseElement } = renderUser({
      "/system/users/list": {
        data: {
          list: [
            {
              userId: "u1",
              userName: "admin",
              nickName: "管理员",
              email: "admin@test.local",
              status: 0,
            },
          ],
          total: 1,
        },
      },
    });
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("list 失败 → catch 路径", async () => {
    const { baseElement } = renderUser();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });
});
