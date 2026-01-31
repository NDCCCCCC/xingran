/**
 * 楼层管理页面
 */

import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import { Table, Button, Space, Card, Alert, message } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import type { Floor } from '@/types';
import { floorApi } from '@/lib/opsApi';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';

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
import { useBuildingOptions, type BuildingOption } from './useBuildingOptions';

// 常量和类型
import type { ViewMode, PageMode } from './constants';
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

  // 初始化加载
  useEffect(() => {
    setTimeout(() => {
      loadFloors();
      loadStatistics();
      loadBuildingOptions();
    }, 0);
  }, [paginationProps.current, paginationProps.pageSize, loadFloors, loadStatistics, loadBuildingOptions]);

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
    } catch (error: any) {
      if (error.errorFields) return;
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
   * 打开平面图编辑器
   */
  const handleEditFloorPlan = useCallback(
    async (floor: Floor) => {
      setCurrentFloor(floor);
      setPageMode('editor');
      setEditMode(false);
      await loadFloorPlanData(floor.id);
    },
    [loadFloorPlanData, setEditMode]
  );

  /**
   * 保存平面图
   */
  const handleSaveFloorPlan = useCallback(
    async (data: any) => {
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
    onEditFloorPlan,
    onDelete: handleDelete,
  });

  // 渲染平面图编辑器视图
  const renderFloorPlanEditor = () => (
    <FloorPlanEditorView
      currentFloor={currentFloor}
      floorPlanData={floorPlanData}
      loading={floorPlanLoading}
      isEditMode={isEditMode}
      onEditModeChange={setEditMode}
      onRefresh={() => loadFloorPlanData(currentFloor?.id || '')}
      onBack={handleBackToList}
      onSave={handleSaveFloorPlan}
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
