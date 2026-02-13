/**
 * 工位管理页面
 */

import { useState, useCallback, useEffect } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Form, Input, Select, Card, Row, Col,
  Statistic, Alert, Radio, Layout, Cascader,
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
import { handleSuccess as showSuccessMessage } from '@/utils/errorHandler';
import { useWorkstationData, useWorkstationModals, useWorkstationView } from './hooks';
import { getWorkstationColumns } from './columns';
import { WorkstationEditModal } from './modals';
import { WorkstationCardView, WorkstationFloorPlanView } from './views';
import { STATUS_OPTIONS, TYPE_OPTIONS } from './constants';
import type { WorkstationStatistics, FloorOption, UserOption, DeptTreeNode } from './types';

// Cascader 选项类型
type CascaderOption = {
  value: string;
  label: string;
  children?: CascaderOption[];
  isLeaf?: boolean;
};
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

  // Cascader 懒加载状态
  const [cascaderOptions, setCascaderOptions] = useState<CascaderOption[]>([]);
  const [loadingCascader, setLoadingCascader] = useState(false);

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

  // 部门变化时重新加载数据（使用 useCallback 稳定依赖）
  useEffect(() => {
    loadWorkstations();
    if (selectedDeptId) {
      loadFloorOptions(selectedDeptId);
    }
  }, [selectedDeptId]);

  const handleSuccess = useCallback(() => {
    refreshData();
  }, [refreshData]);

  const columns = getWorkstationColumns({
    handleEdit: (record: WorkstationOps) => {
      handleOpenModal(record);
    },
    handleDelete: (id: string) => handleDelete(id, handleSuccess),
  });

  const handleDeptChange = (deptId: string) => {
    loadUserOptions(deptId);
  };

  // ==================== Cascader 级联选择相关函数 ====================

  // 加载楼宇列表（级联第一级）
  const loadBuildingsForCascader = useCallback(async (orgId?: string) => {
    try {
      const params: Record<string, unknown> = { current: 1, pageSize: 1000 };
      if (orgId) {
        params.orgId = orgId;
      }
      const { buildingApi } = await import('@/lib/opsApi');
      const buildingResult = await buildingApi.list(params);
      const buildings = buildingResult.data?.list || [];
      return buildings.map((b: { id: string; name: string }) => ({
        value: b.id,
        label: b.name,
        isLeaf: false,
      }));
    } catch (error) {
      console.error('加载楼宇列表失败:', error);
      return [];
    }
  }, []);

  // 加载楼层列表（级联第二级）
  const loadFloorsForCascader = useCallback(async (buildingId: string) => {
    try {
      const { floorApi } = await import('@/lib/opsApi');
      const floorResult = await floorApi.list({ buildingId, current: 1, pageSize: 1000 });
      const floors = floorResult.data?.list || [];
      return floors.map((f) => ({
        value: f.id,
        label: f.name || f.floorNo || '',
        isLeaf: true,
      }));
    } catch (error) {
      console.error('加载楼层列表失败:', error);
      return [];
    }
  }, []);

  // Cascader 懒加载处理函数
  const handleCascaderLoadData = useCallback(async (selectedOptions: CascaderOption[]) => {
    const targetOption = selectedOptions[selectedOptions.length - 1];

    setLoadingCascader(true);

    try {
      if (selectedOptions.length === 1) {
        // 加载楼层（第二级）
        const buildingId = targetOption.value;
        const floors = await loadFloorsForCascader(buildingId);
        targetOption.children = floors;
      }

      // 触发重新渲染
      setCascaderOptions([...cascaderOptions]);
    } finally {
      setLoadingCascader(false);
    }
  }, [cascaderOptions, loadFloorsForCascader]);

  // 初始化 Cascader 选项（加载楼宇列表）
  const initCascaderOptions = useCallback(async () => {
    setLoadingCascader(true);
    try {
      const buildings = await loadBuildingsForCascader(selectedDeptId || undefined);
      setCascaderOptions(buildings);
    } finally {
      setLoadingCascader(false);
    }
  }, [loadBuildingsForCascader, selectedDeptId]);

  // 预加载 Cascader 路径数据（用于编辑回显）
  const preloadCascaderPath = useCallback(async (buildingId: string, floorId: string) => {
    try {
      // 找到对应的楼宇节点
      const buildingIndex = cascaderOptions.findIndex(b => b.value === buildingId);
      if (buildingIndex === -1) return;

      // 加载该楼宇下的楼层列表
      const floors = await loadFloorsForCascader(buildingId);

      // 更新楼宇节点的 children（楼层列表）
      const updatedOptions = [...cascaderOptions];
      (updatedOptions[buildingIndex] as typeof cascaderOptions[0] & { children?: typeof cascaderOptions }).children = floors;

      // 触发重新渲染
      setCascaderOptions(updatedOptions);
    } catch (error) {
      console.error('预加载 Cascader 路径失败:', error);
    }
  }, [cascaderOptions, loadFloorsForCascader]);

  // 打开编辑模态框时初始化 Cascader
  const handleOpenModal = useCallback(async (record?: WorkstationOps) => {
    // 初始化 Cascader 选项（加载楼宇列表）
    await initCascaderOptions();

    if (record) {
      handleEdit(record);

      // 编辑模式：需要预加载级联路径数据
      if (record.floorId) {
        try {
          // 查询楼层信息以获取所属楼宇
          const { floorApi } = await import('@/lib/opsApi');
          const floorResult = await floorApi.get(record.floorId);
          const floor = floorResult.data;

          const buildingId = floor?.buildingId || floor?.buildingCode;
          const floorId = record.floorId;

          if (buildingId && floorId) {
            // 预加载 Cascader 路径数据，确保回显时能找到匹配的节点
            await preloadCascaderPath(buildingId, floorId);

            // 设置表单值为二级路径: [buildingId, floorId]
            workstationForm.setFieldsValue({
              ...record,
              floorId: [buildingId, floorId],
            });
          } else {
            workstationForm.setFieldsValue(record);
          }
        } catch (error) {
          console.error('获取楼层信息失败:', error);
          workstationForm.setFieldsValue(record);
        }
      } else {
        workstationForm.setFieldsValue(record);
      }
    } else {
      // 新增模式
      handleAdd();
      workstationForm.setFieldsValue({ status: 0, type: 0 });
    }

    setModalVisible(true);
  }, [initCascaderOptions, preloadCascaderPath, handleEdit, handleAdd, workstationForm, setModalVisible]);

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
            handleOpenModal(record);
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
          handleOpenModal(workstations.find(w => w.id === ws.id));
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
                  icon={<DeleteOutlined />} style={{ color: '#ff4d4f' }}
                  onClick={() => handleBatchDelete(selectedRowKeys, handleSuccess, resetSelection)}
                >
                  批量删除 ({selectedRowKeys.length})
                </Button>
              )}
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  handleOpenModal(undefined);
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
          deptTreeData={deptTreeData}
          userOptions={userOptions}
          cascaderOptions={cascaderOptions}
          loadingCascader={loadingCascader}
          handleCascaderLoadData={handleCascaderLoadData}
          onOk={async (values) => {
            try {
              // 处理级联选择返回的 floorId（可能是数组）
              const submitValues = { ...values };
              if (Array.isArray(submitValues.floorId)) {
                submitValues.floorId = submitValues.floorId[submitValues.floorId.length - 1];
              }

              if (editingWorkstation) {
                await workstationApi.update(editingWorkstation.id, submitValues);
                showSuccessMessage('更新');
              } else {
                await workstationApi.create(submitValues);
                showSuccessMessage('创建');
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

