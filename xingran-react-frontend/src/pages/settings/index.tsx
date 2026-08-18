/**
 * 用户设置页面（v1.22 Phase 65 收敛版 —— 仅明暗/布局/密度/数据设置）
 * User Settings Page (Light/Dark Mode, Layout, Density & Data)
 *
 * 多主题能力已随 D-01 移除：主题风格选择与颜色自定义入口不再存在，
 * 仅保留明暗模式（THEME-02）与布局/密度（THEME-03）。
 */

import { useEffect, type FC } from "react";
import { App, Card, Form, Select, Switch, Button, Divider, Alert, Radio } from "antd";
import { SunOutlined, MoonOutlined } from "@ant-design/icons";
import { useSettingsStore } from "@/store/settingsStore";
import { getDefaultThemeConfig } from "@/lib/defaultThemeApi";

const { Option } = Select;

const SettingsPage: FC = () => {
  const { message } = App.useApp();
  const { preferences, loading, initialized, updatePreferences } = useSettingsStore();
  const [form] = Form.useForm();

  // 加载设置
  useEffect(() => {
    if (!initialized) {
      useSettingsStore.getState().initialize();
    }
  }, [initialized]);

  // 同步表单值
  useEffect(() => {
    if (initialized && preferences) {
      form.setFieldsValue(preferences);
    }
  }, [initialized, preferences, form]);

  // 保存设置
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      await updatePreferences(values);
      message.success("设置保存成功");
    } catch (_error) {
      message.error("设置保存失败");
    }
  };

  // 重置设置 - 从后端获取管理员配置的默认主题，覆盖当前用户偏好
  const handleReset = async () => {
    try {
      // 获取管理员配置的默认主题（Phase 65 后仅含 mode）
      const defaultTheme = await getDefaultThemeConfig();

      // 用默认明暗模式覆盖表单的主题字段
      form.resetFields();
      form.setFieldsValue({
        ...preferences,
        theme: {
          ...preferences.theme,
          mode: defaultTheme.mode,
        },
      });
    } catch {
      // 获取失败时回退到当前用户偏好
      form.resetFields();
      form.setFieldsValue(preferences);
      message.warning("获取系统默认主题失败，已重置为当前保存值");
    }
  };

  if (!initialized) {
    return <div>加载中...</div>;
  }

  return (
    <div className="p-6">
      <Card title="用户设置" loading={loading}>
        <Alert
          title="配置说明"
          description="所有配置会自动保存到服务器，并在您下次登录时恢复。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form form={form} layout="vertical" initialValues={preferences}>
          <Divider styles={{ content: { margin: 0 } }}>界面设置</Divider>

          {/* 主题模式 - 明暗切换 */}
          <Form.Item
            label="明暗模式"
            name={["theme", "mode"]}
            tooltip="选择系统的颜色模式"
            rules={[{ required: true, message: "请选择明暗模式" }]}
          >
            <Radio.Group>
              <Radio.Button value="light">
                <SunOutlined /> 浅色模式
              </Radio.Button>
              <Radio.Button value="dark">
                <MoonOutlined /> 深色模式
              </Radio.Button>
            </Radio.Group>
          </Form.Item>

          <Divider styles={{ content: { margin: 0 } }}>布局设置</Divider>

          {/* 布局类型 */}
          <Form.Item
            label="布局类型"
            name={["layout", "type"]}
            tooltip="选择系统的布局方式"
            rules={[{ required: true, message: "请选择布局类型" }]}
          >
            <Select onSearch={() => {}}>
              <Option value="classic">经典布局</Option>
              <Option value="hybrid">混合布局</Option>
              <Option value="innovative">创新布局</Option>
            </Select>
          </Form.Item>

          {/* 密度模式 */}
          <Form.Item
            label="密度模式"
            name={["layout", "density"]}
            tooltip="选择界面的紧凑程度"
            rules={[{ required: true, message: "请选择密度模式" }]}
          >
            <Select onSearch={() => {}}>
              <Option value="compact">紧凑</Option>
              <Option value="comfortable">舒适</Option>
              <Option value="spacious">宽松</Option>
            </Select>
          </Form.Item>

          {/* 侧边栏折叠 */}
          <Form.Item
            label="默认折叠侧边栏"
            name={["layout", "sidebar", "collapsed"]}
            valuePropName="checked"
            tooltip="默认折叠侧边栏以节省空间"
          >
            <Switch />
          </Form.Item>

          <Divider styles={{ content: { margin: 0 } }}>数据设置</Divider>

          {/* 默认分页大小 */}
          <Form.Item
            label="默认分页大小"
            name={["data", "defaultPageSize"]}
            tooltip="列表页面默认每页显示的数据条数"
            rules={[{ required: true, message: "请选择默认分页大小" }]}
          >
            <Select onSearch={() => {}}>
              <Option value={10}>10 条/页</Option>
              <Option value={20}>20 条/页</Option>
              <Option value={50}>50 条/页</Option>
              <Option value={100}>100 条/页</Option>
            </Select>
          </Form.Item>

          <Form.Item>
            <Button type="primary" onClick={handleSave} className="mr-2">
              保存设置
            </Button>
            <Button onClick={handleReset}>重置</Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default SettingsPage;
