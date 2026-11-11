import React from "react";
import {
  Card,
  Row,
  Col,
  Form,
  Switch,
  InputNumber,
  Button,
  Descriptions,
  Tag,
  TimePicker,
  Alert,
} from "antd";
// Phase 71 (W5)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
import { BellOutlined, SaveOutlined } from "@ant-design/icons";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import type { DutyConfig as DutyConfigType } from "@/lib/dutyApi";

interface DutyConfigProps {
  config: DutyConfigType | null;
  loading: boolean;
  saving: boolean;
  onSave: (values: {
    reminderEnabled: boolean;
    reminderTime: Dayjs;
    reminderChannels: string[];
    beforeReminderMinutes?: number;
  }) => Promise<boolean>;
}

export const DutyConfig: React.FC<DutyConfigProps> = ({ config, loading, saving, onSave }) => {
  const [form] = Form.useForm();

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      await onSave(values);
    } catch (_error) {
      // 表单验证失败
    }
  };

  React.useEffect(() => {
    if (config) {
      form.setFieldsValue({
        reminderEnabled: config.reminderEnabled,
        reminderTime: config.reminderTime ? dayjs(config.reminderTime, "HH:mm") : undefined,
        reminderChannels: config.reminderChannels ? config.reminderChannels.split(",") : [],
        beforeReminderMinutes: config.beforeReminderMinutes,
      });
    }
  }, [config, form]);

  return (
    <div>
      <Row gutter={24}>
        <Col span={12}>
          <Card title="当前配置" size="small" style={{ height: "100%" }}>
            {config ? (
              <Descriptions column={1} bordered size="small">
                <Descriptions.Item label="提醒状态">
                  {config.reminderEnabled ? (
                    <Tag color="green" icon={<BellOutlined />}>
                      已启用
                    </Tag>
                  ) : (
                    <Tag color="red">已禁用</Tag>
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="提醒时间">{config.reminderTime}</Descriptions.Item>
                <Descriptions.Item label="提醒渠道">
                  {config.reminderChannels ? (
                    config.reminderChannels.split(",").map((ch) => {
                      const map: Record<string, string> = {
                        websocket: "站内通知",
                        email: "邮件",
                        sms: "短信",
                      };
                      return <Tag key={ch}>{map[ch] || ch}</Tag>;
                    })
                  ) : (
                    <Tag>无</Tag>
                  )}
                </Descriptions.Item>
                <Descriptions.Item label="提前提醒">
                  {config.beforeReminderMinutes
                    ? `${config.beforeReminderMinutes}分钟`
                    : "当天提醒"}
                </Descriptions.Item>
              </Descriptions>
            ) : (
              <div style={{ textAlign: "center", padding: "20px" }}>
                {loading ? "加载中..." : "无配置"}
              </div>
            )}
          </Card>
        </Col>
        <Col span={12}>
          <Card title="修改配置" size="small" style={{ height: "100%" }}>
            <Form form={form} layout="vertical">
              <Form.Item label="启用提醒" name="reminderEnabled" valuePropName="checked">
                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
              </Form.Item>
              <Form.Item
                label="提醒时间"
                name="reminderTime"
                rules={[{ required: true, message: "请选择时间" }]}
              >
                <TimePicker style={{ width: "100%" }} format="HH:mm" />
              </Form.Item>
              <Form.Item
                label="提醒渠道"
                name="reminderChannels"
                rules={[{ required: true, message: "请选择渠道" }]}
              >
                {/* 形态分类器②⑦：children 写法改写 options；mode="multiple" 显式 showSearch
                （antd 多选默认开搜索，保持不变） */}
                <AppSelect
                  allowClear
                  mode="multiple"
                  placeholder="选择渠道"
                  showSearch
                  onSearch={() => {}}
                  options={[
                    { value: "websocket", label: "站内通知" },
                    { value: "email", label: "邮件（未开放）", disabled: true },
                    { value: "sms", label: "短信（未开放）", disabled: true },
                  ]}
                />
              </Form.Item>
              <Form.Item label="提前提醒(分钟)" name="beforeReminderMinutes">
                <InputNumber placeholder="当天提醒请留空" style={{ width: "100%" }} min={0} />
              </Form.Item>
              <Form.Item>
                <Button
                  type="primary"
                  icon={<SaveOutlined />}
                  onClick={handleSave}
                  loading={saving}
                  block
                >
                  保存配置
                </Button>
              </Form.Item>
            </Form>
          </Card>
        </Col>
      </Row>

      {/* 说明卡：由原独立页 src/pages/duty/config/index.tsx 并入（配置说明 Tab 语义） */}
      <Card title="注意事项" className="mt-6">
        <Alert
          description={
            <ul className="list-disc pl-4 m-0">
              <li>修改提醒时间后，定时任务需要管理员在系统中重新配置</li>
              <li>站内通知通过 WebSocket 实时推送，请确保保持在线</li>
              <li>邮件和短信通知功能正在开发中</li>
              <li>如需禁用提醒，关闭「启用提醒」开关即可</li>
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
                系统使用定时任务来发送值班提醒，如需修改提醒时间，需要管理员在数据库中更新 Job
                配置：
              </p>
              <pre className="bg-gray-100 p-4 mt-4 rounded">
                {`-- 更新值班提醒任务时间（例如改为早上9点）
UPDATE sys_job
SET cron_expression = '0 0 9 * * ?'
WHERE job_name = '每日值班提醒';

-- Cron表达式说明：0 0 8 * * ?
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

export default DutyConfig;
