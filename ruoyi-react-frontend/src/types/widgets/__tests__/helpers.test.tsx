/**
 * Phase 88 Batch284 — types/widgets/helpers 测试
 */
import { describe, it, expect } from "vitest";
import React from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { asWidgetComponent, createLazyWidget } from "../helpers";

const MockComp: React.FC = () => <div>Mock</div>;

describe("types/widgets/helpers", () => {
  it("asWidgetComponent 返回组件", () => {
    const w = asWidgetComponent(MockComp);
    expect(w).toBeDefined();
  });

  it("createLazyWidget 返回 lazy component", () => {
    const w = createLazyWidget(async () => ({ StatCard: MockComp }), "StatCard");
    expect(w).toBeDefined();
    expect(w.$$typeof).toBeDefined();
  });

  it("createLazyWidget 异步加载", async () => {
    const w = createLazyWidget(async () => ({ My: MockComp }), "My");
    const resolved = await w;
    expect(resolved).toBeDefined();
  });
});
