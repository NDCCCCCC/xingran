/**
 * EChartsWrapper - Lazy-loaded ECharts wrapper
 *
 * Wraps `echarts-for-react` (which transitively pulls in `echarts` + `zrender`,
 * totaling ~1.1MB raw / 376KB gzip baseline) with React.lazy + Suspense so the
 * echarts library is not in the initial bundle.
 *
 * Per D-06 (Wave 2): echarts only loads when a chart widget is rendered.
 * Per D-09 / D-17: Suspense fallback uses AntD Spin with a descriptive tip.
 *
 * Usage:
 *   import ReactECharts from '@/components/charts/EChartsWrapper';
 *   <ReactECharts option={option} style={{ height: 300 }} />
 *
 * The wrapper preserves the `echarts-for-react` component interface.
 */

import { lazy, Suspense, forwardRef, type ComponentProps, type ComponentRef } from "react";
import { Spin } from "antd";

const ReactECharts = lazy(() => import("echarts-for-react"));

// `echarts-for-react` exports a default React component. We accept the same
// props the original accepts (ComponentProps on the lazy module).
type ReactEChartsProps = ComponentProps<typeof ReactECharts>;
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
    <div style={{ marginTop: 8, color: "rgba(0, 0, 0, 0.45)" }}>加载图表...</div>
  </div>
);

export const EChartsWrapper = forwardRef<EChartsRef, ReactEChartsProps>((props, ref) => (
  <Suspense fallback={<Loading />}>
    <ReactECharts {...props} ref={ref} />
  </Suspense>
));

EChartsWrapper.displayName = "EChartsWrapper";

export default EChartsWrapper;
