/**
 * 调班模态框组件
 */

import { Modal, Form } from "antd";
// Phase 71 (W5)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
import { formatDate } from "@/utils/datetime";
import type { DutySchedule } from "@/lib/dutyApi";
import { SWAP_REASON_OPTIONS } from "../constants";

interface SwapScheduleModalProps {
  visible: boolean;
  allSchedules: DutySchedule[];
  onOk: () => void;
  onCancel: () => void;
}

const scheduleOptionLabel = (s: DutySchedule) =>
  `${formatDate(s.scheduleDate)} - ${s.user?.nickname || s.user?.username}`;

export function SwapScheduleModal({
  visible,
  allSchedules,
  onOk,
  onCancel,
}: SwapScheduleModalProps) {
  const [form] = Form.useForm();

  const handleOk = async () => {
    await onOk();
    form.resetFields();
  };

  return (
    <Modal title="调班" open={visible} onOk={handleOk} onCancel={onCancel} destroyOnHidden>
      <Form form={form} layout="vertical">
        <Form.Item name="fromScheduleId" label="原排班" rules={[{ required: true }]}>
          {/* 形态分类器②：children 写法改写 options；原 antd 单选默认禁搜，
              显式 showSearch={false} 保持不变。scheduleId 为 UUID 外键，原 allowClear=true 行为不变 */}
          <AppSelect
            allowClear
            placeholder="选择原排班"
            showSearch={false}
            onSearch={() => {}}
            options={allSchedules.map((s) => ({ value: s.id, label: scheduleOptionLabel(s) }))}
          />
        </Form.Item>
        <Form.Item name="toScheduleId" label="目标排班" rules={[{ required: true }]}>
          <AppSelect
            allowClear
            placeholder="选择目标排班"
            showSearch={false}
            onSearch={() => {}}
            options={allSchedules.map((s) => ({ value: s.id, label: scheduleOptionLabel(s) }))}
          />
        </Form.Item>
        <Form.Item
          name="reason"
          label="调班原因"
          rules={[{ required: true, message: "请输入调班原因" }]}
        >
          <AppSelect
            allowClear
            placeholder="选择原因"
            showSearch={false}
            onSearch={() => {}}
            options={SWAP_REASON_OPTIONS.map((opt) => ({ value: opt.value, label: opt.label }))}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
