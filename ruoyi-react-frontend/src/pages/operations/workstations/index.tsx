/**
 * 工位管理页面
 */

import { useEffect, useState } from 'react';
import type { FC } from 'react';
import {
  Table,
  Button,
  Space,
  Form,
  Input,
  Select,
  Card,
  Row,
  Col,
  Statistic,
  Alert,
  Radio,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  StopOutlined,
  AppstoreOutlined,
  TableOutlined,
  ImportOutlined,
  ExportOutlined,
  DeleteOutlined,
  BgColorsOutlined,
  DesktopOutlined,
} from '@ant-design/icons';
import type { WorkstationOps } from '@/types';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';
import { useWorkstationData, useWorkstationModals, useWorkstationView } from './hooks';
import { getWorkstationColumns } from './columns';
import { WorkstationEditModal } from './modals';
import { WorkstationCardView, WorkstationFloorPlanView } from './views';
import { STATUS_OPTIONS, TYPE_OPTIONS } from './constants';
import type { WorkstationStatistics, FloorOption, UserOption, DeptTreeNode } from './types';

const { Option } = Select;

const WorkstationManagement: FC = () => {
  const [statistics, setStatistics] = useState<WorkstationStatistics>({
    total: 0,
    available: 0,
    occupied: 0,
    maintain: 0,
  });
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [floorOptions, setFloorOptions] = useState<FloorOption[]>([]);
  const [deptTreeData, setDeptTreeData] = useState<DeptTreeNode[]>([]);
  const [userOptions, setUserOptions] = useState<UserOption[]>([]);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading,
    data: workstations,
    total,
    selectedRowKeys,
    searchForm,
    editForm: workstationForm,
    editModalVisible: modalVisible,
    editingItem: editingWorkstation,
    setSelectedRowKeys,
    setEditModalVisible: setModalVisible,
    setEditingItem: setEditingWorkstation,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData: loadWorkstations,
    resetSelection,
  } = useTableManager<WorkstationOps>(
    async (params) => {
      const { workstationApi } = await import('@/lib/opsApi');
      const result = await workstationApi.list(params);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      externalPagination: {
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        setCurrent,
        setPageSize,
      },
    }
  );

  const { loadStatistics, loadFloorOptions, loadDeptOptions, loadUserOptions } = useWorkstationData(
    setStatistics,
    setFloorOptions,
    setDeptTreeData,
    setUserOptions
  );

  const { openModal, closeModal, handleSave, handleDelete, handleBatchDelete } = useWorkstationModals();

  const {
    viewMode,
    setViewMode,
    selectedFloorForPlan,
    floorPlanWorkstations,
    handleFloorChangeForPlan,
    handlePositionUpdate,
    handleFloorPlanEdit,
  } = useWorkstationView(floorOptions);

  useEffect(() => {
    // 使用 setTimeout 避免同步 setState
    setTimeout(() => {
      loadWorkstations();
      loadStatistics();
      loadFloorOptions();
      loadDeptOptions();
      loadUserOptions();
    }, 0);
  }, [paginationProps.current, paginationProps.pageSize]);

  // 操作成功后刷新
  const handleSuccess = () => {
    loadWorkstations();
    loadStatistics();
  };

  const columns = getWorkstationColumns({
    handleEdit: (record: WorkstationOps) => {
      handleEdit(record);
      openModal(record, workstationForm);
    },
    handleDelete: (id: string) => handleDelete(id, handleSuccess),
  });

  const handleDeptChange = (deptId: string) => {
    loadUserOptions(deptId);
  };

  const handleImportSuccess = () => {
    loadWorkstations();
    loadStatistics();
    setImportVisible(false);
  };

  const renderCurrentView = () => {
    if (viewMode === 'table') {
      return (
        <Table
          rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
          columns={columns}
          dataSource={workstations}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
        />
      );
    }

    if (viewMode === 'card') {
      return (
        <WorkstationCardView
          workstations={workstations}
          onEdit={(record) => {
            handleEdit(record);
            openModal(record, workstationForm);
          }}
          onDelete={(id) => handleDelete(id, handleSuccess)}
        />
      );
    }

    // floorplan view
    return (
      <WorkstationFloorPlanView
        selectedFloorForPlan={selectedFloorForPlan}
        floorOptions={floorOptions}
        floorPlanWorkstations={floorPlanWorkstations}
        allWorkstations={workstations}
        onFloorChange={handleFloorChangeForPlan}
        onPositionUpdate={handlePositionUpdate}
        onEdit={(ws) => {
          handleFloorPlanEdit(ws, () => {
            openModal(undefined, workstationForm);
          });
        }}
        onCloseFloorPlan={() => setViewMode('table')}
      />
    );
  };

  return (
    <div>
      {total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Card>
              <Statistic title="总工位数" value={statistics.total} prefix={<DesktopOutlined />} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="空闲"
                value={statistics.available}
                styles={{ content: { color: '#52c41a' } }}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="占用"
                value={statistics.occupied}
                styles={{ content: { color: '#ff4d4f' } }}
                prefix={<StopOutlined />}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="维护"
                value={statistics.maintain}
                styles={{ content: { color: '#faad14' } }}
                valueStyle={{ color: '#faad14' }}
              />
            </Card>
          </Col>
        </Row>
      )}
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            flexWrap: 'wrap',
            gap: '16px',
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="floorId" label="所属楼层">
              <Select placeholder="请选择楼层" allowClear style={{ width: 150 }}>
                {floorOptions.map((f) => (
                  <Option key={f.code} value={f.code}>
                    {f.name}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="name" label="工位名称">
              <Input placeholder="请输入工位名称" allowClear style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }}>
                {STATUS_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    loadWorkstations();
                    loadStatistics();
                  }}
                >
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table">
                <TableOutlined /> 表格
              </Radio.Button>
              <Radio.Button value="card">
                <AppstoreOutlined /> 卡片
              </Radio.Button>
              <Radio.Button value="floorplan">
                <BgColorsOutlined /> 平面图
              </Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>
              导入
            </Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>
              导出
            </Button>
            {selectedRowKeys.length > 0 && (
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={() => handleBatchDelete(selectedRowKeys, handleSuccess, resetSelection)}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => {
                handleAdd();
                openModal(undefined, workstationForm);
              }}
            >
              新增工位
            </Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && (
          <Alert
            message={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个工位，
                <Button
                  type="link"
                  size="small"
                  onClick={() => setSelectedRowKeys([])}
                  style={{ padding: 0 }}
                >
                  取消选择
                </Button>
              </span>
            }
            type="info"
            showIcon
            style={{ marginTop: 12 }}
          />
        )}
      </Card>
      <Card>{renderCurrentView()}</Card>
      <WorkstationEditModal
        open={modalVisible}
        editingWorkstation={editingWorkstation}
        floorOptions={floorOptions}
        deptTreeData={deptTreeData}
        userOptions={userOptions}
        onOk={async () => {
          await handleSave(editingWorkstation, workstationForm, handleSuccess);
        }}
        onCancel={() => {
          setModalVisible(false);
          closeModal(workstationForm);
        }}
        onDeptChange={handleDeptChange}
      />
      <ExcelImport
        entityType="workstation"
        entityName="工位"
        visible={importVisible}
        onClose={() => setImportVisible(false)}
        onImportSuccess={handleImportSuccess}
      />
      <ExcelExport
        entityType="workstation"
        entityName="工位"
        visible={exportVisible}
        onClose={() => setExportVisible(false)}
      />
    </div>
  );
};

export default WorkstationManagement;
