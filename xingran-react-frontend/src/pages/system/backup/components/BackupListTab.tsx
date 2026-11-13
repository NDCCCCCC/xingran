/**
 * 备份列表面板：StatCards（总数/成功/占用空间）+ 工具栏 + Table。
 *
 * 范式对齐 config/index.tsx：useTableManager + usePagination + StatCards + Card 工具栏 +
 * Table + ActionButtons（permission 内建门控）。立即备份 / 导入恢复 / 下载 / 删除均走
 * dbBackupApi；破坏性操作统一用 handleApiError / handleSuccess 出口。
 */
import { useCallback, useEffect, useMemo, useState, type FC } from "react";
import { App, Alert, Button, Card, Space, Table, Tag, Tooltip, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  CloudUploadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
// P66 lint 规则要求新列表页用 PageTable；但 261110-tbq CONTRACT.md 第 5 节明确要求
// 本页沿用 config/index.tsx 的 useTableManager + usePagination 范式，故显式行级豁免。
// eslint-disable-next-line no-restricted-imports
import { usePagination } from "@/hooks/usePagination";
// eslint-disable-next-line no-restricted-imports
import { useTableManager } from "@/hooks/useTableManager";
import ActionButtons from "@/components/shared/ActionButtons";
import { confirmDelete, StatCards, useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { dbBackupApi } from "@/lib/dbBackupApi";
import type { Capabilities } from "@/types/backup";
import { formatDateTime } from "@/utils/datetime";
import { formatFileSize } from "@/utils/tableHelpers";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import type { BackupRecord, BackupStatistics } from "@/types/backup";
import type { AsyncTask } from "../types";
import {
  DOWNLOAD_SENSITIVE_CONFIRM_TEXT,
  RESTORE_RELOGIN_NOTICE,
  SOURCE_TEXT,
  SQLITE_OFFLINE_COMMAND,
  SQLITE_OFFLINE_NOTICE,
  STATUS_TAG_COLOR,
  STATUS_TEXT,
  STORAGE_TEXT,
} from "../types";
import RestoreModal from "./RestoreModal";

interface BackupListTabProps {
  /** 触发父页面异步任务轮询（立即备份 / 导入恢复） */
  onStartTask: (task: AsyncTask) => void;
  /** 父页面任务终态后递增，驱动列表与统计刷新 */
  refreshToken: number;
  /** 是否有任务在跑（禁用重复触发） */
  taskRunning: boolean;
}

const EMPTY_STATISTICS: BackupStatistics = { total: 0, success: 0, usedSpace: 0 };

const BackupListTab: FC<BackupListTabProps> = ({ onStartTask, refreshToken, taskRunning }) => {
  const { message } = App.useApp();
  // 工具栏按钮门控：统一走 useActionPermission（三态 ready / loading / error），不自造 hasPermission
  const permGate = useActionPermission();
  const permExport = permGate.decide(PERMISSION_CODES.systemBackup.export);
  const permRestore = permGate.decide(PERMISSION_CODES.systemBackup.restore);
  const [statistics, setStatistics] = useState<BackupStatistics>(EMPTY_STATISTICS);
  // 能力预检：pg_dump/pg_restore 不可用时提前给出安装/配置指引，避免点了「立即备份」
  // 才看到底层 "executable file not found in $PATH"。
  const [capabilities, setCapabilities] = useState<Capabilities | null>(null);
  const [restoreOpen, setRestoreOpen] = useState(false);

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading,
    data: records,
    loadData,
    handleRefresh: handleTableRefresh,
    handleTableChange,
  } = useTableManager<BackupRecord>(
    async (params) => {
      const result = await dbBackupApi.list({
        current: Number(params.current) || 1,
        pageSize: Number(params.pageSize) || 10,
      });
      return { list: result.list ?? [], total: result.total ?? 0 };
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

  /** 统计卡（独立端点，不受分页钳制） */
  const loadStatistics = useCallback(async () => {
    try {
      setStatistics(await dbBackupApi.statistics());
    } catch (error) {
      handleApiError(error, "加载备份统计", false);
    }
  }, []);

  /** 能力预检（独立调用，不进表格 effect；仅首屏一次） */
  useEffect(() => {
    let cancelled = false;
    void dbBackupApi
      .capabilities()
      .then((cap) => {
        if (!cancelled) setCapabilities(cap);
      })
      .catch(() => {
        /* 预检失败不阻断页面，按钮仍可点击并由后端给出诊断 */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // 首屏 + 分页参数变化 + 任务终态刷新（对齐 config/index.tsx 的表格效果）。
  // loadData（useTableManager 内部 useCallback([])）与 loadStatistics（useCallback([])）
  // 均为稳定引用；分页值取 useMemo 稳定的 paginationProps 字段，effect 不会反复重建。
  useEffect(() => {
    if (paginationProps.current && paginationProps.pageSize) {
      void loadData();
      void loadStatistics();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paginationProps.current, paginationProps.pageSize, refreshToken]);

  /** 刷新（工具栏）：列表 + 统计一起刷新 */
  const handleRefresh = useCallback(() => {
    handleTableRefresh();
    void loadStatistics();
  }, [handleTableRefresh, loadStatistics]);

  /** 立即备份：拿 taskId 交给父页面轮询 */
  const handleExport = useCallback(async () => {
    try {
      const { taskId } = await dbBackupApi.export();
      if (!taskId) {
        message.error("未获取到任务 ID，无法跟踪备份进度");
        return;
      }
      message.info("备份任务已提交，正在后台执行");
      onStartTask({ taskId, kind: "export" });
    } catch (error) {
      handleApiError(error, "触发立即备份");
    }
  }, [message, onStartTask]);

  /** 下载备份 zip：先二次提示「备份包含全库敏感数据」，再走 downloadFile 唯一权威封装 */
  const handleDownload = useCallback((record: BackupRecord) => {
    confirmDelete({
      title: "确认下载备份",
      target: `备份文件（${record.fileName}）`,
      actionLabel: "确认下载",
      actionContext: "下载备份文件",
      contentText: DOWNLOAD_SENSITIVE_CONFIRM_TEXT,
      successText: "已开始下载，请妥善保管全库敏感数据",
      onDelete: async () => {
        await dbBackupApi.download(record.id, record.fileName);
      },
    });
  }, []);

  /** 删除备份记录（confirmDelete 二次确认，权限内建） */
  const handleDelete = useCallback(
    (record: BackupRecord) => {
      confirmDelete({
        target: `备份文件（${record.fileName}）`,
        onDelete: async () => {
          await dbBackupApi.remove(record.id);
          handleSuccess("删除备份");
          handleTableRefresh();
          void loadStatistics();
        },
      });
    },
    [handleTableRefresh, loadStatistics]
  );

  const columns: ColumnsType<BackupRecord> = useMemo(
    () => [
      {
        title: "文件名",
        dataIndex: "fileName",
        key: "fileName",
        width: 240,
        ellipsis: true,
      },
      {
        title: "数据库类型",
        dataIndex: "dbType",
        key: "dbType",
        width: 110,
        align: "center",
      },
      {
        title: "来源",
        dataIndex: "source",
        key: "source",
        width: 110,
        align: "center",
        render: (source: string) => SOURCE_TEXT[source] ?? source ?? "-",
      },
      {
        title: "存储后端",
        dataIndex: "storageType",
        key: "storageType",
        width: 100,
        align: "center",
        render: (storage: string) => STORAGE_TEXT[storage] ?? storage ?? "-",
      },
      {
        title: "大小",
        dataIndex: "fileSize",
        key: "fileSize",
        width: 110,
        align: "right",
        render: (size: number) => formatFileSize(Number(size) || 0),
      },
      {
        title: "状态",
        dataIndex: "status",
        key: "status",
        width: 100,
        align: "center",
        render: (status: string) => (
          <Tag color={STATUS_TAG_COLOR[status] ?? "default"}>
            {STATUS_TEXT[status] ?? status ?? "-"}
          </Tag>
        ),
      },
      {
        title: "创建时间",
        dataIndex: "createdAt",
        key: "createdAt",
        width: 180,
        render: (date: string) => formatDateTime(date),
      },
      {
        title: "sha256",
        dataIndex: "sha256",
        key: "sha256",
        width: 120,
        render: (sha: string) =>
          sha ? (
            <Tooltip title={sha}>
              <span>{sha.slice(0, 8)}…</span>
            </Tooltip>
          ) : (
            "-"
          ),
      },
      {
        title: "操作",
        key: "action",
        width: 150,
        fixed: "right",
        render: (_, record) => (
          <ActionButtons
            actions={[
              {
                key: "download",
                label: "下载",
                icon: <DownloadOutlined />,
                permission: PERMISSION_CODES.systemBackup.export,
                onClick: () => void handleDownload(record),
              },
              {
                key: "delete",
                label: "删除",
                icon: <DeleteOutlined />,
                // task-3 G4 整改：删除按钮改挂独立 system:backup:remove（此前复用 export）。
                // 需与后端 migration_246 F 型播种配套（持 system:backup 父菜单的角色自动获得）。
                danger: true,
                permission: PERMISSION_CODES.systemBackup.remove,
                onClick: () => handleDelete(record),
              },
            ]}
          />
        ),
      },
    ],
    [handleDownload, handleDelete]
  );

  const statItems = useMemo(
    () => [
      { label: "备份总数", value: statistics.total ?? 0, tone: "default" as const },
      { label: "成功备份", value: statistics.success ?? 0, tone: "green" as const },
      {
        label: "占用空间",
        value: formatFileSize(Number(statistics.usedSpace) || 0),
        tone: "gold" as const,
      },
    ],
    [statistics]
  );

  const pgDumpMissing =
    capabilities !== null && capabilities.dbType !== "sqlite" && !capabilities.pgDumpAvailable;
  // 依据 capabilities() 的 dbType 常驻显示 SQLite 离线恢复说明（不依赖失败信息）。
  const isSQLite = capabilities?.dbType === "sqlite";

  return (
    <div>
      {isSQLite && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          title="SQLite 整库恢复需停服离线执行"
          description={
            <div>
              <div>{SQLITE_OFFLINE_NOTICE}</div>
              <div style={{ marginTop: 8 }}>离线命令入口：</div>
              <Typography.Text code copyable={{ text: SQLITE_OFFLINE_COMMAND }}>
                {SQLITE_OFFLINE_COMMAND}
              </Typography.Text>
            </div>
          }
        />
      )}
      {capabilities && !capabilities.pgDumpAvailable && capabilities.dbType !== "sqlite" && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          title="PostgreSQL 备份工具未就绪"
          description={capabilities.hint}
        />
      )}
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        title="备份包含全库敏感数据"
        description="备份包含全库敏感数据（用户口令散列、个人信息、业务数据等），下载即等同于导出整库明文，请按最高敏感级妥善保管。"
      />
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 16 }}
        title="恢复后需重新登录"
        description={RESTORE_RELOGIN_NOTICE}
      />
      <StatCards items={statItems} />

      <Card style={{ marginBottom: 16 }}>
        <Space wrap>
          {permExport.visible && (
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              disabled={taskRunning || permExport.disabled || pgDumpMissing}
              title={
                pgDumpMissing
                  ? "pg_dump 不可用，请先安装或配置可执行文件（见上方提示）"
                  : permExport.hint
              }
              onClick={() => void handleExport()}
            >
              立即备份
            </Button>
          )}
          {permRestore.visible && (
            <Button
              icon={<CloudUploadOutlined />}
              disabled={taskRunning || permRestore.disabled}
              title={permRestore.hint}
              onClick={() => setRestoreOpen(true)}
            >
              导入恢复
            </Button>
          )}
          <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
            刷新
          </Button>
        </Space>
      </Card>

      <Card>
        <Table<BackupRecord>
          rowKey="id"
          columns={columns}
          dataSource={records}
          loading={loading}
          pagination={paginationProps}
          onChange={handleTableChange}
          scroll={{ x: "max-content" }}
        />
      </Card>

      <RestoreModal
        open={restoreOpen}
        taskRunning={taskRunning}
        capabilities={capabilities}
        onClose={() => setRestoreOpen(false)}
        onStartTask={onStartTask}
      />
    </div>
  );
};

export default BackupListTab;
