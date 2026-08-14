import React from "react";
import { Modal, Form, Select, DatePicker, Switch, Alert } from "antd";
import type { DutyPool } from "@/lib/dutyApi";

const { RangePicker } = DatePicker;

interface GenerateScheduleModalProps {
  visible: boolean;
  onCancel: () => void;
  onOk: (values: {
    poolId: string;
    startDate: string;
    endDate: string;
    dutyType: string;
    clearExists: boolean;
  }) => Promise<boolean>;
  pools: DutyPool[];
}

export const GenerateScheduleModal: React.FC<GenerateScheduleModalProps> = ({
  visible,
  onCancel,
  onOk,
  pools,
}) => {
  const [form] = Form.useForm();

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      const success = await onOk({
        poolId: values.poolId,
        startDate: values.dateRange[0].format("YYYY-MM-DD"),
        endDate: values.dateRange[1].format("YYYY-MM-DD"),
        dutyType: values.dutyType,
        clearExists: values.clearExists || false,
      });
      if (success) {
        form.resetFields();
      }
      return success;
    } catch (error) {
      return false;
    }
  };

  return (
    <Modal
      title="生成排班"
      open={visible}
      onOk={handleOk}
      onCancel={() => {
        form.resetFields();
        onCancel();
      }}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" preserve={false}>
        <Form.Item
          name="poolId"
          label="值班池"
          rules={[{ required: true, message: "请选择值班池" }]}
        >
          <Select placeholder="请选择值班池" onSearch={() => {}}>
            {pools.map((pool) => (
              <Select.Option key={pool.id} value={pool.id}>
                {pool.poolName}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item
          name="dateRange"
          label="日期范围"
          rules={[{ required: true, message: "请选择日期范围" }]}
        >
          <RangePicker style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item
          name="dutyType"
          label="值班类型"
          rules={[{ required: true, message: "请选择值班类型" }]}
          initialValue="weekday"
        >
          <Select placeholder="请选择值班类型" onSearch={() => {}}>
            <Select.Option value="weekday">工作日值班</Select.Option>
            <Select.Option value="weekend">周末值班</Select.Option>
            <Select.Option value="holiday">节假日值班</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item
          name="clearExists"
          label="清除已有排班"
          valuePropName="checked"
          initialValue={false}
        >
          <Switch checkedChildren="是" unCheckedChildren="否" />
        </Form.Item>
        <Alert
          title="系统将根据值班池中的成员顺序，按轮询方式自动生成排班记录。只对符合所选值班类型的日期进行排班。"
          type="info"
          showIcon
        />
      </Form>
    </Modal>
  );
};

export default GenerateScheduleModal;
