/**
 * RPA 任务编辑弹窗
 */

import { useLayoutEffect, useState, useCallback } from "react";
import {
  App,
  Modal,
  Form,
  InputNumber,
  Switch,
  Tabs,
  Button,
  Card,
  Row,
  Col,
  Space,
  Tag,
  Typography,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  RobotOutlined,
  AppstoreOutlined,
  VideoCameraOutlined,
} from "@ant-design/icons";
import type { FormInstance } from "antd/es/form";
import type { Task, Action, ActionType } from "@/types/rpa";
import { ACTION_TYPE_OPTIONS, TASK_STATUS_OPTIONS } from "../../constants";
// Phase 71 (W3)：裸 Input/Select/TextArea → 契约组件。
// - 动作类型 Select：固定枚举 → options 改写 + showSearch={false}
// - tags Select：mode="tags"（⑦ 自由输入）→ 显式 showSearch（否则无法输入新标签）
// - 动作描述/选择器/值：受控输入（value + onChange），allowClear 原就存在
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { AIScriptEditor } from "./AIScriptEditor";
// NOCODE-04：编辑器内的模板套用
import {
  TASK_TEMPLATES,
  TEMPLATE_CATEGORY_LABELS,
  instantiateTemplate,
} from "../../wizard/templates";
import type { TaskTemplate } from "../../wizard/templates";

const { Text } = Typography;

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
          {/* 形态分类器②④：动作类型固定枚举（业务禁搜），children → options。
              onChange 值由 string 变 unknown → as ActionType 收敛（选项集合即 ActionType 子集） */}
          <AppSelect
            allowClear
            value={action.type}
            placeholder="动作类型"
            onChange={(value) => onUpdate(index, { ...action, type: value as ActionType })}
            style={{ width: "100%" }}
            showSearch={false}
            options={ACTION_TYPE_OPTIONS}
          />
        </Col>
        <Col span={18}>
          <AppInput
            value={action.description || ""}
            placeholder="动作描述"
            onChange={(e) => onUpdate(index, { ...action, description: e.target.value })}
            style={{ marginBottom: 8 }}
          />
          {action.type !== "navigate" && action.type !== "wait" && (
            <AppInput
              value={action.selector?.value || ""}
              placeholder="选择器 (CSS Selector)"
              onChange={(e) =>
                onUpdate(index, {
                  ...action,
                  selector: { ...action.selector, type: "css", value: e.target.value },
                })
              }
              style={{ marginBottom: 8 }}
            />
          )}
          {(action.type === "fill" || action.type === "select" || action.type === "navigate") && (
            <AppInput
              value={(action.params?.value as string) || ""}
              placeholder="值 (URL/输入内容/选项)"
              onChange={(e) =>
                onUpdate(index, {
                  ...action,
                  params: { ...action.params, value: e.target.value },
                })
              }
            />
          )}
          {action.type === "wait" && (
            <InputNumber
              value={(action.params?.duration as number) || 1000}
              placeholder="等待时长(毫秒)"
              min={100}
              max={60000}
              onChange={(value) =>
                onUpdate(index, {
                  ...action,
                  params: { ...action.params, duration: value || 1000 },
                })
              }
              style={{ width: "100%" }}
            />
          )}
        </Col>
      </Row>
    </Card>
  );
}

export function TaskEditModal({ open, form, editingTask, onOk, onCancel }: TaskEditModalProps) {
  const { message } = App.useApp();
  const [actions, setActions] = useState<Action[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [aiEditorVisible, setAiEditorVisible] = useState(false);
  // NOCODE-04：模板选择器（在编辑器内直接套用开箱模板）
  const [templatePickerVisible, setTemplatePickerVisible] = useState(false);

  // 仅在新增模式下重置表单
  useLayoutEffect(() => {
    if (open && !editingTask?.id) {
      form.resetFields();
      form.setFieldsValue({
        // CONTRACT-03：status 是任务态数字 0=启用 / 1=停用（后端 TaskStatus）。
        // 历史这里传执行态字符串 "pending"，后端 *int 会直接 400，
        // 且「新建任务」的默认状态也落不到真实列上。
        status: 0,
        priority: 50,
        timeout: 300,
        retryOnFailure: true,
        maxRetries: 3,
      });
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
        // 数字 0 必须原样回填（不能写成 falsy 判定，否则 0 会回落默认值）
        status:
          editingTask.status === undefined || editingTask.status === null ? 0 : editingTask.status,
        // tags 是字符串，需要转换为数组（用逗号分隔）
        tags: editingTask.tags
          ? editingTask.tags
              .split(",")
              .map((t) => t.trim())
              .filter(Boolean)
          : [],
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
        // 后端 script 字段为动态 JSON（与前端 Action 接口不同构，含 attributes 等原始字段），
        // 类型边界登记豁免（任务规范允许：动态结构不强行 unknown 化）
        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- 后端动态 JSON 结构边界
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

  const handleUpdateAction = useCallback(
    (index: number, updatedAction: Action) => {
      const newActions = [...actions];
      newActions[index] = updatedAction;
      setActions(newActions);
    },
    [actions]
  );

  const handleRemoveAction = useCallback(
    (index: number) => {
      setActions(actions.filter((_, i) => i !== index));
    },
    [actions]
  );

  const handleAIGenerate = useCallback((generatedActions: Action[]) => {
    setActions(generatedActions);
    setAiEditorVisible(false);
    message.success("AI 生成脚本成功！");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  /**
   * handleApplyTemplate NOCODE-04：把开箱模板套进当前编辑器。
   *
   * 用模板的示例值实例化（用户随后可在动作卡片里逐项改）—— 这比让用户从
   * 空白动作列表开始要快得多，是「一般员工」路径的核心价值。
   */
  const handleApplyTemplate = useCallback(
    (template: TaskTemplate) => {
      const values: Record<string, string> = {};
      for (const p of template.params) {
        values[p.key] = p.example || "";
      }
      const instantiated = instantiateTemplate(template, values);
      setActions(instantiated);
      setTemplatePickerVisible(false);
      // 任务名/描述也一并带上，减少用户填写负担（空着才填，不覆盖用户已输入）
      const currentName = form.getFieldValue("name");
      if (!currentName) form.setFieldsValue({ name: template.name });
      const currentDesc = form.getFieldValue("description");
      if (!currentDesc) form.setFieldsValue({ description: template.summary });
      message.success(`已套用模板「${template.name}」，可逐项修改内容`);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [form]
  );

  /**
   * handleSwitchToRecord NOCODE-04：一键转录制。
   *
   * 语义是「清空手写动作，改走录制」——必须显式确认，避免用户误点丢掉已配置
   * 的动作。录制完成后由上层把结果回填到 actions。
   */
  const handleSwitchToRecord = useCallback(() => {
    const doSwitch = () => {
      setActions([]);
      message.info("已切换为录制方式：请到「RPA 实时画面」开始录制，录完自动回填步骤");
    };
    if (actions.length === 0) {
      doSwitch();
      return;
    }
    // eslint-disable-next-line no-restricted-syntax -- 非删除语义：切换到录制会清空当前动作，属于流程切换确认
    Modal.confirm({
      title: "改用录制方式？",
      content: `当前已配置 ${actions.length} 个动作，切换后会清空，改由录制生成。`,
      okText: "改用录制",
      cancelText: "先不改",
      onOk: doSwitch,
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [actions]);

  const handleOk = useCallback(async () => {
    const values = await form.validateFields();

    // 构建 script 数组格式（后端期望的是 ScriptAction 数组）
    const script = actions.map((action) => ({
      type: action.type,
      selector: action.selector?.value || "",
      value: (action.params?.value as string) || "",
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
        onOk={async () => {
          setSubmitting(true);
          try {
            await handleOk();
          } finally {
            setSubmitting(false);
          }
        }}
        cancelButtonProps={{ disabled: submitting }}
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
                      <AppInput placeholder="请输入任务名称" />
                    </Form.Item>

                    <Form.Item name="description" label="任务描述">
                      <AppTextArea rows={3} placeholder="请输入任务描述" />
                    </Form.Item>

                    <Row gutter={16}>
                      <Col span={12}>
                        {/* CONTRACT-03 裁定：优先级统一为 0-100（后端 TaskPriorityMax
                            同步放宽为 100，SMALLINT 无需迁移）。 */}
                        <Form.Item name="priority" label="优先级(0-100)" initialValue={50}>
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
                        {/* CONTRACT-03：status 用数字枚举 0=启用 / 1=停用，
                            后端指针语义可把停用任务改回启用（0 不再被当「未传」）。 */}
                        <Form.Item name="status" label="任务状态" initialValue={0}>
                          <AppSelect options={TASK_STATUS_OPTIONS} showSearch={false} />
                        </Form.Item>
                      </Col>
                      <Col span={12} />
                    </Row>

                    <Row gutter={16}>
                      <Col span={12}>
                        <Form.Item
                          name="retryOnFailure"
                          label="失败重试"
                          valuePropName="checked"
                          initialValue={true}
                        >
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
                      {/* 形态分类器⑦：mode="tags" 必须显式 showSearch（默认策略按
                          options.length 判定，本处无 options 会被判 ≤8 而关掉搜索/输入） */}
                      <AppSelect
                        allowClear
                        mode="tags"
                        showSearch
                        placeholder="请输入标签"
                        onSearch={() => {}}
                      />
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
                    {/*
                      NOCODE-04 补登项：EditModal.tsx:391 加「模板 / 一键转录制」。
                      两个入口都是「把用户从空白编辑器里救出来」的通道：
                        - 模板：选一个开箱模板，步骤直接填进来（可再改）
                        - 转录制：清空手写动作，改走录制（一般员工更易成功）
                    */}
                    <Button
                      type="link"
                      size="small"
                      icon={<AppstoreOutlined />}
                      onClick={() => setTemplatePickerVisible(true)}
                      style={{ marginLeft: 4 }}
                      data-testid="edit-modal-template"
                    >
                      模板
                    </Button>
                    <Button
                      type="link"
                      size="small"
                      icon={<VideoCameraOutlined />}
                      onClick={handleSwitchToRecord}
                      style={{ marginLeft: 4 }}
                      data-testid="edit-modal-to-record"
                    >
                      一键转录制
                    </Button>
                  </span>
                ),
                children: (
                  <>
                    <div style={{ marginBottom: 16 }}>
                      <Button type="dashed" onClick={handleAddAction} icon={<PlusOutlined />} block>
                        添加动作
                      </Button>
                    </div>

                    <div style={{ maxHeight: 400, overflowY: "auto" }}>
                      {actions.length === 0 ? (
                        <div
                          style={{
                            textAlign: "center",
                            padding: "40px 0",
                            color: "var(--theme-text-tertiary)",
                          }}
                        >
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
        onFallbackToRecord={() => {
          // NOCODE-03 降级路径：AI 不可用 → 直接切到录制（清空并提示），
          // 用户不会卡在「AI 报错 → 重试 → 再报错」的循环里。
          setAiEditorVisible(false);
          handleSwitchToRecord();
        }}
      />

      {/* NOCODE-04：模板选择器 */}
      <Modal
        title="从模板开始"
        open={templatePickerVisible}
        onCancel={() => setTemplatePickerVisible(false)}
        footer={null}
        width={760}
        destroyOnHidden
        data-testid="template-picker-modal"
      >
        <Text type="secondary">选一个最接近你日常工作的模板，套用后再逐项改成自己的内容。</Text>
        <Row gutter={12} style={{ marginTop: 16 }}>
          {TASK_TEMPLATES.map((tpl) => (
            <Col span={8} key={tpl.id}>
              <Card
                size="small"
                hoverable
                data-testid={`template-pick-${tpl.id}`}
                onClick={() => handleApplyTemplate(tpl)}
                style={{ height: "100%" }}
              >
                <Space orientation="vertical" size={6} style={{ width: "100%" }}>
                  <Text strong>{tpl.name}</Text>
                  <Tag>{TEMPLATE_CATEGORY_LABELS[tpl.category]}</Tag>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {tpl.summary}
                  </Text>
                  <Space size={4} wrap>
                    <Tag color="green">{tpl.steps.length} 个步骤</Tag>
                    <Tag>{tpl.estimatedTime}</Tag>
                  </Space>
                </Space>
              </Card>
            </Col>
          ))}
        </Row>
      </Modal>
    </>
  );
}
