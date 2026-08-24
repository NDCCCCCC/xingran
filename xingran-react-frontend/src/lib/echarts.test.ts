/**
 * echarts.ts 按需加载模块测试 (Phase 83-03)
 *
 * 模块职责:向 echarts/core 注册 CustomChart + 4 个 components + CanvasRenderer,
 * 默认导出共享的 echarts core 命名空间。测试验证注册副作用无异常且导出可用
 * (jsdom 无 canvas 2d context,不触发真实 init 渲染)。
 */
import { describe, expect, it } from "vitest";
import echarts from "./echarts";

describe("lib/echarts 按需加载配置", () => {
  it("模块导入副作用(组件注册)执行完成且不抛错", () => {
    // import 本身执行了 echarts.use([...7 项]);到达此断言即注册成功
    expect(true).toBe(true);
  });

  it("默认导出为 echarts core 命名空间(具备 init/use API)", () => {
    expect(typeof echarts.init).toBe("function");
    expect(typeof echarts.use).toBe("function");
  });

  it("注册后的 core 保留常用 API(version/dispose/getInstanceByDom)", () => {
    const core = echarts as unknown as Record<string, unknown>;
    expect(typeof core.version).toBe("string");
    expect((core.version as string).length).toBeGreaterThan(0);
    expect(typeof core.dispose).toBe("function");
    expect(typeof core.getInstanceByDom).toBe("function");
  });
});
