/**
 * 手动排班模态框组件
 */

import { Modal, Form, DatePicker } from "antd";
// Phase 71 (W5)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
import { useState } from "react";
import type { DutyPool, SimpleUser } from "@/lib/dutyApi";
import { DUTY_TYPE_OPTIONS, MANUAL_REASON_OPTIONS } from "../constants";

interface ManualScheduleModalProps {
  visible: boolean;
  pools: DutyPool[];
  users: SimpleUser[];
  onOk: () => void;
  onCancel: () => void;
}

export function ManualScheduleModal({
  visible,
  pools,
  users,
  onOk,
  onCancel,
}: ManualScheduleModalProps) {
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);

  const handleOk = async () => {
    setSubmitting(true);
    try {
      await onOk();
      form.resetFields();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title="手动排班"
      open={visible}
      onOk={handleOk}
      cancelButtonProps={{ disabled: submitting }}
      onCancel={onCancel}
      destroyOnHidden
      width={500}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="poolId"
          label="值班池"
          rules={[{ required: true, message: "请选择值班池" }]}
        >
          {/* 形态分类器②：children 写法改写 options；原 antd 单选默认禁搜，
              显式 showSearch={false} 保持不变。poolId 为 UUID 外键，原 allowClear=true 行为不变 */}
          <AppSelect
            allowClear
            placeholder="选择值班池"
            showSearch={false}
            onSearch={() => {}}
            options={pools.map((pool) => ({ value: pool.id, label: pool.poolName }))}
          />
        </Form.Item>
        <Form.Item
          name="dutyDate"
          label="值班日期"
          rules={[{ required: true, message: "请选择日期" }]}
        >
          <DatePicker style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item
          name="dutyType"
          label="值班类型"
          rules={[{ required: true }]}
          initialValue="weekday"
        >
          <AppSelect
            allowClear
            showSearch={false}
            onSearch={() => {}}
            options={DUTY_TYPE_OPTIONS.map((opt) => ({ value: opt.value, label: opt.label }))}
          />
        </Form.Item>
        <Form.Item
          name="userIds"
          label="值班人员"
          rules={[{ required: true, message: "请选择人员" }]}
        >
          {/* 形态分类器②：children 写法改写 options；mode="multiple" 显式 showSearch
              （antd 多选默认开搜索，保持不变）。userIds 为 UUID 外键，原 allowClear=true 行为不变 */}
          <AppSelect
            allowClear
            mode="multiple"
            placeholder="选择人员"
            showSearch
            onSearch={() => {}}
            options={users.map((user) => ({
              value: user.id,
              label: user.nickname || user.username,
            }))}
          />
        </Form.Item>
        <Form.Item name="reason" label="备注">
          <AppSelect
            allowClear
            placeholder="选择原因（可选）"
            showSearch={false}
            onSearch={() => {}}
            options={MANUAL_REASON_OPTIONS.map((opt) => ({ value: opt.value, label: opt.label }))}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
