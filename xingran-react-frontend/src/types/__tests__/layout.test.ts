/**
 * Phase 88 Batch223 — types/layout 布局类型
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  LayoutType,
  LayoutConfig,
  LayoutState,
  TabItem,
  TabsState,
  DensityMode,
} from "../layout";

describe("types/layout", () => {
  it("LayoutType 3 类", () => {
    const t: LayoutType[] = ["classic", "hybrid", "innovative"];
    expect(t.length).toBe(3);
  });

  it("LayoutConfig shape", () => {
    const cfg: LayoutConfig = {
      id: "classic",
      name: "Classic",
      description: "传统布局",
      features: {
        sidebar: { collapsible: true, width: 240, collapsedWidth: 64, position: "left" },
        header: { fixed: true, height: 56, showBreadcrumb: true, showUserInfo: true },
        tabs: { enabled: true, position: "top", closable: true, draggable: true, persist: true },
        content: { padding: "16px", centered: true, scrollable: true },
      },
    };
    expect(cfg.features.sidebar.width).toBe(240);
  });

  it("TabItem shape 含 icon/pinned", () => {
    const t: TabItem = {
      key: "t1",
      title: "Tab 1",
      path: "/tab1",
      closable: true,
      icon: null,
      pinned: false,
    };
    expect(t.key).toBe("t1");
  });

  it("TabsState shape", () => {
    const s: TabsState = {
      tabs: [{ key: "t1", title: "A", path: "/a", closable: true }],
      activeTab: "t1",
      history: ["/a"],
    };
    expect(s.tabs.length).toBe(1);
  });

  it("LayoutState shape", () => {
    const s: LayoutState = {
      currentLayout: "hybrid",
      sidebarCollapsed: false,
      headerVisible: true,
    };
    expect(s.currentLayout).toBe("hybrid");
  });

  it("DensityMode 3 类", () => {
    const m: DensityMode[] = ["compact", "comfortable", "spacious"];
    expect(m.length).toBe(3);
  });
});
