/**
 * ECharts 按需加载配置
 * 仅导入 MAC 轨迹图所需的模块，减少包体积
 */
import * as echarts from "echarts/core";
import { CustomChart, LineChart, BarChart, PieChart } from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  DataZoomComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

// 注册必需的组件（CustomChart：MAC 轨迹图；LineChart/BarChart/PieChart：dashboard 图表）
echarts.use([
  CustomChart,
  LineChart,
  BarChart,
  PieChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  DataZoomComponent,
  CanvasRenderer,
]);

export default echarts;
