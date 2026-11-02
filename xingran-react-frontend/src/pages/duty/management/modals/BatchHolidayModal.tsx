/**
 * 批量添加节假日模态框组件
 */

import { Modal, Form, DatePicker, App } from "antd";
// Phase 71 (W5)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
import { useState } from "react";
import { BATCH_HOLIDAY_NAME_OPTIONS, MAX_BATCH_DAYS } from "../constants";
import type { BatchHolidayFormValues } from "../types";

const { RangePicker } = DatePicker;

interface BatchHolidayModalProps {
  visible: boolean;
  onOk: (values: BatchHolidayFormValues) => Promise<void>;
  onCancel: () => void;
}

export function BatchHolidayModal({ visible, onOk, onCancel }: BatchHolidayModalProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);

  const handleOk = async () => {
    setSubmitting(true);
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
    } catch (_error) {
      // 表单验证失败
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title="批量添加节假日"
      open={visible}
      onOk={handleOk}
      cancelButtonProps={{ disabled: submitting }}
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
        <Form.Item
          name="holidayName"
          label="名称"
          rules={[{ required: true, message: "请输入名称" }]}
        >
          {/* 形态分类器②：children 写法改写 options；原 antd 单选默认禁搜，显式
              showSearch={false} 保持不变 */}
          <AppSelect
            allowClear
            placeholder="选择假期名称"
            showSearch={false}
            onSearch={() => {}}
            options={BATCH_HOLIDAY_NAME_OPTIONS.map((opt) => ({
              value: opt.value,
              label: opt.label,
            }))}
          />
        </Form.Item>
        <Form.Item
          name="holidayType"
          label="类型"
          rules={[{ required: true }]}
          initialValue="legal"
        >
          <AppSelect
            allowClear
            showSearch={false}
            onSearch={() => {}}
            options={[
              { value: "legal", label: "法定节假日" },
              { value: "custom", label: "自定义" },
            ]}
          />
        </Form.Item>
        <Form.Item name="isOffday" label="是否休息" valuePropName="checked" initialValue={true}>
          <AppSelect
            allowClear
            showSearch={false}
            onSearch={() => {}}
            options={[
              { value: true, label: "休息" },
              { value: false, label: "工作" },
            ]}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
