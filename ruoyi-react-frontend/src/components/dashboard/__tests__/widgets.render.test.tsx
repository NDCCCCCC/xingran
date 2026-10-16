/**
 * Phase 88 batch5 — dashboard widgets 渲染(WidgetRenderer + 各 Widget)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { widgetRegistry } from "../widgets/configs/widgetRegistry";
import type { WidgetConfig } from "@/types/dashboard";

const makeWidget = (type: string): WidgetConfig =>
  ({
    id: `w-${type}`,
    type,
    title: `测试 ${type}`,
    position: { x: 0, y: 0, w: 6, h: 3 },
    dataSource: { type: "static", data: null },
    display: { type },
  }) as WidgetConfig;

describe("dashboard widgets 渲染(真实 hooks)", () => {
  // 逐个 widget type 渲染(Pitfall #4:不整体 mock registry)
  for (const type of Object.keys(widgetRegistry)) {
    it(`WidgetRenderer 渲染 ${type}`, { timeout: 30000 }, async () => {
      const { WidgetRenderer } = await import("../widgets/WidgetRenderer");
      const { rendered } = renderPageWithEndpoints(<WidgetRenderer widget={makeWidget(type)} />, {
        // 静态数据源 widget 不打 API;fallback 兜底其它端点
      });
      await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), {
        timeout: 20000,
      });
    });
  }
});
