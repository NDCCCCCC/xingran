import React from "react";
import { Modal, Form, DatePicker, Switch, Alert } from "antd";
// Phase 71 (W5)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
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
    } catch (_error) {
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
          {/* 形态分类器②：children 写法改写 options；原 antd 单选默认禁搜，显式
              showSearch={false} 保持不变。poolId 为 UUID 外键，原 allowClear=true 行为不变 */}
          <AppSelect
            allowClear
            placeholder="请选择值班池"
            showSearch={false}
            onSearch={() => {}}
            options={pools.map((pool) => ({ value: pool.id, label: pool.poolName }))}
          />
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
          <AppSelect
            allowClear
            placeholder="请选择值班类型"
            showSearch={false}
            onSearch={() => {}}
            options={[
              { value: "weekday", label: "工作日值班" },
              { value: "weekend", label: "周末值班" },
              { value: "holiday", label: "节假日值班" },
            ]}
          />
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
