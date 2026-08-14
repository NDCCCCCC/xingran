/**
 * BaseWidget - Widget 基类组件
 *
 * 所有Widget组件的基础类，提供通用功能和数据获取
 */

import React, { type ReactNode, useState, useCallback, useEffect } from "react";
import { Card, Space, Dropdown, Button, Tooltip, Spin, Empty, Result, Skeleton } from "antd";
import {
  MoreOutlined,
  SettingOutlined,
  ReloadOutlined,
  CloseOutlined,
  DragOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import type { WidgetConfig } from "@/types/dashboard";
import { useDashboardStore } from "@/store/dashboardStore";
import { useWidgetData } from "@/hooks/useWidgetData";

import "./BaseWidget.css";

// 导出BaseWidgetProps类型
export interface BaseWidgetProps {
  /** Widget 配置 */
  widget: WidgetConfig;

  /** Widget 数据（可选，如果提供则不自动获取） */
  data?: unknown;

  /** 加载状态 */
  loading?: boolean;

  /** 错误信息 */
  error?: string | null;

  /** 是否为空数据 */
  empty?: boolean;

  /** 空数据提示信息 */
  emptyMessage?: string;

  /** 子元素 */
  children: ReactNode;

  /** 编辑回调 */
  onEdit?: () => void;

  /** 删除回调 */
  onDelete?: () => void;

  /** 刷新回调 */
  onRefresh?: () => void;

  /** 禁用自动数据获取（子组件自行获取数据） */
  disableDataFetch?: boolean;

  /** 是否首次加载（显示骨架屏） */
  isInitialLoad?: boolean;
}

export const BaseWidget: React.FC<BaseWidgetProps> = ({
  widget,
  data: externalData,
  loading: externalLoading,
  error: externalError,
  empty: externalEmpty = false,
  emptyMessage = "暂无数据",
  children,
  onEdit,
  onDelete,
  onRefresh: externalOnRefresh,
  disableDataFetch = false,
  isInitialLoad = false,
}) => {
  const { viewMode, selectWidget, selectedWidgetId } = useDashboardStore();
  const [isHovered, setIsHovered] = useState(false);
  const [isFirstLoad, setIsFirstLoad] = useState(true);

  // 自动获取数据（如果没有提供外部数据且未禁用自动获取）
  const {
    data: internalData,
    loading: internalLoading,
    error: internalError,
    refresh,
  } = useWidgetData(
    widget,
    { disabled: !!externalData || disableDataFetch } // 如果提供了外部数据或禁用自动获取，则不获取
  );

  // 使用外部或内部的状态
  const data = externalData ?? internalData;
  const loading = externalLoading ?? internalLoading;
  const error = externalError ?? internalError;
  const onRefresh = externalOnRefresh ?? refresh;

  // 检测数据是否为空
  const isEmpty =
    externalEmpty ||
    (!loading &&
      !error &&
      (data === null ||
        data === undefined ||
        (Array.isArray(data) && data.length === 0) ||
        (typeof data === "object" && data !== null && Object.keys(data).length === 0)));

  // 首次加载完成后更新状态
  useEffect(() => {
    if (!loading && isFirstLoad) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setIsFirstLoad(false);
    }
  }, [loading, isFirstLoad]);

  const isSelected = selectedWidgetId === widget.id;
  const isEditable = viewMode === "edit";
  const showSkeleton = (isInitialLoad || isFirstLoad) && loading;

  // 点击处理
  const handleClick = useCallback(() => {
    if (isEditable) {
      selectWidget(isSelected ? null : widget.id);
    }
  }, [isEditable, isSelected, widget.id, selectWidget]);

  // 更多操作菜单
  const menuItems: MenuProps["items"] = [
    {
      key: "refresh",
      icon: <ReloadOutlined />,
      label: "刷新",
      onClick: () => onRefresh?.(),
    },
  ];

  if (isEditable) {
    menuItems.push(
      {
        key: "edit",
        icon: <SettingOutlined />,
        label: "编辑",
        onClick: onEdit,
      },
      {
        type: "divider",
      },
      {
        key: "delete",
        icon: <CloseOutlined />,
        label: "删除",
        danger: true,
        onClick: onDelete,
      }
    );
  }

  return (
    <div
      className={`base-widget ${isSelected ? "base-widget--selected" : ""} ${isEditable ? "base-widget--editable" : ""} ${loading ? "base-widget--loading" : ""}`}
      onClick={handleClick}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <Card
        title={
          <div className="base-widget__title-wrapper">
            {isEditable && (
              <Tooltip title="拖拽移动">
                <DragOutlined className="widget-drag-handle" />
              </Tooltip>
            )}
            <span>{widget.title}</span>
          </div>
        }
        extra={
          <Space size="small">
            {loading && <ReloadOutlined spin />}
            {(isHovered || isSelected) && (
              <Dropdown menu={{ items: menuItems }} trigger={["click"]}>
                <Button
                  type="text"
                  size="small"
                  icon={<MoreOutlined />}
                  onClick={(e) => e.stopPropagation()}
                />
              </Dropdown>
            )}
          </Space>
        }
        variant="borderless"
        className="base-widget__card"
        data-widget-id={widget.id}
        styles={{
          body: {
            height: "calc(100% - 40px)",
            overflow: "auto",
          },
        }}
      >
        {/* 骨架屏 - 首次加载 */}
        {showSkeleton && (
          <div className="base-widget__skeleton">
            <Skeleton active paragraph={{ rows: 4 }} />
          </div>
        )}

        {/* 错误状态 */}
        {!showSkeleton && error && (
          <div className="base-widget__error-container">
            <Result
              status="error"
              title="加载失败"
              subTitle={error}
              extra={
                <Button type="primary" icon={<ReloadOutlined />} onClick={() => onRefresh?.()}>
                  重试
                </Button>
              }
            />
          </div>
        )}

        {/* 空数据状态 */}
        {!showSkeleton && !error && isEmpty && (
          <div className="base-widget__empty">
            <Empty description={emptyMessage} image={Empty.PRESENTED_IMAGE_SIMPLE} />
          </div>
        )}

        {/* 正常内容 */}
        {!showSkeleton && !error && !isEmpty && (
          <>
            <Spin spinning={loading}>{children}</Spin>
            {loading && (
              <div style={{ marginTop: 8, textAlign: "center", color: "rgba(0, 0, 0, 0.45)" }}>
                加载中...
              </div>
            )}
          </>
        )}
      </Card>
    </div>
  );
};

/**
 * BaseWidgetHeader - Widget 头部组件
 */
export const BaseWidgetHeader: React.FC<{
  title: string;
  icon?: ReactNode;
  extra?: ReactNode;
}> = ({ title, icon, extra }) => {
  return (
    <div className="base-widget-header">
      {icon && <div className="base-widget-header__icon">{icon}</div>}
      <h3 className="base-widget-header__title">{title}</h3>
      {extra && <div className="base-widget-header__extra">{extra}</div>}
    </div>
  );
};

/**
 * BaseWidgetContent - Widget 内容组件
 */
export const BaseWidgetContent: React.FC<{
  children: ReactNode;
  loading?: boolean;
  error?: string;
  empty?: boolean;
  emptyMessage?: string;
}> = ({ children, loading, error, empty, emptyMessage = "暂无数据" }) => {
  if (loading) {
    return <div className="base-widget-content base-widget-content--loading">加载中...</div>;
  }

  if (error) {
    return <div className="base-widget-content base-widget-content--error">{error}</div>;
  }

  if (empty) {
    return <div className="base-widget-content base-widget-content--empty">{emptyMessage}</div>;
  }

  return <div className="base-widget-content">{children}</div>;
};

/**
 * BaseWidgetActions - Widget 操作按钮组件
 */
export const BaseWidgetActions: React.FC<{
  onEdit?: () => void;
  onDelete?: () => void;
  onRefresh?: () => void;
  showActions?: boolean;
}> = ({ onEdit, onDelete, onRefresh, showActions = true }) => {
  if (!showActions) return null;

  return (
    <div className="base-widget-actions">
      <Space size="small">
        {onRefresh && (
          <Button type="text" size="small" icon={<ReloadOutlined />} onClick={onRefresh}>
            刷新
          </Button>
        )}
        {onEdit && (
          <Button type="text" size="small" icon={<SettingOutlined />} onClick={onEdit}>
            编辑
          </Button>
        )}
        {onDelete && (
          <Button type="text" size="small" danger icon={<CloseOutlined />} onClick={onDelete}>
            删除
          </Button>
        )}
      </Space>
    </div>
  );
};
