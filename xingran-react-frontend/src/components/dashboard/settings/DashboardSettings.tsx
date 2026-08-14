/**
 * DashboardSettings - 仪表盘设置侧边抽屉
 *
 * 实现仪表盘设置编辑功能，包括名称、描述、权限范围和刷新间隔配置
 * 从右侧滑出，可同时看到仪表盘内容变化
 */

import { useState, useEffect } from "react";
import { App, Drawer, Form, Input, Button, Spin, Divider } from "antd";
import { SaveOutlined, CloseOutlined } from "@ant-design/icons";
import { useDashboardStore } from "@/store/dashboardStore";
import DashboardScopeSelector from "./DashboardScopeSelector";
import RefreshIntervalSelector from "./RefreshIntervalSelector";
import type { DashboardScope, UpdateDashboardRequest } from "@/types/dashboard";

const { TextArea } = Input;

export interface DashboardSettingsProps {
  /** 是否可见 */
  visible: boolean;
  /** 关闭回调 */
  onClose: () => void;
}

/**
 * 仪表盘设置侧边抽屉组件
 */
export const DashboardSettings: React.FC<DashboardSettingsProps> = ({ visible, onClose }) => {
  const { message } = App.useApp();
  const { currentDashboard, updateDashboard } = useDashboardStore();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [scopeConfig, setScopeConfig] = useState<{
    scope: DashboardScope;
    deptId?: string;
    isSystem?: boolean;
  }>({
    scope: "private",
  });

  // 当抽屉打开时初始化表单
  useEffect(() => {
    if (visible && currentDashboard) {
      form.setFieldsValue({
        name: currentDashboard.name,
        description: currentDashboard.description || "",
        refreshInterval: currentDashboard.refreshInterval,
      });
      setScopeConfig({
        scope: currentDashboard.scope || "private",
        deptId: currentDashboard.deptId,
        isSystem: currentDashboard.isSystem,
      });
    }
  }, [visible, currentDashboard, form]);

  // 处理保存
  const handleSave = async () => {
    if (!currentDashboard) {
      message.warning("当前没有活动的仪表盘");
      return;
    }

    try {
      setLoading(true);
      const values = await form.validateFields();

      const updateData: UpdateDashboardRequest = {
        name: values.name,
        description: values.description,
        refreshInterval: values.refreshInterval,
      };

      // 更新仪表盘
      await updateDashboard(currentDashboard.id, updateData);

      message.success("仪表盘设置已保存");
      onClose();
    } catch (error) {
      console.error("保存仪表盘设置失败:", error);
      if (error instanceof Error) {
        message.error(`保存失败: ${error.message}`);
      }
    } finally {
      setLoading(false);
    }
  };

  // 处理取消
  const handleCancel = () => {
    form.resetFields();
    onClose();
  };

  // 处理权限范围变化
  const handleScopeChange = (value: {
    scope: DashboardScope;
    deptId?: string;
    isSystem?: boolean;
  }) => {
    setScopeConfig(value);
  };

  if (!currentDashboard) {
    return null;
  }

  return (
    <Drawer
      title="仪表盘设置"
      placement="right"
      open={visible}
      onClose={handleCancel}
      width={480}
      footer={
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 8 }}>
          <Button icon={<CloseOutlined />} onClick={handleCancel}>
            取消
          </Button>
          <Button type="primary" icon={<SaveOutlined />} loading={loading} onClick={handleSave}>
            保存
          </Button>
        </div>
      }
    >
      <Spin spinning={loading}>
        <Form form={form} layout="vertical" initialValues={currentDashboard}>
          {/* 基本信息 */}
          <Divider titlePlacement="left" plain>
            基本信息
          </Divider>

          <Form.Item
            label="仪表盘名称"
            name="name"
            rules={[
              { required: true, message: "请输入仪表盘名称" },
              { max: 100, message: "名称不能超过 100 个字符" },
            ]}
          >
            <Input placeholder="请输入仪表盘名称" />
          </Form.Item>

          <Form.Item
            label="描述"
            name="description"
            rules={[{ max: 500, message: "描述不能超过 500 个字符" }]}
          >
            <TextArea rows={3} placeholder="请输入仪表盘描述（可选）" />
          </Form.Item>

          {/* 权限设置 */}
          <Divider titlePlacement="left" plain>
            权限设置
          </Divider>

          <Form.Item>
            <DashboardScopeSelector value={scopeConfig} onChange={handleScopeChange} />
          </Form.Item>

          {/* 刷新设置 */}
          <Divider titlePlacement="left" plain>
            刷新设置
          </Divider>

          <Form.Item
            label="刷新间隔"
            name="refreshInterval"
            tooltip="仪表盘数据自动刷新的时间间隔"
            initialValue={300}
          >
            <RefreshIntervalSelector />
          </Form.Item>

          {/* 仪表盘信息 */}
          <Divider titlePlacement="left" plain>
            仪表盘信息
          </Divider>

          <div style={{ color: "var(--theme-text-tertiary, #666)", fontSize: 12 }}>
            <p>
              <strong>ID:</strong> {currentDashboard.id}
            </p>
            <p>
              <strong>创建者:</strong>{" "}
              {currentDashboard.ownerName || currentDashboard.createdBy || "-"}
            </p>
            <p>
              <strong>创建时间:</strong> {currentDashboard.createdAt}
            </p>
            <p>
              <strong>更新时间:</strong> {currentDashboard.updatedAt}
            </p>
            {currentDashboard.isDefault && (
              <p style={{ color: "var(--theme-info, #1890ff)" }}>
                <strong>★ 这是默认仪表盘</strong>
              </p>
            )}
            {currentDashboard.isTemplate && (
              <p style={{ color: "var(--theme-success, #52c41a)" }}>
                <strong>📋 这是一个模板</strong>
              </p>
            )}
          </div>
        </Form>
      </Spin>
    </Drawer>
  );
};

export default DashboardSettings;
