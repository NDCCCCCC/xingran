/**
 * 工具类 hooks 组合测试
 *
 * 覆盖:useCaptcha / useImageUpload / useRoleList / useSidebarDeptFilter /
 * useTabSync / useWallDrawing / useWindowSize。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const mocks = vi.hoisted(() => ({
  getCaptcha: vi.fn(),
  getCaptchaConfig: vi.fn(),
  apiPost: vi.fn(),
  message: { success: vi.fn(), error: vi.fn() },
}));

vi.mock("@/services/captcha", () => ({
  getCaptcha: mocks.getCaptcha,
  getCaptchaConfig: mocks.getCaptchaConfig,
}));

vi.mock("@/lib/api", () => ({
  post: mocks.apiPost,
}));

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  const App = Object.assign(actual.App, {
    useApp: () => ({ message: mocks.message }),
  });
  return { ...actual, App };
});

import { useCaptcha } from "./useCaptcha";
import { useImageUpload } from "./useImageUpload";
import { useRoleList } from "./useRoleList";
import { useSidebarDeptFilter } from "./useSidebarDeptFilter";
import { useTabSync } from "./useTabSync";
import { useWallDrawing } from "./useWallDrawing";
import { useWindowSize } from "./useWindowSize";
import { useTabsStore } from "@/store/tabsStore";
import { useDashboardStore } from "@/store/dashboardStore";
import type { CaptchaConfig, CaptchaResponse } from "@/types/captcha";

const fakeCaptchaConfig = (enabled: string): CaptchaConfig =>
  ({ enabled }) as unknown as CaptchaConfig;

const fakeCaptcha: CaptchaResponse = {
  captchaId: "cap-1",
} as unknown as CaptchaResponse;

describe("useCaptcha", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    mocks.getCaptcha.mockReset();
    mocks.getCaptchaConfig.mockReset();
    mocks.getCaptchaConfig.mockResolvedValue(fakeCaptchaConfig("slider"));
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
  });

  it("挂载加载配置,derive isEnabled/captchaType", async () => {
    const { result } = renderHook(() => useCaptcha());
    await waitFor(() => expect(result.current.config).not.toBeNull());
    expect(result.current.isEnabled).toBe(true);
    expect(result.current.captchaType).toBe("slider");
  });

  it("disabled 配置 → isEnabled=false", async () => {
    mocks.getCaptchaConfig.mockResolvedValue(fakeCaptchaConfig("disabled"));
    const { result } = renderHook(() => useCaptcha());
    await waitFor(() => expect(result.current.config).not.toBeNull());
    expect(result.current.isEnabled).toBe(false);
    expect(result.current.captchaType).toBe("disabled");
  });

  it("配置加载失败走 error 分支", async () => {
    mocks.getCaptchaConfig.mockRejectedValue(new Error("cfg fail"));
    const { result } = renderHook(() => useCaptcha());
    await waitFor(() => expect(consoleSpy).toHaveBeenCalled());
    expect(result.current.config).toBeNull();
  });

  it("loadCaptcha 写入数据;失败向上抛出并复位 loading", async () => {
    mocks.getCaptcha.mockResolvedValue(fakeCaptcha);
    const { result } = renderHook(() => useCaptcha());

    await act(async () => {
      await result.current.loadCaptcha();
    });
    expect(result.current.captchaData).toEqual(fakeCaptcha);
    expect(result.current.loading).toBe(false);

    mocks.getCaptcha.mockRejectedValue(new Error("captcha fail"));
    await act(async () => {
      await expect(result.current.loadCaptcha()).rejects.toThrow("captcha fail");
    });
    expect(result.current.loading).toBe(false);
  });

  it("verifyCaptcha 恒 true(滑动验证在组件内完成)", async () => {
    const { result } = renderHook(() => useCaptcha());
    await act(async () => {
      expect(await result.current.verifyCaptcha("any")).toBe(true);
    });
  });
});

describe("useImageUpload", () => {
  beforeEach(() => {
    mocks.message.success.mockClear();
    mocks.message.error.mockClear();
  });

  it("handleUploadSuccess 写入 imageId/imageUrl 并回调 onSuccess", async () => {
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useImageUpload({ onSuccess }));

    await act(async () => {
      result.current.handleUploadSuccess(
        { uid: "1" } as never,
        {
          id: "file-1",
          storagePath: "abc.png",
        } as never
      );
    });

    expect(result.current.imageId).toBe("file-1");
    expect(result.current.imageUrl).toContain("uploads/abc.png");
    expect(mocks.message.success).toHaveBeenCalledWith("图片上传成功");
    expect(onSuccess).toHaveBeenCalledWith("file-1", expect.stringContaining("uploads/abc.png"));
    expect(result.current.uploading).toBe(false);
  });

  it("handleUploadError 走 message.error + onError(Error 包装)", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageUpload({ onError }));

    act(() => {
      result.current.handleUploadError({ uid: "1" } as never, "plain failure");
    });
    expect(mocks.message.error).toHaveBeenCalledWith("图片上传失败");
    expect(onError).toHaveBeenCalledWith(expect.any(Error));
    expect(onError.mock.calls[0][0].message).toBe("plain failure");
  });

  it("handleUploadChange 空列表清空 id/url;resetUpload 全量复位", () => {
    const { result } = renderHook(() => useImageUpload());

    act(() => result.current.handleUploadChange([{ uid: "1" } as never]));
    expect(result.current.fileList).toHaveLength(1);

    act(() => result.current.handleUploadChange([]));
    expect(result.current.fileList).toEqual([]);
    expect(result.current.imageId).toBeUndefined();
    expect(result.current.imageUrl).toBeUndefined();

    act(() => result.current.setInitialValue("f2", "/uploads/x.png"));
    expect(result.current.imageId).toBe("f2");
    act(() => result.current.resetUpload());
    expect(result.current.imageId).toBeUndefined();
    expect(result.current.fileList).toEqual([]);
  });

  it("setInitialValue 无 fileUrl 时清空列表;空 id 直接短路", () => {
    const { result } = renderHook(() => useImageUpload());

    act(() => result.current.setInitialValue("f3", "/uploads/pic.png"));
    expect(result.current.fileList[0]).toMatchObject({
      uid: "f3",
      status: "done",
      url: "/uploads/pic.png",
    });
    expect(result.current.fileList[0].response).toEqual({
      id: "f3",
      storagePath: "pic.png",
    });

    act(() => result.current.setInitialValue("f4"));
    expect(result.current.imageId).toBe("f4");
    expect(result.current.fileList).toEqual([]);

    act(() => result.current.setInitialValue(""));
    expect(result.current.imageId).toBe("f4"); // 空串短路
  });
});

describe("useRoleList", () => {
  beforeEach(() => {
    mocks.apiPost.mockReset();
  });

  it("拉取角色列表(pageSize 1000)并解包", async () => {
    mocks.apiPost.mockResolvedValue({
      data: {
        list: [{ roleId: "r1", roleName: "管理员", roleKey: "admin", status: 0 }],
        total: 1,
      },
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const { result } = renderHook(() => useRoleList(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.apiPost).toHaveBeenCalledWith("/system/roles/list", {
      current: 1,
      pageSize: 1000,
    });
    expect(result.current.data).toHaveLength(1);
  });

  it("响应缺 list 回退空数组", async () => {
    mocks.apiPost.mockResolvedValue({ data: null });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const { result } = renderHook(() => useRoleList(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([]);
  });
});

describe("useSidebarDeptFilter", () => {
  it("默认无选中;handleDeptSelect 更新并回调", () => {
    const onDeptChange = vi.fn();
    const { result } = renderHook(() => useSidebarDeptFilter({ onDeptChange }));

    expect(result.current.selectedDeptId).toBe("");

    act(() => {
      result.current.handleDeptSelect(["dept-1"], { selected: true, node: {} as never });
    });
    expect(result.current.selectedDeptId).toBe("dept-1");
    expect(onDeptChange).toHaveBeenCalledWith("dept-1");

    // 空选中 → 回退空串
    act(() => {
      result.current.handleDeptSelect([], { selected: false, node: {} as never });
    });
    expect(result.current.selectedDeptId).toBe("");
  });

  it("配置 searchForm 时清空指定字段", () => {
    const setFieldValue = vi.fn();
    const { result } = renderHook(() =>
      useSidebarDeptFilter({
        searchForm: { setFieldValue } as never,
        clearFieldNames: ["username", "status"],
      })
    );

    act(() => {
      result.current.handleDeptSelect(["d1"], { selected: true, node: {} as never });
    });
    expect(setFieldValue).toHaveBeenCalledWith("username", undefined);
    expect(setFieldValue).toHaveBeenCalledWith("status", undefined);

    // setSelectedDeptId 直接暴露
    act(() => result.current.setSelectedDeptId("d2"));
    expect(result.current.selectedDeptId).toBe("d2");
  });
});

describe("useWallDrawing", () => {
  it("startDrawing 网格吸附并回调 onPointsChange", () => {
    const onPointsChange = vi.fn();
    const { result } = renderHook(() => useWallDrawing({ gridSize: 20, onPointsChange }));

    act(() => result.current.startDrawing({ x: 11, y: 9 }));
    expect(result.current.isDrawing).toBe(true);
    expect(result.current.drawPoints[0]).toEqual({ x: 20, y: 0 }); // 吸附到 20 网格
    expect(onPointsChange).toHaveBeenCalledWith([{ x: 20, y: 0 }]);
  });

  it("addPoint 距离过近(<5px)被忽略,足够远则追加", () => {
    const onPointsChange = vi.fn();
    const { result } = renderHook(() => useWallDrawing({ onPointsChange }));

    act(() => result.current.startDrawing({ x: 0, y: 0 }));
    act(() => result.current.addPoint({ x: 1, y: 1 })); // 距离不足
    expect(result.current.drawPoints).toHaveLength(1);

    act(() => result.current.addPoint({ x: 40, y: 0 }));
    expect(result.current.drawPoints).toHaveLength(2);
    expect(onPointsChange).toHaveBeenLastCalledWith([
      { x: 0, y: 0 },
      { x: 40, y: 0 },
    ]);
  });

  it("未开始绘制时 addPoint/updatePreview 短路", () => {
    const { result } = renderHook(() => useWallDrawing());
    act(() => result.current.addPoint({ x: 100, y: 100 }));
    act(() => result.current.updatePreview({ x: 100, y: 100 }));
    expect(result.current.drawPoints).toEqual([]);
    expect(result.current.previewPoint).toBeNull();
  });

  it("updatePreview 更新预览点,shiftKey 禁用角度吸附", () => {
    const { result } = renderHook(() =>
      useWallDrawing({ snapEnabled: false, angleSnapEnabled: true })
    );

    act(() => result.current.startDrawing({ x: 0, y: 0 }));
    act(() => result.current.updatePreview({ x: 33, y: 4 })); // 角度吸附到 0°,距离保持
    const snapped = result.current.previewPoint!;
    expect(snapped.y).toBeCloseTo(0, 5);
    expect(snapped.x).toBeCloseTo(Math.sqrt(33 * 33 + 4 * 4), 5);

    act(() => result.current.updatePreview({ x: 10, y: 10 }, true)); // shift 置禁用(本次仍用旧值)
    // 45° 方向吸附后与原点几乎重合(浮点误差),再调用一次才是真正禁用后的原始点
    act(() => result.current.updatePreview({ x: 33, y: 7 }, true));
    expect(result.current.previewPoint).toEqual({ x: 33, y: 7 });
  });

  it("finishDrawing:两点成直线/多点成折线/单点取消", () => {
    const onComplete = vi.fn();
    const onPointsChange = vi.fn();
    const { result } = renderHook(() =>
      useWallDrawing({ snapEnabled: false, onComplete, onPointsChange })
    );

    // 单点 → 取消,onComplete 不触发
    act(() => result.current.startDrawing({ x: 0, y: 0 }));
    act(() => result.current.finishDrawing());
    expect(onComplete).not.toHaveBeenCalled();
    expect(result.current.isDrawing).toBe(false);
    expect(onPointsChange).toHaveBeenLastCalledWith([]);

    // 两点 → straight
    act(() => result.current.startDrawing({ x: 0, y: 0 }));
    act(() => result.current.addPoint({ x: 40, y: 0 }));
    act(() => result.current.finishDrawing());
    expect(onComplete).toHaveBeenLastCalledWith({
      points: [
        { x: 0, y: 0 },
        { x: 40, y: 0 },
      ],
      type: "straight",
    });

    // 三点 → polyline
    act(() => result.current.startDrawing({ x: 0, y: 0 }));
    act(() => result.current.addPoint({ x: 40, y: 0 }));
    act(() => result.current.addPoint({ x: 40, y: 40 }));
    act(() => result.current.finishDrawing());
    expect(onComplete).toHaveBeenLastCalledWith({
      points: [
        { x: 0, y: 0 },
        { x: 40, y: 0 },
        { x: 40, y: 40 },
      ],
      type: "polyline",
    });
  });

  it("startDrawing 续接已有顶点;cancelDrawing 清空;undoPoint 回退", () => {
    const onPointsChange = vi.fn();
    const { result } = renderHook(() => useWallDrawing({ snapEnabled: false, onPointsChange }));

    act(() =>
      result.current.startDrawing({ x: 0, y: 0 }, [
        { x: -40, y: 0 },
        { x: 0, y: 0 },
      ])
    );
    expect(result.current.drawPoints).toEqual([
      { x: -40, y: 0 },
      { x: 0, y: 0 },
    ]);

    act(() => result.current.addPoint({ x: 40, y: 0 }));
    act(() => result.current.undoPoint());
    expect(result.current.drawPoints).toEqual([
      { x: -40, y: 0 },
      { x: 0, y: 0 },
    ]);

    // 回退到只剩一个点再 undo → 取消绘制
    act(() => result.current.undoPoint());
    expect(result.current.drawPoints).toEqual([{ x: -40, y: 0 }]);
    act(() => result.current.undoPoint());
    expect(result.current.isDrawing).toBe(false);

    act(() => result.current.startDrawing({ x: 0, y: 0 }));
    act(() => result.current.cancelDrawing());
    expect(result.current.isDrawing).toBe(false);
    expect(result.current.drawPoints).toEqual([]);
  });
});

describe("useWindowSize", () => {
  it("初始读 window 尺寸,resize 事件更新", () => {
    const initial = { width: window.innerWidth, height: window.innerHeight };
    const { result } = renderHook(() => useWindowSize());
    expect(result.current).toEqual(initial);

    act(() => {
      Object.defineProperty(window, "innerWidth", { value: 1280, configurable: true });
      Object.defineProperty(window, "innerHeight", { value: 720, configurable: true });
      window.dispatchEvent(new Event("resize"));
    });
    expect(result.current).toEqual({ width: 1280, height: 720 });
  });
});

describe("useTabSync", () => {
  const routerWrapper = ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={["/system/user"]}>{children}</MemoryRouter>
  );

  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
    useTabsStore.setState({ tabs: [], activeTab: "", history: [] });
    useDashboardStore.setState({ currentDashboard: null, widgetDataCache: new Map() });
  });

  afterEach(() => {
    useTabsStore.setState({ tabs: [], activeTab: "", history: [] });
    useDashboardStore.setState({ currentDashboard: null, widgetDataCache: new Map() });
  });

  it("非 dashboard 路由:创建标签(fallback 标题=最后一段),/login 不创建", () => {
    const { rerender } = renderHook(({ path }) => useTabSync(path), {
      initialProps: { path: "/system/user" },
      wrapper: routerWrapper,
    });

    // Effect1 建 user 标签 + Effect2 兜底建固定 dashboard 标签(addTab 会激活新标签)
    const keys = useTabsStore.getState().tabs.map((t) => t.key);
    expect(keys).toEqual(expect.arrayContaining(["/system/user", "/dashboard"]));
    expect(useTabsStore.getState().activeTab).toBe("/dashboard");
    const userTab = useTabsStore.getState().tabs.find((t) => t.key === "/system/user")!;
    expect(userTab.title).toBe("user");

    rerender({ path: "/login" });
    expect(useTabsStore.getState().tabs).toHaveLength(2); // login 不加 tab
  });

  it("dashboard 路由:固定标签 + 激活;/dashboard 首页标题恒为仪表盘", () => {
    renderHook(() => useTabSync("/dashboard"), { wrapper: routerWrapper });

    const state = useTabsStore.getState();
    expect(state.tabs).toHaveLength(1);
    expect(state.tabs[0]).toMatchObject({
      key: "/dashboard",
      title: "仪表盘",
      pinned: true,
      closable: false,
    });
    expect(state.activeTab).toBe("/dashboard");
  });

  it("dashboard 子路由标题跟随 currentDashboard.name", () => {
    useDashboardStore.setState({
      currentDashboard: {
        id: "d1",
        name: "自定义大盘",
      } as never,
    });
    renderHook(() => useTabSync("/dashboard/d1"), { wrapper: routerWrapper });

    const tab = useTabsStore.getState().tabs.find((t) => t.key === "/dashboard");
    expect(tab?.title).toBe("自定义大盘");
  });

  it("Effect2 兜底:无 dashboard 标签时创建固定标签", () => {
    renderHook(() => useTabSync("/system/role"), { wrapper: routerWrapper });
    // Effect2 在挂载时确保 dashboard 标签存在
    const dash = useTabsStore.getState().tabs.find((t) => t.key === "/dashboard");
    expect(dash).toMatchObject({ pinned: true, closable: false });
  });

  it("已有 dashboard 标签状态破损时强制修复为固定", () => {
    useTabsStore.setState({
      tabs: [
        {
          key: "/dashboard",
          title: "仪表盘",
          path: "/dashboard",
          closable: true,
          pinned: false,
        },
      ],
      activeTab: "",
      history: [],
    });
    renderHook(() => useTabSync("/system/role"), { wrapper: routerWrapper });

    const dash = useTabsStore.getState().tabs.find((t) => t.key === "/dashboard");
    expect(dash).toMatchObject({ pinned: true, closable: false });
  });
});
