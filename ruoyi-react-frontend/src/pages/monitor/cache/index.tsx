/**
 * Cache 缓存管理页面
 *
 * Phase 67-T4: 迁移 66 契约组件（PageTable/FormModal）。
 * PR 差异记录（详见 67-SUMMARY）：
 * - 双状态源（caches+setTotal+usePagination+useServerSort 手工管线）→ PageTable 单实例 request 管线
 * - 搜索卡（受控 Input/Select + 搜索/重置/刷新按钮）→ searchFields 3 字段（B 类：按钮文案统一内建 搜索/重置/刷新）
 * - 操作缓存 Modal → FormModal（okText="确定"；68.1-T5 minor#7：FormModal 已补
 *   maskClosable / keyboard 透传位（FormModal 冻结契约 prop），此处按迁移前原 Modal 取值显式声明 false/false；
 *   Form 布局 vertical → FormModal 内建 horizontal labelCol4/wrapperCol20）
 * - 单删 Popconfirm → 命令式 confirmDelete（68.1-T4：Modal 强确认，触发按钮原样保留）
 * - 清空全部缓存 → confirmDelete 非删除变体（actionLabel="清空"，68.1-T4 §1.9）
 * - 导出按钮文案「导出当前筛选结果」：导出携带表格当前筛选（缓存键/缓存类型）同参，
 *   层级（level）因后端 /monitor/cache/export 仅支持 Key/Type（CacheExportParams）而在
 *   前端导出前二次过滤，保证「导出结果 = 当前筛选结果」（68.1-T5 minor#6）
 * - Tabs（列表/缓存统计）移除：统计面板内容为「统计信息」Modal 的真子集，信息零损失（记 PR）
 * - 30s monitor 轮询保留（ref 承接）；列表失败重试由 PageTable 内建 ErrorAlertWithRetry 承接
 */
import { useState, useEffect, useCallback, useMemo, useRef, type FC } from "react";
import { Button, App, Space, Modal, Form, InputNumber, Tag } from "antd";
// Phase 71 (W1)：裸 Input/Select/TextArea → 契约组件（P65 allowClear / P65 Select 契约 / P70 TextArea 契约）
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import {
  PlusOutlined,
  SettingOutlined,
  ExportOutlined,
  DeleteOutlined,
  EyeOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { post, get } from "@/lib/api";
import { PageTable, confirmDelete } from "@/components/shared";
import type { PageTableRenderContext } from "@/components/shared";
import { FormModal } from "@/components/shared/FormModal";
import { useFormModalForm } from "@/components/shared/hooks/useFormModalForm";
import { handleApiError } from "@/utils/errorHandler";
import { ErrorAlertWithRetry } from "@/components/shared";
import { createSorterMeta } from "@/utils/tableHelpers";

// 导入提取的文件
import type { CacheInfo, CacheMonitor } from "./types";
import { TYPE_OPTIONS, OPERATION_OPTIONS, LEVEL_OPTIONS, LEVEL_TAG_CONFIG } from "./constants";
import {
  formatMemorySize,
  formatTTL,
  formatDateTime as cacheFormatDateTime,
  exportCacheAsJson,
} from "./utils";
import MonitorOverview from "./components/MonitorOverview";
import CacheStatsModal from "./components/CacheStatsModal";

// ==================== 表格列定义 ====================

interface CacheTableColumnsProps {
  handleViewDetail: (key: string) => void;
}

function getCacheTableColumns(props: CacheTableColumnsProps): ColumnsType<CacheInfo> {
  const { handleViewDetail: _handleViewDetail } = props;

  return [
    {
      title: "缓存键",
      dataIndex: "key",
      key: "key",
      width: 300,
      ellipsis: true,
      sorter: true,
      render: (text: string) => (
        <code className="bg-gray-100 px-2 py-1 rounded text-xs">{text}</code>
      ),
    },
    {
      title: "缓存类型",
      dataIndex: "type",
      key: "type",
      width: 100,
      sorter: true,
      render: (type: string) => <Tag color="blue">{type}</Tag>,
    },
    {
      title: "层级",
      dataIndex: "location",
      key: "location",
      width: 120,
      sorter: true,
      render: (location: string) => {
        const config = LEVEL_TAG_CONFIG[location];
        if (config) {
          return <Tag color={config.color}>{config.label}</Tag>;
        }
        return <Tag>{location}</Tag>;
      },
    },
    {
      title: "TTL",
      dataIndex: "ttl",
      key: "ttl",
      width: 120,
      sorter: true,
      render: (ttl: number) => <Tag color={ttl > 0 ? "green" : "default"}>{formatTTL(ttl)}</Tag>,
    },
    {
      title: "大小",
      dataIndex: "size",
      key: "size",
      width: 100,
      sorter: true,
      render: (size: number) => formatMemorySize(size),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      render: (value: string) => cacheFormatDateTime(value),
    },
    {
      title: "更新时间",
      dataIndex: "updatedAt",
      key: "updatedAt",
      width: 180,
      sorter: true,
      render: (value: string) => cacheFormatDateTime(value),
    },
  ];
}

// ==================== 主组件 ====================

const CacheManager: FC = () => {
  const { message } = App.useApp();
  const [monitor, setMonitor] = useState<CacheMonitor | null>(null);
  const [monitorError, setMonitorError] = useState<unknown>(null);
  const [operateModalVisible, setOperateModalVisible] = useState(false);
  const [statsModalVisible, setStatsModalVisible] = useState(false);

  // 操作缓存弹窗（FormModal 契约）
  const operateForm = useFormModalForm();
  // PageTable ctx.loadData 转发（操作/清空后刷新列表；devices 页同款 state 承接写法）
  const [reloadRef] = useState<{ current: (() => void) | null }>({ current: null });
  // 68.1-T5 minor#6：最近一次列表查询参数（列表 request 入参），导出须与之同参
  const lastListQueryRef = useRef<{ key?: string; type?: string; level?: string }>({});

  // ==================== 监控数据 ====================

  const fetchMonitor = useCallback(async () => {
    try {
      const result = await post<CacheMonitor>("/monitor/cache/monitor", {});
      setMonitor(result.data || null);
      setMonitorError(null);
    } catch (error) {
      setMonitorError(error);
    }
  }, []);

  // ==================== 初始化（monitor 30s 轮询） ====================

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- fetchMonitor 异步 setMonitor 为既有挂载行为
    fetchMonitor();
    // 设置定时刷新（每30秒）；fetchMonitor 为稳定引用（deps=[]），无需 ref 承接
    const interval = setInterval(() => {
      fetchMonitor();
    }, 30000);

    return () => clearInterval(interval);
  }, [fetchMonitor]);

  // ==================== 缓存操作 ====================

  // handlers 用普通函数（devices 页风格）：内部读 reloadRef.current，避免 useCallback deps 报错
  const handleOperate = async (rawValues: Record<string, unknown>) => {
    const values = rawValues as { key: string; operation: string; value?: string; ttl?: number };
    await post<unknown>("/monitor/cache/operate", {
      ...values,
      ttl: values.ttl || undefined,
    });
    message.success("操作成功");
    setOperateModalVisible(false);
    reloadRef.current?.();
  };

  // 单条删除（68.1-T4 附加条目：原 Popconfirm → 命令式 confirmDelete，触发按钮不变）
  const handleDelete = (key: string) => {
    confirmDelete({
      target: `缓存键 ${key}`,
      onDelete: async () => {
        await post<unknown>("/monitor/cache/batch", { keys: [key], operation: "del" });
        reloadRef.current?.();
      },
    });
  };

  const handleClearAll = () => {
    // 68.1-T4 §1.9 非删除破坏性变体（清空缓存）
    confirmDelete({
      title: "确认清空",
      target: "全部缓存数据",
      actionLabel: "清空",
      actionContext: "清空缓存",
      contentText: "将{target}，操作不可恢复，是否继续？",
      successText: "已清空缓存",
      onDelete: async () => {
        await post("/monitor/cache/clear", {});
        reloadRef.current?.();
        fetchMonitor();
      },
    });
  };

  const handleExport = useCallback(async () => {
    try {
      // 68.1-T5 minor#6：导出携带当前筛选条件（与列表同参），不再导出全量。
      // 后端 /monitor/cache/export 的 CacheExportParams 仅支持 Key / Type 两个过滤位
      // （handler 只取 req.Key / req.Type；service 用 key LIKE + type = 过滤），不支持 level
      // → 层级在导出前由前端对 location 二次过滤，使导出结果与表格当前筛选一致。
      const query = lastListQueryRef.current;
      const result = await post<{ data?: CacheInfo[]; count?: number }>("/monitor/cache/export", {
        key: query.key || undefined,
        type: query.type || undefined,
      });
      const rows = result.data?.data || [];
      const level = query.level && query.level !== "all" ? query.level : "";
      // 层级口径与列表一致：l1/l2 对应 CacheInfo.location；"both" 仅在「全部」下出现
      exportCacheAsJson(level ? rows.filter((row) => row.location === level) : rows);
      message.success("导出成功");
    } catch (error) {
      handleApiError(error, "导出缓存");
    }
  }, [message]);

  const handleViewDetail = useCallback(async (key: string) => {
    try {
      const result = await get<CacheInfo>(`/monitor/cache/${key}`);

      Modal.info({
        title: "缓存详情",
        width: 600,
        content: (
          <div>
            <p>
              <strong>键名：</strong>
              {result.data?.key}
            </p>
            <p>
              <strong>值：</strong>
            </p>
            <pre className="bg-gray-100 p-3 rounded">
              {JSON.stringify(JSON.parse(result.data?.value || "{}"), null, 2)}
            </pre>
            <p>
              <strong>类型：</strong>
              {result.data?.type}
            </p>
            <p>
              <strong>大小：</strong>
              {(result.data?.size || 0).toLocaleString()} 字节
            </p>
            <p>
              <strong>TTL：</strong>
              {formatTTL(result.data?.ttl || -1)}
            </p>
            <p>
              <strong>创建时间：</strong>
              {result.data?.createdAt ? cacheFormatDateTime(result.data.createdAt) : "-"}
            </p>
          </div>
        ),
      });
    } catch (error) {
      handleApiError(error, "获取缓存详情");
    }
  }, []);

  // ==================== 搜索字段 / 排序 ====================

  const searchFields = useMemo(
    () => [
      { type: "input" as const, name: "key", label: "缓存键", placeholder: "缓存键" },
      {
        type: "select" as const,
        name: "type",
        label: "缓存类型",
        options: [...TYPE_OPTIONS],
        placeholder: "缓存类型",
      },
      {
        type: "select" as const,
        name: "level",
        label: "缓存层级",
        options: [...LEVEL_OPTIONS],
        placeholder: "缓存层级",
      },
    ],
    []
  );

  const cacheSorterMetas = useMemo(
    () => [
      createSorterMeta<CacheInfo>("key"),
      createSorterMeta<CacheInfo>("type"),
      createSorterMeta<CacheInfo>("location"),
      createSorterMeta<CacheInfo>("ttl", "number"),
      createSorterMeta<CacheInfo>("size", "number"),
      createSorterMeta<CacheInfo>("createdAt", "date"),
      createSorterMeta<CacheInfo>("updatedAt", "date"),
    ],
    []
  );

  // ==================== 行操作（原操作列整体迁 rowActions；删除走 confirmDelete 强确认） ====================

  // 行操作（普通函数组装；handleDelete/handleViewDetail 稳定闭包语义）
  const buildRowActions = (record: CacheInfo) => [
      {
        key: "view",
        label: "查看详情",
        icon: <EyeOutlined />,
        onClick: () => void handleViewDetail(record.key),
      },
      {
        key: "delete",
        label: "删除",
        icon: <DeleteOutlined />,
        danger: true,
        render: () => (
          <Button
            aria-label="删除缓存键"
            title="删除缓存键"
            type="link"
            icon={<DeleteOutlined />}
            style={{ color: "var(--theme-error)" }}
            size="small"
            onClick={() => handleDelete(record.key)}
          >
            删除
          </Button>
        ),
      },
    ];

  const columns = useMemo(() => getCacheTableColumns({ handleViewDetail }), [handleViewDetail]);

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold mb-4">缓存管理</h1>
      </div>

      {/* FIX-04b: 监控数据加载失败可重试错误条（列表错误由 PageTable 内建承接） */}
      {!!monitorError && <ErrorAlertWithRetry error={monitorError} onRetry={() => void fetchMonitor()} />}

      <PageTable<CacheInfo>
        request={async (params) => {
          // 68.1-T5 minor#6：留存列表查询入参，导出复用（保证导出与当前筛选同参）
          lastListQueryRef.current = params as { key?: string; type?: string; level?: string };
          // level 默认 "all"（SearchField 无 initialValue 支持；filterEmpty 剔除空值后兜底）
          const result = await post<{ list?: CacheInfo[]; total?: number }>("/monitor/cache/list", {
            ...params,
            level: (params as { level?: string }).level || "all",
          });
          return { list: result.data?.list || [], total: result.data?.total || 0 };
        }}
        columns={columns}
        rowKey={(r) => r.key}
        searchFields={searchFields}
        sorterMetas={cacheSorterMetas}
        defaultSort={{ orderByColumn: "createdAt", isAsc: false }}
        rowActions={buildRowActions}
        toolbar={
          <Space wrap>
            <Button
              aria-label="缓存操作"
              title="缓存操作"
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setOperateModalVisible(true)}
            >
              操作缓存
            </Button>
            <Button
              icon={<SettingOutlined />}
              onClick={() => setStatsModalVisible(true)}
              aria-label="查看统计"
              title="查看统计"
            >
              统计信息
            </Button>
            <Button
              icon={<ExportOutlined />}
              onClick={() => void handleExport()}
              aria-label="导出当前筛选结果"
              title="按当前筛选条件（缓存键 / 缓存类型 / 缓存层级）导出 JSON"
            >
              导出当前筛选结果
            </Button>
            <Button
              aria-label="清空全部缓存"
              title="清空全部缓存"
              icon={<DeleteOutlined />}
              style={{ color: "var(--theme-error)" }}
              onClick={handleClearAll}
            >
              清空
            </Button>
          </Space>
        }
      >
        {(ctx: PageTableRenderContext<CacheInfo>) => {
          reloadRef.current = ctx.loadData;
          return <MonitorOverview monitor={monitor} />;
        }}
      </PageTable>

      {/* 操作缓存模态框（FormModal 契约）
          68.1-T5 minor#7：按迁移前原 Modal 取值显式声明 maskClosable / keyboard = false */}
      <FormModal
        title="缓存操作"
        open={operateModalVisible}
        onCancel={() => setOperateModalVisible(false)}
        form={operateForm}
        okText="确定"
        width={500}
        maskClosable={false}
        keyboard={false}
        onFinish={handleOperate}
      >
        <Form.Item label="缓存键" name="key" rules={[{ required: true, message: "请输入缓存键" }]}>
          <AppInput allowClear placeholder="请输入缓存键" />
        </Form.Item>
        <Form.Item
          label="操作类型"
          name="operation"
          rules={[{ required: true, message: "请选择操作类型" }]}
        >
          <AppSelect allowClear options={OPERATION_OPTIONS} onSearch={() => {}} />
        </Form.Item>
        <Form.Item
          label="缓存值"
          name="value"
          dependencies={["operation"]}
          rules={[
            {
              required: true,
              message: "请输入缓存值",
            },
            ({ getFieldValue }) => ({
              validator: (_, value) => {
                if (getFieldValue("operation") === "set" && !value) {
                  return Promise.reject(new Error("请输入缓存值"));
                }
                return Promise.resolve();
              },
            }),
          ]}
        >
          <AppTextArea placeholder="请输入缓存值" rows={4} />
        </Form.Item>
        <Form.Item label="过期时间(秒)" name="ttl" dependencies={["operation"]}>
          <InputNumber placeholder="留空表示永不过期" min={1} style={{ width: "100%" }} />
        </Form.Item>
      </FormModal>

      {/* 统计信息模态框（纯搬迁 components/CacheStatsModal.tsx） */}
      <CacheStatsModal
        open={statsModalVisible}
        onClose={() => setStatsModalVisible(false)}
        monitor={monitor}
      />
    </div>
  );
};

export default CacheManager;
