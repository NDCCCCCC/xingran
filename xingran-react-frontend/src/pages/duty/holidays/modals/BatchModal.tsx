/**
 * Holiday Batch Add Modal
 * 节假日批量新增模态框
 */

import { Modal, Button, DatePicker, Switch } from "antd";
// Phase 71 (W5)：裸 Input/Select → 契约组件（children 形态改写 options）
import { AppInput, AppSelect } from "@/components/ui";
import { useState } from "react";
import { PlusOutlined } from "@ant-design/icons";
import type { DictItem } from "@/hooks/useDict";
import type { BatchHolidayRow } from "../types";
import { HOLIDAY_TYPE_OPTIONS } from "../constants";

export interface HolidayBatchModalProps {
  open: boolean;
  batchHolidays: BatchHolidayRow[];
  /** Phase 69 DICT-03: duty_holiday_type 字典数据（父页面 useDict 拉取后透传；
   *  空数组时下拉回退静态 HOLIDAY_TYPE_OPTIONS） */
  holidayTypeDict?: DictItem[];
  onOk: () => Promise<void>;
  onCancel: () => void;
  onAddRow: () => void;
  onRemoveRow: (index: number) => void;
  onUpdateRow: (index: number, field: string, value: unknown) => void;
}

export function HolidayBatchModal({
  open,
  batchHolidays,
  holidayTypeDict = [],
  onOk,
  onCancel,
  onAddRow,
  onRemoveRow,
  onUpdateRow,
}: HolidayBatchModalProps) {
  const [submitting, setSubmitting] = useState(false);
  return (
    <Modal
      title="批量新增节假日"
      open={open}
      onOk={async () => {
        setSubmitting(true);
        try {
          await onOk();
        } finally {
          setSubmitting(false);
        }
      }}
      cancelButtonProps={{ disabled: submitting }}
      onCancel={onCancel}
      width={800}
      destroyOnHidden
    >
      <div className="mb-4">
        <Button type="dashed" onClick={onAddRow} block icon={<PlusOutlined />}>
          添加一行
        </Button>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full border-collapse">
          <thead>
            <tr className="bg-gray-50">
              <th className="border p-2">日期</th>
              <th className="border p-2">名称</th>
              <th className="border p-2">类型</th>
              <th className="border p-2">是否休息</th>
              <th className="border p-2">操作</th>
            </tr>
          </thead>
          <tbody>
            {batchHolidays.map((row, index) => (
              <tr key={index}>
                <td className="border p-2">
                  <DatePicker
                    value={row.holidayDate}
                    onChange={(date) => onUpdateRow(index, "holidayDate", date)}
                    style={{ width: "100%" }}
                  />
                </td>
                <td className="border p-2">
                  <AppInput allowClear
                    value={row.holidayName}
                    onChange={(e) => onUpdateRow(index, "holidayName", e.target.value)}
                    placeholder="节假日名称"
                  />
                </td>
                <td className="border p-2">
                  {/* 形态分类器②：children 写法改写 options；原 antd 单选默认禁搜，
                      显式 showSearch={false} 保持不变 */}
                  <AppSelect
                    allowClear
                    value={row.holidayType}
                    onChange={(v) => onUpdateRow(index, "holidayType", v)}
                    style={{ width: "100%" }}
                    showSearch={false}
                    onSearch={() => {}}
                    options={
                      holidayTypeDict.length > 0
                        ? holidayTypeDict.map((d) => ({ value: d.dictValue, label: d.dictLabel }))
                        : HOLIDAY_TYPE_OPTIONS.map((opt) => ({ value: opt.value, label: opt.label }))
                    }
                  />
                </td>
                <td className="border p-2 text-center">
                  <Switch
                    checked={row.isOffday}
                    onChange={(v) => onUpdateRow(index, "isOffday", v)}
                    checkedChildren="休息"
                    unCheckedChildren="工作"
                  />
                </td>
                <td className="border p-2 text-center">
                  <Button
                    type="link"
                    size="small"
                    style={{ color: "var(--theme-error)" }}
                    onClick={() => onRemoveRow(index)}
                  >
                    删除
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Modal>
  );
}
