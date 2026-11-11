/**
 * 节假日模态框组件
 */

import { Modal, Form, DatePicker } from "antd";
// Phase 71 (W5)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
// Phase 69 DICT-03：假日类型下拉按字典 duty_holiday_type 渲染（由父页面拉取后透传；
// 未传或字典为空时回落静态常量，模态框本身不依赖 react-query）
import type { DictItem } from "@/hooks/useDict";
import { HOLIDAY_NAME_OPTIONS, HOLIDAY_TYPE_OPTIONS, HOLIDAY_REMARK_OPTIONS } from "../constants";

interface HolidayModalProps {
  visible: boolean;
  onOk: () => void;
  onCancel: () => void;
  /** duty_holiday_type 字典数据（父页面 useDict 拉取后透传；空数组回落静态选项） */
  holidayTypeDict?: DictItem[];
}

export function HolidayModal({ visible, onOk, onCancel, holidayTypeDict = [] }: HolidayModalProps) {
  const [form] = Form.useForm();

  // 假日类型字典（duty_holiday_type）：空数组时回落静态 HOLIDAY_TYPE_OPTIONS
  const holidayTypeOptions =
    holidayTypeDict.length > 0
      ? holidayTypeDict.map((d) => ({ value: d.dictValue, label: d.dictLabel }))
      : HOLIDAY_TYPE_OPTIONS.map((opt) => ({ value: opt.value, label: opt.label }));
  // 字典配了 isDefault 就用它，否则沿用本页既有默认值 custom
  const defaultHolidayType = holidayTypeDict.find((d) => d.isDefault)?.dictValue || "custom";

  const handleOk = async () => {
    await onOk();
    form.resetFields();
  };

  return (
    <Modal title="节假日" open={visible} onOk={handleOk} onCancel={onCancel} destroyOnHidden>
      <Form form={form} layout="vertical">
        <Form.Item
          name="holidayDate"
          label="日期"
          rules={[{ required: true, message: "请选择日期" }]}
        >
          <DatePicker style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item
          name="holidayName"
          label="名称"
          rules={[{ required: true, message: "请输入名称" }]}
        >
          {/* 形态分类器②：children 写法改写 options；原 antd 单选默认禁搜，
              显式 showSearch={false} 保持不变 */}
          <AppSelect
            allowClear
            placeholder="选择或输入"
            showSearch={false}
            options={HOLIDAY_NAME_OPTIONS.map((opt) => ({ value: opt.value, label: opt.label }))}
          />
        </Form.Item>
        <Form.Item
          name="holidayType"
          label="类型"
          rules={[{ required: true }]}
          initialValue={defaultHolidayType}
        >
          <AppSelect allowClear showSearch={false} options={holidayTypeOptions} />
        </Form.Item>
        <Form.Item name="isOffday" label="是否休息" valuePropName="checked" initialValue={true}>
          <AppSelect
            allowClear
            showSearch={false}
            options={[
              { value: true, label: "休息" },
              { value: false, label: "工作" },
            ]}
          />
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <AppSelect
            allowClear
            placeholder="选择备注（可选）"
            showSearch={false}
            options={HOLIDAY_REMARK_OPTIONS.map((opt) => ({ value: opt.value, label: opt.label }))}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
