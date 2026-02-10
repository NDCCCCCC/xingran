/**
 * 工位管理页面
 */

import { useState, useCallback, useEffect } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Form, Input, Select, Card, Row, Col,
  Statistic, Alert, Radio, Layout,
} from 'antd';
import {
  PlusOutlined, SearchOutlined, ReloadOutlined, CheckCircleOutlined,
  StopOutlined, AppstoreOutlined, TableOutlined, ImportOutlined,
  ExportOutlined, DeleteOutlined, BgColorsOutlined, DesktopOutlined,
} from '@ant-design/icons';
import type { WorkstationOps } from '@/types';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { useSidebarDeptFilter } from '@/hooks/useSidebarDeptFilter';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';
import { DeptSidebar } from '@/components/operations/DeptSidebar';
import { StatisticsCards } from '@/components/operations/StatisticsCards';
import { useWorkstationData, useWorkstationModals, useWorkstationView } from './hooks';
import { getWorkstationColumns } from './columns';
import { WorkstationEditModal } from './modals';
import { WorkstationCardView, WorkstationFloorPlanView } from './views';
import { STATUS_OPTIONS, TYPE_OPTIONS } from './constants';
import type { WorkstationStatistics, FloorOption, UserOption, DeptTreeNode } from './types';
import { workstationApi } from '@/lib/opsApi';

const { Option } = Select;
const { Content } = Layout;

interface Statistics {
  total: number;
  available: number;
  occupied: number;
  maintain: number;
}

const WorkstationManagement: FC = () => {
  const [statistics, setStatistics] = useState<Statistics>({
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

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();
  const { selectedDeptId, handleDeptSelect } = useSidebarDeptFilter({
    searchForm: undefined,
    clearFieldNames: ['floorId'],
  });

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
      const searchParams = selectedDeptId
        ? { ...params, orgId: selectedDeptId }
        : params;
      const result = await workstationApi.list(searchParams) as { data?: { list: WorkstationOps[]; total: number } };
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
    }
  );

  const { loadStatistics: loadStatisticsFromHook, loadFloorOptions, loadDeptOptions, loadUserOptions } = useWorkstationData(
    setStatistics,
    setFloorOptions,
    setDeptTreeData,
    setUserOptions
  );

  const { openModal, closeModal, handleSave, handleDelete, handleBatchDelete } = useWorkstationModals(
    loadUserOptions
  );

  const {
    viewMode,
    setViewMode,
    selectedFloorForPlan,
    floorPlanWorkstations,
    handleFloorChangeForPlan,
    handlePositionUpdate,
    handleFloorPlanEdit,
  } = useWorkstationView(floorOptions);

  const refreshData = useCallback(() => {
    loadWorkstations();
    loadStatisticsFromHook();
  }, [loadWorkstations, loadStatisticsFromHook]);

  // 初始化加载
  useEffect(() => {
    loadStatisticsFromHook();
    loadFloorOptions();
    loadDeptOptions();
    loadUserOptions();
  }, [loadStatisticsFromHook, loadFloorOptions, loadDeptOptions, loadUserOptions]);

  useEffect(() => {
    loadWorkstations();
    loadFloorOptions(selectedDeptId);
  }, [selectedDeptId, loadWorkstations, loadFloorOptions]);

  const handleSuccess = useCallback(() => {
    refreshData();
  }, [refreshData]);

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

  const handleImportSuccess = useCallback(() => {
    refreshData();
    setImportVisible(false);
  }, [refreshData]);

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
          // 从完整工位列表中找到对应的数据
          const fullWorkstation = workstations.find(w => w.id === ws.id);
          handleFloorPlanEdit(ws, (record) => {
            // 优先使用完整工位数据，如果没有则使用平面图节点数据
            openModal(fullWorkstation || record, workstationForm);
          });
        }}
        onCloseFloorPlan={() => setViewMode('table')}
      />
    );
  };

  return (
    <Layout style={{ background: '#f0f2f5', minHeight: 'calc(100vh - 64px)' }}>
      <DeptSidebar selectedDeptId={selectedDeptId} onSelect={handleDeptSelect} />
      <Content style={{ padding: '16px', background: '#f0f2f5' }}>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: '总工位数', value: statistics.total, prefix: <DesktopOutlined /> },
            { title: '空闲', value: statistics.available, styles: { content: { color: '#52c41a' } }, prefix: <CheckCircleOutlined /> },
            { title: '占用', value: statistics.occupied, styles: { content: { color: '#ff4d4f' } }, prefix: <StopOutlined /> },
            { title: '维护', value: statistics.maintain, styles: { content: { color: '#faad14' } }, valueStyle: { color: '#faad14' } },
          ]}
        />
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
                    <Option key={f.id} value={f.id}>
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
                  <Button icon={<ReloadOutlined />} onClick={refreshData}>
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
          onOk={async (values) => {
            try {
              // 直接使用传递来的表单值
              if (editingWorkstation) {
                await workstationApi.update(editingWorkstation.id, values);
                handleSuccess('更新');
              } else {
                await workstationApi.create(values);
                handleSuccess('创建');
              }
              // 成功后关闭模态框并刷新
              closeModal(workstationForm);
              setModalVisible(false);
              handleSuccess();
            } catch (error) {
              // 失败时不关闭模态框
              throw error;
            }
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
      </Content>
    </Layout>
  );
};

export default WorkstationManagement;
