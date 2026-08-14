/**
 * Holiday Batch Add Modal
 * 节假日批量新增模态框
 */

import { Modal, Button, DatePicker, Input, Select, Switch } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import type { BatchHolidayRow } from "../types";
import { HOLIDAY_TYPE_OPTIONS } from "../constants";

const { Option } = Select;

export interface HolidayBatchModalProps {
  open: boolean;
  batchHolidays: BatchHolidayRow[];
  onOk: () => Promise<void>;
  onCancel: () => void;
  onAddRow: () => void;
  onRemoveRow: (index: number) => void;
  onUpdateRow: (index: number, field: string, value: unknown) => void;
}

export function HolidayBatchModal({
  open,
  batchHolidays,
  onOk,
  onCancel,
  onAddRow,
  onRemoveRow,
  onUpdateRow,
}: HolidayBatchModalProps) {
  return (
    <Modal
      title="批量新增节假日"
      open={open}
      onOk={onOk}
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
                  <Input
                    value={row.holidayName}
                    onChange={(e) => onUpdateRow(index, "holidayName", e.target.value)}
                    placeholder="节假日名称"
                  />
                </td>
                <td className="border p-2">
                  <Select
                    value={row.holidayType}
                    onChange={(v) => onUpdateRow(index, "holidayType", v)}
                    style={{ width: "100%" }}
                    onSearch={() => {}}
                  >
                    {HOLIDAY_TYPE_OPTIONS.map((opt) => (
                      <Option key={opt.value} value={opt.value}>
                        {opt.label}
                      </Option>
                    ))}
                  </Select>
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
                    style={{ color: "var(--theme-error, #ff4d4f)" }}
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
