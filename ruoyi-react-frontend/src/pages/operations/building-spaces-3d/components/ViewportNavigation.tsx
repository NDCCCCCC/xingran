/**
 * 视图导航组件
 * 备用导航组件，实际返回按钮已集成到各组件内部
 * 此组件保留用于将来的扩展需求
 */

import type { ViewLevel } from "@/store/visualizationStore";

interface ViewportNavigationProps {
  currentView: ViewLevel;
  onBackToMap: () => void;
}

const ViewportNavigation: React.FC<ViewportNavigationProps> = () => {
  // 返回按钮已移至各组件内部，此组件不再渲染任何内容
  return null;
};

export default ViewportNavigation;
