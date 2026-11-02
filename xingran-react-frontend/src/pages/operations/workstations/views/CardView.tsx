/**
 * Workstation Card View
 * 工位卡片视图
 */

import type { CSSProperties } from "react";
import { Row, Col, Card, Tag } from "antd";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import type { WorkstationOps } from "@/types";
import type { ActionPermissionDecision } from "@/components/shared";
import { confirmDelete } from "@/components/shared/confirmDelete";
import {
  getWorkstationTypeText,
  getWorkstationStatusText,
  getWorkstationStatusColor,
} from "../constants";

export interface WorkstationCardViewProps {
  workstations: WorkstationOps[];
  onEdit: (record: WorkstationOps) => void;
  onDelete: (id: string) => void;
  /**
   * Phase 86 W2：卡片视图与表格视图是同一页的两种渲染形态 → 共用页面层同一组决策。
   * 不传 = 不做权限判定（既有调用方行为零变更）。
   */
  editDecision?: ActionPermissionDecision;
  deleteDecision?: ActionPermissionDecision;
}

/** 卡片视图图标动作的禁用态样式（antd `Card.actions` 只接受 ReactNode，无 disabled 位） */
const cardActionStyle = (disabled: boolean): CSSProperties => ({
  opacity: disabled ? 0.45 : 1,
  pointerEvents: disabled ? "none" : "auto",
  cursor: disabled ? "not-allowed" : "pointer",
});

export function WorkstationCardView({
  workstations,
  onEdit,
  onDelete,
  editDecision,
  deleteDecision,
}: WorkstationCardViewProps) {
  if (workstations.length === 0) {
    return (
      <div style={{ textAlign: "center", padding: "40px", color: "var(--theme-text-tertiary)" }}>
        暂无数据
      </div>
    );
  }

  return (
    <Row gutter={[16, 16]}>
      {workstations.map((workstation) => (
        <Col key={workstation.id} xs={24} sm={12} md={8} lg={6}>
          <Card
            hoverable
            actions={[
              ...((editDecision?.visible ?? true)
                ? [
                    <EditOutlined
                      key="edit"
                      title={editDecision?.hint}
                      style={cardActionStyle(editDecision?.disabled ?? false)}
                      onClick={() => {
                        if (editDecision?.disabled) return;
                        onEdit(workstation);
                      }}
                    />,
                  ]
                : []),
              ...((deleteDecision?.visible ?? true)
                ? [
                    <DeleteOutlined
                      key="delete"
                      title={deleteDecision?.hint}
                      style={{
                        ...cardActionStyle(deleteDecision?.disabled ?? false),
                        color: deleteDecision?.disabled
                          ? "var(--theme-text-tertiary)"
                          : "var(--theme-error)",
                      }}
                      onClick={() => {
                        if (deleteDecision?.disabled) return;
                        confirmDelete({
                          target: `该工位（${workstation.name || workstation.id}）`,
                          onDelete: async () => {
                            await onDelete(workstation.id);
                          },
                        });
                      }}
                    />,
                  ]
                : []),
            ]}
          >
            <Card.Meta
              title={
                <div
                  style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}
                >
                  <span>{workstation.name}</span>
                  <Tag color={getWorkstationStatusColor(workstation.status)}>
                    {getWorkstationStatusText(workstation.status)}
                  </Tag>
                </div>
              }
              description={
                <div>
                  <div>
                    <strong>楼层：</strong>
                    {workstation.floorName || workstation.floorId}
                  </div>
                  <div>
                    <strong>类型：</strong>
                    {getWorkstationTypeText(workstation.type)}
                  </div>
                </div>
              }
            />
          </Card>
        </Col>
      ))}
    </Row>
  );
}
