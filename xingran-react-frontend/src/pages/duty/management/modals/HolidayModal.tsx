/**
 * 节假日模态框组件
 */

import { Modal, Form, Select, DatePicker } from "antd";
import { HOLIDAY_NAME_OPTIONS, HOLIDAY_TYPE_OPTIONS, HOLIDAY_REMARK_OPTIONS } from "../constants";

const { Option } = Select;

interface HolidayModalProps {
  visible: boolean;
  onOk: () => void;
  onCancel: () => void;
}

export function HolidayModal({ visible, onOk, onCancel }: HolidayModalProps) {
  const [form] = Form.useForm();

  const handleOk = async () => {
    await onOk();
    form.resetFields();
  };

  return (
    <Modal
      title="节假日"
      open={visible}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item name="holidayDate" label="日期" rules={[{ required: true, message: "请选择日期" }]}>
          <DatePicker style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="holidayName" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
          <Select placeholder="选择或输入" onSearch={() => {}}>
            {HOLIDAY_NAME_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="holidayType" label="类型" rules={[{ required: true }]} initialValue="custom">
          <Select onSearch={() => {}}>
            {HOLIDAY_TYPE_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="isOffday" label="是否休息" valuePropName="checked" initialValue={true}>
          <Select onSearch={() => {}}>
            <Option value={true}>休息</Option>
            <Option value={false}>工作</Option>
          </Select>
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <Select placeholder="选择备注（可选）" allowClear onSearch={() => {}}>
            {HOLIDAY_REMARK_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
}
