/**
 * 配置备份管理页面 - Phase 67 T2 契约组件迁移版
 *
 * 原 1141 行 → PageTable/SearchForm/FormModal 契约组件链路（PR 差异记录）：
 * - Table+usePagination+useServerSort+手工 onChange → PageTable（request 直连
 *   /network/backups/list，返回 {list,total}，SC2 的 setTotal 回填链路整体删除，
 *   分页/排序/错误条（ErrorAlertWithRetry 内建）/刷新由 PageTable 承接）
 * - 列表数据经 groupBackupsByDevice 在 request 内分组（deviceGroups state 不再
 *   经 hook 中转；useBackupData 保留 loadStatistics/loadDevices）
 * - 搜索卡 → searchFields（deviceName input + backupType select）；按钮统一
 *   SearchForm 内建「搜索/重置/刷新」（原「查询」→「搜索」，B 类文案对齐）
 * - 统计卡×4（备份总数/自动备份/手动备份/涉及设备）→ PageTable children 顶部
 * - 创建备份 Modal → FormModal（显式 okText="确定"，不用默认「保存」；
 *   dirtyCheck 默认 true 为新增防护）；handleBackup(hook) 保留原签名复用
 * - 还原选择 Modal/版本列表抽屉/差异对比 Modal/配置内容抽屉 → modals/RestoreModals.tsx
 *   纯搬迁（versionColumns 随迁；B 类豁免两处还原确认「确认恢复配置」「确认恢复到
 *   此版本?」原文保留）
 * - 事实注记：原 renderDiffLine 空 diff 行分支内误嵌 BatchExportModal（:312-337），
 *   迁移时归位到页面根级（仅位置修正，行为不变）
 * - hooks/useBackupData、useBackupModals、useRestoreTask、utils、types 零修改
 */
import { useCallback, useMemo, useRef, useState, type FC } from "react";
import { App, Button, Card, Col, Form, Row, Space, Statistic, Tag } from "antd";
// Phase 71 (W4)：裸 Select/TextArea → 契约组件（children 形态改写 options）
import { AppSelect, AppTextArea } from "@/components/ui";
import type { ColumnsType } from "antd/es/table";
import type { PageResponse, ConfigBackup } from "@/types";
import type { DeviceBackupGroup, DiffLine } from "./types";
import { useBackupData, useBackupDiff, useBackupModals } from "./hooks";
import { useRestoreTask } from "./hooks/useRestoreTask";
import { groupBackupsByDevice } from "./utils";
import RestoreModals from "./modals/RestoreModals";
import PageTable from "@/components/shared/PageTable";
import FormModal from "@/components/shared/FormModal";
import { useFormModalForm } from "@/components/shared/hooks/useFormModalForm";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import ActionButtons from "@/components/shared/ActionButtons";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal, confirmDelete, useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { createSorterMeta } from "@/utils/tableHelpers";
import { formatDateTime } from "@/utils/datetime";
import {
  SaveOutlined,
  CloudUploadOutlined,
  ClockCircleOutlined,
  DatabaseOutlined,
  DiffOutlined,
  DownloadOutlined,
  HistoryOutlined,
} from "@ant-design/icons";

// 备份类型选项（搜索字段用）
const BACKUP_TYPE_SEARCH_OPTIONS = [
  { label: "自动备份", value: "auto" },
  { label: "手动备份", value: "manual" },
];

const ConfigBackupPage: FC = () => {
  const { message } = App.useApp();

  // Phase 86 W3（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  const permAdd = permGate.decide(PERMISSION_CODES.networkBackup.add);
  // ⚠️ 本页**无删除入口**：`networkBackup` 码组里的 `delete` 键（permissionCodes.ts:194）虽已声明，
  // 但配置备份页只有「还原 / 版本列表 / 版本对比 / 创建备份 / 导出」五类入口。
  // 两处 `confirmDelete`（`handleConfirmRestore` / `handleRestore`）与
  // `modals/RestoreModals.tsx` 的「恢复到指定版本」承载的都是**还原（恢复配置）**动作 ——
  // 它们只是复用了 confirmDelete 这个二次确认 helper（形参名 `onDelete` 与业务语义无关）。
  // 因此还原类入口一律用 `networkBackup.restore`（决策变量名 `permRestore`，**不叫 permDelete**）：
  //   ❌ 若误挂 `.delete` → ① 持 restore 无 delete 者被误拒（有权还原却看不到按钮）；
  //                          ② 持 delete 无 restore 者可越权还原（把配置恢复到任意历史版本）。
  // 门控落点：行「还原」走 ActionButtons 的 `permission:` 传参（见下方列定义）；
  // 还原确认的两处触发按钮在 `modals/RestoreModals.tsx`（`permRestore` 在那里决定）。
  // 导出类写入口（数据外带面）：真实触发按钮在 `NetworkExport.tsx:120` 组件内部，
  // 页面上没有独立导出按钮 → 门控点落在 `<NetworkExport>` 元素本身。
  const permExport = permGate.decide(PERMISSION_CODES.networkBackup.export);

  // PageTable loadData 转发引用（useBackupModals onLoad 需在 hook 初始化前可用）
  const reloadRef = useRef<(() => void) | null>(null);
  // PageTable 内部 searchForm 引用（NetworkExport filters 承接当前搜索条件）
  const searchFormRef = useRef<React.ComponentRef<typeof Form> | null>(null);
  // 导出当前页参数（request 回填，NetworkExport 承接原 paginationProps 语义）
  const [exportPage, setExportPage] = useState({ current: 1, pageSize: 10 });

  const [selectedDeviceGroup, setSelectedDeviceGroup] = useState<DeviceBackupGroup | null>(null);
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 创建备份弹窗表单（FormModal 契约）
  const backupForm = useFormModalForm();
  // useBackupData 现仅承载统计/设备（loadBackups 已由 PageTable request 直连承接，
  // searchForm 仅为 hook 必填参数保留空实例）
  const [hookForm] = Form.useForm();

  // 服务端排序:field 与 columns.dataIndex 对齐(PageTable 按 sorterMetas 装配 sorter)
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<DeviceBackupGroup>("deviceId"),
      createSorterMeta<DeviceBackupGroup>("deviceName"),
      createSorterMeta<DeviceBackupGroup>("createdAt", "date"),
    ],
    []
  );

  // 统计 + 设备（列表改由 PageTable request 直连）
  const { devices, statistics, loadDevices, loadStatistics } = useBackupData({
    current: 1,
    pageSize: 10,
    searchForm: hookForm,
  });

  // 列表请求（PageTable 契约：返回 {list,total}；分组逻辑沿用 utils）
  const request = useCallback(
    async (params: Record<string, unknown>) => {
      const { current, pageSize, orderByColumn, isAsc, ...rest } = params as {
        current: number;
        pageSize: number;
        orderByColumn?: string;
        isAsc?: boolean;
      } & Record<string, unknown>;
      const result = await post<PageResponse<ConfigBackup>>("/network/backups/list", {
        current,
        pageSize,
        // v129-recheck WR-02 语义保留：仅显式排序时透传（undefined 时无排序语义）
        ...(orderByColumn !== undefined ? { orderByColumn, isAsc } : {}),
        ...rest,
      });
      setExportPage({ current, pageSize });
      return {
        list: groupBackupsByDevice(result.data?.list || []),
        total: result.data?.total || 0,
      };
    },
    []
  );

  const {
    diffModalVisible,
    diffResult,
    compareBackup1,
    compareBackup2,
    leftScrollRef,
    rightScrollRef,
    openDiffModal,
    closeDiffModal,
    handleLeftScroll,
    handleRightScroll,
  } = useBackupDiff();

  const {
    backupModalVisible,
    restoreModalVisible,
    contentDrawerVisible,
    versionListDrawerVisible,
    selectedBackup,
    selectedRestoreBackup,
    backupContent,
    restoreTaskId,
    openBackupModal,
    closeBackupModal,
    openRestoreModal,
    closeRestoreModal,
    openContentDrawer,
    closeContentDrawer,
    openVersionListDrawer,
    closeVersionListDrawer,
    handleBackup,
    handleRestore: confirmRestore,
  } = useBackupModals({
    onLoad: () => {
      reloadRef.current?.();
      loadStatistics();
    },
  });

  // Phase 93: 恢复任务轮询（restoreTaskId 非空时 3s 轮询，终态自动停止）
  const { task: restoreTask } = useRestoreTask(restoreTaskId);

  // 任务 result JSON 解析（sentLines/totalLines/hashMatched 展示用）
  const restoreResult = useMemo(() => {
    if (!restoreTask?.resultJson) {
      return null;
    }
    try {
      return JSON.parse(restoreTask.resultJson) as {
        sentLines?: number;
        totalLines?: number;
        hashMatched?: boolean;
        failedLine?: string;
      };
    } catch {
      return null;
    }
  }, [restoreTask?.resultJson]);

  // 打开备份弹窗
  const handleOpenBackupModal = () => {
    loadDevices();
    openBackupModal();
  };

  // 打开还原选择模态框
  const handleOpenRestoreModal = (group: DeviceBackupGroup) => {
    setSelectedDeviceGroup(group);
    openRestoreModal(group.latestBackup);
  };

  // 确认还原（68.1-T4 §1.9 非删除破坏性变体：恢复配置）
  const handleConfirmRestore = () => {
    if (!selectedRestoreBackup) {
      message.warning("请选择要还原的版本");
      return;
    }
    confirmDelete({
      title: "确认恢复配置",
      target: `把设备 ${selectedDeviceGroup?.deviceName} 的当前配置恢复到版本 ${selectedRestoreBackup.version}`,
      actionLabel: "恢复",
      actionContext: "恢复配置",
      contentText: "将{target}，操作不可恢复，是否继续？",
      successText: "已恢复配置",
      onDelete: async () => {
        await confirmRestore();
      },
    });
  };

  // 打开版本列表抽屉
  const handleOpenVersionList = (group: DeviceBackupGroup) => {
    setSelectedDeviceGroup(group);
    openVersionListDrawer(group);
  };

  // 打开对比对话框
  const handleOpenDiffModal = (
    group: DeviceBackupGroup,
    backup1?: ConfigBackup,
    backup2?: ConfigBackup
  ) => {
    setSelectedDeviceGroup(group);
    if (group.backups.length < 2) {
      message.info("该设备只有一份备份，无法对比");
      return;
    }
    // 默认：左边是次新版，右边是最新版
    const b1 = backup1 || group.backups[1];
    const b2 = backup2 || group.backups[0];
    openDiffModal(b1, b2);
  };

  // 查看配置内容
  const handleViewContent = async (backup: ConfigBackup) => {
    openContentDrawer(backup);
  };

  // 恢复配置（版本列表抽屉入口——68.1-T4 §1.9 变体收口）
  const handleRestore = (backup: ConfigBackup) => {
    confirmDelete({
      title: "确认恢复配置",
      target: `把设备当前配置恢复到版本 ${backup.version}`,
      actionLabel: "恢复",
      actionContext: "恢复配置",
      contentText: "将{target}，操作不可恢复，是否继续？",
      successText: "已恢复配置",
      onDelete: async () => {
        openRestoreModal(backup);
        // v129-recheck WR-01: 显式传参——本闭包捕获的 confirmRestore 引用
        // selectedRestoreBackup 旧值（null），无参调用会被 null 守卫静默短路
        await confirmRestore(backup);
      },
    });
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

  // 渲染差异行（原 :276-379 逐字保留）
  const renderDiffLine = (line: DiffLine, side: "left" | "right", index: number) => {
    const lineNum = line.lineNum !== undefined ? String(line.lineNum).padStart(4, " ") : "    ";
    const isRemoved = line.type === "removed";
    const isAdded = line.type === "added";
    const isSame = line.type === "same";
    const isEmpty = line.type === "empty";

    // 计算背景色
    // Phase 71 (W4)：裸 hex 收敛主题 token（移出 P65 豁免块后 P68 hex 契约生效）
    let backgroundColor = "transparent";
    if (isRemoved) backgroundColor = "var(--theme-error-bg)";
    else if (isAdded) backgroundColor = "var(--theme-success-bg)";
    else if (isSame) {
      const lines = side === "left" ? diffResult?.leftLines : diffResult?.rightLines;
      const prevLine = index > 0 ? lines![index - 1] : null;
      const nextLine = index < lines!.length - 1 ? lines![index + 1] : null;
      if (
        prevLine?.type === "removed" ||
        prevLine?.type === "added" ||
        nextLine?.type === "removed" ||
        nextLine?.type === "added"
      ) {
        backgroundColor = "var(--theme-bg-secondary)";
      }
    }

    const textColor = isRemoved
      ? "var(--theme-error)"
      : isAdded
        ? "var(--theme-success)"
        : "var(--theme-text-tertiary)";
    const contentColor = isRemoved
      ? "var(--theme-error)"
      : isAdded
        ? "var(--theme-success)"
        : "var(--theme-text-primary)";

    if (isEmpty) {
      return (
        <div key={`${side}-${index}`} style={{ display: "flex", backgroundColor }}>
          <div
            style={{
              width: 50,
              padding: "0 8px",
              textAlign: "right",
              color: "var(--theme-text-tertiary)",
              userSelect: "none",
              flexShrink: 0,
              borderRight: "1px solid var(--theme-border-primary)",
            }}
          >
            {lineNum}
          </div>
          <div
            style={{
              padding: "0 12px",
              whiteSpace: "pre",
              overflowX: "auto",
              flex: 1,
            }}
          />
        </div>
      );
    }

    return (
      <div key={`${side}-${index}`} style={{ display: "flex", backgroundColor }}>
        <div
          style={{
            width: 50,
            padding: "0 8px",
            textAlign: "right",
            color: textColor,
            userSelect: "none",
            flexShrink: 0,
            borderRight: "1px solid var(--theme-border-primary)",
          }}
        >
          {lineNum}
        </div>
        <div
          style={{
            padding: "0 12px",
            whiteSpace: "pre",
            overflowX: "auto",
            color: contentColor,
            flex: 1,
          }}
        >
          {line.content}
        </div>
      </div>
    );
  };

  // 设备列表表格列（按设备分组；操作列三动作保持原位不迁 rowActions）
  const deviceColumns: ColumnsType<DeviceBackupGroup> = [
    {
      title: "设备名称",
      dataIndex: "deviceName",
      key: "deviceName",
      width: 200,
      sorter: true,
    },
    {
      title: "IP地址",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 140,
    },
    {
      title: "备份总数",
      dataIndex: "backupCount",
      key: "backupCount",
      width: 100,
      render: (count: number) => (
        <Tag color="blue" icon={<DatabaseOutlined />}>
          {count} 份
        </Tag>
      ),
    },
    {
      title: "自动/手动",
      key: "backupTypeCount",
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tag color="blue" icon={<ClockCircleOutlined />}>
            {record.autoCount}
          </Tag>
          <Tag color="green" icon={<SaveOutlined />}>
            {record.manualCount}
          </Tag>
        </Space>
      ),
    },
    {
      title: "最新版本",
      dataIndex: "latestBackup",
      key: "latestVersion",
      width: 80,
      render: (backup: ConfigBackup) => <Tag>{backup.version}</Tag>,
    },
    {
      title: "最新备份时间",
      dataIndex: "latestBackup",
      key: "latestBackupTime",
      width: 150,
      render: (backup: ConfigBackup) => formatDateTime(backup.createdAt),
    },
    {
      title: "最新备份原因",
      dataIndex: "latestBackup",
      key: "latestReason",
      width: 150,
      ellipsis: true,
      render: (backup: ConfigBackup) => backup.changeReason || "-",
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 100,
      render: (_, record) => {
        const actions = [
          {
            key: "restore",
            label: "还原",
            icon: <DownloadOutlined />,
            onClick: () => handleOpenRestoreModal(record),
            // Phase 86 W3：还原 = networkBackup.restore（**不是** 删除类码）
            permission: PERMISSION_CODES.networkBackup.restore,
          },
          {
            // Phase 86 W3：读动作（版本列表只读，不落库）→ 不加门控
            key: "versions",
            label: "版本列表",
            icon: <HistoryOutlined />,
            onClick: () => handleOpenVersionList(record),
          },
          {
            key: "diff",
            label: "版本对比",
            icon: <DiffOutlined />,
            onClick: () => handleOpenDiffModal(record),
            permission: PERMISSION_CODES.networkBackup.diff,
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <div>
      <PageTable<DeviceBackupGroup>
        request={request}
        columns={deviceColumns}
        rowKey={(record) => record.deviceId}
        scroll={{ x: 1200 }}
        searchFields={[
          { type: "input", name: "deviceName", label: "设备名称", placeholder: "请输入设备名称" },
          {
            type: "select",
            name: "backupType",
            label: "备份类型",
            options: BACKUP_TYPE_SEARCH_OPTIONS,
            placeholder: "请选择备份类型",
          },
        ]}
        sorterMetas={sorterMetas}
        toolbar={
          <Space>
            {/* Phase 86 W3：创建备份 —— 条件渲染（networkBackup.add） */}
            {permAdd.visible && (
              <Button
                type="primary"
                icon={<CloudUploadOutlined />}
                onClick={handleOpenBackupModal}
                aria-label="创建备份"
                title={permAdd.hint ?? "创建备份"}
                disabled={permAdd.disabled}
              >
                创建备份
              </Button>
            )}
            {/* Phase 86 W3：批量导出 —— 门控点 = `<NetworkExport>` 组件元素本身
                （「导出」按钮在 `NetworkExport.tsx:120` 组件内部，页面上无独立触发按钮）。
                ready 无 networkBackup.export → 整个组件（含其内部「导出」按钮）不在 DOM。 */}
            {permExport.visible && (
              <span title={permExport.hint} aria-disabled={permExport.disabled || undefined}>
                <NetworkExport
                  entityType="backups"
                  entityName="配置备份"
                  filters={Object.fromEntries(
                    Object.entries(
                      (searchFormRef.current?.getFieldsValue() as Record<string, unknown>) ?? {}
                    ).filter(([, v]) => v !== undefined && v !== null && v !== "")
                  )}
                  current={exportPage.current}
                  pageSize={exportPage.pageSize}
                />
              </span>
            )}
          </Space>
        }
      >
        {(ctx) => {
          // PageTable loadData / searchForm 转发（onLoad 刷新 + NetworkExport filters）
          reloadRef.current = ctx.loadData;
          searchFormRef.current = ctx.searchForm;
          return (
            <>
              {/* 统计卡片（原 :552-584 逐字） */}
              <Row gutter={16} style={{ marginBottom: 16 }}>
                <Col span={6}>
                  <Card>
                    <Statistic
                      title="备份总数"
                      value={statistics.total}
                      prefix={<DatabaseOutlined />}
                    />
                  </Card>
                </Col>
                <Col span={6}>
                  <Card>
                    <Statistic
                      title="自动备份"
                      value={statistics.auto}
                      styles={{ content: { color: "var(--theme-info)" } }}
                      prefix={<ClockCircleOutlined />}
                    />
                  </Card>
                </Col>
                <Col span={6}>
                  <Card>
                    <Statistic
                      title="手动备份"
                      value={statistics.manual}
                      styles={{ content: { color: "var(--theme-success)" } }}
                      prefix={<SaveOutlined />}
                    />
                  </Card>
                </Col>
                <Col span={6}>
                  <Card>
                    <Statistic
                      title="涉及设备"
                      value={statistics.devices}
                      prefix={<HistoryOutlined />}
                    />
                  </Card>
                </Col>
              </Row>

              {/* 批量导出 Modal（原误嵌 renderDiffLine 空行分支内，归位到根级） */}
              <BatchExportModal
                visible={batchModalVisible}
                onConfirm={handleBatchExport}
                onCancel={() => setBatchModalVisible(false)}
                loading={batchExporting}
              />

              {/* 还原选择 Modal / 版本列表抽屉 / 差异对比 Modal / 配置内容抽屉 */}
              <RestoreModals
                selectedDeviceGroup={selectedDeviceGroup}
                setSelectedDeviceGroup={setSelectedDeviceGroup}
                restoreModalVisible={restoreModalVisible}
                closeRestoreModal={closeRestoreModal}
                restoreTaskId={restoreTaskId}
                restoreTask={restoreTask}
                restoreResult={restoreResult}
                selectedRestoreBackup={selectedRestoreBackup}
                openRestoreModal={openRestoreModal}
                handleConfirmRestore={handleConfirmRestore}
                versionListDrawerVisible={versionListDrawerVisible}
                closeVersionListDrawer={closeVersionListDrawer}
                handleOpenDiffModal={handleOpenDiffModal}
                handleRestore={handleRestore}
                handleViewContent={handleViewContent}
                diffModalVisible={diffModalVisible}
                closeDiffModal={closeDiffModal}
                compareBackup1={compareBackup1}
                compareBackup2={compareBackup2}
                openDiffModal={openDiffModal}
                diffResult={diffResult}
                renderDiffLine={renderDiffLine}
                leftScrollRef={leftScrollRef}
                rightScrollRef={rightScrollRef}
                handleLeftScroll={handleLeftScroll}
                handleRightScroll={handleRightScroll}
                selectedBackup={selectedBackup}
                contentDrawerVisible={contentDrawerVisible}
                closeContentDrawer={closeContentDrawer}
                backupContent={backupContent}
              />

              {/* 创建备份弹窗 → FormModal（okText 显式「确定」，dirtyCheck 默认；
                  68.1-T5 minor#7：FormModal 已补 maskClosable / keyboard 透传位（对外 prop 名冻结），此处按迁移前
                  原 Modal 取值显式声明 false/false（Phase 64 FIX-02 已收紧口径：不遮罩关闭、
                  不 ESC 关闭；与 BaseEditModal 默认一致，不放宽） */}
              <FormModal
                title="创建配置备份"
                open={backupModalVisible}
                form={backupForm}
                okText="确定"
                width={600}
                maskClosable={false}
                keyboard={false}
                onFinish={async () => {
                  // handleBackup 内部 validateFields + post + 成功后 closeBackupModal；
                  // 失败路径由 hook 内 handleApiError("创建备份") 承接，弹窗保持打开
                  await handleBackup(backupForm);
                }}
                onCancel={() => closeBackupModal(backupForm)}
              >
                <Form.Item
                  name="deviceIds"
                  label="选择设备"
                  rules={[{ required: true, message: "请选择设备" }]}
                >
                  {/* 形态分类器①/②⑦：children 写法改写 options；mode="multiple" 显式 showSearch。
                      原 optionFilterProp="children" 在 options 形态下失效，改由 AppSelect 契约的
                      optionFilterProp="label" 承接（options 均带 label） */}
                  <AppSelect
                    mode="multiple"
                    placeholder="请选择要备份的设备"
                    showSearch
                    onSearch={() => {}}
                    options={devices.map((device) => ({
                      value: device.id,
                      label: `${device.deviceName} (${device.ipAddress})`,
                    }))}
                  />
                </Form.Item>

                <Form.Item name="changeReason" label="变更原因">
                  <AppTextArea rows={3} placeholder="请输入变更原因" />
                </Form.Item>
              </FormModal>
            </>
          );
        }}
      </PageTable>
    </div>
  );
};

export default ConfigBackupPage;
