import { useState, useEffect } from "react";
import type { FC } from "react";
import {
  Card,
  Form,
  Switch,
  TimePicker,
  Select,
  Button,
  App,
  Space,
  Alert,
  Divider,
  Descriptions,
} from "antd";
// Phase 71 (W5)：裸 Input/Select → 契约组件（children 形态改写 options / 复合子节点保留写法）
import { AppInput, AppSelect } from "@/components/ui";
import { SaveOutlined, ReloadOutlined, BellOutlined, SettingOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import { formatDateTime, FORMAT_TIME_MINUTE } from "@/utils/datetime";
import { getDutyConfig, updateDutyConfig, type DutyConfig } from "@/lib/dutyApi";
import { isFormValidationError } from "@/utils/errorHandler";

// 形态分类器⑤专用：提醒渠道下拉为「children + 复合子节点（Space + 图标）」，
// 按 W1/W2 口径保留 children 写法（不改写为 options），故仍需 antd Select.Option。
// 注意：本文件已无裸 <Select> JSX（P65 selector 按 JSXOpeningElement 名匹配，
// Select.Option 为成员表达式，不在命中面），Select 仅用于取 Option。
const { Option } = Select;

const DutyConfigPage: FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<DutyConfig | null>(null);

  const fetchConfig = async () => {
    setLoading(true);
    try {
      const result = await getDutyConfig();
      const configData = result as { code: number; data: DutyConfig };
      setConfig(configData.data);
      form.setFieldsValue({
        reminderEnabled: configData.data.reminderEnabled,
        reminderTime: dayjs(configData.data.reminderTime, "HH:mm"),
        reminderChannels: configData.data.reminderChannels.split(","),
        beforeReminderMinutes: configData.data.beforeReminderMinutes,
      });
    } catch (_error) {
      message.error("获取配置失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfig();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- load/fetch fn from hook is stable enough; disable to avoid loop
  }, []);

  const handleSubmit = async () => {
    try {
      setSaving(true);
      const values = await form.validateFields();
      const data = {
        reminderEnabled: values.reminderEnabled,
        reminderTime: formatDateTime(values.reminderTime, FORMAT_TIME_MINUTE),
        reminderChannels: Array.isArray(values.reminderChannels)
          ? values.reminderChannels.join(",")
          : values.reminderChannels,
        beforeReminderMinutes: values.beforeReminderMinutes,
      };
      await updateDutyConfig(data);
      message.success("配置保存成功");
      fetchConfig();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      message.error("配置保存失败");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <Card
        title={
          <Space>
            <SettingOutlined />
            <span>值班提醒配置</span>
          </Space>
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={fetchConfig} loading={loading}>
              刷新
            </Button>
            <Button type="primary" icon={<SaveOutlined />} onClick={handleSubmit} loading={saving}>
              保存配置
            </Button>
          </Space>
        }
      >
        <Alert
          title="配置说明"
          description="在此处配置值班提醒功能。启用后，系统将在指定时间通过选择的渠道提醒当日值班人员。"
          type="info"
          showIcon
          closable
          className="mb-6"
        />

        {config && (
          <div className="mb-6">
            <Descriptions title="当前配置概览" bordered column={1}>
              <Descriptions.Item label="提醒状态">
                {config.reminderEnabled ? (
                  <span className="text-green-600">已启用</span>
                ) : (
                  <span className="text-red-600">已禁用</span>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="提醒时间">{config.reminderTime}</Descriptions.Item>
              <Descriptions.Item label="提醒渠道">{config.reminderChannels}</Descriptions.Item>
              <Descriptions.Item label="提前提醒">
                {config.beforeReminderMinutes ? `${config.beforeReminderMinutes} 分钟` : "当天提醒"}
              </Descriptions.Item>
            </Descriptions>
          </div>
        )}

        <Divider />

        <Form
          form={form}
          layout="vertical"
          initialValues={{ reminderEnabled: true, reminderChannels: ["websocket"] }}
        >
          <Form.Item label="启用值班提醒" name="reminderEnabled" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>

          <Form.Item
            label="提醒时间"
            name="reminderTime"
            tooltip="系统将在每天的此时发送值班提醒"
            rules={[{ required: true, message: "请选择提醒时间" }]}
          >
            <TimePicker
              format="HH:mm"
              placeholder="请选择提醒时间"
              style={{ width: 200 }}
              showNow={false}
            />
          </Form.Item>

          <Form.Item
            label="提醒渠道"
            name="reminderChannels"
            tooltip="可选择多种提醒方式"
            rules={[{ required: true, message: "请选择提醒渠道" }]}
          >
            {/* 形态分类器⑤⑦：children + 复合子节点（Space/图标），保留写法；
                mode="multiple" 显式 showSearch（antd 多选默认开搜索）；
                显式 optionFilterProp="label" —— 每个 Option 补 label 属性，
                否则契约过滤指向不存在字段、过滤失效 */}
            <AppSelect
              allowClear
              mode="multiple"
              placeholder="请选择提醒渠道"
              showSearch
              optionFilterProp="label"
              onSearch={() => {}}
            >
              <Option value="websocket" label="站内通知（WebSocket）">
                <Space>
                  <BellOutlined />
                  站内通知（WebSocket）
                </Space>
              </Option>
              <Option value="email" label="邮件通知（暂未开放）" disabled>
                <Space>
                  <BellOutlined />
                  邮件通知（暂未开放）
                </Space>
              </Option>
              <Option value="sms" label="短信通知（暂未开放）" disabled>
                <Space>
                  <BellOutlined />
                  短信通知（暂未开放）
                </Space>
              </Option>
            </AppSelect>
          </Form.Item>

          <Form.Item
            label="提前提醒时间"
            name="beforeReminderMinutes"
            tooltip="设置提前多少分钟提醒，留空则表示当天提醒"
          >
            <AppInput
              allowClear
              type="number"
              placeholder="留空表示当天提醒"
              suffix="分钟"
              style={{ width: 200 }}
            />
          </Form.Item>
        </Form>

        <Divider />

        <Alert
          title="注意事项"
          description={
            <ul className="list-disc pl-4 m-0">
              <li>修改提醒时间后，定时任务需要管理员在系统中重新配置</li>
              <li>站内通知通过WebSocket实时推送，请确保保持在线</li>
              <li>邮件和短信通知功能正在开发中</li>
              <li>如需禁用提醒，关闭"启用值班提醒"开关即可</li>
            </ul>
          }
          type="warning"
          showIcon
        />
      </Card>

      <Card title="定时任务配置说明" className="mt-6">
        <Alert
          title="管理员操作指南"
          description={
            <div>
              <p>
                系统使用定时任务来发送值班提醒，如需修改提醒时间，需要管理员在数据库中更新Job配置：
              </p>
              <pre className="bg-gray-100 p-4 mt-4 rounded">
                {`-- 更新值班提醒任务时间（例如改为早上9点）
UPDATE sys_job
SET cron_expression = '0 0 9 * * ?'
WHERE job_name = '每日值班提醒';

-- Cron表达式说明：0 0 8 * * ?
-- 0 0 8 * * ? 表示每天早上8点00分00秒执行
-- 格式：秒 分 时 日 月 周`}
              </pre>
            </div>
          }
          type="info"
          showIcon
        />
      </Card>
    </div>
  );
};

export default DutyConfigPage;
