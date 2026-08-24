/**
 * useColumnConfig 列配置 Hook 测试
 *
 * 覆盖：默认列加载 / localStorage 缓存命中与过期 / 服务端配置转换(transformToColumnConfig) /
 * 健全性检查回退(可见列过少) / saveConfig 成功与失败 / resetConfig /
 * toggleColumn / updateColumnWidth / updateColumnOrder / visibleColumns 派生。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";

const mockMessage = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
}));

const columnConfigApiMock = vi.hoisted(() => ({
  getByPageKey: vi.fn(),
  save: vi.fn(),
  reset: vi.fn(),
}));

vi.mock("@/lib/columnConfigApi", () => ({
  columnConfigApi: columnConfigApiMock,
}));

vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  const App = Object.assign(actual.App, {
    useApp: () => ({ message: mockMessage }),
  });
  return { ...actual, App };
});

import { useColumnConfig, type ColumnConfig } from "./useColumnConfig";

const defaultColumns: ColumnConfig[] = [
  { key: "name", label: "名称", visible: true, order: 1 },
  { key: "status", label: "状态", visible: true, order: 2 },
  { key: "created", label: "创建时间", visible: true, order: 3 },
  { key: "remark", label: "备注", visible: false, order: 4 },
];

const CACHE_KEY = "column_config:test-page";

function renderColumnConfig(overrides: { enableCache?: boolean } = {}) {
  return renderHook(() =>
    useColumnConfig({
      pageKey: "test-page",
      defaultColumns,
      enableCache: overrides.enableCache ?? true,
    })
  );
}

/** minVisible = floor(可见默认列 3 / 2) = 1,可见列 <1 视为损坏 */
function writeCache(data: ColumnConfig[], ageMs = 0) {
  localStorage.setItem(CACHE_KEY, JSON.stringify({ data, timestamp: Date.now() - ageMs }));
}

describe("useColumnConfig", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
    columnConfigApiMock.getByPageKey.mockResolvedValue({ data: [] });
    columnConfigApiMock.save.mockResolvedValue({});
    columnConfigApiMock.reset.mockResolvedValue({});
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
    vi.restoreAllMocks();
  });

  it("服务端无配置时使用默认列并写缓存", async () => {
    columnConfigApiMock.getByPageKey.mockResolvedValue({ data: [] });
    const { result } = renderColumnConfig();

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.config).toEqual(defaultColumns);
    expect(JSON.parse(localStorage.getItem(CACHE_KEY)!).data).toEqual(defaultColumns);
  });

  it("缓存命中(未过期且健全)时直接使用缓存,不覆盖服务端结果", async () => {
    const cached: ColumnConfig[] = [
      { key: "name", label: "名称", visible: true, order: 1 },
      { key: "status", label: "状态", visible: true, order: 2 },
    ];
    writeCache(cached);
    columnConfigApiMock.getByPageKey.mockResolvedValue({
      data: [{ columnKey: "created", visible: true, width: 120 }],
    });

    const { result } = renderColumnConfig();
    // 缓存先应用;随后服务端转换结果覆盖(服务端优先)
    await waitFor(() => expect(result.current.loading).toBe(false));
    const keys = result.current.config.map((c) => c.key);
    expect(keys).toContain("created");
    expect(keys).toContain("name");
  });

  it("缓存过期被清除并回退", async () => {
    writeCache(defaultColumns, 6 * 60 * 1000); // 6 分钟 > 5 分钟 TTL
    columnConfigApiMock.getByPageKey.mockResolvedValue({ data: [] });

    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.config).toEqual(defaultColumns);
  });

  it("缓存损坏(JSON parse 失败)安全回退", async () => {
    localStorage.setItem(CACHE_KEY, "not-valid-json{{{");
    columnConfigApiMock.getByPageKey.mockResolvedValue({ data: [] });

    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.config).toEqual(defaultColumns);
    expect(consoleSpy).toHaveBeenCalled();
  });

  it("缓存可见列过少(不健全)时移除缓存", async () => {
    writeCache([
      { key: "name", label: "名称", visible: false, order: 1 },
      { key: "status", label: "状态", visible: false, order: 2 },
    ]);
    columnConfigApiMock.getByPageKey.mockResolvedValue({ data: [] });

    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(localStorage.getItem(CACHE_KEY)).toBeTruthy(); // 服务端空配置后重新写入默认
    expect(JSON.parse(localStorage.getItem(CACHE_KEY)!).data).toEqual(defaultColumns);
  });

  it("服务端配置转换:合并 visible/width,顺序保持默认插入序", async () => {
    columnConfigApiMock.getByPageKey.mockResolvedValue({
      data: [
        { columnKey: "status", visible: false, width: 90 },
        { columnKey: "name", visible: true, width: 0 },
      ],
    });

    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));

    const status = result.current.config.find((c) => c.key === "status")!;
    expect(status.visible).toBe(false);
    expect(status.width).toBe(90);
    // width 0 回退到默认 width(undefined);顺序保持默认列插入序
    expect(result.current.config.map((c) => c.key)).toEqual([
      "name",
      "status",
      "created",
      "remark",
    ]);
    expect(result.current.config.map((c) => c.order)).toEqual([1, 2, 3, 4]);
    // 结果写缓存
    expect(JSON.parse(localStorage.getItem(CACHE_KEY)!).data[0].key).toBe("name");
  });

  it("服务端配置不健全(可见列不足)回退默认列", async () => {
    columnConfigApiMock.getByPageKey.mockResolvedValue({
      data: [
        { columnKey: "name", visible: false },
        { columnKey: "status", visible: false },
        { columnKey: "created", visible: false },
      ],
    });

    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.config).toEqual(defaultColumns);
  });

  it("loadConfig 抛错时回退默认列并复位 loading", async () => {
    columnConfigApiMock.getByPageKey.mockRejectedValue(new Error("network"));
    const { result } = renderColumnConfig();

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.config).toEqual(defaultColumns);
    expect(consoleSpy).toHaveBeenCalled();
  });

  it("enableCache=false 不读不写 localStorage", async () => {
    columnConfigApiMock.getByPageKey.mockResolvedValue({ data: [] });
    const { result } = renderColumnConfig({ enableCache: false });

    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(localStorage.getItem(CACHE_KEY)).toBeNull();
  });

  it("saveConfig 成功:更新配置+写缓存+message.success", async () => {
    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));

    const newConfig: ColumnConfig[] = [
      { key: "name", label: "名称", visible: true, order: 1, width: 100 },
    ];
    await act(async () => {
      await result.current.saveConfig(newConfig);
    });

    expect(columnConfigApiMock.save).toHaveBeenCalledWith({
      pageKey: "test-page",
      columnConfigs: [{ columnKey: "name", visible: true, width: 100 }],
    });
    expect(result.current.config).toEqual(newConfig);
    expect(result.current.saving).toBe(false);
    expect(mockMessage.success).toHaveBeenCalledWith("列配置保存成功");
    expect(JSON.parse(localStorage.getItem(CACHE_KEY)!).data).toEqual(newConfig);
  });

  it("saveConfig 失败:message.error 并向上抛出", async () => {
    columnConfigApiMock.save.mockRejectedValue(new Error("save failed"));
    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await expect(result.current.saveConfig([])).rejects.toThrow("save failed");
    });
    expect(mockMessage.error).toHaveBeenCalledWith("保存列配置失败，请稍后重试");
    expect(result.current.saving).toBe(false);
  });

  it("resetConfig 成功:回默认列+清缓存+message.success", async () => {
    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));
    writeCache([{ key: "name", label: "名称", visible: true, order: 1 }]);

    await act(async () => {
      await result.current.resetConfig();
    });
    expect(columnConfigApiMock.reset).toHaveBeenCalledWith("test-page");
    expect(result.current.config).toEqual(defaultColumns);
    expect(mockMessage.success).toHaveBeenCalledWith("列配置已重置");
    expect(localStorage.getItem(CACHE_KEY)).toBeNull();
    expect(result.current.saving).toBe(false);
  });

  it("resetConfig 失败:message.error 并向上抛出", async () => {
    columnConfigApiMock.reset.mockRejectedValue(new Error("reset failed"));
    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      await expect(result.current.resetConfig()).rejects.toThrow("reset failed");
    });
    expect(mockMessage.error).toHaveBeenCalledWith("重置列配置失败，请稍后重试");
  });

  it("toggleColumn/updateColumnWidth/updateColumnOrder 更新配置", async () => {
    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));

    act(() => result.current.toggleColumn("remark", true));
    expect(result.current.config.find((c) => c.key === "remark")!.visible).toBe(true);

    act(() => result.current.updateColumnWidth("name", 200));
    expect(result.current.config.find((c) => c.key === "name")!.width).toBe(200);

    const reordered = [...result.current.config].reverse();
    act(() => result.current.updateColumnOrder(reordered));
    expect(result.current.config.map((c) => c.key)).toEqual(reordered.map((c) => c.key));
    expect(result.current.config.map((c) => c.order)).toEqual([1, 2, 3, 4]);
  });

  it("visibleColumns 只含可见列并按 order 排序", async () => {
    const { result } = renderColumnConfig();
    await waitFor(() => expect(result.current.loading).toBe(false));

    const vis = result.current.visibleColumns;
    expect(vis.map((c) => c.key)).toEqual(["name", "status", "created"]);
    expect(vis.every((c) => c.visible)).toBe(true);
  });
});
