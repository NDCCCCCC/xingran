/**
 * Phase 86 W3（RBAC-02）—— 网络设备页操作级权限门控测试。
 *
 * 本页**独有语义 = 门控键降级解析**（`resolveDeviceActionPerm`，v1.24 窗口期兼容）：
 *   权限集含任一 `network:device:*` 动作级码 ⇒ 按动作码逐动作门控；
 *   否则（sys_menu 只播种页面级 `network:device`）⇒ 退回页面级码，
 *   语义 =「能进入设备管理页面的用户可见本页操作入口」，与拆分前行为一致。
 *
 * 覆盖：
 *   ① 降级路径：只持页面级 `network:device` → 编辑 / 采集端口 / 快速创建 / 手动新增仍可见
 *      （**这是本页最重要的回归防线**：改成硬绑动作码会让所有现役角色失去入口）；
 *   ② 动作码路径：持 `network:device:edit` → 编辑可见；只持 `network:device:view`
 *      （环境已播种动作码但未授予 edit）→ 编辑**不在 DOM**（证明已切到动作码判定）；
 *   ③ 未就绪（permissionsLoaded=false）→ 写入口在 DOM 且 disabled + 提示；
 *   ④ 批量导出 = `<NetworkExport>` 的「导出」按钮（`networkDevice.export`，
 *      85-03 新码在 87 播种前无授权来源 → 按码门控、无码不渲染，与后端 fail-closed 同向）。
 *
 * ⚠️ 断言只用 vitest 原生 matcher + 原生 disabled 属性，不用 jest-dom（理由见 credentials 同名文件）。
 * ⚠️ 本页行操作有 5 个 → ActionButtons threshold=3 收进「操作」下拉，需先点开再断言。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor, cleanup, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { mockApiBatch, setGenericFallback } from "@/test/utils/createApiMock";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import DevicePage from "../index";

const DEVICE_ROW = {
  id: "d1",
  deviceName: "switch-01",
  ipAddress: "10.0.0.1",
  deviceType: "switch",
  status: 0,
  onlineStatus: "online",
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
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <DevicePage />
    </QueryClientProvider>
  );
}

/** 点开行操作「操作」下拉，返回其菜单容器（antd Dropdown 挂 body） */
async function openRowActionsMenu(): Promise<Element> {
  const trigger = screen.getAllByRole("button", { name: "操作" }).at(-1);
  if (!trigger) throw new Error("未找到行操作下拉触发器");
  fireEvent.click(trigger);
  await waitFor(() => {
    expect(document.querySelector(".ant-dropdown")).toBeTruthy();
  });
  const menu = document.querySelector(".ant-dropdown");
  if (!menu) throw new Error("行操作下拉未渲染");
  return menu;
}

describe("网络设备页操作级权限门控（Phase 86 W3）", () => {
  beforeEach(() => {
    mockApiBatch([
      {
        endpoint: "/network/devices/list",
        response: { data: { list: [DEVICE_ROW], total: 1 } },
      },
      {
        endpoint: "/network/devices/statistics",
        response: { data: { totalDevices: 1, onlineDevices: 1 } },
      },
      { endpoint: "/network/credentials/list", response: { data: { list: [], total: 0 } } },
      { endpoint: "/system/departments/tree", response: { data: [] } },
      { endpoint: "/system/dict/data/type/network_device_type", response: { data: [] } },
    ]);
    setGenericFallback({ data: {} });
  });

  // confirmDelete 走命令式 Modal.confirm（独立挂 body），RTL cleanup 不覆盖——手动清残留
  afterEach(() => {
    cleanup();
    document.body
      .querySelectorAll(".ant-modal-root, .ant-modal-mask, .ant-dropdown")
      .forEach((node) => node.remove());
  });

  // ==================== ① 降级路径（页面级码） ====================

  it("① 只持页面级 network:device → 编辑/采集端口/快速创建/手动新增仍可见（降级解析保留）", async () => {
    // ⚠️ 这是降级语义的回归防线：87 播种前多数角色只有页面级码，
    // 若把门控硬绑成动作码，这里会立刻变红（入口消失 = 功能回归）。
    setPermissionState(["network:device"]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    // 工具栏写入口（add 走降级解析 → 页面级码放行）
    expect(screen.getByRole("button", { name: /快速创建/ })).toBeTruthy();
    expect(screen.getByRole("button", { name: /新增设备|手动新增/ })).toBeTruthy();

    // 行操作（edit/delete/采集端口 均走降级解析 → 页面级码放行）
    const menu = await openRowActionsMenu();
    const text = menu.textContent ?? "";
    expect(text).toContain("采集端口");
    expect(text).toContain("编辑");
    expect(text).toContain("删除");
  });

  // ==================== ② 动作码路径 ====================

  it("② 持 network:device:edit（环境已播种动作码）→ 编辑可见、删除不在 DOM", async () => {
    setPermissionState(["network:device", "network:device:edit"]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    const menu = await openRowActionsMenu();
    const text = menu.textContent ?? "";
    expect(text).toContain("编辑");
    // 只授 edit 未授 delete → 删除被动作码门控掉
    expect(text).not.toContain("删除");
  });

  it("②-2 只持 network:device:view（动作码已播种但无 edit）→ 编辑不在 DOM", async () => {
    // 权限集含 `network:device:` 前缀的码 ⇒ 降级解析切到动作码模式；
    // 未授予 edit 动作码 ⇒ 编辑入口消失（这正是 #16 的目标语义：只读用户看不到写入口）。
    setPermissionState(["network:device", "network:device:view"]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    // 只剩无门控的只读「详情」→ visible.length = 1 < threshold(3) → 平铺渲染，
    // 不再收进「操作」下拉（下拉触发器本身也不在 DOM）。
    expect(screen.queryByRole("button", { name: "操作" })).toBeNull();
    expect(screen.queryByRole("button", { name: /编辑/ })).toBeNull();
    // 读入口「详情」保持无门控 → 仍在
    expect(screen.getByRole("button", { name: /详情/ })).toBeTruthy();
  });

  // ==================== ③ 未就绪 ====================

  it("③ 未就绪（loading）→ 快速创建/手动新增在 DOM 且 disabled；行编辑在 DOM 且禁用", async () => {
    setPermissionState([], { loaded: false });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    // 未就绪 → 降级解析退回页面级码，三态门控把按钮渲染成 disabled + 提示（入口不消失）
    const quickBtn = screen.getByRole("button", { name: /快速创建/ }) as HTMLButtonElement;
    expect(quickBtn.disabled).toBe(true);
    expect(quickBtn.getAttribute("title") ?? "").toContain("权限");
    const manualBtn = screen.getByRole("button", { name: /新增设备|手动新增/ }) as HTMLButtonElement;
    expect(manualBtn.disabled).toBe(true);

    // 行操作未就绪 → 入口仍在 DOM（下拉存在），菜单项 disabled
    const menu = await openRowActionsMenu();
    expect(menu.textContent ?? "").toContain("编辑");
    expect(menu.querySelectorAll(".ant-dropdown-menu-item-disabled").length).toBeGreaterThan(0);
  });

  it("③-2 未就绪（error）→ 写入口在 DOM 且 disabled", async () => {
    setPermissionState([], { loaded: false, error: new Error("权限接口失败") });
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });

    const quickBtn = screen.getByRole("button", { name: /快速创建/ }) as HTMLButtonElement;
    expect(quickBtn.disabled).toBe(true);
  });

  // ==================== ④ 批量导出（NetworkExport 组件元素门控） ====================

  it("④ 无 networkDevice.export → 「导出」按钮不在 DOM；持该码 → 存在且可用", async () => {
    // 无导出码（只持页面级码 + edit）
    setPermissionState(["network:device", "network:device:edit"]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });
    // 门控点 = <NetworkExport> 元素本身 → 组件不渲染，其内部「导出」按钮不在 DOM
    expect(screen.queryByRole("button", { name: /导出/ })).toBeNull();

    cleanup();
    document.body.querySelectorAll(".ant-dropdown").forEach((n) => n.remove());

    // 持导出码
    setPermissionState([
      "network:device",
      "network:device:edit",
      PERMISSION_CODES.networkDevice.export,
    ]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("switch-01")).toBeTruthy();
    });
    const exportBtn = screen.getByRole("button", { name: /导出/ }) as HTMLButtonElement;
    expect(exportBtn).toBeTruthy();
    expect(exportBtn.disabled).toBe(false);
  });
}, 30000);
