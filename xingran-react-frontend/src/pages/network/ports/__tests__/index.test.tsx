/**
 * Phase 53 W4 — ports/index.tsx canWrite 权限 gating 单元测试 (UI-01)
 *
 * 锁定行为 (D-09 权限源 useMenuStore, ROADMAP #4 笔误纠正):
 * - useMenuStore.permissions 含 "network:port:write" → 端口表"操作"列 (D-01) + "批量配置"按钮 (D-04) 渲染
 * - useMenuStore.permissions 不含 → 两者均不渲染 (canWrite gating)
 *
 * 注: 父组件 ports/index.tsx 用 useTableManager / usePagination / 网络请求等,全量挂载 mock 成本高。
 * 本测试聚焦"权限 → 可见性"的单一行为契约,通过 mock 这些 hook 让组件可挂载,
 * 只断言权限驱动的 UI 元素 (操作列标题 + 批量配置按钮) 是否渲染。
 *
 * 前端 canWrite gating 是 UX 优化,后端 RequirePermissions(["network:port:write"]) 是真相源 (T-53-05)。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { App } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";

// Polyfill: antd v6 uses ResizeObserver (jsdom lacks it)
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (typeof globalThis.ResizeObserver === "undefined") {
  (globalThis as unknown as { ResizeObserver: typeof ResizeObserverStub }).ResizeObserver =
    ResizeObserverStub;
}

// ---- Mocks ----
// (1) useMenuStore — 核心: 测试可控制 permissions
let mockPermissions: string[] = [];
vi.mock("@/store/menuStore", () => ({
  useMenuStore: (selector: (s: { permissions: string[] }) => unknown) =>
    selector({ permissions: mockPermissions }),
}));

// (2) useTableManager — 提供 portStatus + selectedRowKeys + 控制器 (避免真实网络请求)
// ports/index.tsx 从返回值解构: portStatus, selectedRowKeys, setSelectedRowKeys,
// searchForm (必须是真实 antd Form.useForm() 实例 — antd Form 会调用其内部方法),
// loadData, getColumnSortOrder, handleTableChange, handleSearch, handleReset 等
import { Form as AntdForm } from "antd";
vi.mock("@/hooks/useTableManager", () => ({
  useTableManager: () => {
    const [formInstance] = AntdForm.useForm();
    return {
      data: [],
      total: 0,
      current: 1,
      pageSize: 10,
      loading: false,
      selectedRowKeys: [] as React.Key[],
      setSelectedRowKeys: vi.fn(),
      dataSource: [],
      handleSearch: vi.fn(),
      handleReset: vi.fn(),
      handleTableChange: vi.fn(),
      refresh: vi.fn(),
      loadData: vi.fn().mockResolvedValue(undefined),
      getColumnSortOrder: () => null,
      searchForm: formInstance,
    };
  },
}));

// (3) usePagination — 简单 stub (返回 paginationProps 对象 + setter)
vi.mock("@/hooks/usePagination", () => ({
  usePagination: () => ({
    paginationProps: {
      current: 1,
      pageSize: 10,
      total: 0,
      showSizeChanger: true,
      showTotal: (total: number) => `共 ${total} 条`,
      onChange: vi.fn(),
    },
    setCurrent: vi.fn(),
    setPageSize: vi.fn(),
    setTotal: vi.fn(),
  }),
}));

// (4) useServerSort — 简单 stub (ports/index.tsx 可能间接依赖)
vi.mock("@/hooks/useServerSort", () => ({
  useServerSort: () => ({
    orderByColumn: undefined,
    isAsc: true,
    sortOrder: null,
    handleTableChange: vi.fn(),
    resetSort: vi.fn(),
  }),
}));

// (5) API 层 — 避免真实网络请求
vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ code: 0, data: { list: [], total: 0 } }),
  get: vi.fn().mockResolvedValue({ code: 0, data: { list: [], total: 0 } }),
}));

vi.mock("@/lib/api/networkApi", () => ({
  batchExport: vi.fn().mockResolvedValue("test.zip"),
}));

// (6) withErrorHandling — 透传
vi.mock("@/utils/errorHandler", () => ({
  withErrorHandling: async <T,>(fn: () => Promise<T>) => {
    try {
      return await fn();
    } catch {
      return undefined as unknown as T;
    }
  },
}));

// (7) Antd 子组件可能依赖的 store / format
vi.mock("@/utils/datetime", () => ({
  formatDateTime: (s: string) => s,
}));

vi.mock("@/utils/tableHelpers", () => ({
  createSorterMeta: () => [],
  getColumnSortOrder: () => null,
  getSortOrder: () => null,
}));

// ActionButtons default export (default + named)
vi.mock("@/components/shared/ActionButtons", () => ({
  default: ({ actions }: { actions: { key: string; label: string }[] }) => (
    <div data-testid="action-buttons-mock">
      {actions.map((a) => (
        <span key={a.key} data-testid={`action-${a.key}`}>
          {a.label}
        </span>
      ))}
    </div>
  ),
}));

// NetworkExport + BatchExportModal — 避免 Antd Modal 渲染复杂度
vi.mock("@/components/shared/NetworkExport", () => ({
  default: () => <div data-testid="network-export-mock" />,
}));

vi.mock("@/components/shared", async () => {
  const actual = await vi.importActual<typeof import("@/components/shared")>("@/components/shared");
  return {
    ...actual,
    BatchExportModal: () => <div data-testid="batch-export-modal-mock" />,
  };
});

import PortStatusPage from "../index";

function Wrapper({ children }: { children: ReactNode }) {
  return (
    <MemoryRouter>
      <App>{children}</App>
    </MemoryRouter>
  );
}

describe("ports/index.tsx — Phase 53 W4 canWrite gating (UI-01)", () => {
  beforeEach(() => {
    mockPermissions = [];
  });

  it("renders the '操作' column header (th) when permission 'network:port:write' is granted", async () => {
    mockPermissions = ["network:port:write"];

    render(<PortStatusPage />, { wrapper: Wrapper });

    // 等待 mount 完成
    await waitFor(() => {
      expect(screen.getByText("端口总数")).toBeInTheDocument();
    });

    // 操作列标题 (antd Table 同时渲染 th + hidden measurement div, 用 th 缩小匹配范围)
    const opHeaders = document.querySelectorAll(".ant-table-thead th");
    const headerTexts = Array.from(opHeaders).map((th) => th.textContent);
    expect(headerTexts).toContain("操作");

    // 批量配置按钮 (D-04) 也渲染
    expect(screen.getByRole("button", { name: /批量配置/ })).toBeInTheDocument();
  });

  it("does NOT render '操作' column header when permission 'network:port:write' is absent", async () => {
    mockPermissions = [];

    render(<PortStatusPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(screen.getByText("端口总数")).toBeInTheDocument();
    });

    // 操作列 整列消失 (D-09 canWrite gating, 5 个操作 ActionButtons 不渲染)
    // 同时检查 th + 任何文本节点都没 "操作"
    const opHeaders = document.querySelectorAll(".ant-table-thead th");
    const headerTexts = Array.from(opHeaders).map((th) => th.textContent);
    expect(headerTexts).not.toContain("操作");
  });

  it("disables the '批量配置' button when permission is absent (D-04 disabled fallback)", async () => {
    mockPermissions = [];

    render(<PortStatusPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(screen.getByText("端口总数")).toBeInTheDocument();
    });

    // 批量配置按钮存在但 disabled (D-04: !canWrite fallback — 不消失但禁用点击)
    const bulkBtn = screen.getByRole("button", {
      name: /批量配置/,
    }) as HTMLButtonElement;
    expect(bulkBtn.disabled).toBe(true);
  });

  it("disables the '批量配置' button when permission granted but no ports selected", async () => {
    mockPermissions = ["network:port:write"];

    render(<PortStatusPage />, { wrapper: Wrapper });

    await waitFor(() => {
      expect(screen.getByText("端口总数")).toBeInTheDocument();
    });

    // 批量配置按钮存在但 disabled (selectedRowKeys.length === 0)
    const bulkBtn = screen.getByRole("button", {
      name: /批量配置/,
    }) as HTMLButtonElement;
    expect(bulkBtn.disabled).toBe(true);
  });
});
