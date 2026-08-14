import type { FC } from "react";
import { Button, Table, Card, Space, Select, Alert, Upload } from "antd";
import {
  PlusOutlined,
  ReloadOutlined,
  UploadOutlined,
  CalendarOutlined,
  FileExcelOutlined,
} from "@ant-design/icons";
import { useHolidayData, useHolidayModals } from "./hooks";
import { getHolidayColumns } from "./columns";
import { HolidayEditModal, HolidayBatchModal } from "./modals";
import { handleExcelImport, downloadTemplate } from "./utils";

const { Option } = Select;

const DutyHolidayPage: FC = () => {
  // 数据管理
  const { loading, dataSource, year, availableYears, fetchList } = useHolidayData();

  // 模态框和操作管理
  const {
    modalState,
    batchState,
    setBatchHolidays: _setBatchHolidays,
    setModalVisible,
    setBatchModalVisible,
    handleAdd,
    handleEdit,
    handleDelete,
    handleModalOk,
    handleBatchAdd,
    addBatchRow,
    removeBatchRow,
    updateBatchRow,
    handleBatchSubmit,
  } = useHolidayModals({
    year,
    availableYears,
    fetchList,
  });

  // 表格列
  const columns = getHolidayColumns({
    handleEdit,
    handleDelete,
  });

  return (
    <div className="p-6">
      <Card
        title={
          <Space>
            <CalendarOutlined />
            <span>节假日管理 - {year}年</span>
          </Space>
        }
        extra={
          <Space>
            <Select
              value={year}
              onChange={(y) => fetchList(y)}
              style={{ width: 120 }}
              suffixIcon={null}
              onSearch={() => {}}
            >
              {availableYears.map((y) => (
                <Option key={y} value={y}>
                  {y}年
                </Option>
              ))}
            </Select>
            <Button icon={<ReloadOutlined />} onClick={() => fetchList()}>
              刷新
            </Button>
          </Space>
        }
      >
        <div className="mb-4 flex gap-2">
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增节假日
          </Button>
          <Button icon={<UploadOutlined />} onClick={handleBatchAdd}>
            批量新增
          </Button>
          <Button icon={<FileExcelOutlined />} onClick={downloadTemplate}>
            下载导入模板
          </Button>
          <Upload
            accept=".xlsx,.xls"
            showUploadList={false}
            customRequest={(options) =>
              handleExcelImport(
                {
                  file: options.file as File,
                  onSuccess: options.onSuccess as (data: unknown) => void,
                  onError: options.onError as (error: Error) => void,
                },
                fetchList
              )
            }
          >
            <Button icon={<UploadOutlined />} type="primary">
              导入Excel
            </Button>
          </Upload>
        </div>

        <Alert
          title="提示"
          description="支持通过Excel批量导入节假日数据。请先下载模板，按模板格式填写后导入。日期格式支持 YYYY-MM-DD 或 Excel日期格式。"
          type="info"
          showIcon
          closable
          className="mb-4"
        />

        <Table
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          scroll={{ x: 1000 }}
          pagination={false}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <HolidayEditModal
        open={modalState.modalVisible}
        editingRecord={modalState.editingRecord}
        year={year}
        availableYears={availableYears}
        onOk={handleModalOk}
        onCancel={() => {
          // Close modal through hook - will implement properly
          setModalVisible(false);
        }}
      />

      {/* 批量新增弹窗 */}
      <HolidayBatchModal
        open={modalState.batchModalVisible}
        batchHolidays={batchState.batchHolidays}
        onOk={handleBatchSubmit}
        onCancel={() => {
          setBatchModalVisible(false);
        }}
        onAddRow={addBatchRow}
        onRemoveRow={removeBatchRow}
        onUpdateRow={updateBatchRow}
      />
    </div>
  );
};

export default DutyHolidayPage;
