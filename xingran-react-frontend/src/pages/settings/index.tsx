/**
 * 用户设置页面（v1.22 收尾版 —— 仅明暗/布局/密度/数据设置）
 * User Settings Page (Light/Dark Mode, Layout, Density & Data)
 *
 * 多主题能力已随 D-01 移除：主题风格选择与颜色自定义入口不再存在，
 * 仅保留明暗模式（THEME-02）与布局/密度（THEME-03）。
 *
 * v1.22 收尾：移除"重置=获取管理员默认主题"逻辑 —— 后端默认主题页面已删除，
 * 无 sys.theme.default 来源。重置行为简化为"恢复表单到上一次保存的偏好"。
 */

import { useEffect, type FC } from "react";
import { App, Card, Form, Select, Switch, Button, Divider, Radio } from "antd";
import { SunOutlined, MoonOutlined } from "@ant-design/icons";
import { useSettingsStore } from "@/store/settingsStore";

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

  // 重置设置 - 恢复表单到上一次保存的用户偏好（v1.22：不再请求管理员默认主题 API）
  const handleReset = () => {
    form.resetFields();
    form.setFieldsValue(preferences);
  };

  if (!initialized) {
    return <div>加载中...</div>;
  }

  return (
    <div className="p-6">
      <Card title="用户设置" loading={loading}>
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
