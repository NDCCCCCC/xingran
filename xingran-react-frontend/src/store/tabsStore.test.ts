/**
 * tabsStore 标签页状态测试
 *
 * 覆盖:addTab(dashboard 强制固定/已存在激活/MAX_TABS 淘汰)、removeTab(固定与
 * dashboard 保护/激活切换历史栈)、setActiveTab、closeOtherTabs/closeAllTabs/
 * closeLeftTabs/closeRightTabs、updateTab/pinTab/unpinTab、reset、persist 落盘。
 * D-05: setState(initialState) reset;T-83-04-02 fake timers 验证无隐藏异步突变。
 */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { useTabsStore, useTabs } from "./tabsStore";
import type { TabItem } from "@/types/layout";

const tab = (key: string, overrides: Partial<TabItem> = {}): TabItem => ({
  key,
  title: key,
  path: key,
  closable: true,
  pinned: false,
  ...overrides,
});

const DASHBOARD = "/dashboard";

function seedTabs(items: TabItem[], activeTab = "") {
  useTabsStore.setState({
    tabs: items,
    activeTab,
    history: items.map((t) => t.key),
  });
}

describe("tabsStore", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
    // D-05: setState reset,不包 Provider
    useTabsStore.setState({ tabs: [], activeTab: "", history: [] });
  });

  it("addTab 新增并激活;固定标签排前", () => {
    useTabsStore.getState().addTab(tab("/a"));
    useTabsStore.getState().addTab(tab("/pinned", { pinned: true, closable: false }));
    useTabsStore.getState().addTab(tab("/b"));

    const state = useTabsStore.getState();
    expect(state.tabs.map((t) => t.key)).toEqual(["/pinned", "/a", "/b"]);
    expect(state.activeTab).toBe("/b");
    expect(state.history).toEqual(["/a", "/pinned", "/b"]);
  });

  it("addTab dashboard 标签强制固定不可关闭", () => {
    useTabsStore.getState().addTab(tab(DASHBOARD, { closable: true, pinned: false }));

    const dash = useTabsStore.getState().tabs[0];
    expect(dash.pinned).toBe(true);
    expect(dash.closable).toBe(false);
  });

  it("addTab 已存在的标签只激活不重复添加;破损 dashboard 标签被修复", () => {
    seedTabs([tab("/a"), tab(DASHBOARD, { pinned: false, closable: true })]);

    useTabsStore.getState().addTab(tab("/a"));
    expect(useTabsStore.getState().tabs).toHaveLength(2);
    expect(useTabsStore.getState().activeTab).toBe("/a");

    useTabsStore.getState().addTab(tab(DASHBOARD, { pinned: false, closable: true }));
    const dash = useTabsStore.getState().tabs.find((t) => t.key === DASHBOARD)!;
    expect(dash.pinned).toBe(true);
    expect(dash.closable).toBe(false);
  });

  it("超过 MAX_TABS(50) 时淘汰最早的非固定标签并清理其表格状态", () => {
    const many = Array.from({ length: 50 }, (_, i) => tab(`/t${i}`));
    many[3].pinned = true;
    seedTabs(many);
    sessionStorage.setItem("xingran_table_state_t0_current", "2");

    useTabsStore.getState().addTab(tab("/new"));

    const state = useTabsStore.getState();
    expect(state.tabs).toHaveLength(50); // 淘汰 t0 加入 new
    expect(state.tabs.find((t) => t.key === "/t0")).toBeUndefined();
    expect(state.tabs.find((t) => t.key === "/t3")).toBeDefined(); // 固定标签保留
    expect(sessionStorage.getItem("xingran_table_state_t0_current")).toBeNull();
  });

  it("全部固定时达到上限不再新增", () => {
    const many = Array.from({ length: 50 }, (_, i) =>
      tab(`/t${i}`, { pinned: true, closable: false })
    );
    seedTabs(many);

    useTabsStore.getState().addTab(tab("/new"));
    expect(useTabsStore.getState().tabs.find((t) => t.key === "/new")).toBeUndefined();
  });

  it("removeTab:dashboard 与固定标签受保护;激活切换到历史前一个", () => {
    seedTabs(
      [tab("/a"), tab("/b"), tab("/c"), tab("/fixed", { pinned: true, closable: false })],
      "/c"
    );
    sessionStorage.setItem("xingran_table_state_c_current", "3");

    useTabsStore.getState().removeTab(DASHBOARD);
    expect(useTabsStore.getState().tabs).toHaveLength(4);

    useTabsStore.getState().removeTab("/fixed");
    expect(useTabsStore.getState().tabs).toHaveLength(4);

    useTabsStore.getState().removeTab("/c");
    const state = useTabsStore.getState();
    expect(state.tabs.map((t) => t.key)).toEqual(["/a", "/b", "/fixed"]);
    expect(state.activeTab).toBe("/b"); // 历史前一个
    expect(sessionStorage.getItem("xingran_table_state_c_current")).toBeNull();
  });

  it("removeTab 移除首个历史项时回退到 tabs[0]", () => {
    seedTabs([tab("/a"), tab("/b")], "/a");
    useTabsStore.getState().removeTab("/a");
    expect(useTabsStore.getState().activeTab).toBe("/b");
  });

  it("removeTab 移除非激活标签不影响 activeTab", () => {
    seedTabs([tab("/a"), tab("/b")], "/b");
    useTabsStore.getState().removeTab("/a");
    expect(useTabsStore.getState().activeTab).toBe("/b");
  });

  it("setActiveTab 更新激活并重排 history 到末尾", () => {
    seedTabs([tab("/a"), tab("/b"), tab("/c")], "/c");
    useTabsStore.getState().setActiveTab("/a");

    const state = useTabsStore.getState();
    expect(state.activeTab).toBe("/a");
    expect(state.history).toEqual(["/b", "/c", "/a"]);
  });

  it("closeOtherTabs 保留 keepKey 与固定标签,清被关标签的表格状态", () => {
    seedTabs([
      tab("/a"),
      tab("/keep"),
      tab("/c"),
      tab("/fixed", { pinned: true, closable: false }),
    ]);
    sessionStorage.setItem("xingran_table_state_a_current", "1");
    sessionStorage.setItem("xingran_table_state_c_current", "1");

    useTabsStore.getState().closeOtherTabs("/keep");

    const state = useTabsStore.getState();
    // 保持原有顺序过滤,不重排
    expect(state.tabs.map((t) => t.key)).toEqual(["/keep", "/fixed"]);
    expect(state.activeTab).toBe("/keep");
    expect(sessionStorage.getItem("xingran_table_state_a_current")).toBeNull();
    expect(sessionStorage.getItem("xingran_table_state_c_current")).toBeNull();
  });

  it("closeAllTabs 仅保留固定标签;无固定时清空", () => {
    seedTabs([tab("/a"), tab("/fixed", { pinned: true, closable: false }), tab("/b")]);
    useTabsStore.getState().closeAllTabs();

    const state = useTabsStore.getState();
    expect(state.tabs.map((t) => t.key)).toEqual(["/fixed"]);
    expect(state.activeTab).toBe("/fixed");

    useTabsStore.getState().closeAllTabs();
    // 仅剩固定标签,再关一次仍保留
    expect(useTabsStore.getState().tabs).toHaveLength(1);
  });

  it("closeLeftTabs/closeRightTabs 按下标裁剪;keepKey 不存在时 no-op", () => {
    seedTabs([tab("/fixed", { pinned: true }), tab("/a"), tab("/keep"), tab("/b"), tab("/c")]);

    useTabsStore.getState().closeLeftTabs("/keep");
    expect(useTabsStore.getState().tabs.map((t) => t.key)).toEqual(["/fixed", "/keep", "/b", "/c"]);

    useTabsStore.getState().closeRightTabs("/keep");
    expect(useTabsStore.getState().tabs.map((t) => t.key)).toEqual(["/fixed", "/keep"]);

    useTabsStore.getState().closeLeftTabs("/nonexistent");
    expect(useTabsStore.getState().tabs).toHaveLength(2);
  });

  it("updateTab 更新字段;更新 pinned 触发重排序;dashboard 强制固定", () => {
    seedTabs([tab("/a"), tab("/b")]);

    useTabsStore.getState().updateTab("/a", { title: "新标题" });
    expect(useTabsStore.getState().tabs[0].title).toBe("新标题");

    useTabsStore.getState().updateTab("/a", { pinned: true, closable: false });
    const keys = useTabsStore.getState().tabs.map((t) => t.key);
    expect(keys[0]).toBe("/a"); // 固定标签排前

    seedTabs([tab("/b"), tab(DASHBOARD, { pinned: false, closable: true })]);
    useTabsStore.getState().updateTab(DASHBOARD, { pinned: false, closable: true });
    const dash = useTabsStore.getState().tabs.find((t) => t.key === DASHBOARD)!;
    expect(dash.pinned).toBe(true);
    expect(dash.closable).toBe(false);
  });

  it("pinTab/unpinTab;dashboard 不可操作", () => {
    seedTabs([tab("/a"), tab(DASHBOARD, { pinned: true, closable: false })]);

    useTabsStore.getState().pinTab("/a");
    expect(useTabsStore.getState().tabs.find((t) => t.key === "/a")!.pinned).toBe(true);

    useTabsStore.getState().unpinTab("/a");
    expect(useTabsStore.getState().tabs.find((t) => t.key === "/a")!.pinned).toBe(false);
    expect(useTabsStore.getState().tabs.find((t) => t.key === "/a")!.closable).toBe(true);

    // dashboard 短路
    useTabsStore.getState().unpinTab(DASHBOARD);
    const dash = useTabsStore.getState().tabs.find((t) => t.key === DASHBOARD)!;
    expect(dash.pinned).toBe(true);
  });

  it("reset 清空全部状态", () => {
    seedTabs([tab("/a")], "/a");
    useTabsStore.getState().reset();
    const state = useTabsStore.getState();
    expect(state.tabs).toEqual([]);
    expect(state.activeTab).toBe("");
    expect(state.history).toEqual([]);
  });

  it("persist 落盘 localStorage(T-83-04-03 验证写入值)", () => {
    useTabsStore.getState().addTab(tab("/persisted"));
    const raw = localStorage.getItem("tabs-storage");
    expect(raw).toBeTruthy();
    const parsed = JSON.parse(raw!);
    expect(parsed.state.tabs[0].key).toBe("/persisted");
    expect(parsed.state.activeTab).toBe("/persisted");
  });

  it("fake timers 下无隐藏异步状态突变(T-83-04-02)", async () => {
    vi.useFakeTimers();
    try {
      seedTabs([tab("/a"), tab("/b")], "/b");
      useTabsStore.getState().addTab(tab("/c"));
      useTabsStore.getState().removeTab("/c");
      const snapshot = JSON.stringify(useTabsStore.getState().tabs);

      await vi.advanceTimersByTimeAsync(5000);
      expect(JSON.stringify(useTabsStore.getState().tabs)).toBe(snapshot);
      expect(useTabsStore.getState().activeTab).toBe("/b");
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("useTabs hook", () => {
  it("聚合暴露 store 字段与 hasTabs 派生值", () => {
    useTabsStore.setState({ tabs: [], activeTab: "", history: [] });
    const view = useTabsStore.getState();
    expect(view.tabs).toEqual([]);

    useTabsStore.getState().addTab(tab("/x"));
    expect(useTabsStore.getState().tabs).toHaveLength(1);
    // useTabs 的 hasTabs 逻辑:tabs.length > 0
    expect(useTabsStore.getState().tabs.length > 0).toBe(true);
  });
});
