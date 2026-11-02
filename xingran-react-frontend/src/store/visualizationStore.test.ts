/**
 * visualizationStore 运维可视化状态测试
 *
 * 覆盖:视图层级/地图状态/实体选择与 clearSelection/相机状态/
 * 筛选 update+reset/navigate 导航四方法(带/不带经纬度)。
 */
import { describe, it, expect, beforeEach } from "vitest";
import { useVisualizationStore } from "./visualizationStore";
import type { Building, Floor, WorkstationOps } from "@/types/operations";

const building = (id: string, withCoords = true): Building =>
  ({
    id,
    buildingName: `楼-${id}`,
    ...(withCoords ? { longitude: 114.3, latitude: 30.5 } : {}),
  }) as unknown as Building;

const floor = (id: string): Floor => ({ id, floorName: `层-${id}` }) as unknown as Floor;

const workstation = (id: string): WorkstationOps =>
  ({ id, workstationName: `工位-${id}` }) as unknown as WorkstationOps;

describe("visualizationStore", () => {
  beforeEach(() => {
    // 回初始状态(与 store 初始常量一致)
    useVisualizationStore.setState({
      viewLevel: "map",
      mapState: { center: [114.305393, 30.593099], zoom: 7, showLevel: 1 },
      selectedCity: null,
      selectedBuilding: null,
      selectedFloor: null,
      selectedWorkstationOps: null,
      cameraPosition: [0, 50, 100],
      cameraTarget: [0, 0, 0],
      isTransitioning: false,
      filters: {
        city: [],
        buildingStatus: [],
        floorStatus: [],
        workstationTypes: [],
        workstationStatus: [],
      },
    });
  });

  it("初始状态为地图层级 + 默认中心", () => {
    const state = useVisualizationStore.getState();
    expect(state.viewLevel).toBe("map");
    expect(state.mapState.zoom).toBe(7);
    expect(state.cameraPosition).toEqual([0, 50, 100]);
  });

  it("视图层级与地图状态 setters", () => {
    const s = useVisualizationStore.getState();
    s.setViewLevel("building");
    expect(useVisualizationStore.getState().viewLevel).toBe("building");

    s.setMapCenter([120, 30]);
    expect(useVisualizationStore.getState().mapState.center).toEqual([120, 30]);
    expect(useVisualizationStore.getState().mapState.zoom).toBe(7); // 其余字段保持

    s.setMapZoom(12);
    expect(useVisualizationStore.getState().mapState.zoom).toBe(12);

    s.setShowLevel(2);
    expect(useVisualizationStore.getState().mapState.showLevel).toBe(2);
  });

  it("实体选择与 clearSelection", () => {
    const s = useVisualizationStore.getState();
    const b = building("b1");
    s.selectCity("027");
    s.selectBuilding(b);
    s.selectFloor(floor("f1"));
    s.selectWorkstationOps(workstation("w1"));

    let state = useVisualizationStore.getState();
    expect(state.selectedCity).toBe("027");
    expect(state.selectedBuilding).toBe(b);
    expect(state.selectedFloor?.id).toBe("f1");
    expect(state.selectedWorkstationOps?.id).toBe("w1");

    s.clearSelection();
    state = useVisualizationStore.getState();
    expect(state.selectedCity).toBeNull();
    expect(state.selectedBuilding).toBeNull();
    expect(state.selectedFloor).toBeNull();
    expect(state.selectedWorkstationOps).toBeNull();
  });

  it("相机状态 setters", () => {
    const s = useVisualizationStore.getState();
    s.setCameraPosition([1, 2, 3]);
    s.setCameraTarget([4, 5, 6]);
    s.setTransitioning(true);

    const state = useVisualizationStore.getState();
    expect(state.cameraPosition).toEqual([1, 2, 3]);
    expect(state.cameraTarget).toEqual([4, 5, 6]);
    expect(state.isTransitioning).toBe(true);
  });

  it("updateFilters 合并更新;resetFilters 回默认", () => {
    const s = useVisualizationStore.getState();
    s.updateFilters({ city: ["027"], buildingStatus: [0] });
    let state = useVisualizationStore.getState();
    expect(state.filters.city).toEqual(["027"]);
    expect(state.filters.buildingStatus).toEqual([0]);
    expect(state.filters.floorStatus).toEqual([]); // 未指定字段保持

    s.updateFilters({ city: ["020"] });
    state = useVisualizationStore.getState();
    expect(state.filters.city).toEqual(["020"]);

    s.resetFilters();
    expect(useVisualizationStore.getState().filters.city).toEqual([]);
  });

  it("navigateToBuilding:有经纬度时定位地图并进入 building 层级", () => {
    useVisualizationStore.getState().navigateToBuilding(building("b1"));

    const state = useVisualizationStore.getState();
    expect(state.selectedBuilding?.id).toBe("b1");
    expect(state.viewLevel).toBe("building");
    expect(state.isTransitioning).toBe(true);
    expect(state.mapState.center).toEqual([114.3, 30.5]);
    expect(state.mapState.zoom).toBe(14);
    expect(state.mapState.showLevel).toBe(2);
  });

  it("navigateToBuilding:无经纬度时地图状态不变", () => {
    useVisualizationStore.getState().navigateToBuilding(building("b2", false));

    const state = useVisualizationStore.getState();
    expect(state.viewLevel).toBe("building");
    expect(state.mapState.center).toEqual([114.305393, 30.593099]); // 默认中心
    expect(state.mapState.zoom).toBe(7);
  });

  it("navigateToFloor / navigateToWorkstationOps", () => {
    useVisualizationStore.getState().navigateToFloor(floor("f1"));
    let state = useVisualizationStore.getState();
    expect(state.viewLevel).toBe("floor");
    expect(state.isTransitioning).toBe(true);

    useVisualizationStore.getState().navigateToWorkstationOps(workstation("w1"));
    state = useVisualizationStore.getState();
    expect(state.viewLevel).toBe("workstation");
    expect(state.selectedWorkstationOps?.id).toBe("w1");
  });

  it("navigateToMap 全量复位到地图层级", () => {
    const s = useVisualizationStore.getState();
    s.selectCity("027");
    s.selectBuilding(building("b1"));
    s.setViewLevel("building");

    useVisualizationStore.getState().navigateToMap();

    const state = useVisualizationStore.getState();
    expect(state.viewLevel).toBe("map");
    expect(state.selectedCity).toBeNull();
    expect(state.selectedBuilding).toBeNull();
    expect(state.mapState).toEqual({
      center: [114.305393, 30.593099],
      zoom: 7,
      showLevel: 1,
    });
    expect(state.isTransitioning).toBe(true);
  });
});
