/**
 * EChartsWrapper - Lazy-loaded ECharts wrapper
 *
 * Wraps `echarts-for-react/esm/core` (按需 core 入口，仅导入 ECharts 核心，不含全量
 * echarts-for-react 默认导出带来的 `import * as echarts from 'echarts'` 全量引用），
 * 通过 prop 传入 @/lib/echarts 按需注册的 echarts 实例，配合 React.lazy + Suspense
 * 使 echarts 库不在首屏 bundle 中。
 *
 * Per D-06 (Wave 2): echarts only loads when a chart widget is rendered.
 * Per D-09 / D-17: Suspense fallback uses AntD Spin with a descriptive tip.
 *
 * Usage:
 *   import EChartsWrapper from '@/components/charts/EChartsWrapper';
 *   import echarts from '@/lib/echarts';
 *   <EChartsWrapper echarts={echarts} option={option} style={{ height: 300 }} />
 *
 * The wrapper preserves the `echarts-for-react` component interface.
 */

import { lazy, Suspense, forwardRef, type ComponentProps, type ComponentRef } from "react";
import { Spin } from "antd";
import echartsCore from "@/lib/echarts";

// echarts-for-react/esm/core 按需入口（仅 EChartsReactCore 组件，不含全量 echarts 导入）
// echartsCore 从 @/lib/echarts 传入（已注册 line/bar/pie 等图表类型）
const ReactECharts = lazy(() =>
  import("echarts-for-react/esm/core").then((m) => ({ default: m.default }))
);

// `echarts-for-react` exports a default React component. We accept the same
// props the original accepts (ComponentProps on the lazy module).
type ReactEChartsProps = ComponentProps<typeof ReactECharts> & { echarts?: typeof echartsCore };
type EChartsRef = ComponentRef<typeof ReactECharts>;

const Loading = () => (
  <div
    style={{
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
      padding: 24,
      minHeight: 120,
    }}
  >
    <Spin>
      <div style={{ minHeight: 60 }} />
    </Spin>
    <div style={{ marginTop: 8, color: "var(--theme-text-secondary)" }}>加载图表...</div>
  </div>
);

export const EChartsWrapper = forwardRef<EChartsRef, ReactEChartsProps>((props, ref) => (
  <Suspense fallback={<Loading />}>
    <ReactECharts {...props} ref={ref} echarts={echartsCore} />
  </Suspense>
));

EChartsWrapper.displayName = "EChartsWrapper";

export default EChartsWrapper;
