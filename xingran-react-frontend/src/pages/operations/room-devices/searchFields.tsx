import React from "react";
import { Form } from "antd";
import type { SearchField } from "@/components/shared";
import { AppSelect } from "@/components/ui";
import { DEVICE_TYPE_OPTIONS, ROOM_DEVICE_STATUS_OPTIONS } from "./constants";

export interface SearchFieldsOptions {
  selectedDeptId: string | null;
  roomSelectOptions: Array<{ label: string; value: string }>;
}

export const getRoomDeviceSearchFields = ({
  selectedDeptId,
  roomSelectOptions,
}: SearchFieldsOptions): SearchField[] => [
  {
    type: "custom",
    name: "roomId",
    render: (value, onChange) => (
      <Form.Item label="所属机房" style={{ marginBottom: 0 }}>
        <AppSelect
          value={(value as string | undefined) ?? undefined}
          onChange={(v) => onChange(v)}
          placeholder={selectedDeptId ? "请选择机房" : "请先选择部门"}
          disabled={!selectedDeptId}
          showSearch
          style={{ width: 150 }}
          options={roomSelectOptions}
        />
      </Form.Item>
    ),
  },
  {
    type: "input",
    name: "name",
    label: "设备名称",
    placeholder: "请输入设备名称",
  },
  {
    type: "select",
    name: "deviceType",
    label: "设备类型",
    placeholder: "请选择类型",
    options: DEVICE_TYPE_OPTIONS,
  },
  {
    type: "select",
    name: "status",
    label: "状态",
    placeholder: "请选择状态",
    options: ROOM_DEVICE_STATUS_OPTIONS,
  },
];
