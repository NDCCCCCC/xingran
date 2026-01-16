/**
 * 视图导航组件
 * 显示当前视图层级和返回按钮
 */

import { Button, Space, Breadcrumb } from 'antd';
import { ArrowLeftOutlined, HomeOutlined, BuildOutlined, ApartmentOutlined, DesktopOutlined } from '@ant-design/icons';
import type { ViewLevel } from '@/store/visualizationStore';

interface ViewportNavigationProps {
  currentView: ViewLevel;
  onBackToMap: () => void;
}

const VIEW_INFO = {
  map: { title: '湖北省地图', icon: <HomeOutlined /> },
  building: { title: '楼宇详情', icon: <BuildOutlined /> },
  floor: { title: '楼层视图', icon: <ApartmentOutlined /> },
  workstation: { title: '工位平面', icon: <DesktopOutlined /> },
};

const ViewportNavigation: React.FC<ViewportNavigationProps> = ({ currentView, onBackToMap }) => {
  const current = VIEW_INFO[currentView];

  return (
    <div
      style={{
        position: 'absolute',
        top: 16,
        left: 16,
        zIndex: 1000,
        background: 'rgba(255, 255, 255, 0.95)',
        padding: '12px 16px',
        borderRadius: '8px',
        boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
        backdropFilter: 'blur(8px)',
      }}
    >
      <Space direction="vertical" size="small">
        <Breadcrumb>
          <Breadcrumb.Item>
            <HomeOutlined /> 楼宇空间
          </Breadcrumb.Item>
          <Breadcrumb.Item>
            {current.icon} {current.title}
          </Breadcrumb.Item>
        </Breadcrumb>

        {currentView !== 'map' && (
          <Button
            type="primary"
            icon={<ArrowLeftOutlined />}
            onClick={onBackToMap}
            size="small"
          >
            返回地图
          </Button>
        )}
      </Space>
    </div>
  );
};

export default ViewportNavigation;
