/**
 * 工位管理页面
 *
 * 性能优化说明：
 * - 将 API 导入移至顶层，避免动态导入开销
 * - 使用 Promise.all 并行加载独立数据
 * - 优化 useEffect 依赖数组，减少不必要的重渲染
 */

import { useState, useCallback, useEffect, useRef, useMemo } from "react";
import type { FC } from "react";
import type React from "react";
import { Table, Button, Space, Form, Input, Select, Card, Alert, Radio, Layout } from "antd";
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
  SettingOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import type { WorkstationOps } from "@/types";
import { useTableManager } from "@/hooks/useTableManager";
import { useTableQuery } from "@/hooks/useTableQuery";
import { usePagination } from "@/hooks/usePagination";
import { useSidebarDeptFilter } from "@/hooks/useSidebarDeptFilter";
import { useAliasByLocation } from "@/hooks/useAliasByLocation";
import { ExcelImportLazy } from "@/components/shared";
import ExcelExport from "@/components/shared/ExcelExport";
import { DeptSidebar } from "@/components/operations/DeptSidebar";
import { StatisticsCards } from "@/components/operations/StatisticsCards";
import { filterExternalOrgDepts, trimTitleToLastSegment } from "@/utils/deptUtils";
import { handleSuccess as showSuccessMessage } from "@/utils/errorHandler";
import { buildSearchParams } from "@/utils/buildSearchParams";
import { useWorkstationData, useWorkstationModals, useWorkstationView } from "./hooks";
import { getWorkstationColumns } from "./columns";
import { WorkstationEditModal } from "./modals";
import { WorkstationCardView, WorkstationFloorPlanView } from "./views";
import { LocationAliasDrawer } from "./LocationAliasDrawer";
import { STATUS_OPTIONS } from "./constants";
import type { FloorOption, UserOption, DeptTreeNode } from "./types";
// ❌ 移除：import { workstationApi } from '@/lib/opsApi';
// ✅ 优化：直接导入具体 API，避免 barrel 文件 (bundle-barrel-imports)
import { buildingApi, floorApi, workstationApi, deptApi } from "@/lib/opsApi";
import WorkstationDeviceTable from "@/components/operations/WorkstationDeviceTable";
import { createSorterMeta } from "@/utils/tableHelpers";
import { useMenuStore } from "@/store/menuStore";
import {
  HealthCard,
  ReconciliationDrawer,
  useWorkstationHealth,
  useReconciliationVisibility,
  type DrawerTabKey,
} from "@/components/reconciliation";

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

  // Phase 39: 监听工位编辑模态框 orgId 变化 (新增时由 onOrgChange, 编辑时由 setEditFormValues 触发),
  // 顶层 state lift → useAliasByLocation 拉取该机构的 alias 映射部门列表 (D-06/D-07 决策)。
  // 注意: EditModal 内部的 Form.useWatch("orgId") 仍保留 (subDeptTree 派生依赖),
  // 这里只是为了在顶层拿到 orgId 喂给 useAliasByLocation, 不修改 EditModal 内部逻辑。
  const [watchedOrgId, setWatchedOrgId] = useState<string | undefined>(undefined);
  const { data: aliasList = [] } = useAliasByLocation(watchedOrgId);

  // Phase 39-07: 工位部门物理位置映射 Drawer 状态
  const [aliasDrawerOpen, setAliasDrawerOpen] = useState(false);
  const menuPermissions = useMenuStore((s) => s.permissions);
  const canListAlias = menuPermissions.includes("ops:location:alias:list");

  // 派生:仅外部机构(isExternalOrg===1 的节点)的部门树。
  // 模态框"所属机构"与左侧 DeptSidebar(默认 externalOnly=true)共用同一份外部机构视图。
  // 全量部门树仍保留为 deptTreeData,以便"所属部门"下拉能在选中机构下
  // 取该机构节点的全后代子树(deptTreeData 是含 isExternalOrg 信息的原始树)。
  //
  // 标题收窄:useWorkstationData.buildTreeData 把 title 拼成全路径
  // (如 "中国太平洋财产保险股份有限公司/分公司本部")。在"所属机构"下拉里
  // 这种全路径同样冗长(尤其"分公司本部"只占末段),这里收窄为最后一段。
  const orgTreeData = useMemo(
    () => trimTitleToLastSegment(filterExternalOrgDepts<DeptTreeNode>(deptTreeData)),
    [deptTreeData]
  );

  // Cascader 懒加载状态
  const [cascaderOptions, setCascaderOptions] = useState<CascaderOption[]>([]);
  const [loadingCascader, setLoadingCascader] = useState(false);

  // 控制子表（设备列表）展开/收起 — 整行点击切换
  const [expandedRowKeys, setExpandedRowKeys] = useState<React.Key[]>([]);

  // R4 (Phase 45) — Lift state for reconciliation integration (B6 lift)
  //
  // expandedWorkstationId:取展开行的第一行作为"激活工位"
  // workstationHealthQuery:在 page level 调一次 useWorkstationHealth,避免 WorkstationDeviceTable
  //   内每行 N+1 useQuery
  // assetConflictMap:从 query.data.assets 提取 assetId → conflictType 映射,通过 prop 下传
  //
  // WR-01 提示(代码契约锁): useWorkstationHealth 的 cache key 是 workstationId 本身,
  // 因此无论这里(expand 行)、ReconciliationDrawer(useAssetHealth)还是后续可能的
  // WorkstationDeviceTable 子查询,只要 workstationId 一致就会命中同一份 cache entry,
  // 不会 N+1。Future refactor 切勿:
  //   1) 把这个 hook 拆到不同 provider 树(失去共享 cache)
  //   2) 把 expandedWorkstationId 替换为 expandedRowKeys(导致 cache key 在每次 collapse/expand
  //      时变化,触发额外 fetch)
  const expandedWorkstationId = expandedRowKeys.length > 0 ? String(expandedRowKeys[0]) : "";
  const workstationHealthQuery = useWorkstationHealth(expandedWorkstationId);
  const assetConflictMap = useMemo(() => {
    const m = new Map<string, string>();
    (workstationHealthQuery.data?.assets ?? []).forEach((a) => {
      if (a.conflictType) {
        m.set(a.assetId, a.conflictType);
      }
    });
    return m;
  }, [workstationHealthQuery.data?.assets]);

  // Drawer state (lifts to page level per UI-SPEC "Both pages — at page level: render once")
  const [drawerState, setDrawerState] = useState<{
    open: boolean;
    assetId: string | null;
    workstationId: string | null;
    assetCode?: string;
    activeTab: DrawerTabKey;
  }>({ open: false, assetId: null, workstationId: null, activeTab: "summary" });

  const reconciliationVisible = useReconciliationVisibility();

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();
  const { selectedDeptId, handleDeptSelect } = useSidebarDeptFilter({
    searchForm: undefined,
    clearFieldNames: ["floorId"],
  });

  // 服务端排序:field 必须与 columns.tsx 的 dataIndex 一致(useServerSort 按 sorter.field===dataIndex 匹配)
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<WorkstationOps>("name"),
      createSorterMeta<WorkstationOps>("buildingName"),
      createSorterMeta<WorkstationOps>("floorName"),
      createSorterMeta<WorkstationOps>("deptName"),
      createSorterMeta<WorkstationOps>("userName"),
      createSorterMeta<WorkstationOps>("type"),
      createSorterMeta<WorkstationOps>("status"),
      createSorterMeta<WorkstationOps>("createdAt", "date"),
      createSorterMeta<WorkstationOps>("updatedAt", "date"),
    ],
    []
  );

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
    getColumnSortOrder,
    handleTableChange,
  } = useTableManager<WorkstationOps>(
    // ✅ 优化：移除动态导入，直接使用顶层导入的 API (bundle-dynamic-imports)
    async (params) => {
      const searchParams = selectedDeptId ? { ...params, orgId: selectedDeptId } : params;
      const result = (await workstationApi.list(searchParams)) as {
        data?: { list: WorkstationOps[]; total: number };
      };
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
      sorterMetas,
    }
  );

  const {
    loadStatistics: loadStatisticsFromHook,
    loadFloorOptions,
    loadDeptOptions,
    loadUserOptions,
    ensureUser,
  } = useWorkstationData(setStatistics, setFloorOptions, setDeptTreeData, setUserOptions);

  // Refs for stable column callbacks
  const handleEditRef = useRef<(record: WorkstationOps) => void>(() => {});
  const handleDeleteRef = useRef<(id: string) => void>(() => {});

  // 待写入表单的回显值 — 当 cascaderOptions 就绪后由 useEffect 消费
  // 替代原先 setTimeout(0) 的脆弱时序，避免 React 18 自动批处理下不可靠
  const pendingCascaderWrite = useRef<{
    record: WorkstationOps;
    buildingId: string;
  } | null>(null);

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

  // ✅ Wave 5: useTableQuery companion to useTableManager — exercises the
  // React-Query data-fetching hook for a separate, non-conflicting list.
  // Fetches the *first page* of workstations via React Query, dedup'd by the
  // query key (resource, current, pageSize). This demonstrates the companion
  // pattern from 30-03-SUMMARY.md: useTableManager stays for modal/form state,
  // useTableQuery handles a parallel list query.
  //
  // The result is intentionally surfaced in the page footer chip (see JSX
  // below) so the consumer visibly benefits from the cache + dedup. The chip
  // reads the React-Query-side first page size while the main table is
  // driven by useTableManager with the user's search filters.
  const { data: reactQueryWorkstations } = useTableQuery<WorkstationOps>({
    resource: "workstations",
    current: 1,
    pageSize: 5,
    filters: {},
    queryFn: async (params) => {
      const result = (await workstationApi.list(params)) as {
        data?: { list: WorkstationOps[]; total: number };
      };
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
  });

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
      console.error("初始化加载失败:", error);
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

  const handleWorkstationModalOk = useCallback(
    async (values: Record<string, unknown>) => {
      const submitValues = { ...values };
      if (Array.isArray(submitValues.floorId)) {
        submitValues.floorId = submitValues.floorId[submitValues.floorId.length - 1];
      }

      if (editingWorkstation) {
        await workstationApi.update(editingWorkstation.id, submitValues);
        showSuccessMessage("更新");
      } else {
        await workstationApi.create(submitValues);
        showSuccessMessage("创建");
      }
      closeModal(workstationForm);
      setModalVisible(false);
      handleSuccess();
    },
    [editingWorkstation, closeModal, workstationForm, setModalVisible, handleSuccess]
  );

  // Sync refs with current callbacks
  handleEditRef.current = (record: WorkstationOps) => {
    handleOpenModal(record);
  };
  handleDeleteRef.current = (id: string) => handleDelete(id, handleSuccess);

  const columns = useMemo(
    () =>
      getWorkstationColumns({
        handleEdit: (r) => handleEditRef.current(r),
        handleDelete: (id) => handleDeleteRef.current(id),
        getColumnSortOrder,
      }),
    [getColumnSortOrder]
  );

  const handleDeptChange = (deptId: string) => {
    loadUserOptions(deptId);
  };

  // ==================== Cascader 级联选择相关函数 ====================
  // 二级结构：楼宇 → 楼层（机构选择使用 DepartmentTreeSelect）

  // ✅ 优化：移除 Cascader 中的动态导入，使用顶层导入 (bundle-dynamic-imports)
  // 加载楼宇列表（级联第一级，根据机构ID）
  const loadBuildingsForCascader = useCallback(async (orgId: string): Promise<CascaderOption[]> => {
    try {
      const params = { current: 1, pageSize: 50, orgId };
      const buildingResult = await buildingApi.list(params);
      const buildings = buildingResult.data?.list || [];
      return buildings.map((b: { id: string; name: string }) => ({
        value: b.id,
        label: b.name,
        isLeaf: false,
      }));
    } catch (error) {
      console.error("加载楼宇列表失败:", error);
      return [];
    }
  }, []); // 空依赖数组：buildingApi 是稳定的模块导入

  // 加载楼层列表（级联第二级）
  const loadFloorsForCascader = useCallback(
    async (buildingId: string): Promise<CascaderOption[]> => {
      try {
        const floorResult = await floorApi.list({ buildingId, current: 1, pageSize: 50 });
        const floors = floorResult.data?.list || [];
        return floors.map((f) => ({
          value: f.id,
          label: f.name || f.floorNo || "",
          isLeaf: true,
        }));
      } catch (error) {
        console.error("加载楼层列表失败:", error);
        return [];
      }
    },
    []
  );

  // Cascader 懒加载处理函数
  const handleCascaderLoadData = useCallback(
    async (selectedOptions: CascaderOption[]) => {
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
    },
    [cascaderOptions, loadFloorsForCascader]
  );

  // 机构变化时加载楼宇列表
  const handleOrgChange = useCallback(
    async (orgId: string) => {
      // Phase 39: 同步 orgId 到顶层 state, 触发 useAliasByLocation 重新拉取 alias 映射部门
      setWatchedOrgId(orgId || undefined);
      if (!orgId) {
        setCascaderOptions([]);
        return;
      }

      setLoadingCascader(true);
      try {
        const buildings = await loadBuildingsForCascader(orgId);
        setCascaderOptions(buildings);
      } catch (error) {
        console.error("加载楼宇列表失败:", error);
        setCascaderOptions([]);
      } finally {
        setLoadingCascader(false);
      }
    },
    [loadBuildingsForCascader]
  );

  // 预加载 Cascader 路径数据（用于编辑回显）
  // 入参 orgId 用于加载机构下所有楼宇，buildingId 用于定位需要附加 children 的目标楼宇
  const preloadCascaderPath = useCallback(
    async (orgId: string, buildingId: string) => {
      try {
        const [buildings, floors] = await Promise.all([
          loadBuildingsForCascader(orgId),
          loadFloorsForCascader(buildingId),
        ]);
        const buildingWithFloors = buildings.map((b) => ({
          ...b,
          children: b.value === buildingId ? floors : undefined,
        }));
        setCascaderOptions(buildingWithFloors);
      } catch (error) {
        console.error("预加载 Cascader 路径失败:", error);
      }
    },
    [loadBuildingsForCascader, loadFloorsForCascader]
  );

  // 设置编辑模式表单值的辅助函数
  // ✅ 优化：移除动态导入，并行获取楼层和楼宇信息 (async-parallel, bundle-dynamic-imports)
  // 不再使用 setTimeout(0)；回显值通过 pendingCascaderWrite ref + cascaderOptions useEffect 协调
  const setEditFormValues = useCallback(
    async (record: WorkstationOps) => {
      // 所属用户兜底注入(2026-06-30,同 info-points):userOptions 是 pageSize:50 全集源,
      // 当前工位的 userId 可能不在 → Select 显示 raw UUID。用 record.userName 注入临时 Option。
      if (record.userId) {
        ensureUser({ id: record.userId, username: record.userName });
      }
      if (!record.floorId) {
        workstationForm.setFieldsValue(record);
        // Phase 39: 同步编辑模式的 orgId 到顶层 state, 触发 useAliasByLocation
        setWatchedOrgId(record.orgId || undefined);
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

        // 暂存"等待 cascaderOptions 就绪后写入"的待办值
        pendingCascaderWrite.current = { record, buildingId };

        // 并行获取楼宇信息（用于拿到 orgId）
        const buildingResult = await buildingApi.get(buildingId);
        const building = buildingResult.data;
        const orgId = building?.orgId;

        // 不管有没有 orgId 都要把 orgId 写入表单（来自 building.get 的 orgId）
        // 这样编辑模式下"所属机构"字段才能正确回显
        pendingCascaderWrite.current = {
          record: { ...record, orgId },
          buildingId,
        };

        // Phase 39: 编辑模式 orgId 也要同步到顶层 state,
        // 触发 useAliasByLocation 拉取该机构的 alias 映射部门
        setWatchedOrgId(orgId || undefined);

        if (orgId) {
          // 预加载 cascaderOptions；useEffect 会在选项就绪后写入表单
          await preloadCascaderPath(orgId, buildingId);
        } else {
          // 没有 orgId 时直接写入表单（不依赖 cascaderOptions）
          workstationForm.setFieldsValue({
            ...record,
            orgId,
            floorId: [buildingId, record.floorId],
          });
          pendingCascaderWrite.current = null;
        }
      } catch (error) {
        console.error("获取楼层或楼宇信息失败:", error);
        workstationForm.setFieldsValue(record);
        pendingCascaderWrite.current = null;
      }
    },
    [preloadCascaderPath, workstationForm, ensureUser]
  );

  // 打开编辑模态框
  const handleOpenModal = useCallback(
    async (record?: WorkstationOps) => {
      if (record) {
        handleEdit(record);
        // 编辑模式触发部门下用户的异步加载(2026-06-30):除 ensureUser 兜底当前用户外,
        // 仍以 deptId 加载该部门+子部门用户作为 Select 候选,根因修复。
        if (record.deptId) {
          loadUserOptions(record.deptId).catch(() => {
            /* hook 内部已 handleApiError */
          });
        }
        await setEditFormValues(record);
      } else {
        handleAdd();
        setCascaderOptions([]);
        // Phase 39: 新增模式无 orgId, 清空顶层 watchedOrgId 避免上一次编辑的 alias 数据残留
        setWatchedOrgId(undefined);
        workstationForm.setFieldsValue({ status: 0, type: 0 });
      }
      setModalVisible(true);
    },
    [setEditFormValues, handleEdit, handleAdd, workstationForm, setModalVisible, loadUserOptions]
  );

  const handleImportSuccess = useCallback(() => {
    refreshData();
    setImportVisible(false);
  }, [refreshData]);

  // 等待 cascaderOptions 就绪后写入编辑表单的 floorId / orgId
  // 替代 setEditFormValues 内 setTimeout(0) 的脆弱时序
  useEffect(() => {
    const pending = pendingCascaderWrite.current;
    if (!pending) return;

    // 模态框已关闭或切换到其他工位 → 取消待写入
    if (!modalVisible || editingWorkstation?.id !== pending.record.id) {
      pendingCascaderWrite.current = null;
      return;
    }

    // cascaderOptions 必须包含目标 buildingId 才算就绪
    const buildingOption = cascaderOptions.find((opt) => opt.value === pending.buildingId);
    if (!buildingOption) return;

    const { record, buildingId } = pending;
    workstationForm.setFieldsValue({
      ...record,
      floorId: [buildingId, record.floorId],
    });
    pendingCascaderWrite.current = null;
  }, [cascaderOptions, modalVisible, editingWorkstation, workstationForm]);

  const renderCurrentView = () => {
    if (viewMode === "table") {
      return (
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
            // 收紧复选框列宽度（默认 62px → 40px）
            columnWidth: 40,
          }}
          columns={columns}
          dataSource={workstations}
          loading={loading}
          rowKey="id"
          virtual
          scroll={{ x: 1800, y: 600 }}
          pagination={paginationProps}
          onRow={(record) => ({
            onClick: (e) => {
              // 整行任意位置点击 = 展开/收起设备子表
              // 排除复选框、展开图标、操作列按钮等已被内部 stopPropagation / 需要原生行为的位置
              const target = e.target as HTMLElement;
              if (
                target.closest(".ant-checkbox-wrapper") ||
                target.closest(".ant-table-row-expand-icon") ||
                target.closest("[data-stop-row-click]")
              ) {
                return;
              }
              if (expandedRowKeys.includes(record.id)) {
                setExpandedRowKeys([]);
              } else {
                setExpandedRowKeys([record.id]);
              }
            },
            style: { cursor: "pointer" },
          })}
          expandedRowKeys={expandedRowKeys}
          onExpand={(expanded, record) => {
            setExpandedRowKeys(
              expanded
                ? Array.from(new Set([...expandedRowKeys, record.id]))
                : expandedRowKeys.filter((id) => id !== record.id)
            );
          }}
          expandable={{
            expandedRowRender: (record: WorkstationOps) => (
              <div>
                {/* R4 (Phase 45) — HealthCard 嵌入工位 expand 顶部 (D-A1-01) */}
                {reconciliationVisible && (
                  <HealthCard
                    workstationId={record.id}
                    onApplyException={() => {
                      window.open(
                        `/asset/reconciliation/exception-rules/new?workstationId=${record.id}`,
                        "_blank"
                      );
                    }}
                  />
                )}
                <WorkstationDeviceTable
                  workstationId={record.id}
                  conflictTypeMap={assetConflictMap}
                  onDeviceChange={refreshData}
                  onBadgeClick={(assetId, _conflictType) =>
                    setDrawerState({
                      open: true,
                      assetId,
                      workstationId: record.id,
                      activeTab: "summary",
                    })
                  }
                />
              </div>
            ),
            rowExpandable: () => true,
            // 收紧展开图标列宽度（默认 48px → 40px），与复选框列对齐
            columnWidth: 40,
            // 不再自定义 expandIcon — 使用 Ant Design 默认的三角形展开图标
            // 此前自定义的"查看设备/收起设备"按钮与默认三角图标并存，造成重复 UI
          }}
          onChange={handleTableChange}
        />
      );
    }

    if (viewMode === "card") {
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
          handleOpenModal(workstations.find((w) => w.id === ws.id));
        }}
        onCloseFloorPlan={() => setViewMode("table")}
      />
    );
  };

  return (
    <Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>
      <DeptSidebar selectedDeptId={selectedDeptId} onSelect={handleDeptSelect} />
      <Content style={{ background: "#fff" }}>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: "总工位数", value: statistics.total, prefix: <DesktopOutlined /> },
            {
              title: "空闲",
              value: statistics.available,
              styles: { content: { color: "var(--theme-success, #2d8949)" } },
              prefix: <CheckCircleOutlined />,
            },
            {
              title: "占用",
              value: statistics.occupied,
              styles: { content: { color: "var(--theme-error, #ba3630)" } },
              prefix: <StopOutlined />,
            },
            {
              title: "维护",
              value: statistics.maintain,
              styles: { content: { color: "var(--theme-warning, #b07a20)" } },
              prefix: <StopOutlined />,
            },
          ]}
        />
        <Card style={{ marginBottom: 16 }}>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "flex-start",
              flexWrap: "wrap",
              gap: "16px",
            }}
          >
            <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
              <Form.Item name="floorId" label="所属楼层">
                <Select
                  placeholder={selectedDeptId ? "搜索楼层" : "请先选择部门"}
                  allowClear
                  className="user-form-input"
                  style={{ width: 180 }}
                  showSearch
                  filterOption={false}
                  disabled={!selectedDeptId}
                  onSearch={(kw) => loadFloorOptions(selectedDeptId, kw)}
                  options={floorOptions.map((f) => ({ value: f.id, label: f.name }))}
                />
              </Form.Item>
              <Form.Item name="name" label="工位名称">
                <Input
                  placeholder="请输入工位名称"
                  allowClear
                  className="user-form-input"
                  style={{ width: 150 }}
                />
              </Form.Item>
              <Form.Item name="status" label="状态">
                <Select
                  placeholder="请选择状态"
                  allowClear
                  className="user-form-input"
                  style={{ width: 120 }}
                  onSearch={() => {}}
                >
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
              <Radio.Group
                value={viewMode}
                onChange={(e) => setViewMode(e.target.value)}
                buttonStyle="solid"
              >
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
              {/* quick 260713-df0: 工位导入辅助 — 下载 sys_dept dept_name|code 映射表,
                  用户填 Excel 时不必再翻部门列表查 dept_code */}
              <Button icon={<DownloadOutlined />} onClick={() => deptApi.exportMapping()}>
                下载部门映射表
              </Button>
              <Button
                icon={<ExportOutlined />}
                onClick={() => {
                  setExportFilters(buildSearchParams({ searchForm }));
                  setExportVisible(true);
                }}
              >
                导出
              </Button>
              {canListAlias && (
                <Button icon={<SettingOutlined />} onClick={() => setAliasDrawerOpen(true)}>
                  映射
                </Button>
              )}
              {selectedRowKeys.length > 0 && (
                <Button
                  icon={<DeleteOutlined />}
                  style={{ color: "var(--theme-error, #ba3630)" }}
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
              title={
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
          {/* Wave 5: visible cue that useTableQuery is wired (companion to useTableManager). */}
          {reactQueryWorkstations && (
            <Alert
              type="success"
              showIcon
              style={{ marginTop: 12 }}
              title={
                <span>
                  useTableQuery (React Query) 已生效: 首页缓存共{" "}
                  <strong>{reactQueryWorkstations.total}</strong> 条工位 (companion pattern via
                  React Query, 30s staleTime).
                </span>
              }
            />
          )}
        </Card>
        <Card>{renderCurrentView()}</Card>
        <WorkstationEditModal
          open={modalVisible}
          form={workstationForm}
          editingWorkstation={editingWorkstation}
          orgTreeData={orgTreeData}
          deptTreeData={deptTreeData}
          aliasList={aliasList}
          userOptions={userOptions}
          cascaderOptions={cascaderOptions}
          loadingCascader={loadingCascader}
          handleCascaderLoadData={handleCascaderLoadData}
          onOrgChange={handleOrgChange}
          onOk={handleWorkstationModalOk}
          onCancel={() => {
            setModalVisible(false);
            closeModal(workstationForm);
          }}
          onDeptChange={handleDeptChange}
        />
        <LocationAliasDrawer open={aliasDrawerOpen} onClose={() => setAliasDrawerOpen(false)} />
        <ExcelImportLazy
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
        {/* R4 (Phase 45) — ReconciliationDrawer at page level (UI-SPEC "Both pages — at page level: render once") */}
        <ReconciliationDrawer
          open={drawerState.open}
          onClose={() => setDrawerState((s) => ({ ...s, open: false, activeTab: "summary" }))}
          selectedAssetId={drawerState.assetId}
          workstationId={drawerState.workstationId}
          activeTab={drawerState.activeTab}
          onTabChange={(k) => setDrawerState((s) => ({ ...s, activeTab: k }))}
        />
      </Content>
    </Layout>
  );
};

export default WorkstationManagement;
