import React from "react";
import {
  Card,
  Row,
  Col,
  Form,
  Switch,
  Select,
  InputNumber,
  Button,
  Descriptions,
  Tag,
  TimePicker,
} from "antd";
import { BellOutlined, SaveOutlined } from "@ant-design/icons";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import type { DutyConfig as DutyConfigType } from "@/lib/dutyApi";

const { Option } = Select;

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
                {config.beforeReminderMinutes ? `${config.beforeReminderMinutes}分钟` : "当天提醒"}
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
              <Select mode="multiple" placeholder="选择渠道" onSearch={() => {}}>
                <Option value="websocket">站内通知</Option>
                <Option value="email" disabled>
                  邮件（未开放）
                </Option>
                <Option value="sms" disabled>
                  短信（未开放）
                </Option>
              </Select>
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
  );
};

export default DutyConfig;
