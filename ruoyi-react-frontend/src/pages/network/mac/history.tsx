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
 * 颜色与图标:与 MACTrajectoryChart.tsx:24-29 完全一致(D-10)
 *
 * 已知 TODO(由后续 plan 接管):
 * - 14-04:工具栏新增 2 个互斥导出按钮
 * - 14-05:替换内联 Alert/Empty 为 EmptyStateWithAction / ErrorAlertWithRetry
 * - 14-05b:移动端卡片视图最终样式收口
 */

import React, { useState, useEffect, useMemo, useCallback } from 'react';
import {
  Table, Button, Space, Form, Input, InputNumber, Select,
  Tag, Card, Empty, Alert, DatePicker, List, Grid, Spin, Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  SearchOutlined, ReloadOutlined, AppstoreOutlined, TableOutlined,
  CopyOutlined, EyeOutlined, AimOutlined,
} from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import { useTableQuery } from '@/hooks/useTableQuery';
import { useColumnConfig, type ColumnConfig } from '@/hooks/useColumnConfig';
import { queryMACHistory } from '@/lib/api/networkApi';
import type {
  MACHistoryRecord,
  MACHistoryQueryParams,
} from '@/lib/api/networkApi';
import { MACEventsTimeline } from '@/components/network';

const { RangePicker } = DatePicker;
const { useBreakpoint } = Grid;

// 预设时间范围(单位,数量) — D-07 锁定
type PresetKey = '1h' | '24h' | '7d' | '30d' | '90d' | 'custom';
interface Preset {
  key: PresetKey;
  label: string;
  amount: number;
  unit: 'hour' | 'day';
}
const PRESETS: Preset[] = [
  { key: '1h', label: '近 1h', amount: 1, unit: 'hour' },
  { key: '24h', label: '近 24h', amount: 24, unit: 'hour' },
  { key: '7d', label: '近 7d', amount: 7, unit: 'day' },
  { key: '30d', label: '近 30d', amount: 30, unit: 'day' },
  { key: '90d', label: '近 90d', amount: 90, unit: 'day' },
  { key: 'custom', label: '自定义', amount: 0, unit: 'day' },
];

// 事件类型颜色(D-10 锁定,与 MACTrajectoryChart.tsx:24-29 一致)
const EVENT_TAG_COLORS: Record<string, string> = {
  appeared: 'green',
  moved: 'gold',
  disappeared: 'red',
  vlan_changed: 'blue',
};
const EVENT_TYPE_LABELS: Record<string, string> = {
  appeared: '出现',
  moved: '迁移',
  disappeared: '消失',
  vlan_changed: 'VLAN 变更',
};

// 8 列默认配置(D-08 锁定)
const defaultHistoryColumns: ColumnConfig[] = [
  { key: 'time', label: '时间', visible: true, order: 1, width: 160 },
  { key: 'mac', label: 'MAC 地址', visible: true, order: 2, width: 160 },
  { key: 'device', label: '设备', visible: true, order: 3, width: 180 },
  { key: 'port', label: '端口', visible: true, order: 4, width: 130 },
  { key: 'eventType', label: '事件类型', visible: true, order: 5, width: 110 },
  { key: 'vlan', label: 'VLAN', visible: true, order: 6, width: 80 },
  { key: 'status', label: '状态', visible: true, order: 7, width: 80 },
  { key: 'action', label: '操作', visible: true, order: 8, width: 200 },
];

// MAC 规范化(从 trajectory.tsx:63-70 复制)
const normalizeMACAddress = (mac: string): string => {
  const cleaned = mac.replace(/[^a-fA-F0-9]/g, '');
  if (cleaned.length !== 12) return '';
  return cleaned.match(/.{2}/g)?.join(':').toUpperCase() || '';
};

const MACHistoryPage: React.FC = () => {
  const [form] = Form.useForm();
  const [searchParams, setSearchParams] = useSearchParams();
  const breakpoint = useBreakpoint();
  const isMobile = !!breakpoint.xs;

  // 状态
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [activePreset, setActivePreset] = useState<PresetKey>('7d');
  const [customRange, setCustomRange] = useState<[Dayjs, Dayjs] | null>(null);
  const [expandedRowKeys, setExpandedRowKeys] = useState<React.Key[]>([]);

  // 计算当前时间范围(ISO 字符串)用于查询
  const [timeRange, setTimeRange] = useState<{ startTime: string; endTime: string }>(
    () => {
      const preset = PRESETS.find((p) => p.key === '7d')!;
      return {
        startTime: dayjs().subtract(preset.amount, preset.unit).toISOString(),
        endTime: dayjs().toISOString(),
      };
    }
  );

  // URL 参数注入(D-17)
  useEffect(() => {
    const deviceId = searchParams.get('deviceId');
    const portName = searchParams.get('portName');
    const startTime = searchParams.get('startTime');
    const endTime = searchParams.get('endTime');
    const mac = searchParams.get('mac');

    const initial: Record<string, unknown> = {};
    if (deviceId) initial.deviceId = deviceId;
    if (portName) initial.interfaceName = portName;
    if (mac) initial.mac = normalizeMACAddress(mac) || mac;
    form.setFieldsValue(initial);

    if (startTime && endTime) {
      // 视为自定义
      setActivePreset('custom');
      setCustomRange([dayjs(startTime), dayjs(endTime)]);
      setTimeRange({ startTime, endTime });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 切换预设
  const handlePresetClick = useCallback(
    (preset: Preset) => {
      if (preset.key === 'custom') {
        setActivePreset('custom');
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
    []
  );

  // 切换自定义 RangePicker
  const handleCustomRangeChange = useCallback(
    (values: [Dayjs, Dayjs] | null) => {
      setCustomRange(values);
      if (values && values[0] && values[1]) {
        setActivePreset('custom');
        setTimeRange({
          startTime: values[0].toISOString(),
          endTime: values[1].toISOString(),
        });
        setCurrent(1);
      }
    },
    []
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
    isPlaceholderData,
    error,
    refetch,
  } = useTableQuery<MACHistoryRecord>({
    resource: 'mac.history.list',
    current,
    pageSize,
    filters: filters as Record<string, unknown>,
    queryFn: queryMACHistory,
  });

  const list = pageData?.list ?? [];
  const total = pageData?.total ?? 0;

  // 列配置(D-08)
  const { visibleColumns, config } = useColumnConfig({
    pageKey: 'mac.history.list',
    defaultColumns: defaultHistoryColumns,
    enableCache: true,
  });

  // 8 列定义
  const allColumns: Record<string, ColumnsType<MACHistoryRecord>[number]> = useMemo(
    () => ({
      time: {
        title: '时间',
        key: 'time',
        width: 160,
        render: (_: unknown, record: MACHistoryRecord) =>
          dayjs(record.firstSeen).format('YYYY-MM-DD HH:mm:ss'),
      },
      mac: {
        title: 'MAC 地址',
        key: 'mac',
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
        title: '设备',
        key: 'device',
        width: 180,
        ellipsis: true,
        render: (_: unknown, record: MACHistoryRecord) =>
          record.deviceNameSnapshot || record.deviceId,
      },
      port: {
        title: '端口',
        key: 'port',
        width: 130,
        render: (_: unknown, record: MACHistoryRecord) => record.interfaceName,
      },
      eventType: {
        title: '事件类型',
        key: 'eventType',
        width: 110,
        render: (_: unknown, record: MACHistoryRecord) => (
          <Tag color={EVENT_TAG_COLORS[record.eventType] ?? 'default'}>
            {EVENT_TYPE_LABELS[record.eventType] ?? record.eventType}
          </Tag>
        ),
      },
      vlan: {
        title: 'VLAN',
        key: 'vlan',
        width: 80,
        render: (_: unknown, record: MACHistoryRecord) =>
          record.vlanId ?? '-',
      },
      status: {
        title: '状态',
        key: 'status',
        width: 80,
        render: (_: unknown, record: MACHistoryRecord) => (
          <Tag color={record.status === 0 ? 'green' : 'red'}>
            {record.status === 0 ? '正常' : '停用'}
          </Tag>
        ),
      },
      action: {
        title: '操作',
        key: 'action',
        width: 200,
        fixed: 'right' as const,
        render: (_: unknown, record: MACHistoryRecord) => (
          <Space size={4} wrap>
            <Button
              type="link"
              size="small"
              icon={<AimOutlined />}
              onClick={() => {
                const search = new URLSearchParams();
                search.set('mac', record.macAddress);
                search.set('startTime', timeRange.startTime);
                search.set('endTime', timeRange.endTime);
                if (record.deviceId) search.set('deviceId', record.deviceId);
                window.location.href = `/network/mac/trajectory?${search.toString()}`;
              }}
            >
              查看轨迹
            </Button>
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
              {expandedRowKeys.includes(record.id) ? '收起事件' : '查看事件'}
            </Button>
          </Space>
        ),
      },
    }),
    [timeRange, expandedRowKeys]
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
    setActivePreset('7d');
    setCustomRange(null);
    const preset = PRESETS.find((p) => p.key === '7d')!;
    setTimeRange({
      startTime: dayjs().subtract(preset.amount, preset.unit).toISOString(),
      endTime: dayjs().toISOString(),
    });
    setCurrent(1);
    setSearchParams(new URLSearchParams());
  }, [form, setSearchParams]);

  // 复制 MAC
  const copyMAC = (mac: string) => {
    void navigator.clipboard?.writeText(mac);
  };

  // 桌面表格
  const renderTable = () => (
    <Table<MACHistoryRecord>
      rowKey="id"
      size="middle"
      virtual
      scroll={{ x: 1200, y: 600 }}
      dataSource={list}
      columns={tableColumns}
      loading={isLoading || isPlaceholderData}
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
        emptyText: (
          <Empty
            description="该范围内未采集到 MAC 记录,请检查设备是否启用了 MAC 采集/端口采集周期"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          />
        ),
      }}
    />
  );

  // 移动卡片
  const renderCardList = () => {
    if (isLoading && list.length === 0) {
      return (
        <div style={{ textAlign: 'center', padding: '40px' }}>
          <Spin />
        </div>
      );
    }
    if (list.length === 0) {
      return (
        <Empty
          description="该范围内未采集到 MAC 记录"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          style={{ padding: '40px 0' }}
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
              padding: '12px',
              background: '#fff',
              marginBottom: 8,
              borderRadius: 6,
              border: '1px solid #f0f0f0',
            }}
            actions={[
              <Button
                key="trajectory"
                type="link"
                size="small"
                onClick={() => {
                  const search = new URLSearchParams();
                  search.set('mac', record.macAddress);
                  search.set('startTime', timeRange.startTime);
                  search.set('endTime', timeRange.endTime);
                  window.location.href = `/network/mac/trajectory?${search.toString()}`;
                }}
              >
                查看轨迹
              </Button>,
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
                {expandedRowKeys.includes(record.id) ? '收起事件' : '查看事件'}
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
                      onClick={() => copyMAC(record.macAddress)}
                    />
                  </Tooltip>
                  <Tag color={EVENT_TAG_COLORS[record.eventType] ?? 'default'}>
                    {EVENT_TYPE_LABELS[record.eventType] ?? record.eventType}
                  </Tag>
                  <Tag color={record.status === 0 ? 'green' : 'red'}>
                    {record.status === 0 ? '正常' : '停用'}
                  </Tag>
                </Space>
              }
              description={
                <div>
                  <div>设备: {record.deviceNameSnapshot || record.deviceId}</div>
                  <div>端口: {record.interfaceName}</div>
                  <div>VLAN: {record.vlanId ?? '-'}</div>
                  <div>时间: {dayjs(record.firstSeen).format('YYYY-MM-DD HH:mm:ss')}</div>
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
                  type={activePreset === p.key ? 'primary' : 'default'}
                  size="small"
                  onClick={() => handlePresetClick(p)}
                >
                  {p.label}
                </Button>
              ))}
              {activePreset === 'custom' && (
                <RangePicker
                  showTime
                  format="YYYY-MM-DD HH:mm"
                  value={customRange}
                  onChange={handleCustomRangeChange}
                  placeholder={['开始时间', '结束时间']}
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
                  type={activePreset === p.key ? 'primary' : 'default'}
                  size="small"
                  onClick={() => handlePresetClick(p)}
                >
                  {p.label}
                </Button>
              ))}
            </Space>
            {activePreset === 'custom' && (
              <div style={{ marginTop: 8 }}>
                <RangePicker
                  showTime
                  format="YYYY-MM-DD HH:mm"
                  value={customRange}
                  onChange={handleCustomRangeChange}
                  placeholder={['开始时间', '结束时间']}
                />
              </div>
            )}
          </div>
        )}

        <Form
          form={form}
          layout={isMobile ? 'vertical' : 'inline'}
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
                { value: 'appeared', label: '出现' },
                { value: 'moved', label: '迁移' },
                { value: 'disappeared', label: '消失' },
                { value: 'vlan_changed', label: 'VLAN 变更' },
              ]}
            />
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
                { value: 0, label: '正常' },
                { value: 1, label: '停用' },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space>
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
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* 错误兜底(TODO: 14-05 替换为 ErrorAlertWithRetry) */}
      {error && (
        <Alert
          type="error"
          showIcon
          message="查询失败"
          description={(error as Error).message}
          style={{ marginBottom: 16 }}
          action={
            <Button size="small" onClick={() => refetch()}>
              重试
            </Button>
          }
        />
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
            <Tooltip title={isMobile ? '已切换卡片视图' : '切换为表格视图'}>
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
      {false && <div data-config-hint={JSON.stringify(config.slice(0, 2))} aria-hidden />}
    </div>
  );
};

export default MACHistoryPage;
