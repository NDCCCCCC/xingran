/**
 * 批量添加节假日模态框组件
 */

import { Modal, Form, Select, DatePicker, App } from "antd";
import type { FormInstance } from "antd/es/form";
import { BATCH_HOLIDAY_NAME_OPTIONS, MAX_BATCH_DAYS } from "../constants";
import type { BatchHolidayFormValues } from "../types";

const { Option } = Select;
const { RangePicker } = DatePicker;

interface BatchHolidayModalProps {
  visible: boolean;
  onOk: (values: BatchHolidayFormValues) => Promise<void>;
  onCancel: () => void;
}

export function BatchHolidayModal({ visible, onOk, onCancel }: BatchHolidayModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm();

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      const { dateRange } = values;

      const startDate = dateRange[0];
      const endDate = dateRange[1];

      // 限制日期范围最多90天，防止用户误操作创建过多数据
      const daysDiff = endDate.diff(startDate, "day") + 1;
      if (daysDiff > MAX_BATCH_DAYS) {
        message.error(`日期范围不能超过 ${MAX_BATCH_DAYS} 天（当前选择 ${daysDiff} 天）`);
        return;
      }

      await onOk(values);
      form.resetFields();
    } catch (error) {
      // 表单验证失败
    }
  };

  return (
    <Modal
      title="批量添加节假日"
      open={visible}
      onOk={handleOk}
      onCancel={onCancel}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="dateRange"
          label="日期范围"
          rules={[{ required: true, message: "请选择日期范围" }]}
          extra={`最多可选择${MAX_BATCH_DAYS}天`}
        >
          <RangePicker style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="holidayName" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
          <Select placeholder="选择假期名称" onSearch={() => {}}>
            {BATCH_HOLIDAY_NAME_OPTIONS.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="holidayType" label="类型" rules={[{ required: true }]} initialValue="legal">
          <Select onSearch={() => {}}>
            <Option value="legal">法定节假日</Option>
            <Option value="custom">自定义</Option>
          </Select>
        </Form.Item>
        <Form.Item name="isOffday" label="是否休息" valuePropName="checked" initialValue={true}>
          <Select onSearch={() => {}}>
            <Option value={true}>休息</Option>
            <Option value={false}>工作</Option>
          </Select>
        </Form.Item>
      </Form>
    </Modal>
  );
}
