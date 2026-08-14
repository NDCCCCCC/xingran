/**
 * MAC 历史查询主列表页(Phase 14-01)
 *
 * 核心交互:
 * - 时间筛选:6 个预设按钮(近 1h/24h/7d/30d/90d/自定义)+ RangePicker 互斥同步(D-07)
 * - 8 列全列展示 + useColumnConfig 列配置抽屉(D-08)
 * - AntD Table virtual 虚拟滚动 + useTableQuery placeholderData 保持滚动位置(D-06/D-12)
 * - 操作列"查看事件"展开行 → MACEventsTimeline 组件(D-11)
 * - URL 参数注入(D-17):?deviceId=xxx&portName=yyy&startTime=...&endTime=...
 * - 移动端 Grid.useBreakpoint() 自动切换 List 卡片视图(D-05)
 *
 * 锁定端点:POST /network/history/list(D-01)
 * 默认时间范围:近 7d(D-07)
 * 颜色与图标:与 macEventMeta.ts 单一事实源对齐(D-10)
 *
 * 已知 TODO(由后续 plan 接管):
 * - 14-04:工具栏新增 2 个互斥导出按钮
 * - 14-05:替换内联 Alert/Empty 为 EmptyStateWithAction / ErrorAlertWithRetry
 * - 14-05b:移动端卡片视图最终样式收口
 */

import React, { useState, useEffect, useMemo, useCallback } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  Table, Button, Space, Form, Input, InputNumber, Select,
  Tag, Card, DatePicker, List, Grid, Tooltip, Skeleton,
  Typography, App,
} from "antd";
import type { ColumnsType } from "antd/es/table";

type RangePickerOnChange = React.ComponentProps<typeof DatePicker.RangePicker>["onChange"];
import {
  SearchOutlined, ReloadOutlined, AppstoreOutlined, TableOutlined,
  CopyOutlined, EyeOutlined, DownloadOutlined,
} from "@ant-design/icons";
import { useSearchParams, useNavigate } from "react-router-dom";
import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import { useTableQuery } from "@/hooks/useTableQuery";
import { useColumnConfig, type ColumnConfig } from "@/hooks/useColumnConfig";
import { queryMACHistory, exportMACHistory } from "@/lib/api/networkApi";
import type {
  MACHistoryRecord,
  MACHistoryQueryParams,
} from "@/lib/api/networkApi";
import { MACEventsTimeline } from "@/components/network";
import {
  EmptyStateWithAction,
  ErrorAlertWithRetry,
} from "@/components/shared";
import { useMenuStore } from "@/store/menuStore";
import { EVENT_LABEL, EVENT_TAG_COLOR } from "@/components/network/macEventMeta";

const { RangePicker } = DatePicker;
const { useBreakpoint } = Grid;

// 预设时间范围(单位,数量) — D-07 锁定
type PresetKey = "1h" | "24h" | "7d" | "30d" | "90d" | "custom";
interface Preset {
  key: PresetKey;
  label: string;
  amount: number;
  unit: "hour" | "day";
}
const PRESETS: Preset[] = [
  { key: "1h", label: "近 1h", amount: 1, unit: "hour" },
  { key: "24h", label: "近 24h", amount: 24, unit: "hour" },
  { key: "7d", label: "近 7d", amount: 7, unit: "day" },
  { key: "30d", label: "近 30d", amount: 30, unit: "day" },
  { key: "90d", label: "近 90d", amount: 90, unit: "day" },
  { key: "custom", label: "自定义", amount: 0, unit: "day" },
];

// 8 列默认配置(D-08 锁定)
const defaultHistoryColumns: ColumnConfig[] = [
  { key: "time", label: "时间", visible: true, order: 1, width: 160 },
  { key: "mac", label: "MAC 地址", visible: true, order: 2, width: 160 },
  { key: "device", label: "设备", visible: true, order: 3, width: 180 },
  { key: "port", label: "端口", visible: true, order: 4, width: 130 },
  { key: "eventType", label: "事件类型", visible: true, order: 5, width: 110 },
  { key: "vlan", label: "VLAN", visible: true, order: 6, width: 80 },
  { key: "status", label: "状态", visible: true, order: 7, width: 80 },
  { key: "action", label: "操作", visible: true, order: 8, width: 200 },
];

// MAC 规范化
const normalizeMACAddress = (mac: string): string => {
  const cleaned = mac.replace(/[^a-fA-F0-9]/g, "");
  if (cleaned.length !== 12) return "";
  return cleaned.match(/.{2}/g)?.join(":").toUpperCase() || "";
};

const MACHistoryPage: React.FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const breakpoint = useBreakpoint();
  const isMobile = !!breakpoint.xs;

  // 权限(从 menuStore 拉取)— 14-04 导出按钮可见性控制
  const menuPermissions = useMenuStore((s) => s.permissions);
  const hasPermission = (perm: string) => menuPermissions.includes(perm);
  const canExport = hasPermission("network:mac:export");

  // 状态
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const location = useLocation();
  const [activePreset, setActivePreset] = usePersistedStateController<PresetKey>({
    keyPrefix: location.pathname,
    keySuffix: "activePreset",
    defaultValue: "7d",
  });
  const [customRange, setCustomRange] = useState<[Dayjs, Dayjs] | null>(null);
  const [expandedRowKeys, setExpandedRowKeys] = useState<React.Key[]>([]);
  const [exporting, setExporting] = useState<boolean>(false);

  // 计算当前时间范围(ISO 字符串)用于查询
  const [timeRange, setTimeRange] = useState<{ startTime: string; endTime: string }>(
    () => {
      const preset = PRESETS.find((p) => p.key === "7d")!;
      return {
        startTime: dayjs().subtract(preset.amount, preset.unit).toISOString(),
        endTime: dayjs().toISOString(),
      };
    }
  );

  // URL 参数注入(D-17)
  useEffect(() => {
    const deviceId = searchParams.get("deviceId");
    const portName = searchParams.get("portName");
    const startTime = searchParams.get("startTime");
    const endTime = searchParams.get("endTime");
    const mac = searchParams.get("mac");

    const initial: Record<string, unknown> = {};
    if (deviceId) initial.deviceId = deviceId;
    if (portName) initial.interfaceName = portName;
    if (mac) initial.mac = normalizeMACAddress(mac) || mac;
    if (Object.keys(initial).length > 0) form.setFieldsValue(initial);

    if (startTime && endTime) {
      // 视为自定义
      setActivePreset("custom");
      setCustomRange([dayjs(startTime), dayjs(endTime)]);
      setTimeRange({ startTime, endTime });
    }
  }, [searchParams, form, setActivePreset]);

  // 切换预设
  const handlePresetClick = useCallback(
    (preset: Preset) => {
      if (preset.key === "custom") {
        setActivePreset("custom");
        setCustomRange(null);
        return;
      }
      setActivePreset(preset.key);
      setCustomRange(null);
      const start = dayjs().subtract(preset.amount, preset.unit).toISOString();
      const end = dayjs().toISOString();
      setTimeRange({ startTime: start, endTime: end });
      setCurrent(1);
    },
    [setActivePreset]
  );

  // 切换自定义 RangePicker
  const handleCustomRangeChange = useCallback<NonNullable<RangePickerOnChange>>(
    (values) => {
      const arr = values as [Dayjs, Dayjs] | null;
      setCustomRange(arr);
      if (arr && arr[0] && arr[1]) {
        setActivePreset("custom");
        setTimeRange({
          startTime: arr[0].toISOString(),
          endTime: arr[1].toISOString(),
        });
        setCurrent(1);
      }
    },
    [setActivePreset]
  );

  // 搜索参数(交给 useTableQuery)
  const filters = useMemo<Partial<MACHistoryQueryParams>>(
    () => ({
      ...timeRange,
    }),
    [timeRange]
  );

  // useTableQuery:列表分页查询(D-06)
  const {
    data: pageData,
    isLoading,
    isFetching,
    error,
    refetch,
  } = useTableQuery<MACHistoryRecord>({
    resource: "mac.history.list",
    current,
    pageSize,
    filters: filters as Record<string, unknown>,
    queryFn: (params) => queryMACHistory(params as unknown as MACHistoryQueryParams),
  });

  const list = pageData?.list ?? [];
  const total = pageData?.total ?? 0;

  // 列配置(D-08)
  const { visibleColumns, config: _config } = useColumnConfig({
    pageKey: "mac.history.list",
    defaultColumns: defaultHistoryColumns,
    enableCache: true,
  });

  // 8 列定义
  const allColumns: Record<string, ColumnsType<MACHistoryRecord>[number]> = useMemo(
    () => ({
      time: {
        title: "时间",
        key: "time",
        width: 160,
        render: (_: unknown, record: MACHistoryRecord) =>
          dayjs(record.firstSeen).format("YYYY-MM-DD HH:mm:ss"),
      },
      mac: {
        title: "MAC 地址",
        key: "mac",
        width: 160,
        render: (_: unknown, record: MACHistoryRecord) => (
          <Space size={4}>
            <Typography.Text code copyable={{ text: record.macAddress }}>
              {record.macAddress}
            </Typography.Text>
          </Space>
        ),
      },
      device: {
        title: "设备",
        key: "device",
        width: 180,
        ellipsis: true,
        render: (_: unknown, record: MACHistoryRecord) =>
          record.deviceNameSnapshot || record.deviceId,
      },
      port: {
        title: "端口",
        key: "port",
        width: 130,
        render: (_: unknown, record: MACHistoryRecord) => record.interfaceName,
      },
      eventType: {
        title: "事件类型",
        key: "eventType",
        width: 110,
        render: (_: unknown, record: MACHistoryRecord) => (
          <Tag color={EVENT_TAG_COLOR[record.eventType as keyof typeof EVENT_TAG_COLOR] ?? "default"}>
            {EVENT_LABEL[record.eventType as keyof typeof EVENT_LABEL] ?? record.eventType}
          </Tag>
        ),
      },
      vlan: {
        title: "VLAN",
        key: "vlan",
        width: 80,
        render: (_: unknown, record: MACHistoryRecord) =>
          record.vlanId ?? "-",
      },
      status: {
        title: "状态",
        key: "status",
        width: 80,
        render: (_: unknown, record: MACHistoryRecord) => (
          <Tag color={record.status === 0 ? "green" : "red"}>
            {record.status === 0 ? "正常" : "停用"}
          </Tag>
        ),
      },
      action: {
        title: "操作",
        key: "action",
        width: 200,
        fixed: "right" as const,
        render: (_: unknown, record: MACHistoryRecord) => (
          <Space size={4} wrap>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => {
                setExpandedRowKeys((prev) =>
                  prev.includes(record.id)
                    ? prev.filter((k) => k !== record.id)
                    : [...prev, record.id]
                );
              }}
            >
              {expandedRowKeys.includes(record.id) ? "收起事件" : "查看事件"}
            </Button>
          </Space>
        ),
      },
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- navigate from useNavigate is stable
    [expandedRowKeys, navigate]
  );

  // 列配置过滤(根据 visibleColumns 顺序)
  const tableColumns = useMemo<ColumnsType<MACHistoryRecord>>(() => {
    return visibleColumns
      .map((colCfg) => allColumns[colCfg.key])
      .filter((c): c is ColumnsType<MACHistoryRecord>[number] => Boolean(c));
  }, [visibleColumns, allColumns]);

  // 搜索
  const handleSearch = useCallback(() => {
    setCurrent(1);
  }, []);

  // 重置
  const handleReset = useCallback(() => {
    form.resetFields();
    setActivePreset("7d");
    setCustomRange(null);
    const preset = PRESETS.find((p) => p.key === "7d")!;
    setTimeRange({
      startTime: dayjs().subtract(preset.amount, preset.unit).toISOString(),
      endTime: dayjs().toISOString(),
    });
    setCurrent(1);
    setSearchParams(new URLSearchParams());
  }, [form, setSearchParams, setActivePreset]);

  // 14-04 导出按钮处理器 — exportScope = 'current' 时透传当前过滤;'all' 时只保留时间范围
  const handleExport = useCallback(
    async (exportScope: "current" | "all") => {
      if (exporting) return;
      setExporting(true);
      try {
        const formValues = (await form.validateFields()) as Record<string, unknown>;
        // 构造查询参数(必带 startTime / endTime + 分页,后端按 exportScope 决定范围)
        const baseParams: Record<string, unknown> = {
          startTime: timeRange.startTime,
          endTime: timeRange.endTime,
        };
        if (exportScope === "current") {
          // 透传当前过滤条件
          Object.keys(formValues).forEach((k) => {
            const v = formValues[k];
            if (v !== undefined && v !== null && v !== "") {
              baseParams[k] = v;
            }
          });
        }

        const { blob, filename } = await exportMACHistory(
          baseParams as unknown as MACHistoryQueryParams,
          exportScope
        );

        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        message.success(`已导出 ${exportScope === "current" ? "当前查询" : "全量"} 数据`);
      } catch (err) {
        if (err && typeof err === "object" && "errorFields" in err) {
          // form.validateFields 失败,直接返回
          return;
        }
        const msg = err instanceof Error ? err.message : String(err);
        message.error(`导出失败:${msg}`);
      } finally {
        setExporting(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [exporting, form, timeRange]
  );

  // 复制 MAC
  const copyMAC = async (mac: string) => {
    try {
      if (!navigator.clipboard) {
        throw new Error("Clipboard API unavailable");
      }
      await navigator.clipboard.writeText(mac);
      message.success(`已复制 ${mac}`);
    } catch (err) {
      message.error(err instanceof Error ? `复制失败:${err.message}` : "复制失败,请手动复制");
    }
  };

  // 桌面表格
  const renderTable = () => {
    // 首次加载(无数据 + isLoading)— 用 Skeleton 占位,不渲染 Table(D-19)
    if (isLoading && list.length === 0) {
      return (
        <Skeleton
          active
          paragraph={{ rows: 3 }}
          title={false}
          style={{ padding: "24px 12px" }}
        />
      );
    }
    return (
      <Table<MACHistoryRecord>
        rowKey="id"
        size="middle"
        virtual
        scroll={{ x: 1200, y: 600 }}
        dataSource={list}
        columns={tableColumns}
        // 后续分页/筛选不显示骨架,使用 isFetching 触发表头 Spin(D-19)
        loading={isFetching}
        pagination={{
          current,
          pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (page, size) => {
            setCurrent(page);
            setPageSize(size);
          },
        }}
        expandedRowKeys={expandedRowKeys}
        onExpand={(expanded, record) => {
          setExpandedRowKeys((prev) =>
            expanded
              ? [...prev, record.id]
              : prev.filter((k) => k !== record.id)
          );
        }}
        expandedRowRender={(record) => (
          <MACEventsTimeline
            mac={record.macAddress}
            startTime={timeRange.startTime}
            endTime={timeRange.endTime}
            deviceId={record.deviceId}
          />
        )}
        locale={{
          // total === 0 时使用 EmptyStateWithAction 引导去设备管理(D-18)
          emptyText: (
            <EmptyStateWithAction
              description="该范围内未采集到 MAC 记录,请检查设备是否启用了 MAC 采集/端口采集周期"
              actionLabel="前往设备管理"
              actionPath="/network/devices"
            />
          ),
        }}
      />
    );
  };

  // 移动卡片
  const renderCardList = () => {
    if (isLoading && list.length === 0) {
      return (
        <Skeleton
          active
          paragraph={{ rows: 3 }}
          title={false}
          style={{ padding: "24px 12px" }}
        />
      );
    }
    if (list.length === 0) {
      return (
        <EmptyStateWithAction
          description="该范围内未采集到 MAC 记录,请检查设备是否启用了 MAC 采集/端口采集周期"
          actionLabel="前往设备管理"
          actionPath="/network/devices"
        />
      );
    }
    return (
      <List
        dataSource={list}
        renderItem={(record) => (
          <List.Item
            key={record.id}
            style={{
              padding: "12px",
              background: "#fff",
              marginBottom: 8,
              borderRadius: 6,
              border: "1px solid #f0f0f0",
            }}
            actions={[
              <Button
                key="events"
                type="link"
                size="small"
                onClick={() => {
                  setExpandedRowKeys((prev) =>
                    prev.includes(record.id)
                      ? prev.filter((k) => k !== record.id)
                      : [...prev, record.id]
                  );
                }}
              >
                {expandedRowKeys.includes(record.id) ? "收起事件" : "查看事件"}
              </Button>,
            ]}
          >
            <List.Item.Meta
              title={
                <Space size={4} wrap>
                  <Typography.Text code>{record.macAddress}</Typography.Text>
                  <Tooltip title="复制 MAC">
                    <Button
                      type="text"
                      size="small"
                      icon={<CopyOutlined />}
                      onClick={() => { void copyMAC(record.macAddress); }}
                    />
                  </Tooltip>
                  <Tag color={EVENT_TAG_COLOR[record.eventType as keyof typeof EVENT_TAG_COLOR] ?? "default"}>
                    {EVENT_LABEL[record.eventType as keyof typeof EVENT_LABEL] ?? record.eventType}
                  </Tag>
                  <Tag color={record.status === 0 ? "green" : "red"}>
                    {record.status === 0 ? "正常" : "停用"}
                  </Tag>
                </Space>
              }
              description={
                <div>
                  <div>设备: {record.deviceNameSnapshot || record.deviceId}</div>
                  <div>端口: {record.interfaceName}</div>
                  <div>VLAN: {record.vlanId ?? "-"}</div>
                  <div>时间: {dayjs(record.firstSeen).format("YYYY-MM-DD HH:mm:ss")}</div>
                </div>
              }
            />
            {expandedRowKeys.includes(record.id) && (
              <div style={{ marginTop: 12 }}>
                <MACEventsTimeline
                  mac={record.macAddress}
                  startTime={timeRange.startTime}
                  endTime={timeRange.endTime}
                  deviceId={record.deviceId}
                />
              </div>
            )}
          </List.Item>
        )}
      />
    );
  };

  return (
    <div style={{ padding: isMobile ? 12 : 24 }}>
      <Card
        title="MAC 历史查询"
        bordered={false}
        style={{ marginBottom: 16 }}
        extra={
          !isMobile && (
            <Space>
              {PRESETS.map((p) => (
                <Button
                  key={p.key}
                  type={activePreset === p.key ? "primary" : "default"}
                  size="small"
                  onClick={() => handlePresetClick(p)}
                >
                  {p.label}
                </Button>
              ))}
              {activePreset === "custom" && (
                <RangePicker
                  showTime
                  format="YYYY-MM-DD HH:mm"
                  value={customRange}
                  onChange={handleCustomRangeChange}
                  placeholder={["开始时间", "结束时间"]}
                />
              )}
            </Space>
          )
        }
      >
        {/* 移动端时间预设独立一行 */}
        {isMobile && (
          <div style={{ marginBottom: 12 }}>
            <Space wrap>
              {PRESETS.map((p) => (
                <Button
                  key={p.key}
                  type={activePreset === p.key ? "primary" : "default"}
                  size="small"
                  onClick={() => handlePresetClick(p)}
                >
                  {p.label}
                </Button>
              ))}
            </Space>
            {activePreset === "custom" && (
              <div style={{ marginTop: 8 }}>
                <RangePicker
                  showTime
                  format="YYYY-MM-DD HH:mm"
                  value={customRange}
                  onChange={handleCustomRangeChange}
                  placeholder={["开始时间", "结束时间"]}
                />
              </div>
            )}
          </div>
        )}

        <Form
          form={form}
          layout={isMobile ? "vertical" : "inline"}
          onFinish={handleSearch}
        >
          <Form.Item name="mac" label="MAC 地址">
            <Input
              placeholder="AA:BB:CC:DD:EE:FF"
              style={{ width: 180 }}
              allowClear
            />
          </Form.Item>
          <Form.Item name="deviceId" label="设备 ID">
            <Input placeholder="设备 UUID" style={{ width: 220 }} allowClear />
          </Form.Item>
          <Form.Item name="interfaceName" label="端口">
            <Input placeholder="如 GigabitEthernet0/0/1" style={{ width: 180 }} allowClear />
          </Form.Item>
          <Form.Item name="eventType" label="事件类型">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 130 }}
              options={[
                { value: "appeared", label: "出现" },
                { value: "moved", label: "迁移" },
                { value: "disappeared", label: "消失" },
                { value: "vlan_changed", label: "VLAN 变更" },
              ]}
             onSearch={() => {}}/>
          </Form.Item>
          <Form.Item name="vlanId" label="VLAN">
            <InputNumber min={0} max={4094} placeholder="VLAN" style={{ width: 100 }} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select
              placeholder="全部"
              allowClear
              style={{ width: 110 }}
              options={[
                { value: 0, label: "正常" },
                { value: 1, label: "停用" },
              ]}
             onSearch={() => {}}/>
          </Form.Item>
          <Form.Item>
            <Space wrap>
              <Button
                type="primary"
                icon={<SearchOutlined />}
                onClick={handleSearch}
              >
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
              </Button>
              {/* 14-04 注入的导出按钮 — prop 名称 exportScope='current'|'all' 保留,工具栏查询/重置按钮之后位置不变 */}
              {canExport && (
                <>
                  <Button
                    type="primary"
                    icon={<DownloadOutlined />}
                    onClick={() => handleExport("current")}
                    loading={exporting}
                  >
                    导出当前查询
                  </Button>
                  <Button
                    icon={<DownloadOutlined />}
                    onClick={() => handleExport("all")}
                    loading={exporting}
                  >
                    导出全量
                  </Button>
                </>
              )}
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* 错误兜底(14-05b:替换为 ErrorAlertWithRetry — D-20 错误码 1006/1007/500 分级文案) */}
      {error && (
        <div style={{ marginBottom: 16 }}>
          <ErrorAlertWithRetry error={error} onRetry={() => refetch()} />
        </div>
      )}

      <Card
        bordered={false}
        title={
          <Space>
            <span>查询结果</span>
            {!isLoading && (
              <Typography.Text type="secondary">
                共 {total} 条
              </Typography.Text>
            )}
          </Space>
        }
        extra={
          isMobile && (
            <Tooltip title={isMobile ? "已切换卡片视图" : "切换为表格视图"}>
              <Button
                type="text"
                icon={isMobile ? <TableOutlined /> : <AppstoreOutlined />}
                disabled
              />
            </Tooltip>
          )
        }
      >
        {isMobile ? renderCardList() : renderTable()}
      </Card>

      {/* 列配置抽屉(D-08):此处仅占位,具体抽屉 UI 由 14-05 提供 */}
      {/* TODO(14-05):挂载 ColumnConfigModal,传入 config / saveConfig / resetConfig */}
    </div>
  );
};

export default MACHistoryPage;
