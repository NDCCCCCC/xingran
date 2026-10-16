import { create } from "zustand";
import { devtools } from "zustand/middleware";
import type { Building, Floor, WorkstationOps } from "@/types/operations";

// 视图层级类型
export type ViewLevel = "map" | "building" | "floor" | "workstation";

// 地图状态
interface MapState {
  center: [number, number]; // [lng, lat]
  zoom: number;
  showLevel: 1 | 2; // 1=显示城市, 2=显示楼宇
}

// 筛选条件
interface FilterState {
  city: string[];
  buildingStatus: number[];
  floorStatus: number[];
  workstationTypes: number[];
  workstationStatus: number[];
}

// 可视化状态接口
interface VisualizationState {
  // 视图层级
  viewLevel: ViewLevel;

  // 地图状态
  mapState: MapState;

  // 选中的实体
  selectedCity: string | null;
  selectedBuilding: Building | null;
  selectedFloor: Floor | null;
  selectedWorkstationOps: WorkstationOps | null;

  // 相机状态（3D 视图）
  cameraPosition: [number, number, number];
  cameraTarget: [number, number, number];
  isTransitioning: boolean;

  // 筛选条件
  filters: FilterState;

  // Actions
  setViewLevel: (level: ViewLevel) => void;
  setMapCenter: (center: [number, number]) => void;
  setMapZoom: (zoom: number) => void;
  setShowLevel: (level: 1 | 2) => void;

  selectCity: (cityCode: string) => void;
  selectBuilding: (building: Building) => void;
  selectFloor: (floor: Floor) => void;
  selectWorkstationOps: (workstation: WorkstationOps) => void;
  clearSelection: () => void;

  setCameraPosition: (position: [number, number, number]) => void;
  setCameraTarget: (target: [number, number, number]) => void;
  setTransitioning: (isTransitioning: boolean) => void;

  updateFilters: (filters: Partial<FilterState>) => void;
  resetFilters: () => void;

  // 导航方法
  navigateToBuilding: (building: Building) => void;
  navigateToFloor: (floor: Floor) => void;
  navigateToWorkstationOps: (workstation: WorkstationOps) => void;
  navigateToMap: () => void;
}

// 默认地图中心（湖北省中心）
const DEFAULT_MAP_CENTER: [number, number] = [114.305393, 30.593099]; // 武汉市

// 默认筛选条件
const DEFAULT_FILTERS: FilterState = {
  city: [],
  buildingStatus: [],
  floorStatus: [],
  workstationTypes: [],
  workstationStatus: [],
};

// 创建 Store
export const useVisualizationStore = create<VisualizationState>()(
  devtools(
    (set) => ({
      // 初始状态
      viewLevel: "map",

      mapState: {
        center: DEFAULT_MAP_CENTER,
        zoom: 7,
        showLevel: 1,
      },

      selectedCity: null,
      selectedBuilding: null,
      selectedFloor: null,
      selectedWorkstationOps: null,

      cameraPosition: [0, 50, 100],
      cameraTarget: [0, 0, 0],
      isTransitioning: false,

      filters: DEFAULT_FILTERS,

      // 设置视图层级
      setViewLevel: (level) => set({ viewLevel: level }),

      // 地图操作
      setMapCenter: (center) =>
        set((state) => ({
          mapState: { ...state.mapState, center },
        })),

      setMapZoom: (zoom) =>
        set((state) => ({
          mapState: { ...state.mapState, zoom },
        })),

      setShowLevel: (showLevel) =>
        set((state) => ({
          mapState: { ...state.mapState, showLevel },
        })),

      // 选择实体
      selectCity: (cityCode) => set({ selectedCity: cityCode }),

      selectBuilding: (building) => set({ selectedBuilding: building }),

      selectFloor: (floor) => set({ selectedFloor: floor }),

      selectWorkstationOps: (workstation) => set({ selectedWorkstationOps: workstation }),

      clearSelection: () =>
        set({
          selectedCity: null,
          selectedBuilding: null,
          selectedFloor: null,
          selectedWorkstationOps: null,
        }),

      // 相机操作
      setCameraPosition: (position) => set({ cameraPosition: position }),

      setCameraTarget: (target) => set({ cameraTarget: target }),

      setTransitioning: (isTransitioning) => set({ isTransitioning }),

      // 筛选条件
      updateFilters: (newFilters) =>
        set((state) => ({
          filters: { ...state.filters, ...newFilters },
        })),

      resetFilters: () => set({ filters: DEFAULT_FILTERS }),

      // 导航方法
      navigateToBuilding: (building) => {
        set({
          selectedBuilding: building,
          viewLevel: "building",
          isTransitioning: true,
        });

        // 设置地图中心到楼宇位置
        if (building.longitude && building.latitude) {
          set((state) => ({
            mapState: {
              ...state.mapState,
              center: [building.longitude as number, building.latitude as number],
              zoom: 14,
              showLevel: 2,
            },
          }));
        }
      },

      navigateToFloor: (floor) => {
        set({
          selectedFloor: floor,
          viewLevel: "floor",
          isTransitioning: true,
        });
      },

      navigateToWorkstationOps: (workstation) => {
        set({
          selectedWorkstationOps: workstation,
          viewLevel: "workstation",
        });
      },

      navigateToMap: () => {
        set({
          viewLevel: "map",
          mapState: {
            center: DEFAULT_MAP_CENTER,
            zoom: 7,
            showLevel: 1,
          },
          selectedCity: null,
          selectedBuilding: null,
          selectedFloor: null,
          selectedWorkstationOps: null,
          isTransitioning: true,
        });
      },
    }),
    {
      name: "visualization-store",
    }
  )
);
