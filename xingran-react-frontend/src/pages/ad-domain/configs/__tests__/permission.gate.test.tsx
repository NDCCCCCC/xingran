/**
 * Phase 86 W4（RBAC-02）—— AD 配置页（ad-domain/configs）操作级权限门控测试。
 *
 * 覆盖 PLAN 4.3 的四条路径（三路径 + 「ready 空权限集 vs 未就绪」可区分）：
 *   ① 只持 `opsADConfig.list` → 「测试连接」「同步数据」「删除」「新增AD配置」
 *      **均不在 DOM**；
 *   ② 未就绪（permissionsLoaded=false / permissionsError 非空）→ 上述入口
 *      **在 DOM 且 disabled + 提示**（title 含「权限」）；
 *   ③ 持 `.test` → 「测试连接」存在且可用，而「同步数据」**仍不在 DOM**
 *      （逐码断言，不是一揽子）；
 *   ④ ready + **空权限集** → 不渲染（与「未就绪」在数据上可区分）。
 *
 * 另加两条 W4 专项断言：
 *   ⑤ HIGH-1：详情 Drawer 内的「编辑此配置」是**写入口** → 无 `opsADConfig.edit`
 *      时它**不在 DOM**（否则「详情 → 编辑此配置 → 提交」可绕过整页门控）；
 *   ⑥ 表单 Switch 的**随弹窗上下文归所属写动作**口径：只持 `.add`（无 `.edit`）
 *      时，新增弹窗里的三个 Switch 仍可见可用（避免「有 add 无 edit 的用户
 *      新增配置时看不到 SSL/TLS/自动同步字段」的功能退化）。
 *
 * ⚠️ 禁止用 `permissions.length === 0` 判「未就绪」——空权限集与未加载完在数据上不可区分。
 *    本测试用 menuStore 的显式就绪位（permissionsLoaded / permissionsError）构造三态。
 * ⚠️ 断言只用 vitest 原生 matcher（toBeNull / toBe / toContain）+ 原生 disabled 属性，
 *    不用 jest-dom matcher —— `tsconfig.app.json` 排除了测试文件，jest-dom 类型不在
 *    专项类型检查的 types 里，用它会引入类型错误（本文件要求类型错误数 = 0）。
 * ⚠️ 本页行操作有 4 个（测试连接 / 同步数据 / 详情 / 删除）→ ActionButtons 的
 *    threshold=3：可见项 >= 3 时收进「操作」下拉，< 3 时直接平铺。
 *    因此「未就绪」态（4 项全可见）必须先点开「操作」再断言菜单项。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { setGenericFallback } from "@/test/utils/createApiMock";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", () => ({
  createADConfig: vi.fn(),
  updateADConfig: vi.fn(),
  deleteADConfig: vi.fn(),
  testADConnection: vi.fn(),
  syncADData: vi.fn(),
  getADConfigList: vi.fn(() => Promise.resolve({ code: 0, data: { list: [], total: 0 } })),
  // 对账 Tab 的 API（组件读 res.data?.list，返回 undefined 会把空态变成错误态）
  getDeptOUMismatchList: vi.fn(() =>
    Promise.resolve({ data: { list: [], total: 0, current: 1, pageSize: 20 } })
  ),
}));

vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: () => ({ data: [], isLoading: false }),
}));

// useADConfigs 的返回被内联进 mock 工厂（vi.mock 会被提升到模块顶部，
// 不能引用外层 const —— 否则 "Cannot access before initialization"）。
vi.mock("@/hooks/useADConfigs", () => ({
  useADConfigs: () => ({
    configs: [
      {
        id: "cfg-1",
        configName: "公司主AD域",
        serverAddress: "ad.example.com",
        serverPort: 389,
        domainName: "example.com",
        baseDn: "DC=example,DC=com",
        useSsl: true,
        useTls: false,
        syncEnabled: true,
        syncInterval: 3600,
        status: 0,
        lastSyncAt: null,
      },
    ],
    selectedConfig: "cfg-1",
    setSelectedConfig: vi.fn(),
    loading: false,
    fetchConfigs: vi.fn(),
    refreshConfigs: vi.fn(),
    clearSelection: vi.fn(),
  }),
}));

import ADConfigPage from "../index";

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
  return renderWithProviders(<ADConfigPage />, { route: "/ad-domain/configs" });
}

/** 断言某按钮「在 DOM 且 disabled，且 title 带未就绪提示」 */
function expectDisabledWithHint(name: string | RegExp): void {
  const btn = screen.getByRole("button", { name }) as HTMLButtonElement;
  expect(btn).toBeTruthy();
  expect(btn.disabled).toBe(true);
  expect(btn.getAttribute("title") ?? "").toContain("权限");
}

/** 点开行操作「操作」下拉，返回其菜单容器（antd Dropdown 挂 body） */
async function openRowActionsMenu(): Promise<Element> {
  fireEvent.click(screen.getByRole("button", { name: "操作" }));
  await waitFor(() => {
    const menu = document.querySelector(".ant-dropdown");
    expect(menu).toBeTruthy();
  });
  const menu = document.querySelector(".ant-dropdown");
  if (!menu) throw new Error("行操作下拉未渲染");
  return menu;
}

/** 等待配置表格渲染出第一行（行操作挂在该行） */
async function waitForFirstRow(): Promise<void> {
  await waitFor(() => {
    expect(screen.getByText("公司主AD域")).toBeTruthy();
  });
}

describe("AD 配置页操作级权限门控（Phase 86 W4）", () => {
  beforeEach(() => {
    setGenericFallback({ code: 0, data: {} });
  });

  // confirmDelete 走命令式 Modal.confirm（独立挂 body），RTL cleanup 不覆盖——手动清残留
  afterEach(() => {
    cleanup();
    document.body
      .querySelectorAll(".ant-modal-root, .ant-modal-mask, .ant-dropdown, .ant-drawer-root")
      .forEach((node) => node.remove());
  });

  // ==================== ① ready + 只持读码 ====================

  it("① 只持 opsADConfig.list → 测试连接/同步数据/删除/新增配置均不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.opsADConfig.list]);
    renderPage();
    await waitForFirstRow();

    // 工具栏写入口
    expect(screen.queryByText("新增AD配置")).toBeNull();
    // 行操作写入口（详情是读入口 → 保留）
    expect(screen.queryByText("测试连接")).toBeNull();
    expect(screen.queryByText("同步数据")).toBeNull();
    expect(screen.queryByText("删除")).toBeNull();
    expect(screen.getByText("详情")).toBeTruthy();
  });

  // ==================== ④ ready + 空权限集 ====================

  it("④ ready 且空权限集 → 入口不渲染（与「未就绪」可区分）", async () => {
    setPermissionState([], { loaded: true });
    renderPage();
    await waitForFirstRow();

    expect(screen.queryByText("新增AD配置")).toBeNull();
    expect(screen.queryByText("测试连接")).toBeNull();
    expect(screen.queryByText("同步数据")).toBeNull();
    expect(screen.queryByText("删除")).toBeNull();
  });

  // ==================== ② 未就绪（loading / error） ====================

  it("② 未就绪（loading）→ 入口在 DOM 且 disabled + 提示", async () => {
    // permissions 为空但 permissionsLoaded=false —— 这正是「未就绪」态
    setPermissionState([], { loaded: false });
    renderPage();
    await waitForFirstRow();

    // 工具栏「新增AD配置」：页面层条件渲染 + disabled + title 提示
    expectDisabledWithHint(/新增AD配置/);

    // 行操作：未就绪 → 4 项全可见（>= threshold）→ 收进「操作」下拉，菜单项全部 disabled
    const menu = await openRowActionsMenu();
    const text = menu.textContent ?? "";
    expect(text).toContain("测试连接");
    expect(text).toContain("同步数据");
    expect(text).toContain("删除");
    expect(menu.querySelectorAll(".ant-dropdown-menu-item-disabled").length).toBeGreaterThan(0);
    // 未就绪提示挂在菜单项上（title 含「权限」）
    expect(menu.querySelector('[title*="权限"]')).toBeTruthy();
  });

  it("②-2 未就绪（error）→ 入口在 DOM 且 disabled + 提示", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderPage();
    await waitForFirstRow();

    expectDisabledWithHint(/新增AD配置/);
    const menu = await openRowActionsMenu();
    expect(menu.querySelectorAll(".ant-dropdown-menu-item-disabled").length).toBeGreaterThan(0);
    expect(menu.querySelector('[title*="权限"]')).toBeTruthy();
  });

  // ==================== ③ ready + 逐码断言 ====================

  it("③ 持 opsADConfig.test → 「测试连接」存在且可用，而「同步数据」仍不在 DOM", async () => {
    setPermissionState([
      PERMISSION_CODES.opsADConfig.list,
      PERMISSION_CODES.opsADConfig.test,
    ]);
    renderPage();
    await waitForFirstRow();

    // 逐码断言：持 test 不等于持 sync / delete / add
    const testBtn = screen.getByRole("button", { name: /测试连接/ }) as HTMLButtonElement;
    expect(testBtn.disabled).toBe(false);
    expect(screen.queryByText("同步数据")).toBeNull();
    expect(screen.queryByText("删除")).toBeNull();
    expect(screen.queryByText("新增AD配置")).toBeNull();
    // 详情是读入口 → 不受门控影响
    expect(screen.getByText("详情")).toBeTruthy();
  });

  // ==================== ⑤ HIGH-1：详情 Drawer 的「编辑此配置」 ====================

  it("⑤ 详情 Drawer 的「编辑此配置」是写入口 → 无 opsADConfig.edit 时不在 DOM", async () => {
    setPermissionState([PERMISSION_CODES.opsADConfig.list]);
    renderPage();
    await waitForFirstRow();

    fireEvent.click(screen.getByRole("button", { name: /详情/ }));
    // 等详情抽屉内容真正渲染出来（用抽屉标题作锚点，
    // 避免「抽屉根本没打开」被误判成「按钮被门控掉」）
    await waitFor(() => {
      expect(screen.queryByText("AD 配置详情: 公司主AD域")).not.toBeNull();
    });

    // 无 edit 权限 → 详情抽屉里的「编辑此配置」必须不渲染（否则可绕过整页门控提交）
    expect(screen.queryByText("编辑此配置")).toBeNull();
  });

  it("⑤-2 持 opsADConfig.edit → 「编辑此配置」存在且可用", async () => {
    setPermissionState([
      PERMISSION_CODES.opsADConfig.list,
      PERMISSION_CODES.opsADConfig.edit,
    ]);
    renderPage();
    await waitForFirstRow();

    fireEvent.click(screen.getByRole("button", { name: /详情/ }));
    await waitFor(() => {
      expect(screen.queryByText("编辑此配置")).not.toBeNull();
    });

    const editBtn = screen.getByRole("button", { name: /编辑此配置/ }) as HTMLButtonElement;
    expect(editBtn.disabled).toBe(false);
  });

  // ==================== ⑥ 表单 Switch 随弹窗上下文归所属写动作 ====================

  it("⑥ 只持 opsADConfig.add → 新增弹窗的三个 Switch 仍可见可用（不随 edit 归码）", async () => {
    setPermissionState([
      PERMISSION_CODES.opsADConfig.list,
      PERMISSION_CODES.opsADConfig.add,
    ]);
    renderPage();
    await waitForFirstRow();

    fireEvent.click(screen.getByRole("button", { name: /新增AD配置/ }));
    await waitFor(() => {
      expect(screen.queryByText("使用SSL (LDAPS)")).not.toBeNull();
    });

    // 新增弹窗上下文 → 归 opsADConfig.add → 三个 Switch 全部可见
    expect(screen.getByText("使用SSL (LDAPS)")).toBeTruthy();
    expect(screen.getByText("使用TLS (StartTLS)")).toBeTruthy();
    expect(screen.getByText("启用自动同步")).toBeTruthy();

    // 且未被禁用（有 add 权限 → 表单字段可用）
    const switches = document.querySelectorAll(".ant-modal .ant-switch");
    expect(switches.length).toBeGreaterThanOrEqual(3);
    switches.forEach((sw) => {
      expect(sw.classList.contains("ant-switch-disabled")).toBe(false);
    });
  });
}, 30000);
