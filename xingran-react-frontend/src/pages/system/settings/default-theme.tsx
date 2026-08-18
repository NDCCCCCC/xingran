/**
 * 默认主题配置页面
 * Default Theme Configuration Page
 *
 * v1.22 Phase 65（D-01）：多主题移除后仅剩明暗模式配置；
 * 主题风格与自定义颜色入口不再存在。
 */

import { useState, useEffect } from "react";
import { App, Form, Select, Button, Card, Space } from "antd";
import { SyncOutlined, SaveOutlined } from "@ant-design/icons";
import type { ThemeConfiguration } from "@/lib/defaultThemeApi";
import {
  getDefaultThemeConfig,
  setDefaultThemeConfig,
  syncUserThemeToDefault,
} from "@/lib/defaultThemeApi";
import { useSettingsStore } from "@/store/settingsStore";

/**
 * 主题模式选项
 */
const MODE_OPTIONS = [
  { label: "浅色", value: "light" },
  { label: "深色", value: "dark" },
];

const DefaultThemePage: React.FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [currentUserId, setCurrentUserId] = useState<string | null>(null);

  // 获取当前用户ID（用于同步功能）
  useEffect(() => {
    // TODO: 从 auth store 或其他地方获取当前用户ID
    // 这里暂时使用硬编码的用户ID，实际应该从认证状态获取
    setCurrentUserId("chenchao-076");
  }, []);

  // 加载默认主题配置
  useEffect(() => {
    const loadConfig = async () => {
      try {
        setLoading(true);
        const config = await getDefaultThemeConfig();
        form.setFieldsValue({
          mode: config.mode,
        });
      } catch (error) {
        message.error("加载默认主题配置失败");
        console.error("Failed to load default theme config:", error);
      } finally {
        setLoading(false);
      }
    };

    loadConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [form]);

  // 保存默认主题配置
  const handleSave = async (values: { mode: ThemeConfiguration["mode"] }) => {
    try {
      setLoading(true);

      const config: ThemeConfiguration = {
        mode: values.mode,
      };

      await setDefaultThemeConfig(config);
      message.success("默认主题配置已更新");
    } catch (error) {
      message.error("保存默认主题配置失败");
      console.error("Failed to save default theme config:", error);
    } finally {
      setLoading(false);
    }
  };

  // 从当前用户配置同步
  const handleSync = async () => {
    if (!currentUserId) {
      message.warning("无法获取当前用户ID");
      return;
    }

    try {
      setSyncing(true);
      await syncUserThemeToDefault(currentUserId);

      // 重新加载配置
      const config = await getDefaultThemeConfig();
      form.setFieldsValue({
        mode: config.mode,
      });

      message.success("已从当前用户配置同步到默认主题");
    } catch (error) {
      message.error("同步失败");
      console.error("Failed to sync user theme:", error);
    } finally {
      setSyncing(false);
    }
  };

  // 从当前登录用户的设置同步（使用 settingsStore）
  const handleSyncFromCurrentSettings = () => {
    const preferences = useSettingsStore.getState().preferences;
    form.setFieldsValue({
      mode: preferences.theme.mode,
    });
    message.info("已加载当前设置到表单，请点击保存按钮保存");
  };

  return (
    <Card
      title="默认主题配置"
      extra={
        <Space>
          <Button icon={<SyncOutlined />} onClick={handleSyncFromCurrentSettings}>
            从当前设置加载
          </Button>
          <Button
            type="primary"
            icon={<SyncOutlined />}
            loading={syncing}
            onClick={handleSync}
            disabled={!currentUserId}
          >
            从用户 chenchao-076 同步
          </Button>
        </Space>
      }
    >
      <Form form={form} layout="vertical" onFinish={handleSave}>
        <Form.Item
          label="主题模式"
          name="mode"
          rules={[{ required: true, message: "请选择主题模式" }]}
        >
          <Select options={MODE_OPTIONS} placeholder="请选择主题模式" onSearch={() => {}} />
        </Form.Item>

        <Form.Item style={{ marginTop: 24 }}>
          <Space>
            <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={loading}>
              保存配置
            </Button>
            <Button onClick={() => form.resetFields()}>重置</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default DefaultThemePage;
