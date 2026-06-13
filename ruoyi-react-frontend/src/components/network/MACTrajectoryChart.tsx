import React, { useMemo } from 'react';
import ReactECharts from 'echarts-for-react';
import type { EChartsOption } from 'echarts';
import { Empty, Spin } from 'antd';

interface TrajectoryNode {
  mac: string;
  deviceId: string;
  deviceName: string;
  interface: string;
  vlanId?: number;
  startTime: string; // ISO timestamp
  endTime: string; // ISO timestamp
  duration: number;
  eventType: 'appeared' | 'disappeared' | 'moved' | 'vlan_changed';
}

interface MACTrajectoryChartProps {
  data: TrajectoryNode[];
  loading?: boolean;
  onError?: (error: unknown) => void;
}

const EVENT_COLORS = {
  appeared: '#52c41a',     // 绿色
  disappeared: '#ff4d4f',   // 红色
  moved: '#faad14',        // 黄色
  vlan_changed: '#1890ff', // 蓝色
};

const MACTrajectoryChart: React.FC<MACTrajectoryChartProps> = ({
  data,
  loading = false,
  onError
}) => {
  // 处理错误
  if (!loading && (!data || data.length === 0)) {
    return <Empty description="暂无轨迹数据" />;
  }

  // ECharts配置
  const option: EChartsOption = useMemo(() => {
    if (!data || data.length === 0) return {};

    // 按设备分组数据
    const deviceGroups = data.reduce((groups, node) => {
      const device = node.deviceName || 'Unknown Device';
      if (!groups[device]) {
        groups[device] = [];
      }
      groups[device].push(node);
      return groups;
    }, {} as Record<string, TrajectoryNode[]>);

    // 构建Gantt图数据
    const devices = Object.keys(deviceGroups);
    const seriesData = data.map((node, index) => ({
      name: `${node.interface} (VLAN ${node.vlanId})`,
      value: [
        devices.indexOf(node.deviceName), // Y轴：设备索引
        new Date(node.startTime).getTime(), // X轴：开始时间
        new Date(node.endTime).getTime(),  // X轴：结束时间
        node.duration, // 停留时长（用于tooltip）
        node.eventType, // 事件类型（用于颜色）
        node.mac, // MAC地址（用于tooltip）
        node.deviceName, // 设备名（用于tooltip）
      ],
      itemStyle: {
        color: EVENT_COLORS[node.eventType],
      },
    }));

    return {
      title: {
        text: `MAC轨迹: ${data[0]?.mac || 'N/A'}`,
        left: 'center',
      },
      tooltip: {
        trigger: 'item',
        formatter: (params: any) => {
          const data = params.data.value;
          const durationHours = (data[3] / 3600).toFixed(1);
          return `
            <div>
              <strong>MAC: ${data[4]}</strong><br/>
              设备: ${data[5]}<br/>
              端口: ${params.name}<br/>
              停留: ${durationHours}小时<br/>
              事件: ${data[6]}
            </div>
          `;
        },
      },
      grid: {
        left: '15%',
        right: '5%',
        bottom: '15%',
        top: '15%',
      },
      xAxis: {
        type: 'time',
        axisLabel: {
          formatter: '{yyyy}-{MM}-{dd} {HH}:{mm}',
        },
      },
      yAxis: {
        type: 'category',
        data: devices,
        axisLabel: {
          formatter: (value: string) => {
            // 设备名太长时截断
            return value.length > 15 ? value.substring(0, 12) + '...' : value;
          },
        },
      },
      series: [
        {
          type: 'custom',
          renderItem: (params: any, api: any) => {
            const categoryIndex = api.value(0);
            const timeStart = api.value(1);
            const timeEnd = api.value(2);

            const coordStart = api.coord([timeStart, categoryIndex]);
            const coordEnd = api.coord([timeEnd, categoryIndex]);

            const barHeight = 20;
            const barWidth = coordEnd[0] - coordStart[0];

            return {
              type: 'rect',
              shape: {
                x: coordStart[0],
                y: coordStart[1] - barHeight / 2,
                width: barWidth,
                height: barHeight,
              },
              style: {
                fill: api.visual('color'),
              },
            };
          },
          data: seriesData,
        },
      ],
      dataZoom: [
        {
          type: 'slider',
          xAxisIndex: 0,
          bottom: '5%',
        },
      ],
    };
  }, [data]);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin tip="加载轨迹数据..." />
      </div>
    );
  }

  return (
    <div style={{ width: '100%', height: '600px' }}>
      <ReactECharts
        option={option}
        style={{ height: '100%', width: '100%' }}
        opts={{ renderer: 'canvas' }}
        onError={(error) => {
          console.error('ECharts error:', error);
          onError?.(error);
        }}
      />
    </div>
  );
};

export default MACTrajectoryChart;
