/**
 * Phase 67-T4: 缓存监控概览卡（自 index.tsx :489-564 纯搬迁，零 API 修改）
 * 注：裸 hex 色值（#3f8600/#337ab0）按 UI-SPEC B 类豁免原样保留——只搬不增删。
 */
import type { FC } from "react";
import { Card, Col, Divider, Row, Space, Statistic, Tag } from "antd";
import { DatabaseOutlined } from "@ant-design/icons";
import type { CacheMonitor } from "../types";
import { formatMemorySize } from "../utils";

interface MonitorOverviewProps {
  monitor: CacheMonitor | null;
}

const MonitorOverview: FC<MonitorOverviewProps> = ({ monitor }) => (
  <Row gutter={16} className="mb-6">
    {/* L1 内存缓存 */}
    <Col xs={24} sm={12} md={6}>
      <Card size="small" title="L1 内存缓存" extra={<Tag color="blue">L1</Tag>}>
        <Statistic
          title="键数量"
          value={monitor?.l1?.stats?.keyCount || 0}
          prefix={<DatabaseOutlined />}
          styles={{ content: { fontSize: "20px" } }}
        />
        <div className="mt-2">
          <small className="text-gray-500">
            内存: {formatMemorySize(monitor?.l1?.stats?.usedMemory || 0)}
          </small>
        </div>
      </Card>
    </Col>

    {/* L2 Redis缓存 */}
    <Col xs={24} sm={12} md={6}>
      <Card size="small" title="L2 Redis缓存" extra={<Tag color="green">L2</Tag>}>
        <Statistic
          title="键数量"
          value={monitor?.l2?.stats?.keyCount || 0}
          prefix={<DatabaseOutlined />}
          styles={{ content: { fontSize: "20px" } }}
        />
        <div className="mt-2">
          <small className="text-gray-500">
            内存: {formatMemorySize(monitor?.l2?.stats?.usedMemory || 0)}
          </small>
        </div>
      </Card>
    </Col>

    {/* 综合命中率 */}
    <Col xs={24} sm={12} md={6}>
      <Card size="small" title="综合命中率">
        <Statistic
          title="L1命中率"
          value={monitor?.l1?.stats?.hitRate || 0}
          precision={1}
          suffix="%"
          styles={{ content: { fontSize: "20px", color: "var(--theme-success)" } }}
        />
        <Divider className="my-2" />
        <Statistic
          title="L2命中率"
          value={monitor?.l2?.stats?.hitRate || 0}
          precision={1}
          suffix="%"
          styles={{ content: { fontSize: "16px", color: "var(--theme-info)" } }}
        />
      </Card>
    </Col>

    {/* 状态指示 */}
    <Col xs={24} sm={12} md={6}>
      <Card size="small" title="缓存状态">
        <Space orientation="vertical" className="w-full">
          <div className="flex justify-between items-center">
            <span>L1状态:</span>
            <Tag color={monitor?.l1?.status?.connected ? "success" : "error"}>
              {monitor?.l1?.status?.connected ? "正常" : "异常"}
            </Tag>
          </div>
          <div className="flex justify-between items-center">
            <span>L2状态:</span>
            <Tag color={monitor?.l2?.status?.connected ? "success" : "error"}>
              {monitor?.l2?.status?.connected ? "正常" : "异常"}
            </Tag>
          </div>
        </Space>
      </Card>
    </Col>
  </Row>
);

export default MonitorOverview;
