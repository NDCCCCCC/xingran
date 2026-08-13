/**
 * 默认主题配置页面
 * Default Theme Configuration Page
 */

import { useState, useEffect } from "react";
import { App, Form, Select, Button, Card, Space, Divider, ColorPicker } from "antd";
import { SyncOutlined, SaveOutlined } from "@ant-design/icons";
import type { ThemeConfiguration } from "@/types/config";
import { getDefaultThemeConfig, setDefaultThemeConfig, syncUserThemeToDefault } from "@/lib/defaultThemeApi";
import { useSettingsStore } from "@/store/settingsStore";

/**
 * 主题模式选项
 */
const MODE_OPTIONS = [
  { label: "浅色", value: "light" },
  { label: "深色", value: "dark" },
  { label: "自动", value: "auto" },
];

/**
 * 主题风格选项
 */
const STYLE_OPTIONS = [
  { label: "简约", value: "minimal" },
  { label: "玻璃态", value: "glassmorphism" },
  { label: "新拟态", value: "neumorphism" },
  { label: "扁平化 2.0", value: "flat2.0" },
  { label: "奢华静雅", value: "luxury-quiet" },
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
          style: config.style,
          primaryColor: config.customColors?.primary,
          sidebarColor: config.customColors?.sidebar,
        });
      } catch (error) {
        message.error("加载默认主题配置失败");
        console.error("Failed to load default theme config:", error);
      } finally {
        setLoading(false);
      }
    };

    loadConfig();
  }, [form]);

  // 保存默认主题配置
  const handleSave = async (values: any) => {
    try {
      setLoading(true);

      // ColorPicker 返回的是 Color 对象,需要 toHexString() 转成 hex 字符串;
      // 空值或未选择时退化为 undefined,后端 bind 不会报错
      const toHex = (v: any): string | undefined => {
        if (!v) return undefined;
        if (typeof v === "string") return v;
        if (typeof v?.toHexString === "function") return v.toHexString();
        return undefined;
      };

      const config: ThemeConfiguration = {
        mode: values.mode,
        style: values.style,
        customColors: {
          primary: toHex(values.primaryColor),
          sidebar: toHex(values.sidebarColor),
        },
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
        style: config.style,
        primaryColor: config.customColors?.primary,
        sidebarColor: config.customColors?.sidebar,
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
      style: preferences.theme.style,
      primaryColor: preferences.theme.customColors?.primary,
      sidebarColor: preferences.theme.customColors?.sidebar,
    });
    message.info("已加载当前设置到表单，请点击保存按钮保存");
  };

  return (
    <Card
      title="默认主题配置"
      extra={
        <Space>
          <Button
            icon={<SyncOutlined />}
            onClick={handleSyncFromCurrentSettings}
          >
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
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSave}
      >
        <Form.Item
          label="主题模式"
          name="mode"
          rules={[{ required: true, message: "请选择主题模式" }]}
        >
          <Select options={MODE_OPTIONS} placeholder="请选择主题模式"  onSearch={() => {}}/>
        </Form.Item>

        <Form.Item
          label="主题风格"
          name="style"
          rules={[{ required: true, message: "请选择主题风格" }]}
        >
          <Select options={STYLE_OPTIONS} placeholder="请选择主题风格"  onSearch={() => {}}/>
        </Form.Item>

        <Divider>自定义颜色（可选）</Divider>

        <Form.Item
          label="主色调"
          name="primaryColor"
        >
          <ColorPicker showText />
        </Form.Item>

        <Form.Item
          label="侧边栏颜色"
          name="sidebarColor"
        >
          <ColorPicker showText />
        </Form.Item>

        <Form.Item style={{ marginTop: 24 }}>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              icon={<SaveOutlined />}
              loading={loading}
            >
              保存配置
            </Button>
            <Button onClick={() => form.resetFields()}>
              重置
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default DefaultThemePage;
