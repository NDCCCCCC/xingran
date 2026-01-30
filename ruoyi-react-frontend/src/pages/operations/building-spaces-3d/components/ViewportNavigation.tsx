/**
 * 视图导航组件 - 仅用于地图视图的返回按钮（可选）
 */

import { Button } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import type { ViewLevel } from '@/store/visualizationStore';

interface ViewportNavigationProps {
  currentView: ViewLevel;
  onBackToMap: () => void;
}

const ViewportNavigation: React.FC<ViewportNavigationProps> = ({ currentView, onBackToMap }) => {
  // 只在地图视图且需要返回时显示（备用方案，实际返回按钮在各自组件中）
  if (currentView === 'map') {
    return null;
  }

  return null; // 不再显示，返回按钮已移至各组件内部
};

export default ViewportNavigation;
