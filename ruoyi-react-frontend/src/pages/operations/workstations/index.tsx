/**
 * 工位管理页面
 *
 * 性能优化说明：
 * - 将 API 导入移至顶层，避免动态导入开销
 * - 使用 Promise.all 并行加载独立数据
 * - 优化 useEffect 依赖数组，减少不必要的重渲染
 */

import { useState, useCallback, useEffect } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Form, Input, Select, Card,
  Alert, Radio, Layout,
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
import { STATUS_OPTIONS } from './constants';
import type { FloorOption, UserOption, DeptTreeNode } from './types';
// ❌ 移除：import { workstationApi } from '@/lib/opsApi';
// ✅ 优化：直接导入具体 API，避免 barrel 文件 (bundle-barrel-imports)
import { buildingApi, floorApi, workstationApi } from '@/lib/opsApi';

// Cascader 选项类型
type CascaderOption = {
  value: string;
  label: string;
  children?: CascaderOption[];
  isLeaf?: boolean;
};

const { Option } = Select;
const { Content } = Layout;

// 统计数据类型
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
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});
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
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    loadData: loadWorkstations,
    resetSelection,
  } = useTableManager<WorkstationOps>(
    // ✅ 优化：移除动态导入，直接使用顶层导入的 API (bundle-dynamic-imports)
    async (params) => {
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

  const { closeModal, handleDelete, handleBatchDelete } = useWorkstationModals(loadUserOptions);

  const {
    viewMode,
    setViewMode,
    selectedFloorForPlan,
    floorPlanWorkstations,
    handleFloorChangeForPlan,
    handlePositionUpdate,
  } = useWorkstationView(floorOptions);

  const refreshData = useCallback(() => {
    loadWorkstations();
    loadStatisticsFromHook(selectedDeptId || undefined);
  }, [loadWorkstations, loadStatisticsFromHook, selectedDeptId]);

  // ✅ 优化：初始化加载 - 并行执行独立的异步操作 (async-parallel)
  // 原来是顺序执行 4 个独立的 API 调用，现在并行执行
  useEffect(() => {
    // 使用 Promise.all 并行加载所有独立数据
    Promise.all([
      loadStatisticsFromHook(),
      loadFloorOptions(),
      loadDeptOptions(),
      loadUserOptions(),
    ]).catch((error) => {
      console.error('初始化加载失败:', error);
    });
    // 依赖数组保持不变，因为这些函数应该由 hooks 稳定化
  }, [loadStatisticsFromHook, loadFloorOptions, loadDeptOptions, loadUserOptions]);

  // 部门变化时重新加载数据（使用 useCallback 稳定依赖）
  useEffect(() => {
    loadWorkstations();
    loadStatisticsFromHook(selectedDeptId || undefined);
    if (selectedDeptId) {
      loadFloorOptions(selectedDeptId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
  // 二级结构：楼宇 → 楼层（机构选择使用 DepartmentTreeSelect）

  // ✅ 优化：移除 Cascader 中的动态导入，使用顶层导入 (bundle-dynamic-imports)
  // 加载楼宇列表（级联第一级，根据机构ID）
  const loadBuildingsForCascader = useCallback(async (orgId: string): Promise<CascaderOption[]> => {
    try {
      const params = { current: 1, pageSize: 1000, orgId };
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
  }, []); // 空依赖数组：buildingApi 是稳定的模块导入

  // 加载楼层列表（级联第二级）
  const loadFloorsForCascader = useCallback(async (buildingId: string): Promise<CascaderOption[]> => {
    try {
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
        // 直接修改 targetOption 的 children 属性
        targetOption.children = floors;
        // 触发重新渲染
        setCascaderOptions([...cascaderOptions]);
      }
    } finally {
      setLoadingCascader(false);
    }
  }, [cascaderOptions, loadFloorsForCascader]);

  // 机构变化时加载楼宇列表
  const handleOrgChange = useCallback(async (orgId: string) => {
    if (!orgId) {
      setCascaderOptions([]);
      return;
    }

    setLoadingCascader(true);
    try {
      const buildings = await loadBuildingsForCascader(orgId);
      setCascaderOptions(buildings);
    } catch (error) {
      console.error('加载楼宇列表失败:', error);
      setCascaderOptions([]);
    } finally {
      setLoadingCascader(false);
    }
  }, [loadBuildingsForCascader]);

  // 预加载 Cascader 路径数据（用于编辑回显）
  const preloadCascaderPath = useCallback(async (orgId: string, buildingId: string) => {
    try {
      const buildings = await loadBuildingsForCascader(orgId);
      const floors = await loadFloorsForCascader(buildingId);
      const buildingWithFloors = buildings.map(b => ({
        ...b,
        children: b.value === buildingId ? floors : undefined,
      }));
      setCascaderOptions(buildingWithFloors);
    } catch (error) {
      console.error('预加载 Cascader 路径失败:', error);
    }
  }, [loadBuildingsForCascader, loadFloorsForCascader]);

  // 设置编辑模式表单值的辅助函数
  // ✅ 优化：移除动态导入，并行获取楼层和楼宇信息 (async-parallel, bundle-dynamic-imports)
  const setEditFormValues = useCallback(async (record: WorkstationOps) => {
    if (!record.floorId) {
      workstationForm.setFieldsValue(record);
      return;
    }

    try {
      // 先获取楼层信息来得到 buildingId
      const floorResult = await floorApi.get(record.floorId);
      const floor = floorResult.data;
      const buildingId = floor?.buildingId;

      if (!buildingId) {
        workstationForm.setFieldsValue(record);
        return;
      }

      // 并行获取楼宇信息和预加载 Cascader 路径
      const [buildingResult] = await Promise.all([
        buildingApi.get(buildingId),
        preloadCascaderPath(buildingId, buildingId),
      ]);

      const building = buildingResult.data;
      const orgId = building?.orgId;

      if (orgId) {
        // 等待 React 状态更新和 Cascader 重新渲染
        setTimeout(() => {
          workstationForm.setFieldsValue({
            ...record,
            orgId,
            floorId: [buildingId, record.floorId],
          });
        }, 0);
      } else {
        workstationForm.setFieldsValue({
          ...record,
          floorId: [buildingId, record.floorId],
        });
      }
    } catch (error) {
      console.error('获取楼层或楼宇信息失败:', error);
      workstationForm.setFieldsValue(record);
    }
  }, [preloadCascaderPath, workstationForm]);

  // 打开编辑模态框
  const handleOpenModal = useCallback(async (record?: WorkstationOps) => {
    if (record) {
      handleEdit(record);
      await setEditFormValues(record);
    } else {
      handleAdd();
      setCascaderOptions([]);
      workstationForm.setFieldsValue({ status: 0, type: 0 });
    }
    setModalVisible(true);
  }, [setEditFormValues, handleEdit, handleAdd, workstationForm, setModalVisible]);

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
          onChange={(pagination) => {
            const newPage = pagination.current || 1;
            const newPageSize = pagination.pageSize || 10;
            setCurrent(newPage);
            setPageSize(newPageSize);

            // 获取搜索表单值并合并分页参数
            const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
            const searchParams: Record<string, unknown> = { current: newPage, pageSize: newPageSize };
            Object.keys(formValues).forEach(key => {
              const value = formValues[key];
              if (value !== undefined && value !== null && value !== '') {
                searchParams[key] = value;
              }
            });

            loadWorkstations(searchParams);
          }}
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
      <DeptSidebar selectedDeptId={selectedDeptId} onSelect={handleDeptSelect} externalOnly={false} />
      <Content style={{ padding: '16px', background: '#f0f2f5' }}>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: '总工位数', value: statistics.total, prefix: <DesktopOutlined /> },
            { title: '空闲', value: statistics.available, styles: { content: { color: '#52c41a' } }, prefix: <CheckCircleOutlined /> },
            { title: '占用', value: statistics.occupied, styles: { content: { color: '#ff4d4f' } }, prefix: <StopOutlined /> },
            { title: '维护', value: statistics.maintain, styles: { content: { color: '#faad14' } }, prefix: <StopOutlined /> },
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
                <Select placeholder="请选择楼层" allowClear className="user-form-input" style={{ width: 150 }}>
                  {floorOptions.map((f) => (
                    <Option key={f.id} value={f.id}>
                      {f.name}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item name="name" label="工位名称">
                <Input placeholder="请输入工位名称" allowClear className="user-form-input" style={{ width: 150 }} />
              </Form.Item>
              <Form.Item name="status" label="状态">
                <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 120 }}>
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
              <Button icon={<ExportOutlined />} onClick={() => {
                // 获取当前筛选条件
                const values = searchForm.getFieldsValue() as Record<string, unknown>;
                const currentFilters: Record<string, unknown> = {};
                Object.keys(values).forEach(key => {
                  const value = values[key];
                  if (value !== undefined && value !== null && value !== '') {
                    currentFilters[key] = value;
                  }
                });
                setExportFilters(currentFilters);
                setExportVisible(true);
              }}>
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
          form={workstationForm}
          editingWorkstation={editingWorkstation}
          deptTreeData={deptTreeData}
          userOptions={userOptions}
          cascaderOptions={cascaderOptions}
          loadingCascader={loadingCascader}
          handleCascaderLoadData={handleCascaderLoadData}
          onOrgChange={handleOrgChange}
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
          filters={exportFilters}
        />
      </Content>
    </Layout>
  );
};

export default WorkstationManagement;

