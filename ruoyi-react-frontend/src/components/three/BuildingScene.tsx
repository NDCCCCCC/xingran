/**
 * BuildingScene - 3D scene lazy wrapper
 *
 * Wraps the four 3D scene components in `pages/operations/building-spaces-3d/components/`
 * with React.lazy + Suspense so that three.js (and @react-three/*) only loads when the
 * user navigates into the 3D visualization page.
 *
 * Per D-06 (Wave 2): three.js is a heavy lib (894KB raw / 235KB gzip baseline) that
 * should not be in the initial bundle.
 *
 * Per D-09 / D-17: Suspense fallback uses AntD Spin with a descriptive Chinese tip.
 */

import { lazy, Suspense, type FC, type ComponentProps } from "react";
import { Spin } from "antd";

type BuildingModel3DProps = ComponentProps<typeof BuildingModel3D>;
type FloorPlan3DProps = ComponentProps<typeof FloorPlan3D>;

const BuildingModel3D = lazy(
  () => import("@/pages/operations/building-spaces-3d/components/BuildingModel3D")
);
const FloorPlan3D = lazy(
  () => import("@/pages/operations/building-spaces-3d/components/FloorPlan3D")
);
const FloorView3D = lazy(
  () => import("@/pages/operations/building-spaces-3d/components/FloorView3D")
);
const BuildingView3D = lazy(
  () => import("@/pages/operations/building-spaces-3d/components/BuildingView3D")
);

const Loading: FC = () => (
  <div
    style={{
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
      padding: 40,
      minHeight: 240,
    }}
  >
    <Spin size="large">
      <div style={{ minHeight: 120 }} />
    </Spin>
    <div style={{ marginTop: 8, color: "rgba(0, 0, 0, 0.45)" }}>加载 3D 场景...</div>
  </div>
);

export const BuildingModel3DLazy: FC<BuildingModel3DProps> = (props) => (
  <Suspense fallback={<Loading />}>
    <BuildingModel3D {...props} />
  </Suspense>
);

export const FloorPlan3DLazy: FC<FloorPlan3DProps> = (props) => (
  <Suspense fallback={<Loading />}>
    <FloorPlan3D {...props} />
  </Suspense>
);

export const FloorView3DLazy: FC = () => (
  <Suspense fallback={<Loading />}>
    <FloorView3D />
  </Suspense>
);

export const BuildingView3DLazy: FC = () => (
  <Suspense fallback={<Loading />}>
    <BuildingView3D />
  </Suspense>
);
