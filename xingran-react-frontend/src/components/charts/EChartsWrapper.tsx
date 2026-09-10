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

// Bundle echarts-for-react AND the echarts side-effect registration (echarts.use([...]))
// into a single lazy chunk so the ~376KB echarts core is never in the initial bundle.
// echarts.use([...]) is idempotent — calling it multiple times is safe.
const ReactECharts = lazy(() =>
  Promise.all([import("echarts-for-react"), import("@/lib/echarts")]).then(([echartsReact]) => ({
    default: echartsReact.default,
  }))
);

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
    <div style={{ marginTop: 8, color: "var(--theme-text-secondary)" }}>加载图表...</div>
  </div>
);

export const EChartsWrapper = forwardRef<EChartsRef, ReactEChartsProps>((props, ref) => (
  <Suspense fallback={<Loading />}>
    <ReactECharts {...props} ref={ref} />
  </Suspense>
));

EChartsWrapper.displayName = "EChartsWrapper";

export default EChartsWrapper;
