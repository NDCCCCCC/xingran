import React from "react";
import { Card, Table, Button, Space, Tag, Upload } from "antd";
// Phase 71 (W5)：裸 Select → 契约组件（children 形态改写 options）
import { AppSelect } from "@/components/ui";
import { confirmDelete } from "@/components/shared/confirmDelete";
import type { ColumnsType } from "antd/es/table";
import { PlusOutlined, ReloadOutlined, FileExcelOutlined, UploadOutlined } from "@ant-design/icons";
import { formatDate } from "@/utils/datetime";
import type { Holiday } from "@/lib/dutyApi";
import { usePagination } from "@/hooks/usePagination";
import { createSorter } from "@/utils/tableHelpers";
// 节假日类型字典（duty_holiday_type）：父页面 useDict 拉取后透传，组件内只做解析
import type { DictItem } from "@/hooks/useDict";

interface ImportOptions {
  file: File;
  onProgress?: (event: { percent: number }) => void;
  onSuccess?: (response?: unknown) => void;
  onError?: (error: Error) => void;
}

// 节假日类型静态回退映射（字典为空/未命中时使用）
const HOLIDAY_TYPE_COLOR_MAP: Record<string, string> = {
  legal: "red",
  workday: "orange",
  custom: "blue",
};
const HOLIDAY_TYPE_TEXT_MAP: Record<string, string> = {
  legal: "法定节假日",
  workday: "调休工作日",
  custom: "自定义",
};

// antd Tag 预设色（字典 listClass 归一用）
const TAG_COLOR_PRESETS = ["success", "processing", "error", "warning", "default", "info"];

/**
 * 解析节假日类型标签色：字典 listClass 优先 → 静态回退色板 → antd 默认色。
 */
function resolveHolidayTypeColor(dictItem: DictItem | undefined, type: string): string {
  const listClass = dictItem?.listClass;
  if (listClass) {
    // 字典写作 "primary"，antd Tag 预设色名是 "processing"
    const normalized = listClass === "primary" ? "processing" : listClass;
    if (TAG_COLOR_PRESETS.includes(normalized)) return normalized;
  }
  return HOLIDAY_TYPE_COLOR_MAP[type] ?? "default";
}

/**
 * 解析节假日类型文案：字典 dictLabel 优先 → 静态回退映射 → 原始值（不把未知值吞成空）。
 */
function resolveHolidayTypeText(dictItem: DictItem | undefined, type: string): string {
  return dictItem?.dictLabel || HOLIDAY_TYPE_TEXT_MAP[type] || type || "-";
}

interface HolidayManagementProps {
  holidays: Holiday[];
  loading: boolean;
  holidayYear: number | undefined;
  availableYears: number[];
  onYearChange: (year: number) => void;
  onRefresh: () => void;
  onAdd: () => void;
  onBatchAdd: () => void;
  onEdit: (record: Holiday) => void;
  onDelete: (id: string) => void;
  onImport: (options: ImportOptions) => void;
  onDownloadTemplate: () => void;
  /**
   * duty_holiday_type 字典数据（父页面 useDict 拉取后透传）。
   * 字典优先 → 静态回退映射 → 原始值/"-"（未知值不被吞成空）。
   */
  holidayTypeDict?: DictItem[];
}

export const HolidayManagement: React.FC<HolidayManagementProps> = ({
  holidays,
  loading,
  holidayYear,
  availableYears,
  onYearChange,
  onRefresh,
  onAdd,
  onBatchAdd,
  onEdit,
  onDelete,
  onImport,
  onDownloadTemplate,
  holidayTypeDict = [],
}) => {
  // 使用全局分页 hook
  const { paginationProps } = usePagination();
  const columns: ColumnsType<Holiday> = [
    {
      title: "日期",
      dataIndex: "holidayDate",
      key: "holidayDate",
      width: 120,
      sorter: createSorter<Holiday>("holidayDate", "date"),
      render: (date: string) => formatDate(date),
    },
    {
      title: "名称",
      dataIndex: "holidayName",
      key: "holidayName",
      sorter: createSorter<Holiday>("holidayName", "string"),
    },
    {
      title: "类型",
      dataIndex: "holidayType",
      key: "holidayType",
      width: 100,
      sorter: createSorter<Holiday>("holidayType", "string"),
      render: (type: string) => {
        // 字典（duty_holiday_type）命中 → 字典文案；未命中回退静态映射；皆无显示原始值
        const dictItem = holidayTypeDict.find((d) => d.dictValue === type);
        return (
          <Tag color={resolveHolidayTypeColor(dictItem, type)}>
            {resolveHolidayTypeText(dictItem, type)}
          </Tag>
        );
      },
    },
    {
      title: "是否休息",
      dataIndex: "isOffday",
      key: "isOffday",
      width: 100,
      sorter: createSorter<Holiday>("isOffday", "boolean"),
      render: (isOffday: boolean) => (
        <Tag color={isOffday ? "green" : "default"}>{isOffday ? "休息" : "工作"}</Tag>
      ),
    },
    {
      title: "年份",
      dataIndex: "year",
      key: "year",
      width: 80,
      sorter: createSorter<Holiday>("year", "number"),
    },
    {
      title: "备注",
      dataIndex: "remark",
      key: "remark",
      sorter: createSorter<Holiday>("remark", "string"),
      ellipsis: true,
    },
    {
      title: "操作",
      key: "action",
      width: 150,
      fixed: "right",
      render: (_: unknown, record: Holiday) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => onEdit(record)}>
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            danger
            onClick={() => {
              confirmDelete({
                target: `节假日 ${record.holidayName}`,
                onDelete: async () => {
                  await onDelete(record.id);
                },
              });
            }}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Card variant="borderless">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <Space>
          <span>年份：</span>
          {/* 形态分类器②：children 写法改写 options；原 antd 单选默认禁搜，显式
              showSearch={false} 保持不变（年份数可能 ≥8，避免契约自动开搜索） */}
          <AppSelect
            allowClear
            value={holidayYear}
            // AppSelect 泛型被 ComponentProps 擦除 → onChange 值为 unknown，收敛回业务类型
            onChange={(value) => onYearChange(value as number)}
            style={{ width: 120 }}
            showSearch={false}
            onSearch={() => {}}
            options={availableYears.map((y) => ({ value: y, label: `${y}年` }))}
          />
        </Space>
        <Space>
          <Button icon={<PlusOutlined />} onClick={onAdd}>
            新增
          </Button>
          <Button icon={<PlusOutlined />} onClick={onBatchAdd}>
            批量新增
          </Button>
          <Button icon={<FileExcelOutlined />} onClick={onDownloadTemplate}>
            下载模板
          </Button>
          <Upload
            accept=".xlsx,.xls"
            showUploadList={false}
            customRequest={(options) => {
              onImport({
                file: options.file as File,
                onProgress: (event) => options.onProgress?.({ percent: event.percent }),
                onSuccess: (response) => options.onSuccess?.(response),
                onError: (error) => options.onError?.(error),
              });
            }}
          >
            <Button icon={<UploadOutlined />} type="primary">
              导入Excel
            </Button>
          </Upload>
          <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        rowKey="id"
        dataSource={holidays}
        loading={loading}
        pagination={paginationProps}
        columns={columns}
        size="small"
        scroll={{ x: "max-content" }}
      />
    </Card>
  );
};

export default HolidayManagement;
