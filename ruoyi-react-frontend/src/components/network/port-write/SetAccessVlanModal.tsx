/**
 * SetAccessVlanModal — Phase 56 W4 单端口 set_access_vlan Modal (v1.20.1 VLAN-01)
 *
 * 与 PortWriteModal (Phase 53 W4) 同风格: 单 Form Item 主字段 (vlanId InputNumber 1-4094)
 * + reason 预置 Select + 自定义 TextArea 双层结构 (REQUIRED, 非 Optional)。
 *
 * 设计约束 (56-04-PLAN / 56-UI-SPEC):
 * - vlanId InputNumber min=1 max=4094 step=1, Form.Item extra="范围 1-4094 (VLAN 0/4095 保留)"
 * - reason REQUIRED: validateReasonRequired (55-01 WR-02 跨字段 form 参数版)
 * - LANDMINE #5: wrapper 无 try/catch, 组件无 message.error (post() 拦截器统一弹 Toast)
 * - destroyOnHidden + form.resetFields() on open (防上次 vlanId/reason 残留)
 * - D-10: 成功后 showAuditLinkToast 复用 (react-router <a href> + navigate, 非 <Link>)
 * - initialValues vlanId 预填当前 port vlan (DevicePortStatus.vlan)
 *
 * 来源: 56-04-PLAN.md Task 4, 56-PATTERNS.md SetAccessVlanModal 骨架, 56-UI-SPEC §Form Rules
 */

import { useEffect, useState } from "react";
import { Modal, Form, Select, Input, InputNumber, App } from "antd";
import type { MessageInstance } from "antd/es/message/interface";
import { useNavigate } from "react-router-dom";
import type { DevicePortStatus } from "@/types/network";
import { writeSetAccessVlan } from "@/lib/api/networkApi";
import {
  PRESET_REASONS,
  ACTION_TITLE,
  REASON_MIN,
  REASON_MAX,
  REASON_CUSTOM_SENTINEL,
  composeReason,
  validateReasonRequired,
} from "./constants";
import { showAuditLinkToast } from "./PortWriteModal";

export interface SetAccessVlanModalProps {
  open: boolean;
  portRecord: DevicePortStatus | null;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * 单端口 set_access_vlan Modal
 *
 * 端口列表页"操作"列点击"修改 access VLAN"菜单项打开。
 * ports/index.tsx 通过 vlanModalOpen + vlanModalRecord state 控制。
 */
export function SetAccessVlanModal({
  open,
  portRecord,
  onClose,
  onSuccess,
}: SetAccessVlanModalProps) {
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const navigate = useNavigate();
  // IN-03: submitting 防写操作重复提交 (wrapper await 期间用户可重复点"确认执行")
  const [submitting, setSubmitting] = useState(false);

  // CR-03 (2026-07-09 修复): 用 setFieldsValue 替代 resetFields,保留 initialValues
  // 预填的当前 port vlan (D-02 设计契约)。原版 resetFields() 会把 vlanId 字段
  // 清回 undefined,加上 destroyOnHidden Form 每次重挂载 → 用户打开 Modal
  // 永远看到空 VLAN 输入框,违背 D-02 pre-fill 目的。
  // deps 全稳定: open 原始值, form 来自 useForm 稳定 ref, portRecord 来自
  // parent — 但因为外层 destroyOnHidden 触发 component re-mount,这里用
  // 完整 portRecord 引用作为依赖触发 on-open 重设。
  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        vlanId: portRecord?.vlan ?? 1,
        reasonSelect: undefined,
        reasonText: undefined,
      });
    }
  }, [open, portRecord, form]);

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
        console.error("[SetAccessVlanModal] validateFields unexpected error:", err);
        return;
      }

      // 组装最终 reason (预设项直接用 value; custom 项用 reasonText)
      const reason = composeReason(values.reasonSelect, values.reasonText);

      // 防御性: portRecord 应永远非空 (ActionButtons onClick 一定带 record)
      if (!portRecord) return;

      // LANDMINE #5: wrapper reject 由 post() 拦截器统一弹 Toast, 本组件不再 message.error
      await writeSetAccessVlan(
        portRecord.id,
        values.vlanId as number,
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
      title={`${ACTION_TITLE["set_access_vlan"]} - ${portRecord?.interfaceName ?? ""}`}
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
        initialValues={{ vlanId: portRecord?.vlan ?? 1 }}
      >
        {/* vlanId 主字段: InputNumber 1-4094 + 范围提示 */}
        <Form.Item
          name="vlanId"
          label="VLAN ID"
          rules={[
            { required: true, message: "请输入 VLAN ID" },
            {
              type: "number",
              min: 1,
              max: 4094,
              message: "VLAN ID 必须在 1-4094 之间",
            },
          ]}
          extra="范围 1-4094 (VLAN 0/4095 保留)"
        >
          <InputNumber
            min={1}
            max={4094}
            step={1}
            style={{ width: "100%" }}
            placeholder="请输入 1-4094 之间的 VLAN ID"
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

export default SetAccessVlanModal;
