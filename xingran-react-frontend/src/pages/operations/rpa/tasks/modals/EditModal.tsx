/**
 * RPA 任务编辑弹窗
 */

import { useLayoutEffect, useState, useCallback } from "react";
import {
  App, Modal, Form, Input, InputNumber, Switch, Tabs, Button,
  Card, Row, Col, Select,
} from "antd";
import { PlusOutlined, DeleteOutlined, RobotOutlined } from "@ant-design/icons";
import type { FormInstance } from "antd/es/form";
import type { Task, Action } from "@/types/rpa";
import { ACTION_TYPE_OPTIONS } from "../../constants";
import { AIScriptEditor } from "./AIScriptEditor";

const { TextArea } = Input;
const { Option } = Select;

export interface TaskEditModalProps {
  open: boolean;
  form: FormInstance;
  editingTask: Task | null;
  onOk: (values: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
}

// 动作列表项编辑组件
interface ActionItemProps {
  action: Action;
  index: number;
  onUpdate: (index: number, action: Action) => void;
  onRemove: (index: number) => void;
}

function ActionItem({ action, index, onUpdate, onRemove }: ActionItemProps) {
  return (
    <Card
      size="small"
      style={{ marginBottom: 8 }}
      extra={
        <Button
          type="text"
          danger
          size="small"
          icon={<DeleteOutlined />}
          onClick={() => onRemove(index)}
        >
          删除
        </Button>
      }
    >
      <Row gutter={16}>
        <Col span={6}>
          <Select
            value={action.type}
            placeholder="动作类型"
            onChange={(value) =>    onUpdate(index, { ...action, type: value })}
            style={{ width: "100%" }}
           onSearch={() => {}}>
            {ACTION_TYPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Col>
        <Col span={18}>
          <Input
            value={action.description || ""}
            placeholder="动作描述"
            onChange={(e) => onUpdate(index, { ...action, description: e.target.value })}
            style={{ marginBottom: 8 }}
          />
          {action.type !== "navigate" && action.type !== "wait" && (
            <Input
              value={action.selector?.value || ""}
              placeholder="选择器 (CSS Selector)"
              onChange={(e) => onUpdate(index, {
                ...action,
                selector: { ...action.selector, type: "css", value: e.target.value },
              })}
              style={{ marginBottom: 8 }}
            />
          )}
          {(action.type === "fill" || action.type === "select" || action.type === "navigate") && (
            <Input
              value={action.params?.value as string || ""}
              placeholder="值 (URL/输入内容/选项)"
              onChange={(e) => onUpdate(index, {
                ...action,
                params: { ...action.params, value: e.target.value },
              })}
            />
          )}
          {action.type === "wait" && (
            <InputNumber
              value={action.params?.duration as number || 1000}
              placeholder="等待时长(毫秒)"
              min={100}
              max={60000}
              onChange={(value) => onUpdate(index, {
                ...action,
                params: { ...action.params, duration: value || 1000 },
              })}
              style={{ width: "100%" }}
            />
          )}
        </Col>
      </Row>
    </Card>
  );
}

export function TaskEditModal({
  open,
  form,
  editingTask,
  onOk,
  onCancel,
}: TaskEditModalProps) {
  const { message } = App.useApp();
  const [actions, setActions] = useState<Action[]>([]);
  const [aiEditorVisible, setAiEditorVisible] = useState(false);

  // 仅在新增模式下重置表单
  useLayoutEffect(() => {
    if (open && !editingTask?.id) {
      form.resetFields();
      form.setFieldsValue({ status: "pending", priority: 50, timeout: 300, retryOnFailure: true, maxRetries: 3 });
      setActions([]);
    } else if (open && editingTask) {
      // 设置表单值
      form.setFieldsValue({
        name: editingTask.taskName || editingTask.name,
        description: editingTask.description,
        priority: editingTask.priority || 50,
        timeout: editingTask.timeout || 300,
        retryOnFailure: true,
        maxRetries: editingTask.retryCount || editingTask.maxRetries || 3,
        status: editingTask.status === 0 ? "pending" : "disabled",
        // tags 是字符串，需要转换为数组（用逗号分隔）
        tags: editingTask.tags ? editingTask.tags.split(",").map(t => t.trim()).filter(Boolean) : [],
      });

      // 解析脚本动作
      if (editingTask.script) {
        // script 是后端返回的 JSON 数组，需要转换为前端 Action 格式
        let scriptData: unknown[];
        if (typeof editingTask.script === "string") {
          try {
            scriptData = JSON.parse(editingTask.script);
          } catch {
            scriptData = [];
          }
        } else {
          scriptData = editingTask.script as Action[];
        }

        // 转换为前端 Action 格式
        const parsedActions: Action[] = (scriptData as any[]).map((item: any) => ({
          id: `action_${Date.now()}_${Math.random()}`,
          type: item.type || "click",
          selector: { type: "css", value: item.selector || "" },
          params: {
            value: item.value || "",
            duration: item.attributes?.duration,
          },
          description: item.attributes?.description || "",
          timeout: item.timeout || 30000,
        }));
        setActions(parsedActions);
      } else {
        setActions([]);
      }
    }
  }, [open, editingTask, form]);

  const handleAddAction = useCallback(() => {
    const newAction: Action = {
      id: `action_${Date.now()}`,
      type: "click",
      selector: { type: "css", value: "" },
      params: { value: "" },
      description: "",
      timeout: 30000,
    };
    setActions([...actions, newAction]);
  }, [actions]);

  const handleUpdateAction = useCallback((index: number, updatedAction: Action) => {
    const newActions = [...actions];
    newActions[index] = updatedAction;
    setActions(newActions);
  }, [actions]);

  const handleRemoveAction = useCallback((index: number) => {
    setActions(actions.filter((_, i) => i !== index));
  }, [actions]);

  const handleAIGenerate = useCallback((generatedActions: Action[]) => {
    setActions(generatedActions);
    setAiEditorVisible(false);
    message.success("AI 生成脚本成功！");
  }, []);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();

    // 构建 script 数组格式（后端期望的是 ScriptAction 数组）
    const script = actions.map(action => ({
      type: action.type,
      selector: action.selector?.value || "",
      value: action.params?.value as string || "",
      attributes: {
        description: action.description,
        ...(action.params?.duration !== undefined && { duration: action.params.duration }),
      },
      timeout: action.timeout || 30000,
      retry: 0,
    }));

    // 将 tags 数组转换为逗号分隔的字符串
    const tags = Array.isArray(values.tags) ? values.tags.join(",") : "";

    await onOk({
      ...values,
      tags,
      script,
    });
  }, [form, actions, onOk]);

  return (
    <>
      <Modal
        title={editingTask ? "编辑 RPA 任务" : "新增 RPA 任务"}
        open={open}
        onOk={handleOk}
        onCancel={onCancel}
        width={900}
        destroyOnHidden
      >
        <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Tabs
            defaultActiveKey="basic"
            items={[
              {
                key: "basic",
                label: "基本信息",
                children: (
                  <>
                    <Form.Item
                      name="name"
                      label="任务名称"
                      rules={[{ required: true, message: "请输入任务名称" }]}
                    >
                      <Input placeholder="请输入任务名称" />
                    </Form.Item>

                    <Form.Item name="description" label="任务描述">
                      <TextArea rows={3} placeholder="请输入任务描述" />
                    </Form.Item>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="priority" label="优先级" initialValue={50}>
                          <InputNumber min={0} max={100} style={{ width: "100%" }} />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="timeout" label="超时时间(秒)" initialValue={300}>
                          <InputNumber min={10} max={3600} style={{ width: "100%" }} />
                        </Form.Item>
                      </Col>
                    </Row>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item name="retryOnFailure" label="失败重试" valuePropName="checked" initialValue={true}>
                          <Switch />
                        </Form.Item>
                      </Col>
                      <Col span={12}>
                        <Form.Item name="maxRetries" label="最大重试次数" initialValue={3}>
                          <InputNumber min={0} max={10} style={{ width: "100%" }} />
                        </Form.Item>
                      </Col>
                    </Row>

                    <Form.Item name="tags" label="标签">
                      <Select mode="tags" placeholder="请输入标签"  onSearch={() => {}}/>
                    </Form.Item>
                  </>
                ),
              },
              {
                key: "script",
                label: (
                  <span>
                    脚本配置
                    <Button
                      type="link"
                      size="small"
                      icon={<RobotOutlined />}
                      onClick={() => setAiEditorVisible(true)}
                      style={{ marginLeft: 8 }}
                    >
                      AI 生成
                    </Button>
                  </span>
                ),
                children: (
                  <>
                    <div style={{ marginBottom: 16 }}>
                      <Button
                        type="dashed"
                        onClick={handleAddAction}
                        icon={<PlusOutlined />}
                        block
                      >
                        添加动作
                      </Button>
                    </div>

                    <div style={{ maxHeight: 400, overflowY: "auto" }}>
                      {actions.length === 0 ? (
                        <div style={{ textAlign: "center", padding: "40px 0", color: "var(--theme-text-tertiary, #999)" }}>
                          暂无动作，请点击上方按钮添加或使用 AI 生成
                        </div>
                      ) : (
                        actions.map((action, index) => (
                          <ActionItem
                            key={action.id || index}
                            action={action}
                            index={index}
                            onUpdate={handleUpdateAction}
                            onRemove={handleRemoveAction}
                          />
                        ))
                      )}
                    </div>
                  </>
                ),
              },
            ]}
          />
        </Form>
      </Modal>

      <AIScriptEditor
        open={aiEditorVisible}
        onClose={() => setAiEditorVisible(false)}
        onConfirm={handleAIGenerate}
      />
    </>
  );
}
