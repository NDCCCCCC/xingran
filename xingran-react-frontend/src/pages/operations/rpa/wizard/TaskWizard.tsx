/**
 * Phase 156 / NOCODE-04：三步向导
 *
 * 产品形态（261123-v156-research §4）：UiPath StudioX / Power Automate Desktop 的
 * 「录制 → 中文步骤 → 参数 → 一键运行」。**不做流程图编辑器**（那是给专业开发者的，
 * 与目标用户相反）。
 *
 * 三步固定：
 *   ① 说要做什么（自然语言 / 录制 / 模板 三选一）
 *   ② 确认步骤（中文步骤列表，可拖动排序、可改参数）
 *   ③ 选运行方式（立即 / 定时 / 触发）+ 保存
 *
 * 零技术术语纪律：第 ② 步的每一行文字都必须经过 stepLabels.describeAction，
 * 绝不直接渲染 action.type / selector / JSON。测试文件
 * wizard.stepLabels.test.tsx 对本约束做机械断言。
 */

import { useCallback, useMemo, useState } from "react";
import {
  App,
  Alert,
  Button,
  Card,
  Col,
  Divider,
  Empty,
  Form,
  Radio,
  Row,
  Space,
  Steps,
  Tag,
  Typography,
} from "antd";
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  DeleteOutlined,
  HolderOutlined,
  LoginOutlined,
  FormOutlined,
  DownloadOutlined,
  RobotOutlined,
  VideoCameraOutlined,
  AppstoreOutlined,
  CheckOutlined,
} from "@ant-design/icons";
import { DndContext, closestCenter } from "@dnd-kit/core";
import type { DragEndEvent } from "@dnd-kit/core";
import { SortableContext, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { Action } from "@/types/rpa";
// Phase 71 契约组件纪律：禁止裸 antd Input/TextArea（lint 强制）
import { AppInput, AppTextArea } from "@/components/ui";
import {
  TASK_TEMPLATES,
  TEMPLATE_CATEGORY_LABELS,
  getTemplateById,
  instantiateTemplate,
  validateTemplateParams,
} from "./templates";
import type { TaskTemplate } from "./templates";
import { describeAction, containsTechTerm } from "./stepLabels";
import type { WizardDraft, WizardRunMode, WizardSource } from "./payload";
import { runModeLabel } from "./payload";

const { Text, Title } = Typography;

// 类型再导出：调用方（页面 / 测试）从向导入口拿类型更顺手，
// 但真身在 payload.ts（纯函数文件，不触发 react-refresh 规则）。
export type { WizardDraft, WizardRunMode, WizardSource };

/** 第 1 步的三种起点。 */
// WizardSource / WizardRunMode / WizardDraft 已从 payload.ts 引入（见文件顶部）

export interface TaskWizardProps {
  /** 保存草稿（由调用方负责 POST /rpa/tasks） */
  onSave: (draft: WizardDraft) => Promise<void>;
  /** 取消/关闭 */
  onCancel: () => void;
  /**
   * 打开 AI 生成器（第 1 步「用中文描述」）。
   * 由调用方注入，避免向导直接依赖 AIScriptEditor 的挂载位置。
   */
  onOpenAIGenerator?: () => void;
  /**
   * 打开录制（第 1 步「录一段操作」）。
   * NOCODE-03 的降级路径最终也落到这里 —— AI 不可用时用户被引导到同一入口。
   */
  onOpenRecorder?: () => void;
  /** 外部注入的初始动作（录制完成 / AI 生成完成后回填） */
  initialActions?: Action[];
  /** 外部注入的初始描述（AI 降级时带过来的用户描述） */
  initialDescription?: string;
}

/** 模板图标映射（模板里存的是字符串，避免直接存组件）。 */
const TEMPLATE_ICONS: Record<string, React.ReactNode> = {
  LoginOutlined: <LoginOutlined />,
  FormOutlined: <FormOutlined />,
  DownloadOutlined: <DownloadOutlined />,
};

// ==================== 可排序步骤行 ====================

interface SortableStepProps {
  action: Action;
  index: number;
  total: number;
  onMoveUp: (index: number) => void;
  onMoveDown: (index: number) => void;
  onRemove: (index: number) => void;
  onEditValue: (index: number, value: string) => void;
}

/**
 * SortableStep 一个步骤行。
 *
 * 双通道排序（NOCODE-04 要求「可拖动排序」）：
 *   - 拖拽手柄（@dnd-kit，项目已有依赖，零新增）
 *   - 上移/下移按钮（**降级通道**：键盘用户、触屏、以及拖拽失效时仍可操作）
 *
 * 只暴露「业务语义中文名」+「值」两个可编辑面，绝不暴露 selector。
 */
function SortableStep({
  action,
  index,
  total,
  onMoveUp,
  onMoveDown,
  onRemove,
  onEditValue,
}: SortableStepProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: action.id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const label = describeAction(action);
  const editableValue = typeof action.params?.value === "string" ? action.params.value : undefined;
  const techTerm = containsTechTerm(label);

  return (
    <div
      ref={setNodeRef}
      style={style}
      data-testid={`wizard-step-row-${index}`}
      className="flex items-start gap-3 p-3 bg-white border border-gray-200 rounded mb-2"
    >
      <div
        {...attributes}
        {...listeners}
        className="cursor-move text-gray-400 hover:text-gray-600 pt-1"
        data-testid={`wizard-step-row-drag-${index}`}
        title="拖动调整顺序"
      >
        <HolderOutlined />
      </div>

      <div className="flex-1 min-w-0">
        <Space size={4} align="start">
          <Tag color="blue">{index + 1}</Tag>
          <Text strong data-testid={`wizard-step-row-label-${index}`}>
            {label}
          </Text>
        </Space>
        {techTerm && (
          // 自检兜底：若上游 description 漏了技术词，这里显式告警而不是静默展示
          <Text type="warning" style={{ fontSize: 12, display: "block" }}>
            这一步的说明需要优化（请重新录制）
          </Text>
        )}
        {editableValue !== undefined && (
          <AppInput
            size="small"
            style={{ marginTop: 6 }}
            value={editableValue}
            placeholder="填写内容"
            aria-label={`第 ${index + 1} 步的内容`}
            onChange={(e) => onEditValue(index, e.target.value)}
          />
        )}
      </div>

      <Space size={2}>
        {/* 降级通道：上移/下移（拖拽不可用时仍能排序） */}
        <Button
          type="text"
          size="small"
          icon={<ArrowUpOutlined />}
          disabled={index === 0}
          onClick={() => onMoveUp(index)}
          aria-label={`第 ${index + 1} 步上移`}
        />
        <Button
          type="text"
          size="small"
          icon={<ArrowDownOutlined />}
          disabled={index === total - 1}
          onClick={() => onMoveDown(index)}
          aria-label={`第 ${index + 1} 步下移`}
        />
        <Button
          type="text"
          size="small"
          danger
          icon={<DeleteOutlined />}
          onClick={() => onRemove(index)}
          aria-label={`删除第 ${index + 1} 步`}
        />
      </Space>
    </div>
  );
}

// ==================== 向导主体 ====================

export function TaskWizard({
  onSave,
  onCancel,
  onOpenAIGenerator,
  onOpenRecorder,
  initialActions,
  initialDescription,
}: TaskWizardProps) {
  const { message } = App.useApp();

  const [current, setCurrent] = useState(0);
  const [source, setSource] = useState<WizardSource>(initialActions?.length ? "record" : "template");
  const [templateId, setTemplateId] = useState<string>("");
  const [templateValues, setTemplateValues] = useState<Record<string, string>>({});
  const [actions, setActions] = useState<Action[]>(initialActions ?? []);
  const [taskName, setTaskName] = useState<string>("");
  const [description, setDescription] = useState<string>(initialDescription ?? "");
  const [runMode, setRunMode] = useState<WizardRunMode>("immediate");
  const [scheduleCron, setScheduleCron] = useState<string>("0 9 * * *");
  const [triggerEvent, setTriggerEvent] = useState<string>("new-ticket");
  const [saving, setSaving] = useState(false);

  const selectedTemplate = useMemo(
    () => (templateId ? getTemplateById(templateId) : undefined),
    [templateId]
  );

  // ==================== 第 1 步：选起点 ====================

  const handlePickTemplate = useCallback((tpl: TaskTemplate) => {
    setTemplateId(tpl.id);
    setSource("template");
    // 预填示例值，让用户立刻看到「填完长什么样」
    const defaults: Record<string, string> = {};
    for (const p of tpl.params) {
      if (p.example) defaults[p.key] = p.example;
    }
    setTemplateValues(defaults);
    setTaskName(tpl.name);
    setDescription(tpl.summary);
  }, []);

  /**
   * buildTemplateActions 把当前模板 + 参数实例化成动作。
   *
   * 返回动作而不是只 setState：goNext 需要在**同一个 tick** 里拿到结果继续判断
   * 「能不能进第 2 步」。若只 setActions 再读 actions，读到的是闭包里的旧值
   * （空数组），首次点「下一步」会被误判为「没有步骤」而卡住 —— 这是实现期
   * 实测到的真实缺陷。
   */
  const buildTemplateActions = useCallback((): Action[] | null => {
    if (!selectedTemplate) {
      message.warning("请先选择一个模板");
      return null;
    }
    const v = validateTemplateParams(selectedTemplate, templateValues);
    if (!v.ok) {
      message.warning(`请先填写：${v.missing.join("、")}`);
      return null;
    }
    return instantiateTemplate(selectedTemplate, templateValues);
  }, [selectedTemplate, templateValues, message]);

  const goNext = useCallback(() => {
    if (current === 0) {
      let nextActions = actions;

      if (source === "template") {
        const built = buildTemplateActions();
        if (built === null) return;
        nextActions = built;
        setActions(built);
      } else if (nextActions.length === 0) {
        // 录制 / AI 这两条路都要外部组件产出动作；没有动作时给出明确出路，
        // 绝不停在一个「点了下一步但什么也没发生」的死状态。
        if (source === "record") {
          if (onOpenRecorder) {
            onOpenRecorder();
            return;
          }
          message.info("请先录制一段操作，录制完成后会自动回到这里");
          return;
        }
        if (onOpenAIGenerator) {
          onOpenAIGenerator();
          return;
        }
        message.info("请先用中文描述你的任务，生成后会自动回到这里");
        return;
      }

      if (nextActions.length === 0) {
        message.warning("还没有任何步骤，请先选择一种方式生成");
        return;
      }
      setCurrent(1);
      return;
    }

    if (current === 1) {
      if (actions.length === 0) {
        message.warning("步骤列表不能为空");
        return;
      }
      if (!taskName.trim()) {
        // 默认用第一步的模板名/描述，减少用户填写负担
        setTaskName(selectedTemplate?.name || "我的自动化任务");
      }
      setCurrent(2);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [
    current,
    source,
    selectedTemplate,
    buildTemplateActions,
    actions,
    onOpenRecorder,
    onOpenAIGenerator,
    taskName,
  ]);

  const goPrev = useCallback(() => {
    setCurrent((c) => Math.max(0, c - 1));
  }, []);

  // ==================== 第 2 步：排序 / 改参数 ====================

  const handleDragEnd = useCallback((event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    setActions((prev) => {
      const from = prev.findIndex((a) => a.id === active.id);
      const to = prev.findIndex((a) => a.id === over.id);
      if (from < 0 || to < 0) return prev;
      const next = [...prev];
      const [moved] = next.splice(from, 1);
      next.splice(to, 0, moved);
      return next;
    });
  }, []);

  const moveStep = useCallback((index: number, delta: number) => {
    setActions((prev) => {
      const target = index + delta;
      if (target < 0 || target >= prev.length) return prev;
      const next = [...prev];
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  }, []);

  const removeStep = useCallback((index: number) => {
    setActions((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const editStepValue = useCallback((index: number, value: string) => {
    setActions((prev) =>
      prev.map((a, i) =>
        i === index ? { ...a, params: { ...a.params, value } } : a
      )
    );
  }, []);

  // ==================== 第 3 步：保存 ====================

  const handleSave = useCallback(async () => {
    if (actions.length === 0) {
      message.warning("步骤列表不能为空");
      return;
    }
    setSaving(true);
    try {
      await onSave({
        name: taskName.trim() || "我的自动化任务",
        description: description.trim(),
        actions,
        runMode,
        scheduleCron: runMode === "schedule" ? scheduleCron : undefined,
        triggerEvent: runMode === "trigger" ? triggerEvent : undefined,
        source,
        templateId: source === "template" ? templateId : undefined,
      });
    } finally {
      setSaving(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [actions, taskName, description, runMode, scheduleCron, triggerEvent, source, templateId, onSave]);

  // ==================== 渲染 ====================

  const stepItems = [
    { title: "说要做什么", description: "描述、录制或选模板" },
    { title: "确认步骤", description: "按顺序核对每一步" },
    { title: "选运行方式", description: "什么时候自动跑" },
  ];

  return (
    <div data-testid="task-wizard">
      <Steps current={current} items={stepItems} style={{ marginBottom: 24 }} />

      {/* ==================== 第 1 步 ==================== */}
      {current === 0 && (
        <div data-testid="wizard-step-1">
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            title="告诉我你想让电脑帮你做什么，三种方式任选一种"
          />

          {/*
            三选一：**不**用 Radio.Group 包卡片。
            原因（实现期实测）：Radio.Group 的 onChange 与卡片 onClick 会互相
            覆盖受控状态，导致「点了模板卡片但 source 被改回上一次的值」，
            表现为「下一步点不动」。改为纯 Card + 显式选中态，单一真相源
            就是 source 这一个 state。
          */}
          <Row gutter={12}>
            {(
              [
                {
                  key: "describe" as WizardSource,
                  icon: <RobotOutlined />,
                  title: "用中文说",
                  hint: "像跟同事交代工作一样描述，例如「打开报销系统，填入上月差旅金额，点提交」",
                },
                {
                  key: "record" as WizardSource,
                  icon: <VideoCameraOutlined />,
                  title: "录一段操作",
                  hint: "你在电脑上做一遍，系统自动记下来。不用装任何插件。",
                },
                {
                  key: "template" as WizardSource,
                  icon: <AppstoreOutlined />,
                  title: "用现成模板",
                  hint: "从登录、填表、导出这几类常用模板里挑一个，改几个数字就能用。",
                },
              ] as const
            ).map((item) => (
              <Col span={8} key={item.key}>
                <Card
                  size="small"
                  hoverable
                  data-testid={`wizard-source-${item.key}`}
                  data-selected={source === item.key ? "true" : "false"}
                  style={{
                    borderColor:
                      source === item.key ? "var(--theme-border-accent)" : undefined,
                    borderWidth: source === item.key ? 2 : 1,
                  }}
                  onClick={() => setSource(item.key)}
                >
                  <Space orientation="vertical" size={4}>
                    <Space>
                      <Radio checked={source === item.key} />
                      {item.icon}
                      <Text strong>{item.title}</Text>
                    </Space>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {item.hint}
                    </Text>
                  </Space>
                </Card>
              </Col>
            ))}
          </Row>

          {/* 模板选择区 */}
          {source === "template" && (
            <div style={{ marginTop: 20 }} data-testid="wizard-template-picker">
              <Divider titlePlacement="start">选择模板</Divider>
              <Row gutter={12}>
                {TASK_TEMPLATES.map((tpl) => (
                  <Col span={8} key={tpl.id}>
                    <Card
                      size="small"
                      hoverable
                      data-testid={`wizard-template-${tpl.id}`}
                      onClick={() => handlePickTemplate(tpl)}
                      style={{
                        height: "100%",
                        borderColor: templateId === tpl.id ? "var(--theme-border-accent)" : undefined,
                      }}
                    >
                      <Space orientation="vertical" size={6} style={{ width: "100%" }}>
                        <Space>
                          {TEMPLATE_ICONS[tpl.icon]}
                          <Text strong>{tpl.name}</Text>
                          <Tag>{TEMPLATE_CATEGORY_LABELS[tpl.category]}</Tag>
                        </Space>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {tpl.summary}
                        </Text>
                        <Text style={{ fontSize: 12 }}>{tpl.scenario}</Text>
                        <Space size={4} wrap>
                          <Tag color="green">{tpl.steps.length} 个步骤</Tag>
                          <Tag>{tpl.estimatedTime}</Tag>
                        </Space>
                      </Space>
                    </Card>
                  </Col>
                ))}
              </Row>

              {selectedTemplate && (
                <Card size="small" style={{ marginTop: 16 }} data-testid="wizard-template-params">
                  <Title level={5} style={{ marginTop: 0 }}>
                    填写参数
                  </Title>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    只需要改这些内容，其它的系统会自动处理。
                  </Text>
                  <Row gutter={12} style={{ marginTop: 12 }}>
                    {selectedTemplate.params.map((p) => (
                      <Col span={12} key={p.key}>
                        <Form layout="vertical" size="small">
                          <Form.Item
                            label={p.label}
                            required={p.required}
                            style={{ marginBottom: 12 }}
                          >
                            <AppInput
                              value={templateValues[p.key] ?? ""}
                              placeholder={p.placeholder}
                              data-testid={`wizard-param-${p.key}`}
                              onChange={(e) =>
                                setTemplateValues((v) => ({ ...v, [p.key]: e.target.value }))
                              }
                            />
                          </Form.Item>
                        </Form>
                      </Col>
                    ))}
                  </Row>
                </Card>
              )}
            </div>
          )}

          {/* 已生成的动作预览（录制/AI 回填后） */}
          {actions.length > 0 && (
            <Alert
              type="success"
              showIcon
              style={{ marginTop: 16 }}
              title={`已生成 ${actions.length} 个步骤，点「下一步」核对`}
            />
          )}
        </div>
      )}

      {/* ==================== 第 2 步 ==================== */}
      {current === 1 && (
        <div data-testid="wizard-step-2">
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            title="核对下面的步骤：可以拖动调整顺序，也可以改里面的内容"
          />

          {actions.length === 0 ? (
            <Empty description="还没有步骤" />
          ) : (
            <DndContext collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
              <SortableContext
                items={actions.map((a) => a.id)}
                strategy={verticalListSortingStrategy}
              >
                {actions.map((action, index) => (
                  <SortableStep
                    key={action.id}
                    action={action}
                    index={index}
                    total={actions.length}
                    onMoveUp={(i) => moveStep(i, -1)}
                    onMoveDown={(i) => moveStep(i, 1)}
                    onRemove={removeStep}
                    onEditValue={editStepValue}
                  />
                ))}
              </SortableContext>
            </DndContext>
          )}

          <Divider />
          <Form layout="vertical">
            <Form.Item label="任务名称" required>
              <AppInput
                value={taskName}
                placeholder="例如：每月报销单填报"
                data-testid="wizard-task-name"
                onChange={(e) => setTaskName(e.target.value)}
              />
            </Form.Item>
            <Form.Item label="备注（可选）">
              <AppTextArea
                rows={2}
                value={description}
                placeholder="给自己或同事留一句说明"
                onChange={(e) => setDescription(e.target.value)}
              />
            </Form.Item>
          </Form>
        </div>
      )}

      {/* ==================== 第 3 步 ==================== */}
      {current === 2 && (
        <div data-testid="wizard-step-3">
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            title="最后一步：选一个什么时候让它自动跑"
          />
          <Radio.Group
            value={runMode}
            onChange={(e) => setRunMode(e.target.value as WizardRunMode)}
            style={{ width: "100%" }}
          >
            <Space orientation="vertical" style={{ width: "100%" }}>
              <Card size="small" hoverable onClick={() => setRunMode("immediate")}>
                <Space>
                  <Radio value="immediate" />
                  <Text strong>现在就试一次</Text>
                  <Text type="secondary">保存后立刻运行，看看效果对不对</Text>
                </Space>
              </Card>
              <Card size="small" hoverable onClick={() => setRunMode("schedule")}>
                <Space orientation="vertical" style={{ width: "100%" }}>
                  <Space>
                    <Radio value="schedule" />
                    <Text strong>按固定时间自动跑</Text>
                    <Text type="secondary">例如每天早上 9 点</Text>
                  </Space>
                  {runMode === "schedule" && (
                    <Space style={{ marginLeft: 24 }}>
                      <Text>运行时间</Text>
                      <AppInput
                        value={scheduleCron}
                        style={{ width: 220 }}
                        data-testid="wizard-schedule-cron"
                        onChange={(e) => setScheduleCron(e.target.value)}
                        addonAfter="cron 表达式"
                      />
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        常用：0 9 * * *（每天 9 点）、0 9 * * 1（每周一 9 点）
                      </Text>
                    </Space>
                  )}
                </Space>
              </Card>
              <Card size="small" hoverable onClick={() => setRunMode("trigger")}>
                <Space orientation="vertical" style={{ width: "100%" }}>
                  <Space>
                    <Radio value="trigger" />
                    <Text strong>有事发生时自动跑</Text>
                    <Text type="secondary">例如收到新工单时</Text>
                  </Space>
                  {runMode === "trigger" && (
                    <Space style={{ marginLeft: 24 }}>
                      <Text>触发事件</Text>
                      <AppInput
                        value={triggerEvent}
                        style={{ width: 220 }}
                        data-testid="wizard-trigger-event"
                        onChange={(e) => setTriggerEvent(e.target.value)}
                      />
                    </Space>
                  )}
                </Space>
              </Card>
            </Space>
          </Radio.Group>

          <Divider />
          <Card size="small" title="保存前确认" data-testid="wizard-summary">
            <Space orientation="vertical" size={4}>
              <Text>
                任务名称：<Text strong>{taskName || "我的自动化任务"}</Text>
              </Text>
              <Text>
                共 <Text strong>{actions.length}</Text> 个步骤
              </Text>
              <Text>
                运行方式：
                <Text strong>
                  {runModeLabel({ runMode, scheduleCron, triggerEvent })}
                </Text>
              </Text>
            </Space>
          </Card>
        </div>
      )}

      {/* ==================== 底部操作 ==================== */}
      <Divider />
      <Space style={{ width: "100%", justifyContent: "flex-end" }}>
        <Button onClick={onCancel} disabled={saving}>
          取消
        </Button>
        {current > 0 && (
          <Button onClick={goPrev} disabled={saving}>
            上一步
          </Button>
        )}
        {current < 2 && (
          <Button type="primary" onClick={goNext} data-testid="wizard-next">
            下一步
          </Button>
        )}
        {current === 2 && (
          <Button
            type="primary"
            icon={<CheckOutlined />}
            loading={saving}
            onClick={handleSave}
            data-testid="wizard-save"
          >
            保存并完成
          </Button>
        )}
      </Space>
    </div>
  );
}

export default TaskWizard;
