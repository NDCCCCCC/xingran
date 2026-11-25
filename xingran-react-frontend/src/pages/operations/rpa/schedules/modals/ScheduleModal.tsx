/**
 * 定时调度新建/编辑弹窗
 *
 * 员工可见口径（硬约束）：
 *   - 主区域只给三个简单选项：每天 / 每周 / 每隔 N 分钟；
 *   - cron 表达式只出现在**默认折叠**的「高级设置（给熟悉的人用）」里；
 *   - 界面文案不出现「cron」「任务标识」等技术词。
 */

import { useCallback, useLayoutEffect, useMemo, useState } from "react";
import { App, Collapse, Form, InputNumber, Modal, Radio, Row, Col, Space, Switch, Typography } from "antd";
import type { FormInstance } from "antd/es/form";
// Phase 71 契约组件：禁裸 antd Input/Select/TextArea。
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { RemoteSelect } from "@/components/ui/RemoteSelect";
import {
  DEFAULT_INTERVAL_MINUTES,
  DEFAULT_WEEKDAY,
  INTERVAL_MAX_MINUTES,
  INTERVAL_MIN_MINUTES,
  INTERVAL_STEP,
  WEEKDAY_OPTIONS,
  cronToPreset,
  formatClockTime,
  isValidCronShape,
  parseClockTime,
  presetToCron,
  type SimplePreset,
} from "../cronPresets";
import { listTaskBriefs, type ScheduleRecord, type TaskBrief } from "../api";

const { Text } = Typography;

/** 表单值结构 */
export interface ScheduleFormValues {
  /** 调度名称 */
  name: string;
  /** 关联任务标识 */
  taskId: string;
  /** 运行方式：三个简单预设之一，或 custom（高级设置里手写） */
  runMode: SimplePreset | "custom";
  /** 时间文本 "HH:mm" */
  runTime: string;
  /** 星期几：1=周一 … 7=周日 */
  weekday: number;
  /** 每隔 N 分钟 */
  intervalMinutes: number;
  /** 定时规则（cron 表达式，仅高级设置可见） */
  cronExpr: string;
  /** 备注 */
  description?: string;
  /** 是否启用（提交时映射为 0=启用 / 1=停用） */
  enabled: boolean;
}

export interface ScheduleModalProps {
  open: boolean;
  form: FormInstance<ScheduleFormValues>;
  /** 正在编辑的调度（null = 新建） */
  editing: ScheduleRecord | null;
  /** 关联任务候选项（列表页加载的任务列表） */
  taskOptions: TaskBrief[];
  /** 提交（返回已组装好的载荷） */
  onOk: (values: {
    name: string;
    taskId: string;
    cronExpr: string;
    description?: string;
    status: number;
  }) => Promise<void>;
  onCancel: () => void;
}

/** 运行方式（简单选项）的中文标签 */
const RUN_MODE_OPTIONS: Array<{ label: string; value: SimplePreset }> = [
  { label: "每天", value: "daily" },
  { label: "每周", value: "weekly" },
  { label: "每隔若干分钟", value: "interval" },
];

export function ScheduleModal({
  open,
  form,
  editing,
  taskOptions,
  onOk,
  onCancel,
}: ScheduleModalProps) {
  const { message } = App.useApp();
  const [submitting, setSubmitting] = useState(false);

  const runMode = Form.useWatch("runMode", form);
  const runTime = Form.useWatch("runTime", form);
  const weekday = Form.useWatch("weekday", form);
  const intervalMinutes = Form.useWatch("intervalMinutes", form);

  /** 关联任务候选项（编辑时补上当前已选任务，避免只显示内部标识） */
  const initialTaskOptions = useMemo(() => {
    const options = taskOptions.map((task) => ({ label: task.taskName, value: task.id }));
    if (editing?.taskId && !options.some((opt) => opt.value === editing.taskId)) {
      options.push({ label: "已选任务", value: editing.taskId });
    }
    return options;
  }, [taskOptions, editing]);

  /** 打开时初始化表单值（新建给默认「每天 09:00」，编辑从既有配置反推） */
  useLayoutEffect(() => {
    if (!open) return;

    if (!editing) {
      form.resetFields();
      form.setFieldsValue({
        name: "",
        taskId: "",
        runMode: "daily",
        runTime: formatClockTime(9, 0),
        weekday: DEFAULT_WEEKDAY,
        intervalMinutes: DEFAULT_INTERVAL_MINUTES,
        cronExpr: presetToCron("daily", { hour: 9, minute: 0 }),
        description: "",
        enabled: true,
      });
      return;
    }

    const parsed = cronToPreset(editing.cronExpr ?? "");
    const base = {
      name: editing.name ?? "",
      taskId: editing.taskId ?? "",
      description: editing.description ?? "",
      enabled: editing.status === 0,
      runTime: formatClockTime(9, 0),
      weekday: DEFAULT_WEEKDAY,
      intervalMinutes: DEFAULT_INTERVAL_MINUTES,
    };

    if (parsed.preset === "daily") {
      const opts = parsed.opts as { hour: number; minute: number };
      form.setFieldsValue({
        ...base,
        runMode: "daily",
        runTime: formatClockTime(opts.hour, opts.minute),
        cronExpr: editing.cronExpr ?? "",
      });
      return;
    }
    if (parsed.preset === "weekly") {
      const opts = parsed.opts as { weekday: number; hour: number; minute: number };
      form.setFieldsValue({
        ...base,
        runMode: "weekly",
        weekday: opts.weekday,
        runTime: formatClockTime(opts.hour, opts.minute),
        cronExpr: editing.cronExpr ?? "",
      });
      return;
    }
    if (parsed.preset === "interval") {
      const opts = parsed.opts as { minutes: number };
      form.setFieldsValue({
        ...base,
        runMode: "interval",
        intervalMinutes: opts.minutes,
        cronExpr: editing.cronExpr ?? "",
      });
      return;
    }
    // 自定义规则：保留原始表达式，主区域不做任何推断
    form.setFieldsValue({
      ...base,
      runMode: "custom",
      cronExpr: editing.cronExpr ?? "",
    });
  }, [open, editing, form]);

  /**
   * handleCronChange —— 高级设置里手写定时规则后，把能识别的规则回填成简单选项。
   *
   * 识别不出时切到「自定义」：主区域三个选项都不选中，界面提示当前是自定义规则。
   */
  const handleCronChange = useCallback(
    (value: string) => {
      const parsed = cronToPreset(value);
      if (parsed.preset === "custom") {
        form.setFieldValue("runMode", "custom");
        return;
      }
      if (parsed.preset === "daily") {
        const opts = parsed.opts as { hour: number; minute: number };
        form.setFieldsValue({
          runMode: "daily",
          runTime: formatClockTime(opts.hour, opts.minute),
        });
        return;
      }
      if (parsed.preset === "weekly") {
        const opts = parsed.opts as { weekday: number; hour: number; minute: number };
        form.setFieldsValue({
          runMode: "weekly",
          weekday: opts.weekday,
          runTime: formatClockTime(opts.hour, opts.minute),
        });
        return;
      }
      const opts = parsed.opts as { minutes: number };
      form.setFieldsValue({ runMode: "interval", intervalMinutes: opts.minutes });
    },
    [form]
  );

  /** 关联任务远程搜索（按任务名过滤，服务端分页） */
  const fetchTaskOptions = useCallback(async (keyword: string) => {
    const { list } = await listTaskBriefs({
      current: 1,
      pageSize: 50,
      name: keyword ? keyword : undefined,
    });
    return list.map((task) => ({ label: task.taskName, value: task.id }));
  }, []);

  /** 提交：把简单选项换算成定时规则后交给上层 */
  const handleOk = useCallback(async () => {
    const values = await form.validateFields();

    let cronExpr = "";
    if (values.runMode === "custom") {
      cronExpr = (values.cronExpr ?? "").trim();
      if (!isValidCronShape(cronExpr)) {
        message.error("定时规则格式不正确，请检查后重试");
        throw new Error("定时规则格式不正确");
      }
    } else if (values.runMode === "daily") {
      const { hour, minute } = parseClockTime(values.runTime ?? "");
      cronExpr = presetToCron("daily", { hour, minute });
    } else if (values.runMode === "weekly") {
      const { hour, minute } = parseClockTime(values.runTime ?? "");
      cronExpr = presetToCron("weekly", {
        weekday: values.weekday ?? DEFAULT_WEEKDAY,
        hour,
        minute,
      });
    } else {
      cronExpr = presetToCron("interval", {
        minutes: values.intervalMinutes ?? DEFAULT_INTERVAL_MINUTES,
      });
    }

    await onOk({
      name: (values.name ?? "").trim(),
      taskId: values.taskId ?? "",
      cronExpr,
      description: values.description?.trim() || undefined,
      status: values.enabled ? 0 : 1,
    });
  }, [form, message, onOk]);

  return (
    <Modal
      title={editing ? "编辑定时调度" : "新建定时调度"}
      open={open}
      onOk={async () => {
        setSubmitting(true);
        try {
          await handleOk();
        } finally {
          setSubmitting(false);
        }
      }}
      confirmLoading={submitting}
      cancelButtonProps={{ disabled: submitting }}
      onCancel={onCancel}
      width={640}
      destroyOnHidden
      maskClosable={false}
    >
      <Form<ScheduleFormValues>
        form={form}
        layout="horizontal"
        labelCol={{ span: 6 }}
        wrapperCol={{ span: 18 }}
        onValuesChange={(changed) => {
          // 高级设置里手写定时规则 → 回填简单选项
          if (typeof changed.cronExpr === "string") {
            handleCronChange(changed.cronExpr);
          }
        }}
      >
        <Form.Item
          name="name"
          label="名称"
          rules={[{ required: true, message: "请输入调度名称" }]}
        >
          <AppInput placeholder="例如：每天早上生成日报" />
        </Form.Item>

        <Form.Item
          name="taskId"
          label="关联任务"
          rules={[{ required: true, message: "请选择要定时运行的任务" }]}
        >
          <RemoteSelect
            placeholder="搜索并选择要定时运行的任务"
            fetcher={fetchTaskOptions}
            initialOptions={initialTaskOptions}
          />
        </Form.Item>

        <Form.Item name="runMode" label="运行方式">
          <Radio.Group>
            <Space direction="vertical">
              {RUN_MODE_OPTIONS.map((opt) => (
                <Radio key={opt.value} value={opt.value}>
                  {opt.label}
                </Radio>
              ))}
            </Space>
          </Radio.Group>
        </Form.Item>

        {runMode === "daily" && (
          <Form.Item name="runTime" label="运行时间">
            <AppInput type="time" style={{ width: 160 }} />
          </Form.Item>
        )}

        {runMode === "weekly" && (
          <Row gutter={16}>
            <Col span={10}>
              <Form.Item name="weekday" label="星期几">
                <AppSelect options={WEEKDAY_OPTIONS} showSearch={false} />
              </Form.Item>
            </Col>
            <Col span={14}>
              <Form.Item name="runTime" label="运行时间">
                <AppInput type="time" style={{ width: 160 }} />
              </Form.Item>
            </Col>
          </Row>
        )}

        {runMode === "interval" && (
          <Form.Item name="intervalMinutes" label="间隔（分钟）">
            <InputNumber
              min={INTERVAL_MIN_MINUTES}
              max={INTERVAL_MAX_MINUTES}
              step={INTERVAL_STEP}
              style={{ width: 160 }}
            />
          </Form.Item>
        )}

        {runMode === "custom" && (
          <Form.Item label="运行规则">
            <Text type="secondary">
              当前使用的是自定义规则，请在下方「高级设置」中查看或调整。
            </Text>
          </Form.Item>
        )}

        <Form.Item name="description" label="备注说明">
          <AppTextArea rows={2} placeholder="可以写清楚这条调度是做什么用的（选填）" />
        </Form.Item>

        <Form.Item name="enabled" label="启用" valuePropName="checked">
          <Switch checkedChildren="启用" unCheckedChildren="停用" />
        </Form.Item>

        {/*
          高级设置：默认折叠 —— 一般员工不需要看里面的定时规则。
          折叠面板标题用中文说明「给熟悉的人用」，不出现技术术语。
        */}
        <Collapse
          ghost
          defaultActiveKey={[]}
          items={[
            {
              key: "advanced",
              label: "高级设置（给熟悉的人用）",
              children: (
                <>
                  <Form.Item
                    name="cronExpr"
                    label="定时规则"
                    rules={[
                      {
                        validator: (_, value: unknown) => {
                          // 只有「自定义」模式下才校验形状；简单选项产出的规则必然合法
                          if (runMode !== "custom") return Promise.resolve();
                          if (typeof value !== "string" || !isValidCronShape(value)) {
                            return Promise.reject(new Error("定时规则格式不正确"));
                          }
                          return Promise.resolve();
                        },
                      },
                    ]}
                  >
                    <AppInput placeholder="例如：0 9 * * *" />
                  </Form.Item>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    一般不需要修改。留空时系统会按上面选的运行方式自动生成。
                  </Text>
                </>
              ),
            },
          ]}
        />
      </Form>
    </Modal>
  );
}

export default ScheduleModal;
