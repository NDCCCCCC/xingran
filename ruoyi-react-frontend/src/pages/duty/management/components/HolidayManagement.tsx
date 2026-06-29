import React from 'react';
import { Card, Select, Table, Button, Space, Tag, Upload, Modal } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  ReloadOutlined,
  FileExcelOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import { formatDate } from '@/utils/datetime';
import type { Holiday } from '@/lib/dutyApi';
import { usePagination } from '@/hooks/usePagination';
import { createSorter } from '@/utils/tableHelpers';

const { Option } = Select;

interface ImportOptions {
  file: File;
  onProgress?: (event: { percent: number }) => void;
  onSuccess?: (response?: unknown) => void;
  onError?: (error: Error) => void;
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
}) => {
  // 使用全局分页 hook
  const { paginationProps } = usePagination();
  const columns: ColumnsType<Holiday> = [
    {
      title: '日期',
      dataIndex: 'holidayDate',
      key: 'holidayDate',
      width: 120,
      sorter: createSorter<Holiday>('holidayDate', 'date'),
      render: (date: string) => formatDate(date),
    },
    {
      title: '名称',
      dataIndex: 'holidayName',
      key: 'holidayName',
      sorter: createSorter<Holiday>('holidayName', 'string'),
    },
    {
      title: '类型',
      dataIndex: 'holidayType',
      key: 'holidayType',
      width: 100,
      sorter: createSorter<Holiday>('holidayType', 'string'),
      render: (type: string) => {
        const colorMap: Record<string, string> = {
          legal: 'red',
          workday: 'orange',
          custom: 'blue',
        };
        const textMap: Record<string, string> = {
          legal: '法定节假日',
          workday: '调休工作日',
          custom: '自定义',
        };
        return <Tag color={colorMap[type]}>{textMap[type]}</Tag>;
      },
    },
    {
      title: '是否休息',
      dataIndex: 'isOffday',
      key: 'isOffday',
      width: 100,
      sorter: createSorter<Holiday>('isOffday', 'boolean'),
      render: (isOffday: boolean) => (
        <Tag color={isOffday ? 'green' : 'default'}>{isOffday ? '休息' : '工作'}</Tag>
      ),
    },
    {
      title: '年份',
      dataIndex: 'year',
      key: 'year',
      width: 80,
      sorter: createSorter<Holiday>('year', 'number'),
    },
    {
      title: '备注',
      dataIndex: 'remark',
      key: 'remark',
      sorter: createSorter<Holiday>('remark', 'string'),
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      fixed: 'right',
      render: (_: unknown, record: Holiday) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => onEdit(record)}>编辑</Button>
          <Button
            type="link"
            size="small"
            danger
            onClick={() => {
              Modal.confirm({
                title: '确定删除?',
                okText: '确定',
                cancelText: '取消',
                okButtonProps: { danger: true },
                onOk: () => onDelete(record.id),
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
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space>
          <span>年份：</span>
          <Select
            value={holidayYear}
            onChange={onYearChange}
            style={{ width: 120 }}
           onSearch={() => {}}>
            {availableYears.map((y) => (
              <Option key={y} value={y}>{y}年</Option>
            ))}
          </Select>
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
      />
    </Card>
  );
};

export default HolidayManagement;
