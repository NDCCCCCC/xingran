/**
 * LayoutToolbar - 仪表盘布局工具栏
 *
 * 提供仪表盘的操作按钮和工具
 */

import { useCallback, useState, useRef, useEffect } from "react";
import {
  EditOutlined,
  EyeOutlined,
  SaveOutlined,
  ReloadOutlined,
  SettingOutlined,
  AppstoreAddOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import { App, Button, Space, Modal } from "antd";
import { useNavigate } from "react-router-dom";
import { useDashboardStore } from "@/store/dashboardStore";
import type { DashboardViewMode } from "@/store/dashboardStore";
import { DASHBOARD } from "@/constants/routes";

import "./LayoutToolbar.css";

interface LayoutToolbarProps {
  /** 仪表盘ID */
  dashboardId?: string;

  /** 是否显示返回按钮 */
  showBackButton?: boolean;

  /** 自定义操作按钮 */
  extraActions?: React.ReactNode;

  /** 添加Widget回调 */
  onAddWidget?: () => void;
}

export const LayoutToolbar: React.FC<LayoutToolbarProps> = ({
  dashboardId,
  showBackButton = false,
  extraActions,
  onAddWidget,
}) => {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [saving, setSaving] = useState(false);

  const {
    viewMode,
    setViewMode,
    hasUnsavedChanges,
    saveCurrentDashboard,
    resetCurrentDashboard,
    currentDashboard,
  } = useDashboardStore();

  // 使用 ref 存储最新的 viewMode 和 setViewMode，遵循 Vercel React Best Practices: rerender-defer-reads
  const viewModeRef = useRef(viewMode);
  const setViewModeRef = useRef(setViewMode);
  useEffect(() => {
    viewModeRef.current = viewMode;
    setViewModeRef.current = setViewMode;
  }, [viewMode, setViewMode]);

  // 切换视图模式 - 使用 ref 避免依赖变化导致回调重新创建
  const toggleViewMode = useCallback(() => {
    const currentMode = viewModeRef.current;
    const newMode: DashboardViewMode = currentMode === "view" ? "edit" : "view";

    if (newMode === "edit" && dashboardId) {
      // 如果有dashboardId，导航到编辑页面（使用 query 参数）
      navigate(`${DASHBOARD}/${dashboardId}?mode=edit`);
    } else {
      // 否则切换viewMode（用于编辑页面内的预览功能）
      setViewModeRef.current(newMode);
      if (newMode === "view") {
        message.success("已切换到查看模式");
      } else {
        message.info("已切换到编辑模式");
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dashboardId, navigate]); // 减少依赖，只保留必要的 dashboardId 和 navigate

  // 保存仪表盘
  const handleSave = async () => {
    if (!hasUnsavedChanges) {
      message.info("没有需要保存的更改");
      return;
    }

    setSaving(true);
    try {
      await saveCurrentDashboard();
      message.success("保存成功");
    } catch (error) {
      message.error(`保存失败: ${(error as Error).message}`);
    } finally {
      setSaving(false);
    }
  };

  // 重置仪表盘
  const handleReset = async () => {
    if (!hasUnsavedChanges) {
      message.info("没有未保存的更改");
      return;
    }

    Modal.confirm({
      title: "确认重置",
      content: "确定要重置仪表盘吗？所有未保存的更改将丢失。",
      okText: "确定",
      cancelText: "取消",
      onOk: async () => {
        try {
          await resetCurrentDashboard();
          message.success("已重置仪表盘");
        } catch (error) {
          message.error(`重置失败: ${(error as Error).message}`);
        }
      },
    });
  };

  // 添加Widget
  const handleAddWidget = () => {
    if (onAddWidget) {
      onAddWidget();
    } else {
      // TODO: 打开Widget选择器
      message.info("Widget选择器功能待实现");
    }
  };

  // 打开设置
  const handleOpenSettings = () => {
    // TODO: 打开仪表盘设置
    message.info("仪表盘设置功能待实现");
  };

  // 返回
  const handleBack = () => {
    if (hasUnsavedChanges) {
      Modal.confirm({
        title: "确认离开",
        content: "您有未保存的更改，确定要离开吗？",
        okText: "离开",
        cancelText: "取消",
        onOk: () => {
          navigate(DASHBOARD);
        },
      });
    } else {
      navigate(DASHBOARD);
    }
  };

  return (
    <div className="layout-toolbar">
      <div className="layout-toolbar__left">
        {showBackButton && (
          <Button icon={<ArrowLeftOutlined />} onClick={handleBack}>
            返回
          </Button>
        )}
        {currentDashboard && <h2 className="layout-toolbar__title">{currentDashboard.name}</h2>}
      </div>

      <div className="layout-toolbar__center">{extraActions}</div>

      <div className="layout-toolbar__right">
        <Space>
          {viewMode === "edit" && (
            <>
              <Button icon={<AppstoreAddOutlined />} onClick={handleAddWidget}>
                添加Widget
              </Button>
              {hasUnsavedChanges && (
                <Button icon={<ReloadOutlined />} onClick={handleReset}>
                  重置
                </Button>
              )}
              <Button
                type="primary"
                icon={<SaveOutlined />}
                loading={saving}
                disabled={!hasUnsavedChanges}
                onClick={handleSave}
              >
                保存
              </Button>
              <Button icon={<EyeOutlined />} onClick={toggleViewMode}>
                预览
              </Button>
            </>
          )}
          {viewMode === "view" && (
            <>
              <Button icon={<SettingOutlined />} onClick={handleOpenSettings}>
                设置
              </Button>
              <Button type="primary" icon={<EditOutlined />} onClick={toggleViewMode}>
                编辑
              </Button>
            </>
          )}
        </Space>
      </div>
    </div>
  );
};
