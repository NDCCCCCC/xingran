/**
 * Phase 67-T4: 缓存统计信息模态框（自 index.tsx :833-969 纯搬迁，零 API 修改）
 * 注：裸 hex 色值按 B 类豁免原样保留；Tabs「缓存统计」面板内容为本 Modal 的子集
 * （键数量/命中/未命中/命中率/内存使用均覆盖），迁移时 Tabs 移除、入口保留「统计信息」按钮。
 */
import type { FC } from "react";
import { Col, Divider, Modal, Row, Card, Statistic, Tag } from "antd";
import type { CacheMonitor } from "../types";
import { formatMemorySize } from "../utils";

interface CacheStatsModalProps {
  open: boolean;
  onClose: () => void;
  monitor: CacheMonitor | null;
}

const CacheStatsModal: FC<CacheStatsModalProps> = ({ open, onClose, monitor }) => (
  <Modal
    title="缓存统计信息"
    mask={{ closable: false }}
    keyboard={false}
    open={open}
    onCancel={onClose}
    footer={null}
    width={900}
  >
    <Row gutter={16}>
      {/* L1统计 */}
      <Col span={12}>
        <Card
          size="small"
          title={
            <>
              <Tag color="blue">L1</Tag> 内存缓存统计
            </>
          }
        >
          <Statistic
            title="键数量"
            value={monitor?.l1?.stats?.keyCount || 0}
            styles={{ content: { fontSize: "24px" } }}
          />
          <Divider className="my-2" />
          <Row gutter={8}>
            <Col span={12}>
              <Statistic
                title="命中次数"
                value={monitor?.l1?.stats?.hitCount || 0}
                valueStyle={{ fontSize: "18px", color: "var(--theme-success)" }}
              />
            </Col>
            <Col span={12}>
              <Statistic
                title="未命中次数"
                value={monitor?.l1?.stats?.missCount || 0}
                valueStyle={{ fontSize: "18px", color: "var(--theme-error)" }}
              />
            </Col>
          </Row>
          <Divider className="my-2" />
          <Statistic
            title="命中率"
            value={monitor?.l1?.stats?.hitRate || 0}
            precision={2}
            suffix="%"
            styles={{ content: { fontSize: "20px", color: "var(--theme-info)" } }}
          />
          <Divider className="my-2" />
          <Statistic title="内存使用" value={formatMemorySize(monitor?.l1?.stats?.usedMemory || 0)} />
        </Card>
      </Col>

      {/* L2统计 */}
      <Col span={12}>
        <Card
          size="small"
          title={
            <>
              <Tag color="green">L2</Tag> Redis缓存统计
            </>
          }
        >
          <Statistic
            title="键数量"
            value={monitor?.l2?.stats?.keyCount || 0}
            styles={{ content: { fontSize: "24px" } }}
          />
          <Divider className="my-2" />
          <Row gutter={8}>
            <Col span={12}>
              <Statistic
                title="命中次数"
                value={monitor?.l2?.stats?.hitCount || 0}
                valueStyle={{ fontSize: "18px", color: "var(--theme-success)" }}
              />
            </Col>
            <Col span={12}>
              <Statistic
                title="未命中次数"
                value={monitor?.l2?.stats?.missCount || 0}
                valueStyle={{ fontSize: "18px", color: "var(--theme-error)" }}
              />
            </Col>
          </Row>
          <Divider className="my-2" />
          <Statistic
            title="命中率"
            value={monitor?.l2?.stats?.hitRate || 0}
            precision={2}
            suffix="%"
            styles={{ content: { fontSize: "20px", color: "var(--theme-info)" } }}
          />
          <Divider className="my-2" />
          <Statistic title="内存使用" value={formatMemorySize(monitor?.l2?.stats?.usedMemory || 0)} />
        </Card>
      </Col>
    </Row>

    {/* 综合对比 */}
    <Divider />
    <Card size="small" title="综合对比">
      <Row gutter={16}>
        <Col span={8}>
          <Statistic
            title="总键数(L1+L2)"
            value={(monitor?.l1?.stats?.keyCount || 0) + (monitor?.l2?.stats?.keyCount || 0)}
          />
        </Col>
        <Col span={8}>
          <Statistic
            title="总内存使用"
            value={formatMemorySize(
              (monitor?.l1?.stats?.usedMemory || 0) + (monitor?.l2?.stats?.usedMemory || 0)
            )}
          />
        </Col>
        <Col span={8}>
          <Statistic
            title="L1缓存命中率"
            value={monitor?.l1?.stats?.hitRate || 0}
            precision={2}
            suffix="%"
          />
        </Col>
      </Row>
    </Card>
  </Modal>
);

export default CacheStatsModal;
