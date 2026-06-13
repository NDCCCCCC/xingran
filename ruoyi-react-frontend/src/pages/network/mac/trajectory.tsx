import React, { useState } from 'react';
import { Card, DatePicker, Input, Button, Form, Space, Alert, message } from 'antd';
import { useQuery } from '@tanstack/react-query';
import dayjs from 'dayjs';
import MACTrajectoryChart from '@/components/network/MACTrajectoryChart';
import { queryMACTrajectory } from '@/lib/api/networkApi';

const { RangePicker } = DatePicker;

interface TrajectoryQueryParams {
  mac: string;
  start_time: string; // ISO timestamp
  end_time: string;   // ISO timestamp
}

const TrajectoryPage: React.FC = () => {
  const [form] = Form.useForm();
  const [queryParams, setQueryParams] = useState<TrajectoryQueryParams | null>(null);

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

  const handleSearch = async () => {
    try {
      const values = await form.validateFields();
      const { mac, dateRange } = values;

      // 格式化MAC地址（自动规范化）
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
  };

  // MAC地址规范化函数（复用后端逻辑）
  const normalizeMACAddress = (mac: string): string => {
    // 移除所有非十六进制字符
    const cleaned = mac.replace(/[^a-fA-F0-9]/g, '');
    if (cleaned.length !== 12) return '';

    // 格式化为AA:BB:CC:DD:EE:FF
    return cleaned.match(/.{2}/g)?.join(':').toUpperCase() || '';
  };

  return (
    <div style={{ padding: '24px' }}>
      <Card title="MAC地址轨迹查询" bordered={false}>
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

      {error && (
        <Alert
          message="查询失败"
          description={(error as Error).message}
          type="error"
          showIcon
          style={{ marginTop: '16px' }}
        />
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
    </div>
  );
};

export default TrajectoryPage;
