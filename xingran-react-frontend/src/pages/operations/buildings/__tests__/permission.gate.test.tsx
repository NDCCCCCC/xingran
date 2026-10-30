/**
 * Phase 86 W2（RBAC-02）—— 楼宇管理页操作级权限门控测试。
 *
 * 覆盖 PLAN 2.4 的四条路径（三路径 + 「ready 空权限集 vs 未就绪」可区分）：
 *   ① 只持 `opsBuilding.query` → 新增/编辑/删除/批量删除/导入/导出**均不在 DOM**；
 *   ② 未就绪（permissionsLoaded=false / permissionsError 非空）→ 上述入口**在 DOM 且 disabled**
 *      且带提示（title 含「权限」）；
 *   ③ 持 `opsBuilding.add/.edit/.delete/.import/.export` → 入口存在且可用；
 *   ④ ready + **空权限集** → 不渲染（与「未就绪」在数据上可区分）。
 *
 * ⚠️ 禁止用 `permissions.length === 0` 判「未就绪」——空权限集与未加载完在数据上不可区分。
 *    本测试用 menuStore 的显式就绪位（permissionsLoaded / permissionsError）构造三态。
 * ⚠️ 断言只用 vitest 原生 matcher（toBeNull / toBe / toContain）+ 原生 disabled 属性，
 *    不用 jest-dom matcher —— `tsconfig.app.json` 排除了测试文件，jest-dom 类型不在
 *    专项类型检查的 types 里，用它会引入类型错误（本文件要求类型错误数 = 0）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

// 页面数据层走 opsApi（不是 @/lib/api 的裸 post），沿用既有 index.render.test.tsx 的
// mock 形态：整模块替换 @/lib/opsApi，避免任何真实网络请求。
vi.mock("@/lib/opsApi", () => ({
  buildingApi: {
    list: vi.fn(),
    statistics: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    batch: vi.fn(),
    downloadTemplate: vi.fn(),
    export: vi.fn(),
    import: vi.fn(),
    get: vi.fn(),
  },
  deptApi: {
    tree: vi.fn(() => Promise.resolve({ data: [] })),
    exportMapping: vi.fn(),
  },
  floorApi: { list: vi.fn(() => Promise.resolve({ data: { list: [] } })) },
  workstationApi: { list: vi.fn(() => Promise.resolve({ data: { list: [] } })) },
}));

// Excel 导入/导出弹层是 shared 组件（本波零 diff），这里替换为轻量占位，
// 让测试聚焦「触发按钮」的门控（modal 容器是否渲染不是本测试的判定对象）。
vi.mock("@/components/shared/ExcelImportLazy", () => ({
  default: () => null,
}));
vi.mock("@/components/shared/ExcelExport", () => ({
  default: () => null,
}));

/**
 * Phase 86 W4 —— **测试隔离缺陷修复**（既有测试语义适配，登记在 86-STATE.md）：
 *
 * 本文件此前**未** mock `useDeptTree`，而 `buildings/index.tsx:107` 直接消费它
 * → `getDeptTree()`（`@/lib/dutyApi`）→ `@/lib/api` 的**真实** post 请求。
 * 当开发机**有后端在跑**（实测 :8080 / :9000 LISTENING）时，该请求返回
 * `INVALID_TOKEN` → `@/lib/api` 的失效处理链调用 `menuStore.clearMenus()`
 * → `permissionsLoaded` 被重置为 **false** → 门控翻成「未就绪」态
 * → 用例 ③ 的 `addBtn.disabled` 变成 `true` → **假红**。
 *
 * 即该用例的成败取决于「本机有没有后端在跑」，属**环境依赖型 flaky**。
 * 修法与同目录既有测试（`hooks.test.tsx:17` / `useDepartmentData.test.tsx:17`）一致：
 * mock 掉 `useDeptTree`，切断真实网络请求。
 * ⚠️ **未删任何断言**，未改门控语义；仅补齐测试隔离。
 */
vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: vi.fn(() => ({ data: [], isLoading: false })),
}));

import BuildingPage from "../index";
import { buildingApi } from "@/lib/opsApi";

const BUILDING_ROW = {
  id: "b1",
  name: "一号楼",
  code: "B1",
  address: "北京市朝阳区",
  status: 0,
  floorCount: 5,
};

/** 设置权限集与显式就绪位（构造 ready / loading / error 三态） */
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
  return renderWithProviders(<BuildingPage />, { queryClient: qc });
}

/** 断言某按钮「在 DOM 且 disabled，且 title 带未就绪提示」 */
function expectDisabledWithHint(name: string | RegExp): void {
  const btn = screen.getByRole("button", { name }) as HTMLButtonElement;
  expect(btn).toBeTruthy();
  expect(btn.disabled).toBe(true);
  expect(btn.getAttribute("title") ?? "").toContain("权限");
}

describe("楼宇管理页操作级权限门控（Phase 86 W2）", () => {
  beforeEach(() => {
    vi.mocked(buildingApi.list).mockResolvedValue({
      data: { list: [BUILDING_ROW], total: 1 },
    } as never);
    vi.mocked(buildingApi.statistics).mockResolvedValue({
      data: { total: 1, active: 1, inactive: 0 },
    } as never);
  });

  // ==================== ① ready + 只持读码 ====================

  it("① 只持 opsBuilding.query → 新增/导入/导出不在 DOM，行编辑/删除也不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.opsBuilding.query]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("一号楼")).toBeTruthy();
    });

    expect(screen.queryByRole("button", { name: /新增楼宇/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /导入/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /导出/ })).toBeNull();
    // 行操作（ActionButtons 内建三态：ready 无码 → 不渲染）
    expect(screen.queryByRole("button", { name: /编辑/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /删除/ })).toBeNull();
    // 批量删除：PageTable 的 batchDelete 条件传参 → 不传时 rowSelection 不渲染
    expect(screen.queryAllByRole("checkbox")).toHaveLength(0);
  });

  // ==================== ④ ready + 空权限集 ====================

  it("④ ready 且空权限集 → 入口不渲染（与「未就绪」可区分）", async () => {
    setPermissionState([], { loaded: true });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("一号楼")).toBeTruthy();
    });

    expect(screen.queryByRole("button", { name: /新增楼宇/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /导入/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /编辑/ })).toBeNull();
  });

  // ==================== ② 未就绪（loading / error） ====================

  it("② 未就绪（loading）→ 入口在 DOM 且 disabled + 提示", async () => {
    // permissions 为空但 permissionsLoaded=false —— 这正是「未就绪」态
    setPermissionState([], { loaded: false });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("一号楼")).toBeTruthy();
    });

    expectDisabledWithHint(/新增楼宇/);
    expectDisabledWithHint(/导入/);
    expectDisabledWithHint(/导出/);
    // 行操作同样「渲染但 disabled」（ActionButtons 内建三态）。
    // ⚠️ 未就绪时普通按钮的 aria-label 被覆盖为 hint 文案
    //    （`ActionButtons.tsx:87-88` `aria-label={decision.hint ?? action.label}`），
    //    而 `render` 形态（删除走 createDeleteAction → ConfirmDelete）被包在
    //    `<span title={hint} aria-disabled="true">` 里（`ActionButtons.tsx:69-76`）。
    //    因此分两种形态断言：编辑按 hint 名定位；删除按「被 aria-disabled 包裹」定位。
    const editBtn = screen.getByRole("button", { name: /权限信息加载中/ }) as HTMLButtonElement;
    expect(editBtn.disabled).toBe(true);
    expect(editBtn.getAttribute("title") ?? "").toContain("权限");

    const deleteEntry = screen.getByRole("button", { name: /删除/ });
    expect(deleteEntry.closest('[aria-disabled="true"]')).toBeTruthy();
    expect(deleteEntry.closest('[aria-disabled="true"]')?.getAttribute("title") ?? "").toContain(
      "权限"
    );
  });

  it("②-2 未就绪（error）→ 入口在 DOM 且 disabled + 提示", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("一号楼")).toBeTruthy();
    });

    expectDisabledWithHint(/新增楼宇/);
    expectDisabledWithHint(/导入/);
    expectDisabledWithHint(/导出/);
  });

  // ==================== ③ ready + 全写码 ====================

  it("③ 持 add/edit/delete/import/export → 入口存在且可用", async () => {
    setPermissionState([
      PERMISSION_CODES.opsBuilding.query,
      PERMISSION_CODES.opsBuilding.add,
      PERMISSION_CODES.opsBuilding.edit,
      PERMISSION_CODES.opsBuilding.delete,
      PERMISSION_CODES.opsBuilding.import,
      PERMISSION_CODES.opsBuilding.export,
    ]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("一号楼")).toBeTruthy();
    });

    const addBtn = screen.getByRole("button", { name: /新增楼宇/ }) as HTMLButtonElement;
    expect(addBtn.disabled).toBe(false);
    const importBtn = screen.getByRole("button", { name: /导入/ }) as HTMLButtonElement;
    expect(importBtn.disabled).toBe(false);
    const exportBtn = screen.getByRole("button", { name: /导出/ }) as HTMLButtonElement;
    expect(exportBtn.disabled).toBe(false);
    const editBtn = screen.getByRole("button", { name: /编辑/ }) as HTMLButtonElement;
    expect(editBtn.disabled).toBe(false);
    expect(screen.getByRole("button", { name: /删除/ })).toBeTruthy();
    // 批量删除可用 → PageTable 内建 rowSelection 出现
    expect(screen.queryAllByRole("checkbox").length).toBeGreaterThan(0);
  });
}, 30000);
