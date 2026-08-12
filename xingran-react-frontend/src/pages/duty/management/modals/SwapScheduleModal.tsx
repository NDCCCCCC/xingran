/**
 * 调班模态框组件
 */

import { Modal, Form, Select } from "antd";
import { formatDate } from "@/utils/datetime";
import type { DutySchedule } from "@/lib/dutyApi";
import { SWAP_REASON_OPTIONS } from "../constants";

const { Option } = Select;

interface SwapScheduleModalProps {
  visible: boolean;
  allSchedules: DutySchedule[];
  onOk: () => void;
  onCancel: () => void;
}

export function SwapScheduleModal({ visible, allSchedules, onOk, onCancel }: SwapScheduleModalProps) {
  const [form] = Form.useForm();

  const handleOk = async () => {
    await onOk();
    form.resetFields();
  };

  return (
    <Modal
      title="调班"
      open={visible}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item name="fromScheduleId" label="原排班" rules={[{ required: true }]}>
          <Select placeholder="选择原排班" onSearch={() => {}}>
            {allSchedules.map((s) => (
              <Option key={s.id} value={s.id}>
                {formatDate(s.scheduleDate)} - {s.user?.nickname || s.user?.username}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="toScheduleId" label="目标排班" rules={[{ required: true }]}>
          <Select placeholder="选择目标排班" onSearch={() => {}}>
            {allSchedules.map((s) => (
              <Option key={s.id} value={s.id}>
                {formatDate(s.scheduleDate)} - {s.user?.nickname || s.user?.username}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="reason" label="调班原因" rules={[{ required: true, message: "请输入调班原因" }]}>
          <Select placeholder="选择原因" onSearch={() => {}}>
            {SWAP_REASON_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
}
