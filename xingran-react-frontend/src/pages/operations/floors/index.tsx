/**
 * 楼层管理页面
 */

import { useState, useEffect, useCallback, useMemo } from "react";
import type { FC } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { Table, Button, Card, Alert, Layout } from "antd";
import type { Floor, Building } from "@/types";
import type { FloorPlanData } from "@/components/cad-editor/types";
import { floorApi, buildingApi } from "@/lib/opsApi";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { useSidebarDeptFilter } from "@/hooks/useSidebarDeptFilter";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import ExcelExport from "@/components/shared/ExcelExport";
import { useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { DeptSidebar } from "@/components/operations/DeptSidebar";
import { StatisticsCards } from "@/components/operations/StatisticsCards";
import { createSorterMeta } from "@/utils/tableHelpers";
import {
  FloorCardView,
  FloorSearchForm,
  FloorModal,
  FloorPlanEditorView,
  createFloorTableColumns,
} from "./components";
import { useFloorPlanEditor } from "./useFloorPlanEditor";
import { useFloorStatistics } from "./useFloorStatistics";
import { useBuildingOptions } from "./useBuildingOptions";
import { useDepartmentData } from "../buildings/useDepartmentData";
import type { ViewMode, PageMode } from "./constants";
import type { FloorOption } from "./components";
import { DEFAULT_FORM_VALUES } from "./constants";

const { Content } = Layout;

const FloorManagement: FC = () => {
  // Phase 86 W2（RBAC-02）：操作级权限门控（三态 ready/loading/error）。
  // 权限码单一真相源 = src/constants/permissionCodes.ts；ops 码组只有 delete 键 → 删除用 .delete。
  const permGate = useActionPermission();
  const permEdit = permGate.decide(PERMISSION_CODES.opsFloor.edit);
  const permDelete = permGate.decide(PERMISSION_CODES.opsFloor.delete);
  const permAdd = permGate.decide(PERMISSION_CODES.opsFloor.add);
  const permImport = permGate.decide(PERMISSION_CODES.opsFloor.import);
  const permExport = permGate.decide(PERMISSION_CODES.opsFloor.export);

  // ⚠️ 平面图保存 = **四码合并口径**（取逻辑与：四码全持才可用）。
  // 依据 `useFloorPlanEditor.ts:249-255`：一次 saveFloorPlan 会同时
  //   ① saveWalls → opsWall.*；② saveDoors → opsDoor.*；③ saveTexts → 平面图；④
  //   saveWorkstations → **opsWorkstation.edit**（同一次保存里改写工位位置）。
  // 漏掉 opsWorkstation.edit 即越权（持平面图三码但无工位写权者仍能改工位位置）。
  // 「查看平面图」行操作打开的就是这个**可保存**的编辑器 → 共用同一决策对象。
  const permFloorPlanEdit = permGate.decide(PERMISSION_CODES.opsFloorPlan.edit);
  const permWall = permGate.decide(PERMISSION_CODES.opsWall.edit);
  const permDoor = permGate.decide(PERMISSION_CODES.opsDoor.edit);
  const permWorkstationEdit = permGate.decide(PERMISSION_CODES.opsWorkstation.edit);
  // 四码合并：任一码「未就绪」→ 渲染但禁用（三态语义优先）；四码全就绪 → 全持才可用；
  // 全就绪但缺任一码 → 不渲染（visible=false）。
  const planDecisions = [permFloorPlanEdit, permWall, permDoor, permWorkstationEdit];
  const planNotReady = planDecisions.some((d) => d.disabled);
  const floorPlanDecision = planNotReady
    ? {
        visible: true,
        disabled: true,
        hint: planDecisions.find((d) => d.hint)?.hint,
      }
    : planDecisions.every((d) => d.visible)
      ? { visible: true, disabled: false, hint: undefined }
      : { visible: false, disabled: false, hint: undefined };

  const location = useLocation();
  const [pageMode, setPageMode] = usePersistedStateController<PageMode>({
    keyPrefix: location.pathname,
    keySuffix: "pageMode",
    defaultValue: "list",
  });
  const [currentFloor, setCurrentFloor] = useState<Floor | null>(null);
  const [viewMode, setViewMode] = usePersistedStateController<ViewMode>({
    keyPrefix: location.pathname,
    keySuffix: "viewMode",
    defaultValue: "table",
  });
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();
  const { buildingOptions, loadBuildingOptions } = useBuildingOptions();
  const departmentData = useDepartmentData();

  const { selectedDeptId, handleDeptSelect } = useSidebarDeptFilter({
    searchForm: undefined,
    clearFieldNames: ["buildingId"],
  });

  // 筛选状态
  const [filterBuildingOptions, setFilterBuildingOptions] = useState<Building[]>([]);

  // 模态框状态
  const [modalDeptId, setModalDeptId] = useState<string>("");
  const [modalBuildingOptions, setModalBuildingOptions] = useState<Building[]>([]);

  // 编辑器模式状态
  const [buildingOptionsByDept, setBuildingOptionsByDept] = useState<Building[]>([]);
  const [selectedDeptIdForEditor, setSelectedDeptIdForEditor] = useState<string>("");
  const [selectedBuildingId, setSelectedBuildingId] = useState<string>("");
  const [selectedDeptName, setSelectedDeptName] = useState<string>("");
  const [selectedBuildingName, setSelectedBuildingName] = useState<string>("");
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

  // 服务端排序:field 对应后端 floorAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Floor>("name"),
      createSorterMeta<Floor>("floorNo"),
      createSorterMeta<Floor>("buildingName"),
      createSorterMeta<Floor>("area"),
      createSorterMeta<Floor>("orderNum"),
      createSorterMeta<Floor>("status"),
      createSorterMeta<Floor>("createdAt", "date"),
    ],
    []
  );

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
    getColumnSortOrder,
    handleTableChange,
  } = useTableManager<Floor>(
    async (params) => {
      const searchParams = selectedDeptId ? { ...params, orgId: selectedDeptId } : params;
      const result = await floorApi.list(searchParams);
      return {
        list: result.data?.list || [],
        total: result.data?.total || 0,
      };
    },
    {
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
      sorterMetas,
    }
  );

  // 初始化加载和部门变化时重新加载
  useEffect(() => {
    Promise.all([loadStatistics(), loadBuildingOptions(), departmentData.loadDepartments()]);
    loadFloors();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- departmentData object recreated each render
  }, [loadFloors, loadStatistics, loadBuildingOptions, departmentData.loadDepartments]);

  useEffect(() => {
    loadFloors();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedDeptId]);

  const handleSave = useCallback(async () => {
    try {
      const values = (await floorForm.validateFields()) as Partial<Floor>;

      if (editingFloor) {
        await floorApi.update(editingFloor.id, values);
        handleSuccess("更新");
      } else {
        await floorApi.create(values);
        handleSuccess("创建");
      }

      setModalVisible(false);
      setEditingFloor(null);
      setModalDeptId("");
      setModalBuildingOptions([]);
      loadFloors();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    }
  }, [floorForm, editingFloor, loadFloors, loadStatistics, setModalVisible, setEditingFloor]);

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await floorApi.delete(id);
        handleSuccess("删除");
        loadFloors();
        loadStatistics();
      } catch (error) {
        handleApiError(error, "删除");
      }
    },
    [loadFloors, loadStatistics]
  );

  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) return;

    try {
      await floorApi.batch("delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      loadFloors();
      loadStatistics();
    } catch (error) {
      handleApiError(error, "批量删除");
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
            const result = await buildingApi.list({
              orgId: building.orgId,
              current: 1,
              pageSize: 50,
            });
            let options = result.data?.list || [];
            // 楼宇兜底注入(2026-06-30,同 info-points):pageSize:50 可能不包含当前
            // buildingId → 模态框 Select 显示 raw UUID。buildingApi.get 已返回完整对象,
            // 用它注入一条临时 Option,确保 Select 一定能找到 label。
            if (!options.find((b) => b.id === building.id)) {
              options = [...options, building];
            }
            setModalBuildingOptions(options);
          }
        } catch (error) {
          console.error("加载楼宇信息失败:", error);
        }
      } else {
        handleAdd();
        floorForm.setFieldsValue(DEFAULT_FORM_VALUES);
        if (selectedDeptId) {
          setModalDeptId(selectedDeptId);
          try {
            const result = await buildingApi.list({
              orgId: selectedDeptId,
              current: 1,
              pageSize: 50,
            });
            setModalBuildingOptions(result.data?.list || []);
          } catch (error) {
            console.error("加载楼宇选项失败:", error);
          }
        } else {
          setModalDeptId("");
          setModalBuildingOptions([]);
        }
      }
    },
    [handleEdit, handleAdd, floorForm, selectedDeptId]
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

  const handleDeptSelectWithBuildingLoad = useCallback(
    async (deptId: string) => {
      setFilterBuildingOptions([]);
      searchForm.setFieldValue("buildingId", undefined);

      if (deptId) {
        try {
          const result = await buildingApi.list({ orgId: deptId, current: 1, pageSize: 50 });
          setFilterBuildingOptions(result.data?.list || []);
        } catch (error) {
          console.error("加载筛选楼宇选项失败:", error);
        }
      }
    },
    [searchForm]
  );

  // 模态框部门变化处理
  const handleModalDepartmentChange = useCallback(
    async (deptId: string) => {
      setModalDeptId(deptId || "");
      setModalBuildingOptions([]);
      floorForm.setFieldValue("buildingId", undefined);

      if (deptId) {
        try {
          const result = await buildingApi.list({ orgId: deptId, current: 1, pageSize: 50 });
          setModalBuildingOptions(result.data?.list || []);
        } catch (error) {
          console.error("加载模态框楼宇选项失败:", error);
        }
      }
    },
    [floorForm]
  );

  // 筛选楼宇变化处理
  const handleFilterBuildingChange = useCallback(
    async (_buildingId: string) => {
      // 触发搜索以应用楼宇筛选
      handleSearch();
    },
    [selectedDeptId, handleSearch] // eslint-disable-line react-hooks/exhaustive-deps -- selectedDeptId is intentional
  );

  // 加载指定部门的楼宇选项（用于编辑器模式）
  const loadBuildingOptionsByDept = useCallback(async (deptId: string): Promise<Building[]> => {
    try {
      const result = await buildingApi.list({ orgId: deptId, current: 1, pageSize: 50 });
      const buildings = result.data?.list || [];

      setBuildingOptionsByDept(buildings);
      return buildings;
    } catch (error) {
      console.error("加载楼宇选项失败:", error);
      return [];
    }
  }, []);

  // 加载指定楼宇的楼层选项（用于编辑器模式）
  const loadFloorOptionsByBuilding = useCallback(
    async (buildingId: string): Promise<FloorOption[]> => {
      try {
        const result = await floorApi.list({ buildingId, current: 1, pageSize: 50 });
        const floors = result.data?.list || [];

        const options = floors.map((f: Floor) => ({
          id: f.id,
          name: f.name || "",
          floorNo: Number(f.floorNo) || 0,
        }));

        setFloorOptions(options);
        return options;
      } catch (error) {
        console.error("加载楼层选项失败:", error);
        return [];
      }
    },
    []
  );

  // 编辑器模式：楼层变化
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
        console.error("加载楼层失败:", error);
      }
    },
    [loadFloorPlanData, setEditMode]
  );

  // 编辑器模式：楼宇变化
  const handleEditorBuildingChange = useCallback(
    async (buildingId: string) => {
      if (!buildingId) return;

      setSelectedBuildingId(buildingId);

      const building = buildingOptionsByDept.find((b) => b.id === buildingId);
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

  // 编辑器模式：部门变化
  const handleEditorDepartmentChange = useCallback(
    async (deptId: string, deptName: string) => {
      if (!deptId) return;

      setSelectedDeptIdForEditor(deptId);
      setSelectedDeptName(deptName);
      setSelectedBuildingId("");
      setSelectedBuildingName("");

      const buildings = await loadBuildingOptionsByDept(deptId);

      if (buildings.length > 0) {
        await handleEditorBuildingChange(buildings[0].id);
      }
    },
    [loadBuildingOptionsByDept, handleEditorBuildingChange]
  );

  const handleEditFloorPlan = useCallback(
    async (floor: Floor) => {
      setCurrentFloor(floor);
      setPageMode("editor");
      setEditMode(false);

      // loadFloorPlanData 与 building/选项加载并行，编辑器 loading 只绑 loadFloorPlanData
      const loadFloorPlan = loadFloorPlanData(floor.id);

      try {
        const buildingResult = await buildingApi.get(floor.buildingId);
        const building = buildingResult.data;

        if (building) {
          setSelectedBuildingId(building.id);
          setSelectedBuildingName(building.name);
          setSelectedDeptIdForEditor(building.orgId);
          setSelectedDeptName(building.orgName || "");

          // 两个选项加载在拿到 building 后并行
          await Promise.all([
            loadBuildingOptionsByDept(building.orgId),
            loadFloorOptionsByBuilding(building.id),
          ]);
        }
      } catch (error) {
        console.error("加载楼宇信息失败:", error);
      }

      await loadFloorPlan;
    },
    [
      loadFloorPlanData,
      setEditMode,
      setPageMode,
      loadFloorOptionsByBuilding,
      loadBuildingOptionsByDept,
    ]
  );

  const handleSaveFloorPlan = useCallback(
    async (data: FloorPlanData) => {
      if (!currentFloor) return;
      await saveFloorPlan(data, currentFloor);
    },
    [currentFloor, saveFloorPlan]
  );

  const handleBackToList = useCallback(() => {
    setPageMode("list");
    setCurrentFloor(null);
    resetFloorPlan();
  }, [resetFloorPlan, setPageMode]);

  const columns = createFloorTableColumns({
    onEdit: openModal,
    onEditFloorPlan: handleEditFloorPlan,
    onDelete: handleDelete,
    getColumnSortOrder,
    editPermission: PERMISSION_CODES.opsFloor.edit,
    deletePermission: PERMISSION_CODES.opsFloor.delete,
    // 「查看平面图」与平面图保存按钮共用同一个四码合并决策对象
    floorPlanDecision,
  });

  const renderFloorPlanEditor = () => (
    <FloorPlanEditorView
      currentFloor={currentFloor}
      floorPlanData={floorPlanData}
      loading={floorPlanLoading}
      isEditMode={isEditMode}
      departments={departmentData.departments}
      buildingOptions={buildingOptionsByDept}
      floorOptions={floorOptions}
      selectedDeptId={selectedDeptIdForEditor}
      selectedDeptName={selectedDeptName}
      selectedBuildingId={selectedBuildingId}
      selectedBuildingName={selectedBuildingName}
      onEditModeChange={setEditMode}
      onRefresh={() => loadFloorPlanData(currentFloor?.id || "")}
      onBack={handleBackToList}
      onSave={handleSaveFloorPlan}
      onDepartmentChange={(deptId) => handleEditorDepartmentChange(deptId, "")}
      onBuildingChange={handleEditorBuildingChange}
      onFloorChange={handleFloorChange}
      // 四码合并决策（与行操作「查看平面图」共用同一对象）
      editDecision={floorPlanDecision}
    />
  );

  const renderListView = () => (
    <Layout style={{ background: "var(--theme-bg-secondary)", minHeight: "calc(100vh - 64px)" }}>
      <DeptSidebar
        selectedDeptId={selectedDeptId}
        onSelect={(keys, info) => {
          handleDeptSelect(keys, info);
          const deptId = keys.length > 0 ? (keys[0] as string) : "";
          handleDeptSelectWithBuildingLoad(deptId);
        }}
      />
      <Content style={{ background: "#fff" }}>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: "总楼层数", value: statistics.total },
            {
              title: "正常",
              value: statistics.active,
              styles: { content: { color: "var(--theme-success)" } },
            },
            {
              title: "停用",
              value: statistics.inactive,
              styles: { content: { color: "var(--theme-error)" } },
            },
          ]}
        />

        <FloorSearchForm
          form={searchForm}
          buildingOptions={buildingOptions}
          viewMode={viewMode}
          loading={loading}
          buildingOptionsByDept={filterBuildingOptions}
          selectedDeptId={selectedDeptId}
          disabled={pageMode === "editor"}
          onSearch={handleSearch}
          onReset={handleReset}
          onRefresh={handleRefresh}
          onViewModeChange={setViewMode}
          // Phase 86 W2 通道①：`onImport`/`onExport` 由必填放宽为可选 —— ready 无权限 → 不传
          // → 触发按钮不渲染（不在 DOM）；未就绪 → 传回调 + `importDisabled` → 渲染但 disabled
          onImport={permImport.visible ? () => setImportVisible(true) : undefined}
          onExport={
            permExport.visible
              ? () => {
                  // 获取当前筛选条件
                  const values = searchForm.getFieldsValue() as Record<string, unknown>;
                  const currentFilters: Record<string, unknown> = {};
                  Object.keys(values).forEach((key) => {
                    const value = values[key];
                    if (value !== undefined && value !== null && value !== "") {
                      currentFilters[key] = value;
                    }
                  });
                  setExportFilters(currentFilters);
                  setExportVisible(true);
                }
              : undefined
          }
          importDisabled={permImport.disabled}
          exportDisabled={permExport.disabled}
          notReadyHint={permImport.hint ?? permExport.hint}
          // Phase 86 W2 通道②：`onBatchDelete` 保持必填（noop 兜底）+
          // `showBatchDelete` 决定渲不渲染 + `batchDeleteDisabled` 决定禁不禁用
          onBatchDelete={() => {
            if (!permDelete.visible) return;
            handleBatchDelete();
          }}
          showBatchDelete={permDelete.visible}
          batchDeleteDisabled={permDelete.disabled}
          onAdd={() => {
            if (!permAdd.visible) return;
            openModal();
          }}
          onBuildingChange={handleFilterBuildingChange}
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
          {viewMode === "table" ? (
            <Table
              rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
              columns={columns}
              dataSource={floors}
              loading={loading}
              rowKey="id"
              pagination={paginationProps}
              onChange={handleTableChange}
            />
          ) : (
            <FloorCardView
              floors={floors}
              onEdit={openModal}
              onEditFloorPlan={handleEditFloorPlan}
              onDelete={handleDelete}
              // 卡片视图与表格视图共用页面层同一组门控决策
              editDecision={permEdit}
              deleteDecision={permDelete}
              floorPlanDecision={floorPlanDecision}
            />
          )}
        </Card>

        <FloorModal
          visible={modalVisible}
          editingFloor={editingFloor}
          buildingOptions={buildingOptions}
          departments={departmentData.departments}
          buildingOptionsByDept={modalBuildingOptions}
          selectedDeptId={modalDeptId}
          form={floorForm}
          onOk={handleSave}
          onCancel={() => {
            setModalVisible(false);
            setEditingFloor(null);
            setModalDeptId("");
            setModalBuildingOptions([]);
          }}
          onDepartmentChange={handleModalDepartmentChange}
        />

        <ExcelImportLazy
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
          filters={exportFilters}
        />
      </Content>
    </Layout>
  );

  return <div>{pageMode === "editor" ? renderFloorPlanEditor() : renderListView()}</div>;
};

export default FloorManagement;
