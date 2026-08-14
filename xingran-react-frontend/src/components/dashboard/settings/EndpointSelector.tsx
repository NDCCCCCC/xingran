/**
 * API端点选择器组件
 * API Endpoint Selector Component
 *
 * 提供友好的下拉选择界面，让用户选择API端点而非手动输入
 */
import { useState, useEffect, useCallback } from "react";
import { Select, Tag, Space, Spin, Empty } from "antd";
import type { SelectProps } from "antd";
import {
  FileTextOutlined,
  CloudServerOutlined,
  MonitorOutlined,
  UserOutlined,
  ApartmentOutlined,
  TeamOutlined,
  BellOutlined,
  MenuOutlined,
  BookOutlined,
  LoginOutlined,
  FileSearchOutlined,
  ClockCircleOutlined,
  SettingOutlined,
  UsergroupAddOutlined,
} from "@ant-design/icons";
import { dashboardService } from "@/services/dashboardService";
import type { EndpointCategory, EndpointDetail, WidgetType } from "@/types/dashboard";

// 图标映射
const iconMap: Record<string, React.ComponentType> = {
  FileTextOutlined,
  CloudServerOutlined,
  MonitorOutlined,
  UserOutlined,
  ApartmentOutlined,
  TeamOutlined,
  BellOutlined,
  MenuOutlined,
  BookOutlined,
  LoginOutlined,
  FileSearchOutlined,
  ClockCircleOutlined,
  SettingOutlined,
  UsergroupAddOutlined,
};

interface EndpointSelectorProps extends Omit<
  SelectProps<any>,
  "options" | "children" | "onChange"
> {
  /** 选中的端点路由 */
  value?: string;
  /** 变化回调 */
  onChange?: (value: string, endpoint: EndpointDetail) => void;
  /** Widget类型（用于过滤支持的端点） */
  widgetType?: WidgetType;
  /** 是否在加载时自动获取端点列表 */
  autoLoad?: boolean;
}

/**
 * API端点选择器
 */
export const EndpointSelector: React.FC<EndpointSelectorProps> = ({
  value,
  onChange,
  widgetType,
  autoLoad = true,
  ...selectProps
}) => {
  const [loading, setLoading] = useState(false);
  const [categories, setCategories] = useState<EndpointCategory[]>([]);
  const [selectedEndpoint, setSelectedEndpoint] = useState<EndpointDetail | null>(null);

  // 加载端点列表
  const loadEndpoints = useCallback(async () => {
    setLoading(true);
    try {
      if (widgetType) {
        // 根据Widget类型过滤
        const data = await dashboardService.getEndpointsByWidgetType(widgetType);
        setCategories(data);
      } else {
        // 获取所有可用端点
        const data = await dashboardService.getAvailableEndpoints();
        setCategories(data);
      }
    } catch (error) {
      console.error("加载API端点列表失败:", error);
    } finally {
      setLoading(false);
    }
  }, [widgetType]);

  // 自动加载
  useEffect(() => {
    if (autoLoad) {
      loadEndpoints();
    }
  }, [loadEndpoints, autoLoad]);

  // 处理选择变化
  const handleChange = (route: string) => {
    // 查找选中的端点详情
    for (const category of categories) {
      const endpoint = category.endpoints.find((e) => e.route === route);
      if (endpoint) {
        setSelectedEndpoint(endpoint);
        onChange?.(route, endpoint);
        break;
      }
    }
  };

  // 获取方法标签颜色
  const getMethodColor = (method: string): string => {
    switch (method.toUpperCase()) {
      case "GET":
        return "blue";
      case "POST":
        return "green";
      case "PUT":
        return "orange";
      case "DELETE":
        return "red";
      default:
        return "default";
    }
  };

  // 渲染图标
  const renderIcon = (iconName: string) => {
    const IconComponent = iconMap[iconName];
    return IconComponent ? <IconComponent /> : null;
  };

  // 渲染选项
  const renderOptions = () => {
    if (loading) {
      return [
        <Select.Option key="loading" disabled value="">
          <Space>
            <Spin size="small" />
            <span>加载中...</span>
          </Space>
        </Select.Option>,
      ];
    }

    if (!categories || categories.length === 0) {
      return [
        <Select.Option key="empty" disabled value="">
          <Empty description="暂无可用端点" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        </Select.Option>,
      ];
    }

    return categories.map((category) => (
      <Select.OptGroup
        key={category.module}
        label={
          <Space>
            {renderIcon(category.icon)}
            <span>{category.category}</span>
          </Space>
        }
      >
        {category.endpoints.map((endpoint) => (
          <Select.Option
            key={`${endpoint.method}:${endpoint.route}`}
            value={endpoint.route}
            label={endpoint.displayName}
          >
            <Space style={{ justifyContent: "space-between", width: "100%" }}>
              <Space>
                <Tag color={getMethodColor(endpoint.method)} variant="filled">
                  {endpoint.method}
                </Tag>
                <span>{endpoint.displayName}</span>
              </Space>
              <span style={{ color: "var(--theme-text-tertiary, #999)", fontSize: 12 }}>
                {endpoint.route}
              </span>
            </Space>
          </Select.Option>
        ))}
      </Select.OptGroup>
    ));
  };

  return (
    <Select
      value={value}
      onChange={handleChange}
      loading={loading}
      placeholder="选择API端点"
      showSearch
      optionFilterProp="label"
      filterOption={(input, option) => {
        // 支持搜索显示名称和路由
        const label = (option?.label as string) || "";
        const route = (option?.value as string) || "";
        return (
          label.toLowerCase().includes(input.toLowerCase()) ||
          route.toLowerCase().includes(input.toLowerCase())
        );
      }}
      {...selectProps}
      onSearch={() => {}}
    >
      {renderOptions()}
    </Select>
  );
};

export default EndpointSelector;
