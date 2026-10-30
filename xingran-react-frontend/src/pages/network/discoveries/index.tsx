/**
 * 设备发现管理 — Phase 67 T1 存量迁移（PageTable 契约，405 → 契约化）
 *
 * 迁移自双 hook（usePagination + useServerSort）+ 手工 Table/搜索卡实现；
 * 列表数据层/分页/排序/三态/删除确认全部由 PageTable 内建承接。
 *
 * PR 差异记录（66/67 契约内允许变化，67-UI-SPEC §1.2）：
 * - B 类：删除确认文案对齐 65 模板（原 columns.tsx 裸 Modal.confirm「确认删除?/确定/取消」
 *   → ConfirmDelete「确认删除 / 将删除{taskName}，删除后不可恢复，是否继续？/删除/取消」）；
 *   成功文案 helper「已删除」替代原「删除成功」；onOk Promise 天然获得提交中防重。
 * - C 类 1：状态筛选接线生效（原 Select 未受控，选择不进请求参数；现经 searchFields
 *   schema 进入 filters.status）——缺陷修复。
 * - C 类 2：搜索按钮「查询→搜索」；新增「重置」按钮（SearchForm 内建）。
 * - retry 联动（M1-error 裁决）：request 内联动 loadStatistics——错误重试/搜索/刷新
 *   均同时刷新列表 + 统计（副作用：翻页也会刷新统计，原翻页不刷，可感知差异趋强不趋弱）。
 * - 轮询保持：useDiscoveryPolling 继续工作（列表行经 request 回传驻留 pollList state，
 *   onPoll 桥接 PageTable ctx.loadData）；CreateModal/ResultModal/导出原样未迁。
 */
import { useCallback, useRef, useState } from "react";
import type { FC } from "react";
import { Button, Row, Col, Card, Statistic, App, Form } from "antd";
import {
  PlusOutlined,
  EyeOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from "@ant-design/icons";
import { useDiscoveryData, useDiscoveryPolling } from "./hooks";
import { getDiscoveryColumns } from "./columns";
import { CreateModal, ResultModal } from "./modals";
import { STATUS_OPTIONS } from "./constants";
import { parseIPRanges } from "./utils";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import type { FormInstance } from "antd/es/form";
import type { PageResponse } from "@/types";
import type { DeviceDiscovery } from "@/types";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal, useActionPermission } from "@/components/shared";
import { PageTable } from "@/components/shared";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { createSorterMeta } from "@/utils/tableHelpers";
import { handleApiError, isFormValidationError } from "@/utils/errorHandler";

const statusSearchOptions = STATUS_OPTIONS.map((opt) => ({ label: opt.label, value: opt.value }));

const DeviceDiscoveryPage: FC = () => {
  const { message } = App.useApp();

  // Phase 86 W3（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  const permAdd = permGate.decide(PERMISSION_CODES.networkDiscovery.add);
  // 导出类写入口（数据外带面）：真实触发按钮在 `NetworkExport.tsx:120` 组件内部，
  // 页面上没有独立导出按钮 → 门控点落在 `<NetworkExport>` 元素本身。
  const permExport = permGate.decide(PERMISSION_CODES.networkDiscovery.export);
  // Phase 86 W3 补漏：ResultModal 的「导入设备」按钮（`modals/ResultModal.tsx:50-60`
  // 的 `onClick={onImport}`）会 `POST /network/discoveries/{id}/import` **落库**把发现
  // 设备导入为正式设备 —— 属写入口，原施工图未列（页级扫描器亦无感）。
  // 域内无独立 import 码 → 就近归 `networkDiscovery.add`（与「新增发现任务」同码）。
  const permImport = permGate.decide(PERMISSION_CODES.networkDiscovery.add);

  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);
  const [form] = Form.useForm();

  // PageTable ctx.loadData 提升到删除/创建/轮询回调
  const reloadRef = useRef<() => void>(() => {});
  // 轮询桥接：request 回传的最新列表行驻留此处，供 useDiscoveryPolling 判断 running 任务
  const [pollList, setPollList] = useState<DeviceDiscovery[]>([]);
  // 导出当前页参数：request 入参回传（PageTable 契约外经 state 透传给 NetworkExport）
  const [exportPage, setExportPage] = useState({ current: 1, pageSize: 10 });

  // 统计/部门/结果弹窗数据（PageTable 契约外业务；hook 文件零改动）
  const {
    discoveredDevices,
    resultsLoading,
    departments,
    statistics,
    modalState,
    currentDiscovery,
    setModalState,
    setCurrentDiscovery,
    loadStatistics,
    loadDepartments,
    loadDiscoveryResults,
  } = useDiscoveryData({ current: 1, pageSize: 10 });

  // 服务端排序:field 与 columns.dataIndex 对齐（经 PageTable sorterMetas 透传）
  const sorterMetas = [
    createSorterMeta<DeviceDiscovery>("taskName"),
    createSorterMeta<DeviceDiscovery>("discoveryType"),
    createSorterMeta<DeviceDiscovery>("status"),
    createSorterMeta<DeviceDiscovery>("startedAt"),
    createSorterMeta<DeviceDiscovery>("completedAt"),
  ];

  // 列表请求（PageTable 唯一数据入口）；同时联动统计刷新（retry = 列表 + 统计都刷）
  const request = useCallback(
    async (params: Record<string, unknown>) => {
      void loadStatistics();
      setExportPage({
        current: (params.current as number) ?? 1,
        pageSize: (params.pageSize as number) ?? 10,
      });
      const result = await post<PageResponse<DeviceDiscovery>>("/network/discoveries/list", params);
      const list = result.data?.list || [];
      setPollList(list);
      return { list, total: result.data?.total || 0 };
    },
    [loadStatistics]
  );

  // 轮询管理（onPoll 桥接 PageTable 内部重载；行为与迁前 loadDiscoveries 等价）
  const handlePoll = useCallback(() => reloadRef.current(), []);
  useDiscoveryPolling({ discoveries: pollList, onPoll: handlePoll });

  // 打开创建模态框
  const openModal = async () => {
    // 先打开弹窗，再后台加载部门列表，避免阻塞弹窗显示
    form.resetFields();
    form.setFieldsValue({ snmpPort: 161, discoveryType: "snmp" });
    setModalState((prev) => ({ ...prev, modalVisible: true }));
    // 后台加载部门（不阻塞弹窗打开）
    loadDepartments();
  };

  // 创建发现任务
  const handleCreate = async (form: FormInstance<unknown>) => {
    try {
      const formValues = await form.validateFields();
      const values = formValues as {
        ipRanges?: string;
        taskName: string;
        discoveryType: string;
        snmpCommunity?: string;
        snmpPort?: number;
        groupId?: string;
        autoImport?: boolean;
      };
      const ipRangesText = values.ipRanges || "";
      const ipRanges = parseIPRanges(ipRangesText);

      if (ipRanges.length === 0) {
        message.error("请输入有效的IP范围");
        return;
      }

      const requestData = {
        taskName: values.taskName,
        discoveryType: values.discoveryType,
        ipRanges: ipRanges,
        snmpCommunity: values.snmpCommunity || "",
        snmpPort: values.snmpPort || 161,
        groupId: values.groupId || null,
        autoImport: values.autoImport || false,
      };

      await post("/network/discoveries/create", requestData);
      message.success("发现任务已创建");
      setModalState((prev) => ({ ...prev, modalVisible: false }));
      form.resetFields();
      reloadRef.current();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return;
      }
      handleApiError(error, "创建任务");
    }
  };

  // 删除任务（经 rowActions createDeleteAction→ConfirmDelete；成功文案由 helper
  // 「已删除」承接，失败 helper 内 handleApiError 且确认框保持打开可重试）
  const handleDelete = async (id: string) => {
    await post(`/network/discoveries/${id}/delete`, {});
    reloadRef.current();
  };

  // 查看发现结果
  const handleViewResult = async (record: DeviceDiscovery) => {
    setCurrentDiscovery(record);
    // FIX-01: 先开弹窗再加载，加载期间内嵌表显示 loading 蒙层
    setModalState((prev) => ({ ...prev, resultModalVisible: true }));
    await loadDiscoveryResults((record as DeviceDiscovery).id);
  };

  // 导入发现的设备
  const handleImport = async () => {
    if (!currentDiscovery) return;
    try {
      await post(`/network/discoveries/${currentDiscovery.id}/import`, {});
      message.success("导入成功");
      setModalState((prev) => ({ ...prev, resultModalVisible: false }));
      reloadRef.current();
    } catch (_error) {
      message.error("导入失败");
    }
  };

  // 批量导出
  const handleBatchExport = async (entityTypes: string[]) => {
    setBatchExporting(true);
    try {
      const filename = await batchExport(entityTypes, {});
      message.success(`批量导出成功，文件: ${filename}`);
      setBatchModalVisible(false);
    } catch (error: unknown) {
      message.error(`批量导出失败：${(error as Error).message}`);
    } finally {
      setBatchExporting(false);
    }
  };

  const columns = getDiscoveryColumns();

  return (
    <div>
      <PageTable<DeviceDiscovery>
        request={request}
        columns={columns}
        rowKey={(record) => record.id}
        scroll={{ x: 1500 }}
        searchFields={[
          {
            type: "select",
            name: "status",
            label: "状态",
            options: statusSearchOptions,
            placeholder: "请选择状态",
          },
        ]}
        sorterMetas={sorterMetas}
        rowActions={(record) => [
          {
            // Phase 86 W3：读动作（查看结果只读，不落库）→ 不加门控
            key: "view-result",
            label: "查看结果",
            icon: <EyeOutlined />,
            onClick: () => handleViewResult(record),
            disabled: record.status === "pending",
          },
          createDeleteAction(record.id, handleDelete, {
            target: record.taskName,
            permission: PERMISSION_CODES.networkDiscovery.delete,
          }),
        ]}
        toolbar={
          <>
            {/* Phase 86 W3：批量导出 —— 门控点 = `<NetworkExport>` 组件元素本身
                （「导出」按钮在 `NetworkExport.tsx:120` 组件内部，页面上无独立触发按钮）。
                ready 无 networkDiscovery.export → 整个组件（含其内部「导出」按钮）不在 DOM。 */}
            {permExport.visible && (
              <span title={permExport.hint} aria-disabled={permExport.disabled || undefined}>
                <NetworkExport
                  entityType="discoveries"
                  entityName="设备发现"
                  filters={{}}
                  current={exportPage.current}
                  pageSize={exportPage.pageSize}
                />
              </span>
            )}
            {/* Phase 86 W3：新增任务 —— 条件渲染（networkDiscovery.add） */}
            {permAdd.visible && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={openModal}
                aria-label="新增任务"
                title={permAdd.hint ?? "新增任务"}
                disabled={permAdd.disabled}
              >
                创建发现任务
              </Button>
            )}
          </>
        }
      >
        {() => (
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={4}>
              <Card>
                <Statistic title="任务总数" value={statistics.total} prefix={<CloudServerOutlined />} />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="待执行"
                  value={statistics.pending}
                  styles={{ content: { color: "var(--theme-text-tertiary)" } }}
                  prefix={<ClockCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="扫描中"
                  value={statistics.running}
                  styles={{ content: { color: "var(--theme-info)" } }}
                  prefix={<CloudServerOutlined />}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="已完成"
                  value={statistics.completed}
                  styles={{ content: { color: "var(--theme-success)" } }}
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="失败"
                  value={statistics.failed}
                  styles={{ content: { color: "var(--theme-error)" } }}
                  prefix={<CloseCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic title="发现设备" value={statistics.totalDevices} />
              </Card>
            </Col>
          </Row>
        )}
      </PageTable>

      {/* 批量导出 Modal */}
      <BatchExportModal
        visible={batchModalVisible}
        onConfirm={handleBatchExport}
        onCancel={() => setBatchModalVisible(false)}
        loading={batchExporting}
      />

      {/* 创建发现任务模态框 */}
      <CreateModal
        open={modalState.modalVisible}
        departments={departments}
        onOk={handleCreate}
        onCancel={() => setModalState((prev) => ({ ...prev, modalVisible: false }))}
      />

      {/* 发现结果模态框 */}
      <ResultModal
        open={modalState.resultModalVisible}
        currentDiscovery={currentDiscovery}
        discoveredDevices={discoveredDevices}
        loading={resultsLoading}
        onImport={handleImport}
        importVisible={permImport.visible}
        importDisabled={permImport.disabled}
        importHint={permImport.hint}
        onClose={() => setModalState((prev) => ({ ...prev, resultModalVisible: false }))}
      />
    </div>
  );
};

export default DeviceDiscoveryPage;
