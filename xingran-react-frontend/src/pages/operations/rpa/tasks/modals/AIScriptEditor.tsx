/**
 * AI 脚本编辑器组件
 * 使用 AI 从自然语言描述生成 RPA 脚本
 */

import { useState, useCallback } from "react";
import {
  App, Modal, Form, Input, Button, Steps, Card, Space, Alert, Spin, Tag,
  Divider, Typography,
} from "antd";
import {
  RobotOutlined, ThunderboltOutlined, CheckOutlined,
  LoadingOutlined,
} from "@ant-design/icons";
import type { Action } from "@/types/rpa";
import { ACTION_TYPE_TEXT_MAP } from "../../constants";

const { TextArea } = Input;
const { Text } = Typography;

export interface AIScriptEditorProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (actions: Action[]) => void;
}

// 示例生成的动作（模拟 AI 返回结果）
const mockGeneratedActions: Action[] = [
  {
    id: "action_1",
    type: "navigate",
    description: "打开登录页面",
    selector: { type: "css", value: "" },
    params: { value: "https://example.com/login" },
    timeout: 30000,
  },
  {
    id: "action_2",
    type: "fill",
    description: "输入用户名",
    selector: { type: "css", value: "#username" },
    params: { value: "${username}" },
    timeout: 10000,
  },
  {
    id: "action_3",
    type: "fill",
    description: "输入密码",
    selector: { type: "css", value: "#password" },
    params: { value: "${password}" },
    timeout: 10000,
  },
  {
    id: "action_4",
    type: "click",
    description: "点击登录按钮",
    selector: { type: "css", value: 'button[type="submit"]' },
    timeout: 10000,
  },
  {
    id: "action_5",
    type: "screenshot",
    description: "验证登录成功",
    selector: { type: "css", value: ".user-profile" },
    timeout: 10000,
  },
];

export function AIScriptEditor({ open, onClose, onConfirm }: AIScriptEditorProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [generatedActions, setGeneratedActions] = useState<Action[]>([]);

  const handleGenerate = useCallback(async () => {
    const description = form.getFieldValue("description");
    if (!description?.trim()) {
      message.warning("请输入任务描述");
      return;
    }

    setLoading(true);
    try {
      // TODO: 调用后端 AI API 生成脚本
      // const result = await post('/rpa/ai/generate', { description });
      // setGeneratedActions(result.data.script.actions);

      // 模拟 API 调用
      await new Promise(resolve => setTimeout(resolve, 1500));
      setGeneratedActions(mockGeneratedActions);
      message.success("AI 生成脚本成功！");
    } catch (_error) {
      message.error("AI 生成失败，请重试");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [form]);

  const handleConfirm = useCallback(() => {
    if (generatedActions.length === 0) {
      message.warning("请先生成脚本");
      return;
    }
    onConfirm(generatedActions);
    form.resetFields();
    setGeneratedActions([]);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [generatedActions, onConfirm, form]);

  const handleCancel = useCallback(() => {
    form.resetFields();
    setGeneratedActions([]);
    onClose();
  }, [form, onClose]);

  return (
    <Modal
      title={
        <Space>
          <RobotOutlined />
          <span>AI 脚本生成器</span>
        </Space>
      }
      open={open}
      onCancel={handleCancel}
      width={800}
      footer={
        <Space>
          <Button onClick={handleCancel}>取消</Button>
          <Button
            type="primary"
            icon={<ThunderboltOutlined />}
            loading={loading}
            onClick={handleGenerate}
            disabled={generatedActions.length > 0}
          >
            {loading ? "生成中..." : "生成脚本"}
          </Button>
          <Button
            type="primary"
            icon={<CheckOutlined />}
            onClick={handleConfirm}
            disabled={generatedActions.length === 0}
          >
            确认使用
          </Button>
        </Space>
      }
    >
      <div style={{ padding: "16px 0" }}>
        <Alert
          message="使用自然语言描述您的自动化任务，AI 将为您生成相应的脚本步骤"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form form={form} layout="vertical">
          <Form.Item
            name="description"
            label="任务描述"
            rules={[{ required: true, message: "请描述您的自动化任务" }]}
          >
            <TextArea
              rows={4}
              placeholder="例如：打开登录页面，输入用户名和密码，点击登录按钮，验证登录成功"
              disabled={loading || generatedActions.length > 0}
            />
          </Form.Item>
        </Form>

        {generatedActions.length > 0 && (
          <>
            <Divider>生成的脚本步骤</Divider>
            <Steps
              direction="vertical"
              current={-1}
              items={generatedActions.map((action, index) => ({
                title: (
                  <Space>
                    <Tag color="blue">{ACTION_TYPE_TEXT_MAP[action.type] || action.type}</Tag>
                    <Text strong>{action.description}</Text>
                  </Space>
                ),
                description: (
                  <Card size="small" style={{ marginTop: 8 }}>
                    <Space direction="vertical" style={{ width: "100%" }}>
                      {action.selector?.value && (
                        <Text code style={{ fontSize: 12 }}>
                          选择器: {action.selector.value}
                        </Text>
                      )}
                      {action.params?.value !== undefined && action.params?.value !== null && (
                        <Text code style={{ fontSize: 12 }}>
                          值: {String(action.params.value)}
                        </Text>
                      )}
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        超时: {action.timeout}ms
                      </Text>
                    </Space>
                  </Card>
                ),
                icon: index < generatedActions.length ? <CheckOutlined /> : undefined,
              }))}
            />

            <Alert
              message="请检查生成的脚本步骤，确认无误后点击「确认使用」按钮"
              type="success"
              showIcon
              style={{ marginTop: 16 }}
            />
          </>
        )}

        {loading && (
          <div style={{ textAlign: "center", padding: "40px 0" }}>
            <Spin
              indicator={<LoadingOutlined style={{ fontSize: 48 }} spin />}
              tip="AI 正在生成脚本，请稍候..."
            />
          </div>
        )}
      </div>
    </Modal>
  );
}
