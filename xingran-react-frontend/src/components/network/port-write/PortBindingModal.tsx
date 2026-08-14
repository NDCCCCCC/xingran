/**
 * PortBindingModal — Phase 56 W4 单端口 port_binding Modal (v1.20.1 BIND-01/02)
 *
 * 与 PortWriteModal (Phase 53 W4) + SetAccessVlanModal (Phase 56 W4) 同风格:
 * 3 个主字段 (op Radio.Group + ipAddress Input + macAddress Input optional)
 * + reason 预置 Select + 自定义 TextArea 双层结构 (REQUIRED)。
 *
 * 设计约束 (56-04-PLAN / 56-UI-SPEC):
 * - op Radio.Group buttonStyle="solid" + BIND_OPS options (default "add")
 * - ipAddress Input required + IPV4_REGEX pattern rule (UX hint)
 * - macAddress Input optional + MAC_REGEX pattern rule (UX hint)
 * - reason REQUIRED: validateReasonRequired (55-01 WR-02 跨字段 form 参数版)
 * - LANDMINE #5: wrapper 无 try/catch, 组件无 message.error (post() 拦截器统一弹 Toast)
 * - destroyOnHidden + form.resetFields() on open (防上次 op/ip/mac/reason 残留)
 * - D-10: 成功后 showAuditLinkToast 复用
 * - macAddress undefined 表示仅 IP 绑定 (后端 service 按 op=add/remove 路由)
 *
 * 来源: 56-04-PLAN.md Task 5, 56-PATTERNS.md PortBindingModal 骨架, 56-UI-SPEC §Form Rules
 */

import { useEffect, useState } from "react";
import { Modal, Form, Select, Input, Radio, App } from "antd";
import { useNavigate } from "react-router-dom";
import type { DevicePortStatus } from "@/types/network";
import { writePortBinding } from "@/lib/api/networkApi";
import {
  PRESET_REASONS,
  ACTION_TITLE,
  REASON_MIN,
  REASON_MAX,
  REASON_CUSTOM_SENTINEL,
  IPV4_REGEX,
  MAC_REGEX,
  BIND_OPS,
  composeReason,
  validateReasonRequired,
} from "./constants";
import { showAuditLinkToast } from "./PortWriteModal";

export interface PortBindingModalProps {
  open: boolean;
  portRecord: DevicePortStatus | null;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 单端口 port_binding Modal
 *
 * 端口列表页"操作"列点击"端口绑定"菜单项打开。
 * ports/index.tsx 通过 bindModalOpen + bindModalRecord state 控制。
 */
export function PortBindingModal({
  open,
  portRecord,
  onClose,
  onSuccess,
}: PortBindingModalProps) {
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const navigate = useNavigate();
  // IN-03: submitting 防写操作重复提交 (wrapper await 期间用户可重复点"确认执行")
  const [submitting, setSubmitting] = useState(false);

  // WR-05 (2026-07-09 修复): 用 setFieldsValue 替代 resetFields,与 CR-03
  // SetAccessVlanModal 修复一致 — resetFields() 会清空 op/ip/mac 字段到 undefined,
  // 加上 destroyOnHidden Form 每次重挂载 → 用户每次重开 Modal 看到空表单
  // (op Radio.Group 无选中、ip/mac 输入框空),需手动重填。
  // 这里 op 默认 "add" 是显式默认值(BIND_OPS[0]),与原 initialValues 一致。
  // deps 全稳定: open 原始值, form 来自 useForm 稳定 ref — CLAUDE.md useEffect 纪律满足
  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        op: "add",
        ipAddress: "",
        macAddress: undefined,
        reasonSelect: undefined,
        reasonText: undefined,
      });
    }
  }, [open, form]);

  const handleOk = async (): Promise<void> => {
    setSubmitting(true);
    try {
      let values: Record<string, unknown>;
      try {
        values = await form.validateFields();
      } catch (err) {
        // validateFields 失败抛 errorFields, antd 自动标红字段, 不需 Toast
        if (err && typeof err === "object" && "errorFields" in err) return;
        // WR-01: 非预期校验异常不再 throw 冒泡到 antd onOk 形成未处理 Promise rejection
        console.error("[PortBindingModal] validateFields unexpected error:", err);
        return;
      }

      // 组装最终 reason (预设项直接用 value; custom 项用 reasonText)
      const reason = composeReason(values.reasonSelect, values.reasonText);

      // 防御性: portRecord 应永远非空 (ActionButtons onClick 一定带 record)
      if (!portRecord) return;

      // macAddress: 空字符串 → undefined (仅 IP 绑定); 非空字符串透传给后端 service 归一化
      const macAddressRaw = values.macAddress as string | undefined;
      const macAddress =
        typeof macAddressRaw === "string" && macAddressRaw.trim().length > 0
          ? macAddressRaw.trim()
          : undefined;

      // LANDMINE #5: wrapper reject 由 post() 拦截器统一弹 Toast, 本组件不再 message.error
      await writePortBinding(
        portRecord.id,
        values.op as "add" | "remove",
        values.ipAddress as string,
        macAddress,
        reason ?? ""
      );

      // D-10 成功 Toast 含"查看审计日志"链接 (复用 PortWriteModal helper)
      showAuditLinkToast(message, navigate);

      form.resetFields();
      onSuccess();
      onClose();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={`${ACTION_TITLE["port_binding"]} - ${portRecord?.interfaceName ?? ""}`}
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      destroyOnHidden
      width={520}
      okText="确认执行"
      cancelText="取消"
      okButtonProps={{ loading: submitting }}
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{ op: "add", ipAddress: "", macAddress: "" }}
      >
        {/* op 主字段: Radio.Group buttonStyle="solid", default "add" */}
        <Form.Item
          name="op"
          label="操作"
          rules={[{ required: true, message: "请选择绑定操作" }]}
        >
          <Radio.Group
            buttonStyle="solid"
            options={BIND_OPS as unknown as Array<{ label: string; value: string }>}
          />
        </Form.Item>

        {/* ipAddress 主字段: Input required + IPV4_REGEX pattern (UX hint) */}
        <Form.Item
          name="ipAddress"
          label="IP 地址"
          rules={[
            { required: true, message: "请输入 IP 地址" },
            {
              pattern: IPV4_REGEX,
              message: "请输入合法 IPv4 地址（如 10.62.25.5）",
            },
          ]}
        >
          <Input placeholder="例如 10.62.25.5" allowClear />
        </Form.Item>

        {/* macAddress 主字段: Input optional + MAC_REGEX pattern (UX hint) */}
        <Form.Item
          name="macAddress"
          label="MAC 地址（可选）"
          rules={[
            {
              pattern: MAC_REGEX,
              message: "请输入合法 MAC 地址（如 AA:BB:CC:DD:EE:FF）",
            },
          ]}
          extra="不填则仅 IP 绑定；后端 service 会归一化为各厂商格式"
        >
          <Input
            placeholder="例如 AA:BB:CC:DD:EE:FF（不填则仅 IP 绑定）"
            allowClear
          />
        </Form.Item>

        {/* D-02 reason 字段 — 外层 reasonSelect Select + 内层 reasonText TextArea (仅 __custom__ 时展开) */}
        <Form.Item
          name="reasonSelect"
          label="操作原因"
          rules={[
            {
              required: true,
              validator: (rule: unknown, value: unknown) =>
                validateReasonRequired(rule, value, form),
            },
          ]}
        >
          {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
          <Select
            placeholder="请选择操作原因"
            options={PRESET_REASONS.map((opt) => ({
              label: opt.label,
              value: opt.value,
            }))}
          />
        </Form.Item>

        {/* shouldUpdate 监听 reasonSelect 值, 仅选 __custom__ 时展开 TextArea */}
        <Form.Item shouldUpdate noStyle>
          {({ getFieldValue }) =>
            getFieldValue("reasonSelect") === REASON_CUSTOM_SENTINEL ? (
              <Form.Item
                name="reasonText"
                label="自定义原因"
                rules={[
                  {
                    max: REASON_MAX,
                    message: `操作原因不超过 ${REASON_MAX} 个字符`,
                  },
                ]}
              >
                <Input.TextArea
                  rows={2}
                  maxLength={REASON_MAX}
                  showCount
                  placeholder={`请输入操作原因（${REASON_MIN}-${REASON_MAX} 字符）`}
                />
              </Form.Item>
            ) : null
          }
        </Form.Item>
      </Form>
    </Modal>
  );
}

export default PortBindingModal;
