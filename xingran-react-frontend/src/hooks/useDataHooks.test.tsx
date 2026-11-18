/**
 * 数据类 hooks 组合测试
 *
 * 覆盖:useADConfigs / useAliasByLocation / useDashboard / useDeptTree /
 * useDict / useExceptionList / useReconciliationWebSocket / useWidgetPolling。
 * React Query hooks 用 QueryClientProvider wrapper(按需注入,D-05);
 * API 依赖整模块 vi.mock(D-07 业务轨道)。
 */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const mocks = vi.hoisted(() => ({
  getADConfigList: vi.fn(),
  deptOptions: vi.fn(),
  reconciliation: {
    summary: vi.fn(),
    byConflictType: vi.fn(),
    bySeverity: vi.fn(),
    healthTrend: vi.fn(),
    topUnresolved: vi.fn(),
    exceptionList: vi.fn(),
    exceptionRuleStats: vi.fn(),
  },
  getDeptTree: vi.fn(),
  apiPost: vi.fn(),
  // v1.54：useDeptTree 按当前登录人隔离缓存，测试需可控切换 userID
  currentUserID: "user-a" as string | null,
  getBatchWidgetData: vi.fn(),
  buildWebSocketUrl: vi.fn(() => "ws://test/ws/notices"),
}));

vi.mock("@/lib/adDomainApi", () => ({
  getADConfigList: mocks.getADConfigList,
}));

// v1.54：useDeptTree 从 authStore 读当前用户 ID 作为 queryKey 维度。
vi.mock("@/store/authStore", () => ({
  useAuthStore: (selector: (s: { user: { id: string } | null }) => unknown) =>
    selector({ user: mocks.currentUserID ? { id: mocks.currentUserID } : null }),
}));

vi.mock("@/lib/opsApi", () => ({
  workstationApi: { deptOptions: mocks.deptOptions },
}));

vi.mock("@/lib/assetApi", () => ({
  reconciliationApi: mocks.reconciliation,
}));

vi.mock("@/lib/dutyApi", () => ({
  getDeptTree: mocks.getDeptTree,
}));

vi.mock("@/lib/api", () => ({
  post: mocks.apiPost,
}));

vi.mock("@/services/dashboardService", () => ({
  dashboardService: { getBatchWidgetData: mocks.getBatchWidgetData },
}));

vi.mock("@/lib/noticeApi", () => ({
  buildWebSocketUrl: mocks.buildWebSocketUrl,
}));

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  const App = Object.assign(actual.App, {
    useApp: () => ({ message: { error: vi.fn(), success: vi.fn() } }),
  });
  return { ...actual, App };
});

import { useADConfigs } from "./useADConfigs";
import { useAliasByLocation } from "./useAliasByLocation";
import { useDashboard, useExceptionRuleStats } from "./useDashboard";
import { useDeptTree, useInvalidateDept } from "./useDeptTree";
import { useDict } from "./useDict";
import { useExceptionList } from "./useExceptionList";
import { useReconciliationWebSocket } from "./useReconciliationWebSocket";
import { useWidgetPolling } from "./useWidgetPolling";
import { useDashboardStore } from "@/store/dashboardStore";
import type { ADConfig } from "@/lib/adDomainApi";

/** 每个测试新建独立 QueryClient,避免缓存串扰 */
function createQueryWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { wrapper, queryClient };
}

const adConfig = (id: string): ADConfig => ({
  id,
  configName: `cfg-${id}`,
  serverAddress: "ldap://x",
  serverPort: 389,
  domainName: "x.com",
  baseDn: "dc=x",
  useSsl: false,
  useTls: false,
  syncEnabled: false,
  syncInterval: 0,
  status: 0,
  createdAt: "",
  createdBy: "",
});

describe("useADConfigs", () => {
  beforeEach(() => {
    mocks.getADConfigList.mockReset();
  });

  it("挂载即拉取启用配置并自动选中第一个", async () => {
    mocks.getADConfigList.mockResolvedValue({
      code: 0,
      data: { list: [adConfig("a1"), adConfig("a2")], total: 2 },
    });

    const { result } = renderHook(() => useADConfigs());

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(mocks.getADConfigList).toHaveBeenCalledWith({
      current: 1,
      pageSize: 100,
      status: 0,
    });
    expect(result.current.configs).toHaveLength(2);
    expect(result.current.selectedConfig).toBe("a1");
  });

  it("enabledOnly=false 不带 status 参数;带排序参数", async () => {
    mocks.getADConfigList.mockResolvedValue({
      code: 0,
      data: { list: [adConfig("a1")], total: 1 },
    });
    const { result } = renderHook(() =>
      useADConfigs({ enabledOnly: false, autoSelectFirst: false })
    );

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(mocks.getADConfigList).toHaveBeenCalledWith({
      current: 1,
      pageSize: 100,
    });
    expect(result.current.selectedConfig).toBe("");

    await act(async () => {
      result.current.fetchConfigs("configName", false);
    });
    expect(mocks.getADConfigList).toHaveBeenLastCalledWith({
      current: 1,
      pageSize: 100,
      orderByColumn: "configName",
      isAsc: false,
    });
  });

  it("接口异常时 message.error 且 loading 复位", async () => {
    mocks.getADConfigList.mockRejectedValue(new Error("net"));

    const { result } = renderHook(() => useADConfigs());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.configs).toEqual([]);
  });

  it("refreshConfigs / setSelectedConfig / clearSelection", async () => {
    mocks.getADConfigList.mockResolvedValue({
      code: 0,
      data: { list: [adConfig("a1")], total: 1 },
    });
    const { result } = renderHook(() => useADConfigs());
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.setSelectedConfig("manual"));
    expect(result.current.selectedConfig).toBe("manual");

    act(() => result.current.clearSelection());
    expect(result.current.selectedConfig).toBe("");

    await act(async () => {
      result.current.refreshConfigs();
    });
    expect(mocks.getADConfigList).toHaveBeenCalledTimes(2);
  });

  it("code!=0 或空列表不写 configs", async () => {
    mocks.getADConfigList.mockResolvedValue({ code: 500, data: null });
    const { result } = renderHook(() => useADConfigs());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.configs).toEqual([]);
  });
});

describe("useAliasByLocation", () => {
  beforeEach(() => {
    mocks.deptOptions.mockReset();
  });

  it("locationId 为空时 enabled=false 不发请求,返回空数组", async () => {
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useAliasByLocation(undefined), { wrapper });

    // enabled:false → queryFn 不执行
    expect(mocks.deptOptions).not.toHaveBeenCalled();
    expect(result.current.isEnabled).toBe(false);
  });

  it("有 locationId 时拉取部门映射列表", async () => {
    mocks.deptOptions.mockResolvedValue([{ deptId: "d1", deptName: "映射部门", isAlias: true }]);
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useAliasByLocation("loc-1"), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.deptOptions).toHaveBeenCalledWith("loc-1");
    expect(result.current.data).toEqual([{ deptId: "d1", deptName: "映射部门", isAlias: true }]);
  });
});

describe("useDashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.reconciliation.summary.mockResolvedValue({ total: 1 });
    mocks.reconciliation.byConflictType.mockResolvedValue({ typeA: 1 });
    mocks.reconciliation.bySeverity.mockResolvedValue({ critical: 1 });
    mocks.reconciliation.healthTrend.mockResolvedValue([{ date: "2026-08-01", score: 90 }]);
    mocks.reconciliation.topUnresolved.mockResolvedValue([]);
  });

  it("5 个查询并行加载并聚合 isLoading/isError", async () => {
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useDashboard(7), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.isError).toBe(false);
    expect(result.current.summary.data).toEqual({ total: 1 });
    expect(result.current.byConflictType.data).toEqual({ typeA: 1 });
    expect(result.current.bySeverity.data).toEqual({ critical: 1 });
    expect(result.current.healthTrend.data).toHaveLength(1);
    expect(mocks.reconciliation.topUnresolved).toHaveBeenCalledWith(10);
  });

  it("单个查询失败 → isError 聚合为 true", async () => {
    mocks.reconciliation.summary.mockRejectedValue(new Error("boom"));
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useDashboard(), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });

  it("useExceptionRuleStats 拉取规则命中统计", async () => {
    mocks.reconciliation.exceptionRuleStats.mockResolvedValue([]);
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useExceptionRuleStats(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.reconciliation.exceptionRuleStats).toHaveBeenCalled();
  });
});

describe("useDeptTree", () => {
  beforeEach(() => {
    // v1.54 起 useDeptTree 直接走 @/lib/api 的 post（scoped 收口参数由 hook 内部
    // 组装），不再经 dutyApi.getDeptTree —— mock 点从 getDeptTree 换成 apiPost。
    mocks.apiPost.mockReset();
  });

  it("拉取部门树并解包 data 字段", async () => {
    mocks.apiPost.mockResolvedValue({
      code: 0,
      data: [{ deptId: "1", deptName: "总部", children: [] }],
    });
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useDeptTree(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([{ deptId: "1", deptName: "总部", children: [] }]);
    // 非 scoped 调用（默认）不带收口参数
    expect(mocks.apiPost).toHaveBeenCalledWith("/system/departments/tree", {});
  });

  it("响应缺 data 时回退空数组", async () => {
    mocks.apiPost.mockResolvedValue({ code: 0 });
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useDeptTree(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([]);
  });

  it("scoped: true 时请求体携带收口参数且走独立缓存键", async () => {
    mocks.apiPost.mockResolvedValue({
      code: 0,
      data: [{ deptId: "2", deptName: "本机构", children: [] }],
    });
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useDeptTree({ scoped: true }), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([{ deptId: "2", deptName: "本机构", children: [] }]);
    expect(mocks.apiPost).toHaveBeenCalledWith("/system/departments/tree", { scoped: true });
  });

  it("切换登录用户后 scoped 部门树缓存隔离（不复用上一账号的全量树）", async () => {
    // 场景：超管(user-a)先看全量树 → 退出换机构管理员(user-b)登录。
    // queryKey 必须带 userID，否则 user-b 命中 user-a 的缓存条目看到全量。
    mocks.currentUserID = "user-a";
    mocks.apiPost.mockResolvedValue({
      code: 0,
      data: [{ deptId: "1", deptName: "总部（全量）", children: [] }],
    });
    const { wrapper, queryClient } = createQueryWrapper();
    const first = renderHook(() => useDeptTree({ scoped: true }), { wrapper });
    await waitFor(() => expect(first.result.current.isSuccess).toBe(true));
    expect(first.result.current.data?.[0].deptName).toBe("总部（全量）");

    // 切换用户 + 换返回值（模拟服务端按新用户收口）
    mocks.currentUserID = "user-b";
    mocks.apiPost.mockResolvedValue({
      code: 0,
      data: [{ deptId: "2", deptName: "武汉中支（收口）", children: [] }],
    });
    const second = renderHook(() => useDeptTree({ scoped: true }), { wrapper });
    await waitFor(() => expect(second.result.current.data?.[0]?.deptName).toBe("武汉中支（收口）"));

    // 两个账号的缓存条目并存，互不覆盖
    const keys = queryClient
      .getQueryCache()
      .getAll()
      .map((q) => JSON.stringify(q.queryKey));
    expect(keys.some((k) => k.includes("user-a"))).toBe(true);
    expect(keys.some((k) => k.includes("user-b"))).toBe(true);
    mocks.currentUserID = "user-a";
  });

  it("useInvalidateDept 失效 dept 前缀查询", async () => {
    const { wrapper, queryClient } = createQueryWrapper();
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useInvalidateDept(), { wrapper });

    await act(async () => {
      await result.current();
    });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["dept"] });
  });
});

describe("useDict", () => {
  beforeEach(() => {
    mocks.apiPost.mockReset();
  });

  it("按 dictType 拉取字典数据(pageSize 1000)并解包 list", async () => {
    mocks.apiPost.mockResolvedValue({
      data: { list: [{ id: "1", dictLabel: "男", dictValue: "0" }], total: 1 },
    });
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useDict("sys_user_sex"), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocks.apiPost).toHaveBeenCalledWith("/system/dicts/data/list", {
      dictType: "sys_user_sex",
      current: 1,
      pageSize: 1000,
    });
    expect(result.current.data).toHaveLength(1);
  });

  it("响应缺 list 回退空数组", async () => {
    mocks.apiPost.mockResolvedValue({ data: null });
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useDict("empty_type"), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([]);
  });
});

describe("useExceptionList", () => {
  beforeEach(() => {
    mocks.reconciliation.exceptionList.mockReset();
  });

  it("分页查询异常列表", async () => {
    mocks.reconciliation.exceptionList.mockResolvedValue({
      list: [{ exceptionId: "e1" }],
      total: 1,
      current: 1,
      pageSize: 20,
    });
    const { wrapper } = createQueryWrapper();
    const params = { current: 1, pageSize: 20 };
    const { result } = renderHook(() => useExceptionList(params), { wrapper });

    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.data?.list).toHaveLength(1);
    expect(result.current.isError).toBe(false);
  });

  it("查询失败 → isError", async () => {
    mocks.reconciliation.exceptionList.mockRejectedValue(new Error("bad"));
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useExceptionList({ current: 1, pageSize: 20 }), {
      wrapper,
    });

    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});

describe("useReconciliationWebSocket", () => {
  it("enabled=true 构造 WS URL 并返回初始 disconnected 状态", () => {
    const queryClient = new QueryClient();
    const onCriticalEvent = vi.fn();
    const { result } = renderHook(() =>
      useReconciliationWebSocket({ queryClient, onCriticalEvent })
    );

    expect(mocks.buildWebSocketUrl).toHaveBeenCalled();
    expect(result.current.status).toBe("disconnected");
  });

  it("enabled=false 时空 URL 不构造连接", () => {
    const queryClient = new QueryClient();
    const { result } = renderHook(() =>
      useReconciliationWebSocket({ queryClient, enabled: false })
    );

    expect(result.current.status).toBe("disconnected");
    act(() => result.current.disconnect());
    expect(result.current.status).toBe("disconnected");
  });
});

describe("useWidgetPolling", () => {
  beforeEach(() => {
    mocks.getBatchWidgetData.mockReset();
    useDashboardStore.setState({ widgetDataCache: new Map() });
  });

  it("挂载即拉取未缓存 widget 并写回缓存", async () => {
    mocks.getBatchWidgetData.mockResolvedValue(new Map([["w1", { value: 1 }]]));
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }), {
      wrapper,
    });

    await waitFor(() => expect(mocks.getBatchWidgetData).toHaveBeenCalledWith(["w1"]));
    await waitFor(() => expect(result.current.lastRefreshTime).not.toBeNull());
    expect(useDashboardStore.getState().getCachedWidgetData("w1")).toEqual({
      value: 1,
    });
    expect(result.current.loading).toBe(false);
    expect(result.current.isPaused).toBe(false);
  });

  it("空 widgetIds 不触发请求", async () => {
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: [], interval: 60 }), {
      wrapper,
    });
    expect(mocks.getBatchWidgetData).not.toHaveBeenCalled();
    expect(result.current.loading).toBe(false);
  });

  it("pause/resume 切换 isPaused", async () => {
    mocks.getBatchWidgetData.mockResolvedValue(new Map());
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }), {
      wrapper,
    });

    act(() => result.current.pause());
    expect(result.current.isPaused).toBe(true);
    act(() => result.current.resume());
    expect(result.current.isPaused).toBe(false);
  });

  it("refresh 触发重拉,失败不抛错", async () => {
    mocks.getBatchWidgetData.mockResolvedValue(new Map([["w1", { v: 1 }]]));
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }), {
      wrapper,
    });
    await waitFor(() => expect(mocks.getBatchWidgetData).toHaveBeenCalledWith(["w1"]));

    mocks.getBatchWidgetData.mockClear();
    mocks.getBatchWidgetData.mockRejectedValue(new Error("ws down"));
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      await act(async () => {
        await result.current.refresh();
      });
      expect(mocks.getBatchWidgetData).toHaveBeenCalledWith(["w1"]);
      // 等待 refetch 错误完成:React Query refetch 期间 isFetching 短暂为 true,
      // 错误处理完后 isFetching=false,loading 回归 false
      await waitFor(() => expect(result.current.loading).toBe(false));
    } finally {
      consoleSpy.mockRestore();
    }
  });

  it("widgetIds 非空且 React Query 启用:挂载即拉取", async () => {
    // useWidgetPolling 新实现下: React Query 控制轮询,无独立"缓存跳过"概念
    // (dashboardStore L1 只写不读 — React Query 用自己的 staleTime)
    mocks.getBatchWidgetData.mockResolvedValue(new Map());
    const { wrapper } = createQueryWrapper();
    const { result } = renderHook(() => useWidgetPolling({ widgetIds: ["w1"], interval: 60 }), {
      wrapper,
    });
    await waitFor(() => expect(mocks.getBatchWidgetData).toHaveBeenCalledWith(["w1"]));
    await waitFor(() => expect(result.current.loading).toBe(false));
  });
});
