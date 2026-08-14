/**
 * 默认仪表盘视图
 * 优先级：用户默认 > 系统仪表盘 > 欢迎页
 */
import { useEffect, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { useDashboardStore } from "@/store/dashboardStore";
import { DashboardView } from "./DashboardView";
import { Empty, Button, Space, Tag } from "antd";
import { PlusOutlined, UnorderedListOutlined } from "@ant-design/icons";
import { DASHBOARD } from "@/constants/routes";

interface DashboardHomeProps {
  isEmbedded?: boolean; // 是否作为嵌入视图显示
}

export const DashboardHome: React.FC<DashboardHomeProps> = ({ isEmbedded = false }) => {
  const navigate = useNavigate();
  const { defaultDashboard, defaultDashboardLoading, fetchDefaultDashboard } = useDashboardStore();

  useEffect(() => {
    fetchDefaultDashboard();
  }, [fetchDefaultDashboard]);

  if (defaultDashboardLoading) {
    return (
      <div style={{ padding: "48px", textAlign: "center" }}>
        <div>加载中...</div>
      </div>
    );
  }

  if (!defaultDashboard) {
    return (
      <div style={{ padding: "48px", textAlign: "center" }}>
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <div>
              <div style={{ fontSize: 16, marginBottom: 8 }}>您还没有设置默认仪表盘</div>
              <div style={{ color: "var(--theme-text-tertiary, #999)" }}>
                创建仪表盘或从列表中选择一个设为默认
              </div>
            </div>
          }
        >
          <Space>
            <Button type="primary" icon={<PlusOutlined />}>
              创建仪表盘
            </Button>
            <Button
              icon={<UnorderedListOutlined />}
              onClick={() => navigate(`${DASHBOARD}?mode=list`)}
            >
              查看仪表盘列表
            </Button>
          </Space>
        </Empty>
      </div>
    );
  }

  // 显示系统仪表盘标识
  const isSystem = defaultDashboard.isSystem;

  return (
    <div>
      {isSystem && !isEmbedded && (
        <div style={{ marginBottom: 16 }}>
          <Tag color="blue">系统仪表盘</Tag>
        </div>
      )}
      <DashboardView dashboardId={defaultDashboard.id} isHome={true} />
    </div>
  );
};
