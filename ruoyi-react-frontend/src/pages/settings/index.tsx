import React, { useEffect } from 'react';
import {
  Card,
  Form,
  Select,
  Switch,
  Button,
  message,
  Divider,
} from 'antd';
import { useSettingsStore } from '@/store/settingsStore';

const { Option } = Select;

const SettingsPage: React.FC = () => {
  const { preferences, loading, loadPreferences, updatePreferences } = useSettingsStore();
  const [form] = Form.useForm();

  // 加载设置
  useEffect(() => {
    loadPreferences();
  }, [loadPreferences]);

  // 表单值更新时同步到 store
  const handleFormChange = (_: any, allValues: any) => {
    // 仅在用户手动修改时更新，不触发 API
    form.setFieldsValue(allValues);
  };

  // 保存设置
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      await updatePreferences(values);
      message.success('设置保存成功');
    } catch (error) {
      message.error('设置保存失败');
    }
  };

  // 重置设置
  const handleReset = () => {
    form.resetFields();
    form.setFieldsValue(preferences);
  };

  return (
    <div className="p-6">
      <Card title="系统设置" loading={loading}>
        <Form
          form={form}
          layout="vertical"
          initialValues={preferences}
          onValuesChange={handleFormChange}
        >
          <Divider orientation="left">界面设置</Divider>

          <Form.Item
            label="主题"
            name="theme"
            tooltip="选择系统界面主题"
          >
            <Select>
              <Option value="light">浅色主题</Option>
              <Option value="dark">深色主题</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="语言"
            name="language"
            tooltip="选择系统显示语言"
          >
            <Select>
              <Option value="zh-CN">简体中文</Option>
              <Option value="en-US">English</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="侧边栏折叠"
            name="sidebarCollapsed"
            valuePropName="checked"
            tooltip="默认折叠侧边栏以节省空间"
          >
            <Switch />
          </Form.Item>

          <Divider orientation="left">数据设置</Divider>

          <Form.Item
            label="默认分页大小"
            name="pageSize"
            tooltip="列表页面默认每页显示的数据条数"
          >
            <Select>
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
            <Button onClick={handleReset}>
              重置
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default SettingsPage;
