/**
 * 手动排班模态框组件
 */

import { Modal, Form, Select, DatePicker } from "antd";
import type { DutyPool, SimpleUser } from "@/lib/dutyApi";
import { DUTY_TYPE_OPTIONS, MANUAL_REASON_OPTIONS } from "../constants";

const { Option } = Select;

interface ManualScheduleModalProps {
  visible: boolean;
  pools: DutyPool[];
  users: SimpleUser[];
  onOk: () => void;
  onCancel: () => void;
}

export function ManualScheduleModal({ visible, pools, users, onOk, onCancel }: ManualScheduleModalProps) {
  const [form] = Form.useForm();

  const handleOk = async () => {
    await onOk();
    form.resetFields();
  };

  return (
    <Modal
      title="手动排班"
      open={visible}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
      width={500}
    >
      <Form form={form} layout="vertical">
        <Form.Item name="poolId" label="值班池" rules={[{ required: true, message: "请选择值班池" }]}>
          <Select placeholder="选择值班池" onSearch={() => {}}>
            {pools.map((pool) => (
              <Option key={pool.id} value={pool.id}>{pool.poolName}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="dutyDate" label="值班日期" rules={[{ required: true, message: "请选择日期" }]}>
          <DatePicker style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="dutyType" label="值班类型" rules={[{ required: true }]} initialValue="weekday">
          <Select onSearch={() => {}}>
            {DUTY_TYPE_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="userIds" label="值班人员" rules={[{ required: true, message: "请选择人员" }]}>
          <Select mode="multiple" placeholder="选择人员" onSearch={() => {}}>
            {users.map((user) => (
              <Option key={user.id} value={user.id}>{user.nickname || user.username}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="reason" label="备注">
          <Select placeholder="选择原因（可选）" allowClear onSearch={() => {}}>
            {MANUAL_REASON_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
}
