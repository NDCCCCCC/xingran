/**
 * Holiday Edit Modal
 * 节假日编辑模态框
 */

import { Form, Input, InputNumber, Select, Switch, Modal, DatePicker } from "antd";
import type { FormInstance } from "antd/es/form";
import type { Holiday } from "@/lib/dutyApi";
import { HOLIDAY_TYPE_OPTIONS } from "../constants";
import dayjs from "dayjs";
import { useEffect } from "react";

const { Option } = Select;
const { TextArea } = Input;

export interface HolidayEditModalProps {
  open: boolean;
  editingRecord: Holiday | null;
  year: number | undefined;
  availableYears: number[];
  onOk: (form: FormInstance<unknown>) => Promise<void>;
  onCancel: () => void;
}

export function HolidayEditModal({
  open,
  editingRecord,
  year,
  availableYears,
  onOk,
  onCancel,
}: HolidayEditModalProps) {
  const [form] = Form.useForm();

  const getDefaultYear = () => year ?? availableYears[0] ?? new Date().getFullYear();

  // 当编辑记录变化时，更新表单
  useEffect(() => {
    if (open) {
      if (editingRecord) {
        // 编辑模式：设置现有数据
        form.setFieldsValue({
          holidayDate: dayjs(editingRecord.holidayDate),
          holidayName: editingRecord.holidayName,
          isOffday: editingRecord.isOffday,
          holidayType: editingRecord.holidayType,
          year: editingRecord.year,
          remark: editingRecord.remark,
        });
      } else {
        // 新增模式：设置默认值
        form.setFieldsValue({
          holidayDate: dayjs(),
          isOffday: true,
          holidayType: "legal",
          year: getDefaultYear(),
        });
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- getDefaultYear defined in render
  }, [open, editingRecord, form, year, availableYears]);

  return (
    <Modal
      title={editingRecord ? "编辑节假日" : "新增节假日"}
      open={open}
      onOk={() => onOk(form)}
      onCancel={onCancel}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" preserve={false}>
        <Form.Item
          name="holidayDate"
          label="日期"
          rules={[{ required: true, message: "请选择日期" }]}
        >
          <DatePicker style={{ width: "100%" }} />
        </Form.Item>

        <Form.Item
          name="holidayName"
          label="节假日名称"
          rules={[{ required: true, message: "请输入节假日名称" }]}
        >
          <Input placeholder="请输入节假日名称" />
        </Form.Item>

        <Form.Item
          name="holidayType"
          label="类型"
          rules={[{ required: true, message: "请选择类型" }]}
        >
          <Select placeholder="请选择类型" onSearch={() => {}}>
            {HOLIDAY_TYPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item name="isOffday" label="是否休息日" valuePropName="checked">
          <Switch checkedChildren="休息" unCheckedChildren="工作" />
        </Form.Item>

        <Form.Item name="year" label="年份" rules={[{ required: true, message: "请输入年份" }]}>
          <InputNumber min={2020} max={2030} style={{ width: "100%" }} />
        </Form.Item>

        <Form.Item name="remark" label="备注">
          <TextArea rows={3} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>
  );
}
