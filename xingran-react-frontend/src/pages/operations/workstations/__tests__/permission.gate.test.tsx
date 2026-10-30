/**
 * Phase 86 W2（RBAC-02）—— 工位管理页操作级权限门控测试。
 *
 * 覆盖 PLAN 2.4 的四条路径（三路径 + 「ready 空权限集 vs 未就绪」可区分）：
 *   ① 只持 `opsWorkstation.query` → 新增/编辑/删除/批量删除/导入/导出/「下载部门映射表」
 *      /「映射」（alias 抽屉入口）**均不在 DOM**；
 *   ② 未就绪（permissionsLoaded=false / permissionsError 非空）→ 上述入口**在 DOM 且 disabled**
 *      + 提示（title 含「权限」）；
 *   ③ 持 `opsWorkstation.add/.edit/.delete/.import/.export` + `opsLocationAlias.list`
 *      → 入口存在且可用（含 alias 抽屉入口）；
 *   ④ ready + **空权限集** → 不渲染（与「未就绪」可区分）。
 *
 * 同时覆盖两条「接法」：
 *   - 行编辑/删除走 `columns.tsx` 的列工厂（决策经列 props 传入 → ActionButtons 内建门控）；
 *   - alias 抽屉入口走页面层条件渲染（`permAliasList.visible`）。
 *
 * ⚠️ 本目录**没有** index 渲染测试（`__tests__/hooks.test.tsx` 注释记载：workstations
 *    index.tsx 因 jsdom 死锁不做整页渲染测试）。本文件是本目录第一个整页渲染测试，
 *    因此把三个重型懒加载子件（平面图编辑器 / ReconciliationDrawer / LocationAliasDrawer）
 *    与 Excel 弹层一并替换为轻量占位 —— 门控断言对象是**工具栏与行操作按钮**，
 *    不依赖这些子件内部实现。
 * ⚠️ 断言只用 vitest 原生 matcher + 原生 disabled 属性，不用 jest-dom matcher
 *    （见 buildings 同款说明：测试文件被 tsconfig.app.json 排除，jest-dom 类型不在
 *    types 里，用它会引入类型错误）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/opsApi", () => ({
  buildingApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [] } })),
  },
  floorApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [] } })),
    get: vi.fn(() => Promise.resolve({ data: null })),
  },
  workstationApi: {
    list: vi.fn(),
    statistics: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    updatePositions: vi.fn(),
    delete: vi.fn(),
    batch: vi.fn(),
    export: vi.fn(),
    import: vi.fn(),
  },
  deptApi: {
    tree: vi.fn(() => Promise.resolve({ data: [] })),
    exportMapping: vi.fn(),
  },
  locationAliasApi: {
    list: vi.fn(() => Promise.resolve({ data: { list: [], total: 0 } })),
    create: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// 重型子件替换为轻量占位（见文件头说明）：门控断言对象是工具栏/行操作按钮
vi.mock("@/components/shared/ExcelImportLazy", () => ({ default: () => null }));
vi.mock("@/components/shared/ExcelExport", () => ({ default: () => null }));
vi.mock("@/components/reconciliation", () => ({
  HealthCard: () => null,
  ReconciliationDrawer: () => null,
  useWorkstationHealth: () => ({ data: null }),
  useReconciliationVisibility: () => false,
}));
vi.mock("../LocationAliasDrawer", () => ({
  LocationAliasDrawer: ({ open }: { open: boolean }) =>
    open ? <div data-testid="alias-drawer" /> : null,
}));
vi.mock("@/components/operations/WorkstationDeviceTable", () => ({ default: () => null }));
vi.mock("@/components/operations/DeptSidebar", () => ({ DeptSidebar: () => null }));
vi.mock("@/components/operations/StatisticsCards", () => ({ StatisticsCards: () => null }));
vi.mock("../views/FloorPlanView", () => ({
  WorkstationFloorPlanView: () => null,
}));
vi.mock("../views/CardView", () => ({
  WorkstationCardView: () => null,
}));
vi.mock("../modals", () => ({
  WorkstationEditModal: () => null,
}));
// ⚠️ 本目录既有测试注释记载：workstations index.tsx 在 jsdom 里会死锁
// （Three.js lazy 加载 + 持久化状态跨测串扰）。本文件把非门控相关的外围 hook
// 与数据层一并替换为最小桩，让整页渲染只保留「工具栏 + 行操作按钮」这一层。
vi.mock("@/hooks/useTableQuery", () => ({
  useTableQuery: () => ({ data: undefined }),
}));
vi.mock("@/hooks/useDict", () => ({
  useDict: () => ({ data: [] }),
}));
vi.mock("@/hooks/useAliasByLocation", () => ({
  useAliasByLocation: () => ({ data: [] }),
}));
vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: () => ({ data: [], isLoading: false }),
  useInvalidateDept: () => () => {},
}));
vi.mock("../hooks", () => ({
  // ⚠️ 不要在 render 期调 setStatistics：每次渲染都 setState 新对象会触发
  // React「Too many re-renders」无限循环。统计卡片已被替换为占位组件，无需注入数据。
  useWorkstationData: () => ({
    loadStatistics: () => {},
    loadFloorOptions: () => {},
    loadDeptOptions: () => {},
    loadUserOptions: () => {},
    ensureUser: () => {},
    loadDepartments: () => {},
    loadUsers: () => {},
    buildTreeData: () => [],
    handleOrgChange: () => {},
    handleDeptChange: () => {},
  }),
  useWorkstationModals: () => ({
    closeModal: () => {},
    handleDelete: () => {},
    handleBatchDelete: () => {},
  }),
  useWorkstationView: () => ({
    viewMode: "table",
    setViewMode: () => {},
    selectedFloorForPlan: "",
    floorPlanWorkstations: [],
    handleFloorChangeForPlan: () => {},
    handlePositionUpdate: async () => {},
  }),
}));

import WorkstationPage from "../index";
import { workstationApi } from "@/lib/opsApi";

const WS_ROW = {
  id: "ws1",
  name: "A-01",
  buildingName: "一号楼",
  floorName: "1层",
  deptName: "运维部",
  userName: "张三",
  type: 0,
  status: 0,
};

function setPermissionState(
  permissions: string[],
  state: { loaded?: boolean; error?: Error } = {}
): void {
  useMenuStore.setState({
    permissions,
    permissionsLoaded: state.loaded ?? true,
    permissionsError: state.error ?? null,
  });
}

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(<WorkstationPage />, { queryClient: qc });
}

/** 断言某按钮「在 DOM 且 disabled，且 title 带未就绪提示」 */
function expectDisabledWithHint(name: string | RegExp): void {
  const btn = screen.getByRole("button", { name }) as HTMLButtonElement;
  expect(btn).toBeTruthy();
  expect(btn.disabled).toBe(true);
  expect(btn.getAttribute("title") ?? "").toContain("权限");
}

describe("工位管理页操作级权限门控（Phase 86 W2）", () => {
  beforeEach(() => {
    vi.mocked(workstationApi.list).mockResolvedValue({
      data: { list: [WS_ROW], total: 1 },
    } as never);
    vi.mocked(workstationApi.statistics).mockResolvedValue({
      data: { total: 1, available: 1, occupied: 0, maintain: 0 },
    } as never);
  });

  // ==================== ① ready + 只持读码 ====================

  it("① 只持 opsWorkstation.query → 全部写入口（含导出类）不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.opsWorkstation.query]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("A-01")).toBeTruthy();
    });

    // 工具栏：新增 / 导入 / 导出 / 下载部门映射表 / 映射（alias 抽屉入口）
    expect(screen.queryByRole("button", { name: /新增工位/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /导入/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /导出/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /下载部门映射表/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /映射$/ })).toBeNull();
    // 行操作（columns.tsx 列工厂 → ActionButtons 内建三态：ready 无码 → 不渲染）
    expect(screen.queryByRole("button", { name: /编辑/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /删除/ })).toBeNull();
    // 批量删除（页面层条件渲染）→ 未选中时也不该有
    expect(screen.queryByRole("button", { name: /批量删除/ })).toBeNull();
    // 平面图视图入口：拖动工位会写 updatePositions → 归 opsWorkstation.edit，
    // ready 无码 → 「平面图」选项不在 DOM
    expect(screen.queryByRole("radio", { name: /平面图/ })).toBeNull();
  });

  // ==================== ④ ready + 空权限集 ====================

  it("④ ready 且空权限集 → 入口不渲染（与「未就绪」可区分）", async () => {
    setPermissionState([], { loaded: true });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("A-01")).toBeTruthy();
    });

    expect(screen.queryByRole("button", { name: /新增工位/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /下载部门映射表/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /映射$/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /编辑/ })).toBeNull();
  });

  // ==================== ② 未就绪（loading / error） ====================

  it("② 未就绪（loading）→ 入口在 DOM 且 disabled + 提示（含 alias 抽屉入口）", async () => {
    setPermissionState([], { loaded: false });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("A-01")).toBeTruthy();
    });

    expectDisabledWithHint(/新增工位/);
    expectDisabledWithHint(/导入/);
    expectDisabledWithHint(/导出/);
    // NEW-02：导出类写入口「下载部门映射表」同样「渲染但 disabled」
    expectDisabledWithHint(/下载部门映射表/);
    // alias 抽屉入口（旧式 canListAlias 已迁三态）→ 未就绪时渲染但 disabled
    expectDisabledWithHint(/映射$/);
    // 行编辑/行删除：两者都是 columns.tsx 里的普通 action（onClick 形态，无 render），
    // 未就绪时 aria-label 被覆盖为 hint 文案（ActionButtons.tsx:87-88）→ 按 hint 定位，
    // 断言「行操作至少 2 个入口都在 DOM 且 disabled」。
    const hintedBtns = screen.getAllByRole("button", {
      name: /权限信息加载中/,
    }) as HTMLButtonElement[];
    expect(hintedBtns.length).toBeGreaterThanOrEqual(2); // 行编辑 + 行删除
    hintedBtns.forEach((btn) => {
      expect(btn.disabled).toBe(true);
      expect(btn.getAttribute("title") ?? "").toContain("权限");
    });
    // 平面图视图入口（写 updatePositions → opsWorkstation.edit）：未就绪时渲染但 disabled
    const floorPlanRadio = screen.getByRole("radio", {
      name: /平面图/,
    }) as HTMLInputElement;
    expect(floorPlanRadio.disabled).toBe(true);
    // antd Radio.Button 的 title 落在包裹的 label 上（不在 input 本身）
    expect(floorPlanRadio.closest("label")?.getAttribute("title") ?? "").toContain("权限");
  });

  it("②-2 未就绪（error）→ 入口在 DOM 且 disabled + 提示", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("A-01")).toBeTruthy();
    });

    expectDisabledWithHint(/新增工位/);
    expectDisabledWithHint(/导入/);
    expectDisabledWithHint(/下载部门映射表/);
    expectDisabledWithHint(/映射$/);
  });

  // ==================== ③ ready + 全写码 ====================

  it("③ 持 add/edit/delete/import/export + alias.list → 入口存在且可用", async () => {
    setPermissionState([
      PERMISSION_CODES.opsWorkstation.query,
      PERMISSION_CODES.opsWorkstation.add,
      PERMISSION_CODES.opsWorkstation.edit,
      PERMISSION_CODES.opsWorkstation.delete,
      PERMISSION_CODES.opsWorkstation.import,
      PERMISSION_CODES.opsWorkstation.export,
      PERMISSION_CODES.opsLocationAlias.list,
    ]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("A-01")).toBeTruthy();
    });

    const addBtn = screen.getByRole("button", { name: /新增工位/ }) as HTMLButtonElement;
    expect(addBtn.disabled).toBe(false);
    const importBtn = screen.getByRole("button", { name: /导入/ }) as HTMLButtonElement;
    expect(importBtn.disabled).toBe(false);
    const exportBtn = screen.getByRole("button", { name: /导出/ }) as HTMLButtonElement;
    expect(exportBtn.disabled).toBe(false);
    // NEW-02：导出类写入口在持 export 时可用
    const mappingBtn = screen.getByRole("button", { name: /下载部门映射表/ }) as HTMLButtonElement;
    expect(mappingBtn.disabled).toBe(false);
    // alias 抽屉入口在持 opsLocationAlias.list 时可用
    const aliasBtn = screen.getByRole("button", { name: /映射$/ }) as HTMLButtonElement;
    expect(aliasBtn.disabled).toBe(false);
    // 行编辑/删除可用
    expect(screen.getByRole("button", { name: /编辑/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /删除/ })).toBeTruthy();
  });
}, 30000);
