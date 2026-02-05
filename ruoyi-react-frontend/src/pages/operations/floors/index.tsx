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

import {
  FloorCardView,
  FloorSearchForm,
  FloorModal,
  FloorPlanEditorView,
  FloorStatisticsCards,
  createFloorTableColumns,
} from './components';

import { useFloorPlanEditor } from './useFloorPlanEditor';
import { useFloorStatistics } from './useFloorStatistics';
import { useBuildingOptions } from './useBuildingOptions';

import type { ViewMode, PageMode } from './constants';
import type { FloorOption } from './components';
import { DEFAULT_FORM_VALUES } from './constants';

const FloorManagement: FC = () => {
  const [pageMode, setPageMode] = useState<PageMode>('list');
  const [currentFloor, setCurrentFloor] = useState<Floor | null>(null);
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();
  const { buildingOptions, loadBuildingOptions } = useBuildingOptions();

  const [departmentTreeData, setDepartmentTreeData] = useState<TreeNode[]>([]);
  const [buildingOptionsByDept, setBuildingOptionsByDept] = useState<Building[]>([]);
  const [selectedDeptId, setSelectedDeptId] = useState<string>('');
  const [selectedBuildingId, setSelectedBuildingId] = useState<string>('');
  const [selectedDeptName, setSelectedDeptName] = useState<string>('');
  const [selectedBuildingName, setSelectedBuildingName] = useState<string>('');

  const [filterDeptId, setFilterDeptId] = useState<string>('');
  const [filterBuildingOptions, setFilterBuildingOptions] = useState<Building[]>([]);
  const [modalDeptId, setModalDeptId] = useState<string>('');
  const [modalBuildingOptions, setModalBuildingOptions] = useState<Building[]>([]);
  const [floorOptions, setFloorOptions] = useState<FloorOption[]>([]);

  const { statistics, loadStatistics } = useFloorStatistics();

  const {
    floorPlanData,
    floorPlanLoading,
    isEditMode,
    setEditMode,
    loadFloorPlanData,
    saveFloorPlan,
    resetFloorPlan,
  } = useFloorPlanEditor(currentFloor);

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

  const isInitialized = useRef(false);
  const prevPageRef = useRef({ current: 1, pageSize: 10 });

  useEffect(() => {
    const { current, pageSize } = paginationProps;
    const prev = prevPageRef.current;
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
  }, [paginationProps.current, paginationProps.pageSize, loadFloors, loadStatistics, loadBuildingOptions]);

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

      setModalVisible(false);
      setEditingFloor(null);
      setModalDeptId('');
      setModalBuildingOptions([]);
      loadFloors();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, '操作');
    }
  }, [floorForm, editingFloor, loadFloors, loadStatistics]);

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

  const openModal = useCallback(
    async (record?: Floor) => {
      if (record) {
        handleEdit(record);
        floorForm.setFieldsValue(record);

        try {
          const buildingResult = await buildingApi.get(record.buildingId);
          const building = buildingResult.data;

          if (building) {
            setModalDeptId(building.orgId);
            const result = await buildingApi.list({ orgId: building.orgId, current: 1, pageSize: 1000 });
            setModalBuildingOptions(result.data?.list || []);
          }
        } catch (error) {
          console.error('加载楼宇信息失败:', error);
        }
      } else {
        handleAdd();
        floorForm.setFieldsValue(DEFAULT_FORM_VALUES);
        setModalDeptId('');
        setModalBuildingOptions([]);
      }
    },
    [handleEdit, handleAdd, floorForm]
  );

  const handleImportSuccess = useCallback(() => {
    loadFloors();
    loadStatistics();
    setImportVisible(false);
  }, [loadFloors, loadStatistics]);

  const handleRefresh = useCallback(() => {
    loadFloors();
    loadStatistics();
  }, [loadFloors, loadStatistics]);

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

  const handleFilterDepartmentChange = useCallback(
    async (deptId: string) => {
      setFilterDeptId(deptId || '');
      setFilterBuildingOptions([]);
      searchForm.setFieldValue('buildingId', undefined);

      if (deptId) {
        try {
          const result = await buildingApi.list({ orgId: deptId, current: 1, pageSize: 1000 });
          setFilterBuildingOptions(result.data?.list || []);
        } catch (error) {
          console.error('加载筛选楼宇选项失败:', error);
        }
      }
    },
    [searchForm]
  );

  const handleModalDepartmentChange = useCallback(
    async (deptId: string) => {
      setModalDeptId(deptId || '');
      setModalBuildingOptions([]);
      floorForm.setFieldValue('buildingId', undefined);

      if (deptId) {
        try {
          const result = await buildingApi.list({ orgId: deptId, current: 1, pageSize: 1000 });
          setModalBuildingOptions(result.data?.list || []);
        } catch (error) {
          console.error('加载模态框楼宇选项失败:', error);
        }
      }
    },
    [floorForm]
  );

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

  const handleBuildingChange = useCallback(
    async (buildingId: string) => {
      if (!buildingId) return;

      setSelectedBuildingId(buildingId);

      const building = buildingOptionsByDept.find(b => b.id === buildingId);
      if (building) {
        setSelectedBuildingName(building.name);
      }

      const options = await loadFloorOptionsByBuilding(buildingId);

      if (options.length > 0) {
        await handleFloorChange(options[0].id);
      }
    },
    [loadFloorOptionsByBuilding, buildingOptionsByDept, handleFloorChange]
  );

  const handleDepartmentChange = useCallback(
    async (deptId: string) => {
      if (!deptId) return;

      setSelectedDeptId(deptId);
      setSelectedBuildingId('');
      setSelectedBuildingName('');

      const findDeptName = (nodes: TreeNode[], id: string): string | undefined => {
        for (const node of nodes) {
          if (node.value === id) return node.label;
          if (node.children) {
            const found = findDeptName(node.children, id);
            if (found) return found;
          }
        }
        return undefined;
      };
      const deptName = findDeptName(departmentTreeData, deptId);
      if (deptName) {
        setSelectedDeptName(deptName);
      }

      const buildings = await loadBuildingOptionsByDept(deptId);

      if (buildings.length > 0) {
        await handleBuildingChange(buildings[0].id);
      }
    },
    [loadBuildingOptionsByDept, departmentTreeData, handleBuildingChange]
  );

  const handleEditFloorPlan = useCallback(
    async (floor: Floor) => {
      setCurrentFloor(floor);
      setPageMode('editor');
      setEditMode(false);

      try {
        const buildingResult = await buildingApi.get(floor.buildingId);
        const building = buildingResult.data;

        if (building) {
          setSelectedBuildingId(building.id);
          setSelectedBuildingName(building.name);
          setSelectedDeptId(building.orgId);

          const findDeptName = (nodes: TreeNode[], id: string): string | undefined => {
            for (const node of nodes) {
              if (node.value === id) return node.label;
              if (node.children) {
                const found = findDeptName(node.children, id);
                if (found) return found;
              }
            }
            return undefined;
          };
          const deptName = findDeptName(departmentTreeData, building.orgId);
          if (deptName) {
            setSelectedDeptName(deptName);
          }

          await loadBuildingOptionsByDept(building.orgId);
          await loadFloorOptionsByBuilding(building.id);
        }
      } catch (error) {
        console.error('加载楼宇信息失败:', error);
      }

      await loadFloorPlanData(floor.id);
    },
    [loadFloorPlanData, setEditMode, loadFloorOptionsByBuilding, loadBuildingOptionsByDept, departmentTreeData]
  );

  const handleSaveFloorPlan = useCallback(
    async (data: Record<string, unknown>) => {
      if (!currentFloor) return;
      await saveFloorPlan(data, currentFloor);
    },
    [currentFloor, saveFloorPlan]
  );

  const handleBackToList = useCallback(() => {
    setPageMode('list');
    setCurrentFloor(null);
    resetFloorPlan();
  }, [resetFloorPlan]);

  const columns = createFloorTableColumns({
    onEdit: openModal,
    onEditFloorPlan: handleEditFloorPlan,
    onDelete: handleDelete,
  });

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
      selectedDeptName={selectedDeptName}
      selectedBuildingId={selectedBuildingId}
      selectedBuildingName={selectedBuildingName}
      onEditModeChange={setEditMode}
      onRefresh={() => loadFloorPlanData(currentFloor?.id || '')}
      onBack={handleBackToList}
      onSave={handleSaveFloorPlan}
      onDepartmentChange={handleDepartmentChange}
      onBuildingChange={handleBuildingChange}
      onFloorChange={handleFloorChange}
    />
  );

  const renderListView = () => (
    <div>
      <FloorStatisticsCards statistics={statistics} show={total > 10} />

      <FloorSearchForm
        form={searchForm}
        buildingOptions={buildingOptions}
        viewMode={viewMode}
        loading={loading}
        departmentTreeData={departmentTreeData}
        buildingOptionsByDept={filterBuildingOptions}
        selectedDeptId={filterDeptId}
        disabled={pageMode === 'editor'}
        onSearch={handleSearch}
        onReset={handleReset}
        onRefresh={handleRefresh}
        onViewModeChange={setViewMode}
        onImport={() => setImportVisible(true)}
        onExport={() => setExportVisible(true)}
        onBatchDelete={handleBatchDelete}
        onAdd={() => openModal()}
        onDepartmentChange={handleFilterDepartmentChange}
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
        departmentTreeData={departmentTreeData}
        buildingOptionsByDept={modalBuildingOptions}
        selectedDeptId={modalDeptId}
        form={floorForm}
        onOk={handleSave}
        onCancel={() => {
          setModalVisible(false);
          setEditingFloor(null);
          setModalDeptId('');
          setModalBuildingOptions([]);
        }}
        onDepartmentChange={handleModalDepartmentChange}
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
