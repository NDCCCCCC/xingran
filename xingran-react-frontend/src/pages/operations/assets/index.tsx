/**
 * 资产管理页面 — Phase 68.1 T6（PageTable 全契约形态）
 *
 * 迁移自页面级数据层实现：数据层 / 搜索区 / 分页 / 三态 / 删除确认全部
 * 收敛到 PageTable（UI-SPEC §3.2），本页只保留业务块（统计卡片、列配置、
 * 导入/下载模板/导出、行展开从属组件、对账抽屉）。
 *
 * 列定义仍是 `columnsSchema.ts` 的单一真理源：可见列/宽度经 useColumnConfig
 * 计算后作为 columns 传入，行操作列由 PageTable 的 rowActions 统一追加。
 */

import { formatDateTime, formatDate } from "@/utils/datetime";
import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import type { FC, Key } from "react";
import type { FormInstance } from "antd";
import { App, Button, Space, Tag, Card, Tooltip } from "antd";
import {
  PlusOutlined,
  ImportOutlined,
  ExportOutlined,
  DownloadOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { Asset } from "@/types/operations";
import { assetApi, excelApi } from "@/lib/opsApi";
import { handleApiError } from "@/utils/errorHandler";
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import { useColumnConfig } from "@/hooks/useColumnConfig";
import { ColumnConfigModal } from "@/components/shared/ColumnConfigModal";
import { createSorterMeta } from "@/utils/tableHelpers";
import { PageTable, useActionPermission } from "@/components/shared";
import type { PageTableRenderContext, SearchField } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import {
  ReconciliationDrawer,
  useReconciliationVisibility,
  type DrawerTabKey,
} from "@/components/reconciliation";
import { defaultAssetColumns, type AssetColumnConfig } from "./columnsSchema";
import ComponentListTab from "./components/ComponentListTab";

// 统计数据类型
interface Statistics {
  total: number;
  normal: number;
  stopped: number;
  nbf: number;
}

// 设备类型选项
interface DeviceTypeOption {
  value: string;
  count: number;
}

// 导出类型供其他文件复用(避免 columnsSchema.ts 直接被外引,保持单向依赖)
export type { AssetColumnConfig };

const AssetList: FC = () => {
  // Phase 86 W2（RBAC-02）：操作级权限门控（三态）。ops 码组只有 delete 键 → 删除用 .delete。
  const permGate = useActionPermission();
  const permDelete = permGate.decide(PERMISSION_CODES.opsAsset.delete);
  const permAdd = permGate.decide(PERMISSION_CODES.opsAsset.add);
  const permImport = permGate.decide(PERMISSION_CODES.opsAsset.import);
  // ⚠️ 导出类写入口（数据外带面）：本页有 2 处 —— 「下载模板」与「导出」，
  // 均归 opsAsset.export（本页导出走 handleExport 内联按钮，无 ExcelExport 容器）。
  const permExport = permGate.decide(PERMISSION_CODES.opsAsset.export);

  const { message } = App.useApp();
  const [statistics, setStatistics] = useState<Statistics>({
    total: 0,
    normal: 0,
    stopped: 0,
    nbf: 0,
  });
  const [importVisible, setImportVisible] = useState(false);
  const [templateDownloading, setTemplateDownloading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [deviceTypeOptions, setDeviceTypeOptions] = useState<DeviceTypeOption[]>([]);
  const [deviceCategoryOptions, setDeviceCategoryOptions] = useState<DeviceTypeOption[]>([]);
  const [showColumnConfig, setShowColumnConfig] = useState(false);

  // R4 (Phase 45) — ReconciliationDrawer state (lifts to page level per UI-SPEC)
  const [drawerState, setDrawerState] = useState<{
    open: boolean;
    assetId: string | null;
    workstationId: string | null;
    assetCode?: string;
    activeTab: DrawerTabKey;
  }>({ open: false, assetId: null, workstationId: null, activeTab: "summary" });

  const _reconciliationVisible = useReconciliationVisibility();
  void _reconciliationVisible;

  // PageTable render-prop 上下文提升到业务回调（删除/导入成功后刷新 + 导出取筛选）
  const reloadRef = useRef<() => void>(() => {});
  const searchFormRef = useRef<FormInstance | null>(null);

  // 服务端排序:field 对应后端 assetAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Asset>("devicesn"),
      createSorterMeta<Asset>("deviceTypeName"),
      createSorterMeta<Asset>("lastInventoryDate", "date"),
    ],
    []
  );

  // 列表请求：查询条件（current/pageSize/filters/排序）由 PageTable 内部组装后透传
  const request = useCallback(async (params: Record<string, unknown>) => {
    const result = (await assetApi.list(params)) as { data?: { list: Asset[]; total: number } };
    return { list: result.data?.list || [], total: result.data?.total || 0 };
  }, []);

  // 列配置 Hook
  const {
    config,
    visibleColumns,
    saving: configSaving,
    saveConfig,
    resetConfig,
  } = useColumnConfig({
    pageKey: "asset.list",
    defaultColumns: defaultAssetColumns,
    enableCache: true,
  });

  // 加载统计数据(专用 COUNT 端点,真实 status/nbf_status 计数,不再用伪造比例)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await assetApi.statistics();
      setStatistics(result ?? { total: 0, normal: 0, stopped: 0, nbf: 0 });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  // 加载设备类型选项
  const loadDeviceTypes = useCallback(async () => {
    try {
      const result = await assetApi.getDeviceTypes();
      if (result.code === 0 && result.data) {
        setDeviceTypeOptions(result.data);
      }
    } catch (error) {
      console.error("加载设备类型失败:", error);
      // 降级到固定选项
      setDeviceTypeOptions([
        { value: "台式机", count: 0 },
        { value: "笔记本", count: 0 },
        { value: "服务器", count: 0 },
        { value: "打印机", count: 0 },
        { value: "交换机", count: 0 },
        { value: "路由器", count: 0 },
        { value: "其他", count: 0 },
      ]);
    }
  }, []);

  // 加载设备种类选项
  const loadDeviceCategories = useCallback(async () => {
    try {
      const result = await assetApi.getDeviceCategories();
      if (result.code === 0 && result.data) {
        setDeviceCategoryOptions(result.data);
      }
    } catch (error) {
      console.error("加载设备种类失败:", error);
      // 降级到固定选项
      setDeviceCategoryOptions([
        { value: "计算机设备", count: 0 },
        { value: "网络设备", count: 0 },
        { value: "其他设备", count: 0 },
      ]);
    }
  }, []);

  // 列表首载由 PageTable 内建（mount 即发起请求）；此处只载统计与筛选选项。
  // 三个 loader 均为 useCallback([]) 稳定引用，deps 不会触发重复请求。
  useEffect(() => {
    loadStatistics();
    loadDeviceTypes();
    loadDeviceCategories();
  }, [loadStatistics, loadDeviceTypes, loadDeviceCategories]);

  // 单删（CreateDeleteAction → ConfirmDelete；失败由 confirmDelete 统一反馈后 rethrow）
  const handleDelete = useCallback(
    async (id: string) => {
      await assetApi.delete(id);
      reloadRef.current();
      loadStatistics();
    },
    [loadStatistics]
  );

  // 批删（确认/成功提示由 PageTable → BatchDeleteButton → confirmDelete 承担；
  // 成功后 PageTable 自动清空选中并重载列表，这里只补统计同步）
  const handleBatchDelete = useCallback(
    async (keys: Key[]) => {
      await assetApi.batch("delete", { ids: keys });
      loadStatistics();
    },
    [loadStatistics]
  );

  const handleExport = async () => {
    setExporting(true);
    try {
      const searchValues = (searchFormRef.current?.getFieldsValue() ?? {}) as Record<
        string,
        unknown
      >;
      const params: Record<string, unknown> = {
        current: 1,
        pageSize: 10000,
        ...searchValues,
      };

      await assetApi.excel.export(params);
      message.success("导出成功");
    } catch (_error) {
      message.error("导出失败");
    } finally {
      setExporting(false);
    }
  };

  const columns: ColumnsType<Asset> = useMemo(
    () => [
      // order=1
      {
        title: "设备序列号",
        dataIndex: "devicesn",
        key: "devicesn",
        width: 150,
        fixed: "left",
        sorter: true,
        render: (text) => (
          <Tooltip title={text}>
            <span style={{ cursor: "pointer" }}>{text}</span>
          </Tooltip>
        ),
      },
      // order=2
      { key: "sequenceNo", title: "序列号", dataIndex: "sequenceNo", width: 120, ellipsis: true },
      // order=3
      {
        key: "fixAssetNo",
        title: "固定资产编号",
        dataIndex: "fixAssetNo",
        width: 120,
        ellipsis: true,
      },
      // order=4
      {
        key: "deviceModelName",
        title: "设备型号",
        dataIndex: "deviceModelName",
        width: 120,
        ellipsis: true,
      },
      // order=5
      {
        key: "deviceTypeName",
        title: "设备类型",
        dataIndex: "deviceTypeName",
        width: 100,
        ellipsis: true,
        sorter: true,
      },
      // order=6
      {
        key: "deviceCategorySecondName",
        title: "设备中类",
        dataIndex: "deviceCategorySecondName",
        width: 120,
        ellipsis: true,
      },
      // order=7
      {
        key: "deviceBasicTypeName",
        title: "是否固定资产",
        dataIndex: "deviceBasicTypeName",
        width: 100,
        ellipsis: true,
      },
      // order=9
      {
        key: "usefulDeptName",
        title: "所属部门",
        dataIndex: "usefulDeptName",
        width: 120,
        ellipsis: true,
      },
      // order=13
      { key: "machineIp", title: "加域IP", dataIndex: "machineIp", width: 120, ellipsis: true },
      // order=14
      {
        key: "mac1",
        title: "有线MAC",
        dataIndex: "mac1",
        width: 140,
        ellipsis: true,
        render: (text) => (text ? String(text).toUpperCase() : "-"),
      },
      // order=35
      {
        key: "useStatusLabel",
        title: "使用状态",
        dataIndex: "useStatusLabel",
        width: 100,
        ellipsis: true,
      },
      // order=43
      { key: "remark", title: "备注", dataIndex: "remark", width: 200, ellipsis: true },
      // order=44
      {
        key: "signOrgnoName",
        title: "归属机构",
        dataIndex: "signOrgnoName",
        width: 150,
        ellipsis: true,
      },
      // order=45
      { key: "nowUserName", title: "责任人", dataIndex: "nowUserName", width: 100, ellipsis: true },
      // order=46
      {
        key: "nowUserDeptCode",
        title: "部门编码",
        dataIndex: "nowUserDeptCode",
        width: 120,
        ellipsis: true,
      },
      // order=47
      {
        key: "deviceUserName",
        title: "领取人",
        dataIndex: "deviceUserName",
        width: 100,
        ellipsis: true,
      },
      // order=48
      {
        key: "status",
        title: "状态",
        dataIndex: "status",
        width: 80,
        render: (s: number) => (
          <Tag color={s === 0 ? "green" : "red"}>{s === 0 ? "正常" : "停用"}</Tag>
        ),
      },
      // order=49
      {
        key: "nbfStatus",
        title: "拟报废",
        dataIndex: "nbfStatus",
        width: 80,
        render: (s: number) => (
          <Tag color={s === 1 ? "orange" : "default"}>{s === 1 ? "是" : "否"}</Tag>
        ),
      },
      // order=50
      {
        key: "drawingDate",
        title: "接收日期",
        dataIndex: "drawingDate",
        width: 120,
        render: (date) => formatDate(date),
        ellipsis: true,
      },
      // order=51
      {
        key: "machineUptime",
        title: "最后上线",
        dataIndex: "machineUptime",
        width: 150,
        render: (date) => formatDateTime(date),
        ellipsis: true,
      },
      // order=52
      {
        key: "lastInventoryDate",
        title: "盘点日期",
        dataIndex: "lastInventoryDate",
        width: 120,
        ellipsis: true,
        sorter: true,
        render: (date) => formatDate(date),
      },
      // 设备信息扩展 (order=54 设备渠道, order=55 设备属性)
      { key: "qudaoName", title: "设备渠道", dataIndex: "qudaoName", width: 100, ellipsis: true },
      {
        key: "attributeValue",
        title: "设备属性",
        dataIndex: "attributeValue",
        width: 120,
        ellipsis: true,
      },
      // 部门与用户扩展 (order=53 受益部门)
      { key: "deptName", title: "受益部门", dataIndex: "deptName", width: 120, ellipsis: true },
      // 网络信息扩展 (order=63 加域标识, order=64 无线MAC)
      { key: "machineBs", title: "加域标识", dataIndex: "machineBs", width: 100, ellipsis: true },
      {
        key: "mac2",
        title: "无线MAC",
        dataIndex: "mac2",
        width: 140,
        ellipsis: true,
        render: (text) => (text ? String(text).toUpperCase() : "-"),
      },
      // 归属与责任扩展 (order=58 使用机构, 59 使用人, 60 责任人岗位, 61 用途, 62 子用途, 66 APP扫码账号, 68 APP扫码地理位置)
      { key: "orgnoName", title: "使用机构", dataIndex: "orgnoName", width: 150, ellipsis: true },
      { key: "outerUser", title: "使用人", dataIndex: "outerUser", width: 100, ellipsis: true },
      {
        key: "nowUserJobName",
        title: "责任人岗位",
        dataIndex: "nowUserJobName",
        width: 120,
        ellipsis: true,
      },
      {
        key: "usingTypeName",
        title: "用途",
        dataIndex: "usingTypeName",
        width: 100,
        ellipsis: true,
      },
      {
        key: "subUsingTypeName",
        title: "子用途",
        dataIndex: "subUsingTypeName",
        width: 100,
        ellipsis: true,
      },
      { key: "userName", title: "APP扫码账号", dataIndex: "userName", width: 100, ellipsis: true },
      {
        key: "scanSite",
        title: "APP扫码地理位置",
        dataIndex: "scanSite",
        width: 200,
        ellipsis: true,
      },
      // 状态与日期扩展 (order=56 发放日期, 57 入库日期, 65 最后上线账号, 67 APP扫码时间, 69 Y07更新时间)
      {
        key: "useDate",
        title: "发放日期",
        dataIndex: "useDate",
        width: 120,
        render: (date) => formatDate(date),
        ellipsis: true,
      },
      {
        key: "storageDatetime",
        title: "入库日期",
        dataIndex: "storageDatetime",
        width: 120,
        render: (date) => formatDate(date),
        ellipsis: true,
      },
      {
        key: "machineUserId",
        title: "最后上线账号",
        dataIndex: "machineUserId",
        width: 120,
        ellipsis: true,
      },
      {
        key: "lastUpdateDate",
        title: "APP扫码时间",
        dataIndex: "lastUpdateDate",
        width: 150,
        render: (date) => formatDateTime(date),
        ellipsis: true,
      },
      {
        key: "y07UpdateTime",
        title: "Y07更新时间",
        dataIndex: "y07UpdateTime",
        width: 150,
        render: (date) => formatDateTime(date),
        ellipsis: true,
      },
      // 标识与盘点 (order=70 新设备标识, 71 异常标识, 72 盘点结果)
      {
        key: "newFlagLabel",
        title: "新设备标识",
        dataIndex: "newFlagLabel",
        width: 100,
        ellipsis: true,
      },
      {
        key: "errorFlagName",
        title: "异常标识",
        dataIndex: "errorFlagName",
        width: 100,
        ellipsis: true,
      },
      {
        key: "inventoryResult",
        title: "盘点结果",
        dataIndex: "inventoryResult",
        width: 100,
        ellipsis: true,
      },
      // R4 (Phase 45) — 对账健康列(行内徽标,assets 列表)
      // CR-01 修复:ops_asset 工作站绑定未在 /ops/asset/list API 响应中暴露(workstation_id
      // 不在 SELECT 列表,无法 lift useWorkstationHealth 到页面层)。在此后端限制解除前,
      // 渲染 "-" 占位以避免显示错误的 healthy 绿色徽标(原 conflictType={null} 会让
      // HealthBadge 默认渲染绿色圆点,误导用户以为所有资产均健康)。后续若需要真实
      // 对账状态需扩展 Asset DTO 增加 workstationId 字段,然后才能复用 workstations 页面的
      // lift-up 模式(assetConflictMap)。
      {
        title: "对账健康",
        key: "reconciliation",
        width: 96,
        render: (_: unknown, _record: Asset) => <>-</>,
      },
    ],
    []
  );

  // 根据列配置过滤和排序列
  const tableColumns = useMemo(() => {
    const allColumnsMap = new Map<string, ColumnsType<Asset>[number]>();
    columns.forEach((col) => allColumnsMap.set(col.key as string, col));

    const visibleCols = visibleColumns
      .map((colConfig) => {
        const col = allColumnsMap.get(colConfig.key);
        return {
          ...col,
          width: colConfig.width || col?.width,
        };
      })
      .filter((col) => col !== undefined);

    return visibleCols as ColumnsType<Asset>;
  }, [columns, visibleColumns]);

  // 行操作：删除（数量/顺序与迁移前一致，唯一入口 createDeleteAction → ConfirmDelete；
  // target 传对象名，作为确认弹窗正文与 aria-label 的对象标识，按钮文字固定「删除」。
  // 操作列由 PageTable 统一追加在列尾部）
  const rowActions = useCallback(
    (record: Asset) => [
      createDeleteAction(record.id, handleDelete, {
        target: `资产 ${record.devicesn || record.id}`,
        permission: PERMISSION_CODES.opsAsset.delete,
      }),
    ],
    [handleDelete]
  );

  // 搜索区 schema：字段名与后端查询参数一一对应
  // (devicesn / deviceModelName / deviceTypeName / deviceCategorySecondName / status / nbfStatus)
  const searchFields = useMemo<SearchField[]>(
    () => [
      { type: "input", name: "devicesn", label: "设备序列号", placeholder: "请输入设备序列号" },
      { type: "input", name: "deviceModelName", label: "型号", placeholder: "请输入型号" },
      {
        type: "select",
        name: "deviceTypeName",
        label: "设备类型",
        placeholder: "请选择设备类型",
        options: deviceTypeOptions.map((option) => ({
          label: `${option.value} (${option.count})`,
          value: option.value,
        })),
      },
      {
        type: "select",
        name: "deviceCategorySecondName",
        label: "设备种类",
        placeholder: "请选择设备种类",
        options: deviceCategoryOptions.map((option) => ({
          label: `${option.value} (${option.count})`,
          value: option.value,
        })),
      },
      {
        type: "select",
        name: "status",
        label: "状态",
        placeholder: "请选择状态",
        options: [
          { label: "正常", value: 0 },
          { label: "停用", value: 1 },
        ],
      },
      {
        type: "select",
        name: "nbfStatus",
        label: "拟报废",
        placeholder: "请选择",
        options: [
          { label: "否", value: 0 },
          { label: "是", value: 1 },
        ],
      },
    ],
    [deviceTypeOptions, deviceCategoryOptions]
  );

  // 业务区：统计卡片（PageTable render-prop 顶部；同时把 loadData / searchForm
  // 提升给业务回调，供删除/导入后刷新与导出取筛选）
  const renderBusinessArea = ({ loadData, searchForm }: PageTableRenderContext<Asset>) => {
    reloadRef.current = loadData;
    searchFormRef.current = searchForm;
    return (
      <div className="grid grid-cols-4 gap-4 mb-6">
        <Card>
          <div className="text-2xl font-bold">{statistics.total}</div>
          <div className="text-gray-500">总资产数</div>
        </Card>
        <Card>
          <div className="text-2xl font-bold text-green-600">{statistics.normal}</div>
          <div className="text-gray-500">正常状态</div>
        </Card>
        <Card>
          <div className="text-2xl font-bold text-red-600">{statistics.stopped}</div>
          <div className="text-gray-500">停用状态</div>
        </Card>
        <Card>
          <div className="text-2xl font-bold text-orange-600">{statistics.nbf}</div>
          <div className="text-gray-500">拟报废</div>
        </Card>
      </div>
    );
  };

  return (
    <div className="p-6">
      <PageTable<Asset>
        request={request}
        columns={tableColumns}
        rowKey={(record) => record.id}
        searchFields={searchFields}
        sorterMetas={sorterMetas}
        rowActions={rowActions}
        batchDelete={
          permDelete.visible ? { onDelete: handleBatchDelete, target: "资产" } : undefined
        }
        toolbar={
          <Space>
            {/* Phase 63 STOP-06 遗留：本页未渲染资产表单弹窗，新增入口保持可见
                （迁移前由列表数据层内建弹窗承接，但该弹窗从未在本页渲染） */}
            {permAdd.visible && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                disabled={permAdd.disabled}
                title={permAdd.hint}
              >
                新增资产
              </Button>
            )}
            {/* STOP-06: 下载模板兑现真实下载（原为假入口，复用导入弹层）
                Phase 86 W2（NEW-02）：这是**导出类写入口**（数据外带面 —— 模板可反推
                导入字段口径），不因「本页无 ExcelExport 容器」而被排除 → 归 opsAsset.export */}
            {permExport.visible && (
              <Button
                icon={<DownloadOutlined />}
                loading={templateDownloading}
                disabled={permExport.disabled}
                title={permExport.hint}
                onClick={async () => {
                  setTemplateDownloading(true);
                  try {
                    await excelApi.downloadTemplate("asset");
                  } catch (error) {
                    handleApiError(error, "下载模板");
                  } finally {
                    setTemplateDownloading(false);
                  }
                }}
              >
                下载模板
              </Button>
            )}
            {/* Excel 门控点 = 触发按钮（modal 容器保留无条件渲染） */}
            {permImport.visible && (
              <Button
                icon={<ImportOutlined />}
                onClick={() => setImportVisible(true)}
                disabled={permImport.disabled}
                title={permImport.hint}
              >
                导入
              </Button>
            )}
            {permExport.visible && (
              <Button
                icon={<ExportOutlined />}
                loading={exporting}
                onClick={handleExport}
                disabled={permExport.disabled}
                title={permExport.hint}
              >
                导出
              </Button>
            )}
            <Button icon={<SettingOutlined />} onClick={() => setShowColumnConfig(true)}>
              列设置
            </Button>
          </Space>
        }
        // fixed 列（devicesn fixed:left）配套 scroll.x；行展开渲染「从属组件清单」Tab
        scroll={{ x: 4200, y: 600 }}
        virtual
        expandable={{
          // Phase 48 Wave 3 (D-07): 行展开渲染「从属组件清单」Tab。
          // 仅主设备行可展开(componentType 为空即主设备);组件行本身不会
          // 出现在此列表(后端默认 component_type IS NULL 过滤已生效)。
          expandedRowRender: (record: Asset) => <ComponentListTab parentAssetId={record.id} />,
          rowExpandable: (record: Asset) => !record.componentType,
        }}
      >
        {renderBusinessArea}
      </PageTable>

      {/* 导入弹窗 */}
      <ExcelImportLazy
        entityType="asset"
        entityName="资产"
        visible={importVisible}
        onClose={() => setImportVisible(false)}
        onImportSuccess={() => {
          setImportVisible(false);
          reloadRef.current();
          loadStatistics();
        }}
      />

      {/* 列配置模态框 */}
      <ColumnConfigModal
        visible={showColumnConfig}
        config={config}
        defaultConfig={defaultAssetColumns}
        onSave={saveConfig}
        onReset={resetConfig}
        onClose={() => setShowColumnConfig(false)}
        saving={configSaving}
      />

      {/* R4 (Phase 45) — ReconciliationDrawer at page level
          Plan 02 / SC9: onApplyException 省略让抽屉自带的内联 navigate 接管,自动携带
          workstationId + assetId + assetIp + conflictType 四个 query 参数。
      */}
      <ReconciliationDrawer
        open={drawerState.open}
        onClose={() => setDrawerState((s) => ({ ...s, open: false, activeTab: "summary" }))}
        selectedAssetId={drawerState.assetId}
        workstationId={drawerState.workstationId}
        assetCode={drawerState.assetCode}
        activeTab={drawerState.activeTab}
        onTabChange={(k) => setDrawerState((s) => ({ ...s, activeTab: k }))}
      />
    </div>
  );
};

export default AssetList;
