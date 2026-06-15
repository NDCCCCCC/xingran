import React, { useState, useEffect, useCallback } from 'react';
import {
  Card, DatePicker, Input, Button, Form, Space, Alert, Drawer, message,
} from 'antd';
import { useQuery } from '@tanstack/react-query';
import { useLocation } from 'react-router-dom';
import dayjs from 'dayjs';
import type { Dayjs } from 'dayjs';
import MACTrajectoryChart from '@/components/network/MACTrajectoryChart';
import MACEventsTimeline from '@/components/network/MACEventsTimeline';
import { queryMACTrajectory } from '@/lib/api/networkApi';
import { EmptyStateWithAction, ErrorAlertWithRetry } from '@/components/shared';

const { RangePicker } = DatePicker;

// 时间预设(与 14-01 列表页一致,D-07 锁定)
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

interface TrajectoryQueryParams {
  mac: string;
  start_time: string; // ISO timestamp
  end_time: string;   // ISO timestamp
}

const TrajectoryPage: React.FC = () => {
  const [form] = Form.useForm();
  const location = useLocation();
  const [queryParams, setQueryParams] = useState<TrajectoryQueryParams | null>(null);
  const [activePreset, setActivePreset] = useState<PresetKey>('7d');
  const [customRange, setCustomRange] = useState<[Dayjs, Dayjs] | null>(null);
  const [drawerOpen, setDrawerOpen] = useState<boolean>(true);

  // 查询轨迹数据
  const {
    data: trajectoryData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['macTrajectory', queryParams],
    queryFn: () => queryMACTrajectory(queryParams!),
    enabled: !!queryParams, // 有参数时才查询
    staleTime: 5 * 60 * 1000, // 5分钟缓存
  });

  // MAC地址规范化函数(复用后端逻辑)
  const normalizeMACAddress = (mac: string): string => {
    const cleaned = mac.replace(/[^a-fA-F0-9]/g, '');
    if (cleaned.length !== 12) return '';
    return cleaned.match(/.{2}/g)?.join(':').toUpperCase() || '';
  };

  // 预设 → 设置表单 dateRange + activePreset
  const handlePresetClick = useCallback(
    (preset: Preset) => {
      if (preset.key === 'custom') {
        setActivePreset('custom');
        setCustomRange(null);
        return;
      }
      setActivePreset(preset.key);
      setCustomRange(null);
      const start = dayjs().subtract(preset.amount, preset.unit);
      const end = dayjs();
      form.setFieldsValue({ dateRange: [start, end] });
    },
    [form]
  );

  const handleCustomRangeChange = useCallback(
    (values: [Dayjs, Dayjs] | null) => {
      setCustomRange(values);
      if (values && values[0] && values[1]) {
        setActivePreset('custom');
        form.setFieldsValue({ dateRange: values });
      }
    },
    [form]
  );

  // URL 参数预填(D-17 锁定)— 进入页面时读取 ?mac=...&startTime=...&endTime=...&deviceId=...
  useEffect(() => {
    const sp = new URLSearchParams(location.search);
    const urlMac = sp.get('mac');
    const startTime = sp.get('startTime');
    const endTime = sp.get('endTime');
    // deviceId 仅供未来扩展;当前不参与查询
    // const deviceId = sp.get('deviceId');

    const initial: Record<string, unknown> = {};
    if (urlMac) initial.mac = normalizeMACAddress(urlMac) || urlMac;
    if (startTime && endTime) {
      initial.dateRange = [dayjs(startTime), dayjs(endTime)];
      setActivePreset('custom');
      setCustomRange([dayjs(startTime), dayjs(endTime)]);
    }
    if (Object.keys(initial).length > 0) {
      form.setFieldsValue(initial);
    }

    // URL 包含完整查询条件时,自动触发查询(14-01 跳转场景)
    if (urlMac && startTime && endTime) {
      const normalizedMAC = normalizeMACAddress(urlMac);
      if (normalizedMAC) {
        setQueryParams({
          mac: normalizedMAC,
          start_time: dayjs(startTime).toISOString(),
          end_time: dayjs(endTime).toISOString(),
        });
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSearch = async () => {
    try {
      const values = await form.validateFields();
      const { mac, dateRange } = values;

      // 格式化MAC地址(自动规范化)
      const normalizedMAC = normalizeMACAddress(mac);
      if (!normalizedMAC) {
        message.error('请输入有效的MAC地址');
        return;
      }

      setQueryParams({
        mac: normalizedMAC,
        start_time: dateRange[0].toISOString(),
        end_time: dateRange[1].toISOString(),
      });
    } catch (err) {
      console.error('Form validation failed:', err);
    }
  };

  const handleReset = () => {
    form.resetFields();
    setQueryParams(null);
    setActivePreset('7d');
    setCustomRange(null);
    const preset = PRESETS.find((p) => p.key === '7d')!;
    form.setFieldsValue({
      dateRange: [dayjs().subtract(preset.amount, preset.unit), dayjs()],
    });
  };

  return (
    <div style={{ padding: '24px' }}>
      <Card title="MAC地址轨迹查询" bordered={false}>
        {/* 时间预设按钮组(D-07) */}
        <Space wrap style={{ marginBottom: 12 }}>
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
          <div style={{ marginBottom: 12 }}>
            <RangePicker
              showTime
              format="YYYY-MM-DD HH:mm"
              value={customRange}
              onChange={handleCustomRangeChange}
              placeholder={['开始时间', '结束时间']}
            />
          </div>
        )}
        <Form form={form} layout="inline">
          <Form.Item
            name="mac"
            label="MAC地址"
            rules={[
              { required: true, message: '请输入MAC地址' },
              {
                pattern: /^[A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}$/,
                message: 'MAC地址格式不正确',
              },
            ]}
          >
            <Input
              placeholder="AA:BB:CC:DD:EE:FF"
              style={{ width: 200 }}
              onChange={(e) => {
                // 实时格式化输入
                const formatted = normalizeMACAddress(e.target.value);
                if (formatted) {
                  e.target.value = formatted;
                }
              }}
              onBlur={(e) => {
                // 失焦自动规范化(增强 UX)
                const formatted = normalizeMACAddress(e.target.value);
                if (formatted) {
                  form.setFieldValue('mac', formatted);
                }
              }}
            />
          </Form.Item>

          <Form.Item
            name="dateRange"
            label="时间范围"
            initialValue={[dayjs().subtract(7, 'days'), dayjs()]}
            rules={[{ required: true, message: '请选择时间范围' }]}
          >
            <RangePicker
              showTime
              format="YYYY-MM-DD HH:mm"
              style={{ width: 400 }}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button onClick={handleReset}>
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* 14-05b:错误态替换为 ErrorAlertWithRetry(D-20 错误码 1006/1007/500 分级文案) */}
      {error && (
        <div style={{ marginTop: '16px' }}>
          <ErrorAlertWithRetry error={error} onRetry={() => refetch()} />
        </div>
      )}

      {/* 14-05b:数据为空时 EmptyStateWithAction 提示(D-18 引导)— queryParams 已就绪但轨迹数据为空 */}
      {!error && queryParams && !isLoading && (!trajectoryData || trajectoryData.length === 0) && (
        <div style={{ marginTop: '16px' }}>
          <EmptyStateWithAction
            description="该 MAC 在此时间范围内无轨迹数据"
            actionLabel="查看事件时间线"
            actionPath="#timeline"
          />
        </div>
      )}

      <div style={{ marginTop: '16px' }}>
        <MACTrajectoryChart
          data={trajectoryData || []}
          loading={isLoading}
          onError={(err) => {
            console.error('Trajectory chart error:', err);
          }}
        />
      </div>

      {/* 右侧事件时间线 Drawer(14-01 组件复用) */}
      <Drawer
        title="事件时间线"
        placement="right"
        width={420}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        mask={false}
        destroyOnClose={false}
      >
        {queryParams ? (
          <MACEventsTimeline
            mac={queryParams.mac}
            startTime={queryParams.start_time}
            endTime={queryParams.end_time}
          />
        ) : (
          <Alert
            type="info"
            showIcon
            message="请先执行查询"
            description="查询后将在此显示 MAC 在该时间范围内的事件时间线"
          />
        )}
      </Drawer>

      {/* Drawer 关闭后显示"显示时间线"按钮 */}
      {!drawerOpen && (
        <Button
          type="primary"
          style={{ position: 'fixed', right: 24, top: 96, zIndex: 100 }}
          onClick={() => setDrawerOpen(true)}
        >
          显示时间线
        </Button>
      )}
    </div>
  );
};

export default TrajectoryPage;
