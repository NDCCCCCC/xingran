/**
 * Phase 88 Batch324 — types/widgets/helpers 测试
 */
import { describe, it, expect } from "vitest";
import { asWidgetComponent, createLazyWidget } from "../helpers";

describe("types/widgets/helpers", () => {
  it("asWidgetComponent 返回传入组件", () => {
    const Comp: any = () => null;
    const wrapped = asWidgetComponent(Comp);
    expect(wrapped).toBe(Comp);
  });

  it("asWidgetComponent 类型转换", () => {
    function MyWidget() {
      return null;
    }
    const wrapped = asWidgetComponent(MyWidget);
    expect(typeof wrapped).toBe("function");
  });

  it("createLazyWidget 接受 importFn + exportName", () => {
    const fakeImport = async () => ({
      MyWidget: () => null,
    });
    const Lazy = createLazyWidget(fakeImport, "MyWidget");
    expect(Lazy).toBeDefined();
    expect(typeof Lazy).toBe("object");
  });
});
