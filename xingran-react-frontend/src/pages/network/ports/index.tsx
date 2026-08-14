import type { FC } from "react";
import {
  Table,
  Button,
  Space,
  Form,
  Input,
  Select,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  App,
  Empty,
  Skeleton,
  Typography,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  CloudSyncOutlined,
  ApiOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  ArrowLeftOutlined,
} from "@ant-design/icons";
import type { DevicePortStatus, NetworkDevice } from "@/types";
import type { PortWriteAction } from "@/types/network";
import { post, get } from "@/lib/api";
import { batchExport, getPortMACBundle, type PortMACBundle } from "@/lib/api/networkApi";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useEffect, useState, useMemo } from "react";
import { useTableManager } from "@/hooks/useTableManager";
import { withErrorHandling } from "@/utils/errorHandler";
import { formatDateTime } from "@/utils/datetime";
import { usePagination } from "@/hooks/usePagination";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal, ErrorAlertWithRetry } from "@/components/shared";
import ActionButtons, { type ActionButton } from "@/components/shared/ActionButtons";
import { SettingOutlined } from "@ant-design/icons";
import { createSorterMeta } from "@/utils/tableHelpers";
import { useMenuStore } from "@/store/menuStore";
import { PortWriteModal } from "@/components/network/port-write/PortWriteModal";
import { BulkWriteDrawer } from "@/components/network/port-write/BulkWriteDrawer";
import { SetAccessVlanModal } from "@/components/network/port-write/SetAccessVlanModal";
import { PortBindingModal } from "@/components/network/port-write/PortBindingModal";
import { EVENT_LABEL, EVENT_TAG_COLOR, type MACEventType } from "@/components/network/macEventMeta";

const { Option } = Select;

// ==================== PortMACPanel (quick 260712-vpj, D-05/D-06) ====================
//
// 单端口 MAC 展示子组件,集成到 ports 页 expandedRowRender 内。
// - bundle 由父组件的 macBundleCache(Record<portId, PortMACBundle>)提供
// - bundle === undefined 表示未展开过,显示 Skeleton
// - current.length > 0 展示当前 MAC 区(每条 Tag + VLAN Tag)
// - current 空且端口 down 时 fallback 展示最近一条历史 MAC(单条 + 事件 Tag + 时间)
// - current 空且端口 up 时展示 Empty
//
// 不调 message.error(getPortMACBundle 已把 error 收集到 bundle.error, 此处仅渲染)。

interface PortMACPanelProps {
  portId: string;
  deviceId: string;
  interfaceName: string;
  adminStatus: string;
  operStatus: string;
  bundle?: PortMACBundle;
  load: () => void;
}

const PortMACPanel: FC<PortMACPanelProps> = ({ bundle, adminStatus, operStatus, load }) => {
  if (bundle === undefined) {
    return <Skeleton active paragraph={{ rows: 2 }} />;
  }

  if (bundle.error && bundle.current.length === 0 && !bundle.recentHistory) {
    return <ErrorAlertWithRetry error={bundle.error} onRetry={load} />;
  }

  const isDown = adminStatus === "down" || operStatus === "down";

  if (bundle.current.length > 0) {
    return (
      <Space direction="vertical" size={8} style={{ width: "100%" }}>
        <Typography.Text strong>当前 MAC 地址({bundle.current.length})</Typography.Text>
        <Space size={[8, 8]} wrap>
          {bundle.current.map((m) => (
            <Space key={m.id} size={4}>
              <Tag color="blue">{m.macAddress}</Tag>
              {m.vlanId != null && <Tag>VLAN {m.vlanId}</Tag>}
            </Space>
          ))}
        </Space>
      </Space>
    );
  }

  if (isDown && bundle.recentHistory) {
    const recent = bundle.recentHistory;
    const eventType = (recent.eventType ?? "appeared") as MACEventType;
    return (
      <Space direction="vertical" size={8} style={{ width: "100%" }}>
        <Typography.Text strong>最近一次出现过的 MAC</Typography.Text>
        <Space size={8} wrap>
          <Tag color="blue">{recent.macAddress}</Tag>
          <Tag color={EVENT_TAG_COLOR[eventType]}>{EVENT_LABEL[eventType]}</Tag>
          <Typography.Text type="secondary">{formatDateTime(recent.firstSeen)}</Typography.Text>
          {recent.vlanId != null && <Tag>VLAN {recent.vlanId}</Tag>}
        </Space>
      </Space>
    );
  }

  return <Empty description="该端口暂无 MAC 数据" />;
};

const PortStatusPage: FC = () => {
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const deviceIdFromUrl = searchParams.get("deviceId") || "";
  const isFromDevice = !!deviceIdFromUrl;

  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [collecting, setCollecting] = useState(false);

  // D-09: 权限源 menu store (非 auth store, ROADMAP #4 笔误纠正) — 后端 RequirePermissions 是真相源
  const menuPermissions = useMenuStore((s) => s.permissions);
  const hasPermission = (perm: string) => menuPermissions.includes(perm);
  const canWrite = hasPermission("network:port:write");

  // Phase 53 W4: 单端口写 Modal + 批量 Drawer state
  const [writeModalOpen, setWriteModalOpen] = useState(false);
  const [writeModalAction, setWriteModalAction] = useState<PortWriteAction>("shutdown");
  const [writeModalRecord, setWriteModalRecord] = useState<DevicePortStatus | null>(null);
  const [bulkWriteDrawerOpen, setBulkWriteDrawerOpen] = useState(false);
  // D-07: batchInProgress 由 BulkWriteDrawer onExecutingChange 上抛, 禁刷新+采集 (LANDMINE #4 同类竞态)
  const [batchInProgress, setBatchInProgress] = useState(false);

  // Phase 56 W4: set_access_vlan + port_binding 单端口 Modal state (v1.20.1)
  const [vlanModalOpen, setVlanModalOpen] = useState(false);
  const [vlanModalRecord, setVlanModalRecord] = useState<DevicePortStatus | null>(null);
  const [bindModalOpen, setBindModalOpen] = useState(false);
  const [bindModalRecord, setBindModalRecord] = useState<DevicePortStatus | null>(null);

  // 单端口操作入口 (ActionButtons onClick → openWriteModal(action, record))
  const openWriteModal = (action: PortWriteAction, record: DevicePortStatus) => {
    setWriteModalAction(action);
    setWriteModalRecord(record);
    setWriteModalOpen(true);
  };

  // Phase 56 W4: 2 个新 action 的单端口 Modal opener (独立 Modal 因字段差异大)
  const openVlanModal = (record: DevicePortStatus): void => {
    setVlanModalRecord(record);
    setVlanModalOpen(true);
  };
  const openBindModal = (record: DevicePortStatus): void => {
    setBindModalRecord(record);
    setBindModalOpen(true);
  };

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    up: 0,
    down: 0,
    dot1xEnabled: 0,
  });

  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // quick 260712-vpj D-05: 展开行 MAC bundle 懒加载缓存
  // - 首次展开才 fetch,折叠再展开命中缓存不重发请求
  // - 普通 Record 即可(端口列表分页最多 10-100 条,无需 LRU)
  const [macBundleCache, setMacBundleCache] = useState<Record<string, PortMACBundle>>({});
  const [expandedRowKeys, setExpandedRowKeys] = useState<string[]>([]);

  // 按需加载某端口的 MAC bundle(命中 cache 直接返回;否则 fetch 并写入 cache)
  const loadPortMACBundle = async (
    portId: string,
    deviceId: string,
    interfaceName: string
  ): Promise<void> => {
    if (macBundleCache[portId]) return;
    const bundle = await getPortMACBundle(deviceId, interfaceName);
    setMacBundleCache((prev) => ({ ...prev, [portId]: bundle }));
  };

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 对应后端 portStatusAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<DevicePortStatus>("interfaceName"),
      createSorterMeta<DevicePortStatus>("adminStatus"),
      createSorterMeta<DevicePortStatus>("operStatus"),
      createSorterMeta<DevicePortStatus>("vlan", "number"),
      createSorterMeta<DevicePortStatus>("collectedAt", "date"),
    ],
    []
  );

  const {
    loading,
    data: portStatus,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    loadData: loadPortStatus,
    getColumnSortOrder,
    handleTableChange,
    handleSearch,
    handleReset,
    handleRefresh,
    orderByColumn: _orderByColumn,
    isAsc: _isAsc,
  } = useTableManager<DevicePortStatus>(
    async (params) => {
      const formValues = searchForm.getFieldsValue();
      const values = formValues as Record<string, unknown>;
      const result = (await post("/network/ports/list", {
        current: params.current ?? paginationProps.current ?? 1,
        pageSize: params.pageSize ?? paginationProps.pageSize ?? 10,
        ...values,
        // 如果从设备页面跳转过来，强制使用 URL 中的 deviceId
        ...(isFromDevice ? { deviceId: deviceIdFromUrl } : {}),
        // useTableManager 内部通过 params 已带 current/pageSize;排序由其内部 ref
        // 写入 params.orderByColumn/isAsc(随 sorterMetas 启用后),这里显式展开保证不被覆盖
        ...(params.orderByColumn
          ? { orderByColumn: params.orderByColumn, isAsc: params.isAsc }
          : {}),
      })) as { data: { list: DevicePortStatus[]; total: number } };
      return result.data;
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

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = (await get("/network/ports/statistics")) as {
        data?: {
          totalRecords?: number;
          upPortsCount?: number;
          downPortsCount?: number;
          dot1xEnabledCount?: number;
        };
      };
      const stats = result.data || {};
      setStatistics({
        total: stats.totalRecords || 0,
        up: stats.upPortsCount || 0,
        down: stats.downPortsCount || 0,
        dot1xEnabled: stats.dot1xEnabledCount || 0,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  };

  // 加载设备列表
  const loadDevices = async () => {
    try {
      const result = (await post("/network/devices/list", {
        current: 1,
        pageSize: 50,
      })) as { data?: { list: NetworkDevice[] } };
      setDevices(result.data?.list || []);
    } catch (error) {
      console.error("加载设备列表失败:", error);
    }
  };

  useEffect(() => {
    // 如果从设备页面跳转过来，初始化表单的 deviceId
    if (isFromDevice && deviceIdFromUrl) {
      searchForm.setFieldsValue({ deviceId: deviceIdFromUrl });
    }
    Promise.all([loadStatistics(), loadDevices()]);
    loadPortStatus();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 采集所有设备端口状态
  const handleCollectAll = async () => {
    setCollecting(true);
    await withErrorHandling(
      async () => {
        await post("/network/ports/collect-all", {});
        return "采集任务已创建";
      },
      {
        onSuccess: () => {
          loadPortStatus();
          loadStatistics();
        },
        onError: () => {
          setCollecting(false);
        },
      }
    );
    setCollecting(false);
  };

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的数据");
      return;
    }
    await withErrorHandling(
      async () => {
        await post("/network/ports/batch-delete", { ids: selectedRowKeys });
        return "批量删除成功";
      },
      {
        onSuccess: () => {
          setSelectedRowKeys([]);
          loadPortStatus();
          loadStatistics();
        },
      }
    );
  };

  const handleBatchExport = async (entityTypes: string[]) => {
    setBatchExporting(true);
    try {
      const filename = await batchExport(entityTypes, {}); // 可以根据需要添加筛选条件
      message.success(`批量导出成功，文件: ${filename}`);
      setBatchModalVisible(false);
    } catch (error) {
      // 55-01 IN-01: 项目 TS 严格风格 — 用 instanceof Error 收窄, 避免非 Error reject
      // 时显示 "undefined"
      const msg = error instanceof Error ? error.message : String(error);
      message.error(`批量导出失败：${msg}`);
    } finally {
      setBatchExporting(false);
    }
  };

  // 表格列
  const columns: ColumnsType<DevicePortStatus> = [
    {
      title: "接口名称",
      dataIndex: "interfaceName",
      key: "interfaceName",
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder("interfaceName"),
    },
    {
      title: "描述",
      dataIndex: "description",
      key: "description",
      width: 180,
      ellipsis: true,
      render: (desc: string) => desc || "-",
    },
    {
      title: "端口状态",
      dataIndex: "operStatus",
      key: "operStatus",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("operStatus"),
      render: (status: string) => {
        const color = status === "up" ? "success" : "default";
        return <Tag color={color}>{status?.toUpperCase() || "-"}</Tag>;
      },
    },
    {
      title: "管理员激活",
      dataIndex: "adminStatus",
      key: "adminStatus",
      width: 100,
      sorter: true,
      sortOrder: getColumnSortOrder("adminStatus"),
      render: (status: string) => {
        const color = status === "up" ? "processing" : "default";
        return <Tag color={color}>{status?.toUpperCase() || "-"}</Tag>;
      },
    },
    {
      title: "VLAN",
      dataIndex: "vlan",
      key: "vlan",
      width: 80,
      sorter: true,
      sortOrder: getColumnSortOrder("vlan"),
      render: (vlan?: number) => vlan ?? "-",
    },
    {
      title: "类型",
      dataIndex: "portType",
      key: "portType",
      width: 80,
      render: (type?: string) => type || "-",
    },
    {
      title: "802.1X",
      key: "dot1x",
      width: 100,
      render: (_, record) => (
        <Space size="small">
          <Tag color={record.dot1xEnabled ? "blue" : "default"}>
            {record.dot1xEnabled ? "启用" : "未启用"}
          </Tag>
          {record.dot1xEnabled && (
            <Tag color={record.dot1xPortStatus === "authorized" ? "success" : "warning"}>
              {record.dot1xPortStatus}
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: "端口安全",
      key: "portSecurity",
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tag color={record.portSecurityEnabled ? "purple" : "default"}>
            {record.portSecurityEnabled ? "启用" : "未启用"}
          </Tag>
          {record.portSecurityEnabled && record.portSecurityMode && (
            <Tag>{record.portSecurityMode}</Tag>
          )}
        </Space>
      ),
    },
    {
      title: "采集时间",
      dataIndex: "collectedAt",
      key: "collectedAt",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder("collectedAt"),
      render: (date: string) => (date ? formatDateTime(date) : "-"),
    },
    // Phase 53 W4: 操作列 (D-01, canWrite gating — D-09/ROADMAP #4 笔误纠正)
    // 5 action 全调 openWriteModal, 共用 PortWriteModal 切 action prop
    ...(canWrite
      ? [
          {
            title: "操作",
            key: "portWriteAction",
            fixed: "right" as const,
            width: 100,
            render: (_: unknown, record: DevicePortStatus) => {
              const actions: ActionButton[] = [
                {
                  key: "shutdown",
                  label: "关闭端口",
                  onClick: () => openWriteModal("shutdown", record),
                },
                {
                  key: "undo_shutdown",
                  label: "启用端口",
                  onClick: () => openWriteModal("undo_shutdown", record),
                },
                {
                  key: "description",
                  label: "修改描述",
                  onClick: () => openWriteModal("description", record),
                },
                {
                  key: "dot1x_enable",
                  label: "启用 802.1X",
                  onClick: () => openWriteModal("dot1x_enable", record),
                },
                {
                  key: "dot1x_disable",
                  label: "停用 802.1X",
                  onClick: () => openWriteModal("dot1x_disable", record),
                },
                // v1.20.1: 2 个新 action (5 → 7), 独立 Modal 因字段差异大 (参 SetAccessVlanModal / PortBindingModal)
                {
                  key: "set_access_vlan",
                  label: "修改 access VLAN",
                  onClick: () => openVlanModal(record),
                },
                { key: "port_binding", label: "端口绑定", onClick: () => openBindModal(record) },
              ];
              return <ActionButtons actions={actions} />;
            },
          },
        ]
      : []),
  ];

  return (
    <div>
      {/* 返回按钮（仅当从设备页面跳转时显示） */}
      {isFromDevice && (
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate("/network/devices")}
          style={{ marginBottom: 16 }}
        >
          返回设备管理
        </Button>
      )}

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic title="端口总数" value={statistics.total} prefix={<ApiOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="UP端口"
              value={statistics.up}
              styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="DOWN端口"
              value={statistics.down}
              styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
              prefix={<WarningOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="802.1X启用"
              value={statistics.dot1xEnabled}
              styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
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
            <Form.Item name="deviceId" label="设备">
              <Select
                placeholder="请选择设备"
                allowClear
                className="user-form-input"
                style={{ width: 200 }}
                showSearch
                optionFilterProp="label"
                onSearch={() => {}}
              >
                {devices.map((device) => (
                  <Option key={device.id} value={device.id} label={device.deviceName}>
                    {device.deviceName}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="interfaceName" label="接口">
              <Input
                placeholder="请输入接口名称"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="adminStatus" label="管理状态">
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
              >
                <Option value="up">UP</Option>
                <Option value="down">DOWN</Option>
              </Select>
            </Form.Item>
            <Form.Item name="operStatus" label="操作状态">
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
              >
                <Option value="up">UP</Option>
                <Option value="down">DOWN</Option>
              </Select>
            </Form.Item>
            <Form.Item name="dot1xEnabled" label="802.1X状态">
              <Select
                placeholder="请选择"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
              >
                <Option value={true}>已启用</Option>
                <Option value={false}>未启用</Option>
              </Select>
            </Form.Item>
            <Form.Item name="portSecurityEnabled" label="端口安全">
              <Select
                placeholder="请选择"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
              >
                <Option value={true}>已启用</Option>
                <Option value={false}>未启用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  查询
                </Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>
                  重置
                </Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={handleRefresh}
                  disabled={batchInProgress}
                >
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <NetworkExport
              entityType="ports"
              entityName="端口状态"
              filters={(() => {
                const values = searchForm.getFieldsValue() as Record<string, unknown>;
                const filtered: Record<string, unknown> = {};
                Object.keys(values).forEach((key) => {
                  const value = values[key];
                  if (value !== undefined && value !== null && value !== "") {
                    filtered[key] = value;
                  }
                });
                return filtered;
              })()}
              current={paginationProps.current}
              pageSize={paginationProps.pageSize}
            />
            <Button
              icon={<CloudSyncOutlined />}
              onClick={handleCollectAll}
              loading={collecting}
              disabled={batchInProgress}
            >
              采集所有设备
            </Button>
            <Button
              icon={<DeleteOutlined />}
              onClick={handleBatchDelete}
              disabled={selectedRowKeys.length === 0}
              style={{ color: "var(--theme-error, #ff4d4f)" }}
            >
              批量删除 ({selectedRowKeys.length})
            </Button>
            {/* Phase 53 W4 D-04: 批量配置入口 — 与批量删除 UX 一致, 额外加 !canWrite 防 */}
            <Button
              icon={<SettingOutlined />}
              onClick={() => setBulkWriteDrawerOpen(true)}
              disabled={selectedRowKeys.length === 0 || !canWrite}
              type="primary"
            >
              批量配置 ({selectedRowKeys.length})
            </Button>
          </Space>
          {/* 批量导出 Modal */}

          <BatchExportModal
            visible={batchModalVisible}

            onConfirm={handleBatchExport}

            onCancel={() => setBatchModalVisible(false)}

            loading={batchExporting}
          />
        </div>
      </Card>

      {/* 端口状态表格 */}
      <Card>
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={columns}
          dataSource={portStatus}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1400 }}
          pagination={paginationProps}
          onChange={handleTableChange}
          expandable={{
            expandedRowKeys,
            onExpand: (expanded, record) => {
              if (expanded) {
                setExpandedRowKeys((prev) =>
                  prev.includes(record.id) ? prev : [...prev, record.id]
                );
                // 命中 cache 不重发(quick 260712-vpj D-05)
                void loadPortMACBundle(record.id, record.deviceId, record.interfaceName);
              } else {
                setExpandedRowKeys((prev) => prev.filter((k) => k !== record.id));
              }
            },
            expandedRowRender: (record) => (
              <div style={{ padding: 16, background: "#fafafa" }}>
                <p>
                  <strong>接口详情:</strong>
                </p>
                <p>管理状态: {record.adminStatus?.toUpperCase() || "-"}</p>
                <p>操作状态: {record.operStatus?.toUpperCase() || "-"}</p>
                {record.dot1xEnabled && <p>802.1X状态: {record.dot1xPortStatus}</p>}
                {record.portSecurityEnabled && (
                  <>
                    <p>安全模式: {record.portSecurityMode || "未设置"}</p>
                    <p>最大MAC数: {record.maxMACCount || "未限制"}</p>
                    <p>当前MAC数: {record.currentMACCount || 0}</p>
                  </>
                )}
                {/* quick 260712-vpj D-05/D-06: 端口 MAC 展示(懒加载,折叠再展开命中 cache) */}
                <div style={{ marginTop: 12 }}>
                  <PortMACPanel
                    portId={record.id}
                    deviceId={record.deviceId}
                    interfaceName={record.interfaceName}
                    adminStatus={record.adminStatus}
                    operStatus={record.operStatus}
                    bundle={macBundleCache[record.id]}
                    load={() => {
                      void loadPortMACBundle(record.id, record.deviceId, record.interfaceName);
                    }}
                  />
                </div>
              </div>
            ),
          }}
        />
      </Card>

      {/* Phase 53 W4: 单端口写 Modal + 批量 Drawer (D-01 / D-04) */}
      <PortWriteModal
        open={writeModalOpen}
        action={writeModalAction}
        portRecord={writeModalRecord}
        onClose={() => setWriteModalOpen(false)}
        onSuccess={() => {
          loadPortStatus();
          loadStatistics();
        }}
      />
      {/* Phase 56 W4: v1.20.1 2 个新单端口 Modal (set_access_vlan + port_binding) */}
      <SetAccessVlanModal
        open={vlanModalOpen}
        portRecord={vlanModalRecord}
        onClose={() => setVlanModalOpen(false)}
        onSuccess={() => {
          loadPortStatus();
          loadStatistics();
        }}
      />
      <PortBindingModal
        open={bindModalOpen}
        portRecord={bindModalRecord}
        onClose={() => setBindModalOpen(false)}
        onSuccess={() => {
          loadPortStatus();
          loadStatistics();
        }}
      />
      <BulkWriteDrawer
        open={bulkWriteDrawerOpen}
        selectedPorts={portStatus.filter((p) => selectedRowKeys.includes(p.id))}
        onClose={() => setBulkWriteDrawerOpen(false)}
        onSuccess={() => {
          loadPortStatus();
          loadStatistics();
        }}
        onExecutingChange={setBatchInProgress}
      />
    </div>
  );
};

export default PortStatusPage;
