/**
 * 楼层管理页面
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import type { FC } from 'react';
import { Table, Button, Card, Alert } from 'antd';
import type { Floor, Department, Building } from '@/types';
import type { BaseResponse } from '@/types';
import { floorApi, buildingApi } from '@/lib/opsApi';
import { post } from '@/lib/api';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { handleApiError, handleSuccess, isFormValidationError } from '@/utils/errorHandler';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';
import type { TreeNode } from '@/components/shared/DepartmentTreeSelect';

// 组件
import {
  FloorCardView,
  FloorSearchForm,
  FloorModal,
  FloorPlanEditorView,
  FloorStatisticsCards,
  createFloorTableColumns,
} from './components';

// Hooks
import { useFloorPlanEditor } from './useFloorPlanEditor';
import { useFloorStatistics } from './useFloorStatistics';
import { useBuildingOptions } from './useBuildingOptions';

// 常量和类型
import type { ViewMode, PageMode } from './constants';
import type { FloorOption } from './components';
import { DEFAULT_FORM_VALUES } from './constants';

const FloorManagement: FC = () => {
  // 页面模式状态
  const [pageMode, setPageMode] = useState<PageMode>('list');
  const [currentFloor, setCurrentFloor] = useState<Floor | null>(null);

  // 视图模式状态
  const [viewMode, setViewMode] = useState<ViewMode>('table');

  // 弹窗状态
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);

  // 分页
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 楼宇选项
  const { buildingOptions, loadBuildingOptions } = useBuildingOptions();

  // 部门树数据状态（仅外部机构）
  const [departmentTreeData, setDepartmentTreeData] = useState<TreeNode[]>([]);

  // 楼宇选项状态（用于编辑器中的切换）
  const [buildingOptionsByDept, setBuildingOptionsByDept] = useState<Building[]>([]);

  // 当前选中的部门和楼宇ID（用于编辑器）
  const [selectedDeptId, setSelectedDeptId] = useState<string>('');
  const [selectedBuildingId, setSelectedBuildingId] = useState<string>('');

  // 楼层选项状态（用于编辑器中的切换）
  const [floorOptions, setFloorOptions] = useState<FloorOption[]>([]);

  // 统计数据
  const { statistics, loadStatistics } = useFloorStatistics();

  // 平面图编辑器
  const {
    floorPlanData,
    floorPlanLoading,
    isEditMode,
    setEditMode,
    loadFloorPlanData,
    saveFloorPlan,
    resetFloorPlan,
  } = useFloorPlanEditor(currentFloor);

  // 表格管理
  const {
    loading,
    data: floors,
    total,
    selectedRowKeys,
    searchForm,
    editForm: floorForm,
    editModalVisible: modalVisible,
    editingItem: editingFloor,
    setSelectedRowKeys,
    setEditModalVisible: setModalVisible,
    setEditingItem: setEditingFloor,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData: loadFloors,
    resetSelection,
  } = useTableManager<Floor>(
    async (params) => {
      const result = await floorApi.list(params);
      return {
        list: result.data?.list || [],
        total: result.data?.total || 0,
      };
    },
    {
      externalPagination: {
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        setCurrent,
        setPageSize,
        setTotal,
      },
    }
  );

  // 使用 ref 跟踪初始化状态，避免循环请求
  const isInitialized = useRef(false);
  const prevPageRef = useRef({ current: 1, pageSize: 10 });

  // 初始化加载
  useEffect(() => {
    const { current, pageSize } = paginationProps;
    const prev = prevPageRef.current;

    // 检查分页是否真的发生了变化
    const pageChanged = prev.current !== current || prev.pageSize !== pageSize;

    if (!isInitialized.current || pageChanged) {
      isInitialized.current = true;
      prevPageRef.current = { current, pageSize };

      setTimeout(() => {
        loadFloors();
        loadStatistics();
        loadBuildingOptions();
        loadExternalDepartmentTree();
      }, 0);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paginationProps.current, paginationProps.pageSize]);

  /**
   * 保存楼层
   */
  const handleSave = useCallback(async () => {
    try {
      const values = await floorForm.validateFields();

      if (editingFloor) {
        await floorApi.update(editingFloor.id, values);
        handleSuccess('更新');
      } else {
        await floorApi.create(values);
        handleSuccess('创建');
      }

      handleModalClose();
      loadFloors();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, '操作');
    }
  }, [floorForm, editingFloor, handleModalClose, loadFloors, loadStatistics]);

  /**
   * 删除楼层
   */
  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await floorApi.delete(id);
        handleSuccess('删除');
        loadFloors();
        loadStatistics();
      } catch (error) {
        handleApiError(error, '删除');
      }
    },
    [loadFloors, loadStatistics]
  );

  /**
   * 批量删除
   */
  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) return;

    try {
      await floorApi.batch('delete', { ids: selectedRowKeys });
      handleSuccess('批量删除');
      resetSelection();
      loadFloors();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '批量删除');
    }
  }, [selectedRowKeys, resetSelection, loadFloors, loadStatistics]);

  /**
   * 打开编辑弹窗
   */
  const openModal = useCallback(
    (record?: Floor) => {
      if (record) {
        handleEdit(record);
        floorForm.setFieldsValue(record);
      } else {
        handleAdd();
        floorForm.setFieldsValue(DEFAULT_FORM_VALUES);
      }
    },
    [handleEdit, handleAdd, floorForm]
  );

  /**
   * 导入成功回调
   */
  const handleImportSuccess = useCallback(() => {
    loadFloors();
    loadStatistics();
    setImportVisible(false);
  }, [loadFloors, loadStatistics]);

  /**
   * 刷新数据
   */
  const handleRefresh = useCallback(() => {
    loadFloors();
    loadStatistics();
  }, [loadFloors, loadStatistics]);

  /**
   * 过滤出外部机构部门（isExternalOrg === 1）
   */
  const filterExternalOrgDepts = useCallback((data: Department[]): Department[] => {
    if (!data || !Array.isArray(data)) {
      return [];
    }

    const filterNodes = (nodes: Department[]): Department[] => {
      return nodes.reduce<Department[]>((acc, node) => {
        const filteredChildren = node.children && node.children.length > 0
          ? filterNodes(node.children)
          : [];

        const isExternal = node.isExternalOrg === 1;
        const hasExternalDescendants = filteredChildren.length > 0;

        if (isExternal || hasExternalDescendants) {
          acc.push({
            ...node,
            children: filteredChildren.length > 0 ? filteredChildren : undefined,
          });
        }

        return acc;
      }, []);
    };

    return filterNodes(data);
  }, []);

  /**
   * 将部门数据转换为 TreeSelect 需要的格式
   */
  const convertToTreeSelectData = useCallback((data: Department[]): TreeNode[] => {
    if (!data || !Array.isArray(data)) {
      return [];
    }

    return data.map(item => ({
      title: item.deptName,
      value: item.id,
      key: item.id,
      children: item.children && item.children.length > 0
        ? convertToTreeSelectData(item.children)
        : undefined,
    }));
  }, []);

  /**
   * 加载外部机构部门树数据
   */
  const loadExternalDepartmentTree = useCallback(async () => {
    try {
      const result = await post<BaseResponse<Department[]>>('/system/departments/tree');
      const deptData = filterExternalOrgDepts(result.data ?? []);
      const treeData = convertToTreeSelectData(deptData);
      setDepartmentTreeData(treeData);
    } catch (error) {
      console.error('加载外部机构部门树失败:', error);
    }
  }, [filterExternalOrgDepts, convertToTreeSelectData]);

  /**
   * 加载指定部门的楼宇选项
   * 返回楼宇列表以便后续处理
   */
  const loadBuildingOptionsByDept = useCallback(async (deptId: string): Promise<Building[]> => {
    try {
      const result = await buildingApi.list({ orgId: deptId, current: 1, pageSize: 1000 });
      const buildings = result.data?.list || [];

      setBuildingOptionsByDept(buildings);
      return buildings;
    } catch (error) {
      console.error('加载楼宇选项失败:', error);
      return [];
    }
  }, []);

  /**
   * 加载指定楼宇的楼层选项
   * 返回楼层列表以便后续处理
   */
  const loadFloorOptionsByBuilding = useCallback(async (buildingId: string): Promise<FloorOption[]> => {
    try {
      const result = await floorApi.list({ buildingId, current: 1, pageSize: 1000 });
      const floors = result.data?.list || [];

      const options = floors.map((f: Floor) => ({
        id: f.id,
        name: f.name,
        floorNo: f.floorNo,
      }));

      setFloorOptions(options);
      return options;
    } catch (error) {
      console.error('加载楼层选项失败:', error);
      return [];
    }
  }, []);

  /**
   * 楼层切换处理
   */
  const handleFloorChange = useCallback(
    async (floorId: string) => {
      if (!floorId) return;

      try {
        const result = await floorApi.get(floorId);
        const newFloor = result.data;
        if (newFloor) {
          setCurrentFloor(newFloor);
          setEditMode(false);
          await loadFloorPlanData(newFloor.id);
        }
      } catch (error) {
        console.error('加载楼层失败:', error);
      }
    },
    [loadFloorPlanData, setEditMode]
  );

  /**
   * 楼宇切换处理
   * 加载新楼宇的楼层列表，并自动选择第一个楼层
   */
  const handleBuildingChange = useCallback(
    async (buildingId: string) => {
      if (!buildingId) return;

      setSelectedBuildingId(buildingId);

      // 加载新楼宇的楼层列表
      const options = await loadFloorOptionsByBuilding(buildingId);

      // 自动选择第一个楼层
      if (options.length > 0) {
        await handleFloorChange(options[0].id);
      }
    },
    [loadFloorOptionsByBuilding, handleFloorChange]
  );

  /**
   * 部门切换处理
   * 加载新部门的楼宇列表，并自动选择第一个楼宇和楼层
   */
  const handleDepartmentChange = useCallback(
    async (deptId: string) => {
      if (!deptId) return;

      setSelectedDeptId(deptId);
      setSelectedBuildingId('');

      // 加载新部门的楼宇列表
      const buildings = await loadBuildingOptionsByDept(deptId);

      // 自动选择第一个楼宇
      if (buildings.length > 0) {
        await handleBuildingChange(buildings[0].id);
      }
    },
    [loadBuildingOptionsByDept, handleBuildingChange]
  );

  /**
   * 打开平面图编辑器
   */
  const handleEditFloorPlan = useCallback(
    async (floor: Floor) => {
      setCurrentFloor(floor);
      setPageMode('editor');
      setEditMode(false);

      // 获取楼宇信息以获取部门ID
      try {
        const buildingResult = await buildingApi.get(floor.buildingId);
        const building = buildingResult.data;

        if (building) {
          setSelectedBuildingId(building.id);
          setSelectedDeptId(building.orgId);

          // 加载该部门的楼宇列表
          await loadBuildingOptionsByDept(building.orgId);

          // 加载该楼宇的楼层列表
          await loadFloorOptionsByBuilding(building.id);
        }
      } catch (error) {
        console.error('加载楼宇信息失败:', error);
      }

      await loadFloorPlanData(floor.id);
    },
    [loadFloorPlanData, setEditMode, loadFloorOptionsByBuilding, loadBuildingOptionsByDept]
  );

  /**
   * 保存平面图
   */
  const handleSaveFloorPlan = useCallback(
    async (data: Record<string, unknown>) => {
      if (!currentFloor) return;
      await saveFloorPlan(data, currentFloor);
    },
    [currentFloor, saveFloorPlan]
  );

  /**
   * 返回列表视图
   */
  const handleBackToList = useCallback(() => {
    setPageMode('list');
    setCurrentFloor(null);
    resetFloorPlan();
  }, [resetFloorPlan]);

  // 表格列配置
  const columns = createFloorTableColumns({
    onEdit: openModal,
    onEditFloorPlan: handleEditFloorPlan,
    onDelete: handleDelete,
  });

  // 渲染平面图编辑器视图
  const renderFloorPlanEditor = () => (
    <FloorPlanEditorView
      currentFloor={currentFloor}
      floorPlanData={floorPlanData}
      loading={floorPlanLoading}
      isEditMode={isEditMode}
      departmentTreeData={departmentTreeData}
      buildingOptions={buildingOptionsByDept}
      floorOptions={floorOptions}
      selectedDeptId={selectedDeptId}
      selectedBuildingId={selectedBuildingId}
      onEditModeChange={setEditMode}
      onRefresh={() => loadFloorPlanData(currentFloor?.id || '')}
      onBack={handleBackToList}
      onSave={handleSaveFloorPlan}
      onDepartmentChange={handleDepartmentChange}
      onBuildingChange={handleBuildingChange}
      onFloorChange={handleFloorChange}
    />
  );

  // 渲染列表视图
  const renderListView = () => (
    <div>
      <FloorStatisticsCards statistics={statistics} show={total > 10} />

      <FloorSearchForm
        form={searchForm}
        buildingOptions={buildingOptions}
        viewMode={viewMode}
        loading={loading}
        onSearch={handleSearch}
        onReset={handleReset}
        onRefresh={handleRefresh}
        onViewModeChange={setViewMode}
        onImport={() => setImportVisible(true)}
        onExport={() => setExportVisible(true)}
        onBatchDelete={handleBatchDelete}
        onAdd={() => openModal()}
        selectedCount={selectedRowKeys.length}
      />

      {selectedRowKeys.length > 0 && (
        <Alert
          title={
            <span>
              已选择 <strong>{selectedRowKeys.length}</strong> 个楼层，
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
          style={{ marginTop: 12, marginBottom: 12 }}
        />
      )}

      <Card>
        {viewMode === 'table' ? (
          <Table
            rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
            columns={columns}
            dataSource={floors}
            loading={loading}
            rowKey="id"
            pagination={paginationProps}
          />
        ) : (
          <FloorCardView
            floors={floors}
            onEdit={openModal}
            onEditFloorPlan={handleEditFloorPlan}
            onDelete={handleDelete}
          />
        )}
      </Card>

      <FloorModal
        visible={modalVisible}
        editingFloor={editingFloor}
        buildingOptions={buildingOptions}
        form={floorForm}
        onOk={handleSave}
        onCancel={() => {
          setModalVisible(false);
          floorForm.resetFields();
          setEditingFloor(null);
        }}
      />

      <ExcelImport
        entityType="floor"
        entityName="楼层"
        visible={importVisible}
        onClose={() => setImportVisible(false)}
        onImportSuccess={handleImportSuccess}
      />

      <ExcelExport
        entityType="floor"
        entityName="楼层"
        visible={exportVisible}
        onClose={() => setExportVisible(false)}
      />
    </div>
  );

  return (
    <div>{pageMode === 'editor' ? renderFloorPlanEditor() : renderListView()}</div>
  );
};

export default FloorManagement;
