/**
 * 楼层卡片视图组件
 */

import type { FC, CSSProperties } from "react";
import { Row, Col, Card, Tag } from "antd";
import { EditOutlined, BgColorsOutlined, DeleteOutlined } from "@ant-design/icons";
import type { Floor } from "@/types";
import type { ActionPermissionDecision } from "@/components/shared";
import { confirmDelete } from "@/components/shared/confirmDelete";

interface FloorCardViewProps {
  floors: Floor[];
  onEdit: (floor: Floor) => void;
  onEditFloorPlan: (floor: Floor) => void;
  onDelete: (id: string) => void;
  /**
   * Phase 86 W2：卡片视图与表格视图是同一页的两种渲染形态 → 共用页面层同一组决策。
   * 不传 = 不做权限判定（既有调用方行为零变更）。
   */
  editDecision?: ActionPermissionDecision;
  deleteDecision?: ActionPermissionDecision;
  /** 「查看平面图」→ 与平面图保存按钮共用同一四码合并决策对象。 */
  floorPlanDecision?: ActionPermissionDecision;
}

/** 卡片视图图标动作的禁用态样式（antd `Card.actions` 只接受 ReactNode，无 disabled 位） */
const cardActionStyle = (disabled: boolean): CSSProperties => ({
  opacity: disabled ? 0.45 : 1,
  pointerEvents: disabled ? "none" : "auto",
  cursor: disabled ? "not-allowed" : "pointer",
});

export const FloorCardView: FC<FloorCardViewProps> = ({
  floors,
  onEdit,
  onEditFloorPlan,
  onDelete,
  editDecision,
  deleteDecision,
  floorPlanDecision,
}) => {
  if (floors.length === 0) {
    return (
      <div style={{ textAlign: "center", padding: "40px", color: "var(--theme-text-tertiary)" }}>
        暂无数据
      </div>
    );
  }

  return (
    <Row gutter={[16, 16]}>
      {floors.map((floor) => (
        <Col key={floor.id} xs={24} sm={12} md={8} lg={6}>
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
                        onEdit(floor);
                      }}
                    />,
                  ]
                : []),
              ...((floorPlanDecision?.visible ?? true)
                ? [
                    <BgColorsOutlined
                      key="floorPlan"
                      title={floorPlanDecision?.hint ?? "查看平面图"}
                      style={cardActionStyle(floorPlanDecision?.disabled ?? false)}
                      onClick={() => {
                        if (floorPlanDecision?.disabled) return;
                        onEditFloorPlan(floor);
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
                          target: `该楼层（${floor.name || `${floor.floorNo}层`}）`,
                          onDelete: async () => {
                            await onDelete(floor.id);
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
                  style={{
                    display: "flex",
                    justifyContent: "space-between",
                    alignItems: "center",
                  }}
                >
                  <span>{floor.name || `${floor.floorNo}层`}</span>
                  <Tag color={floor.status === 0 ? "success" : "error"}>
                    {floor.status === 0 ? "正常" : "停用"}
                  </Tag>
                </div>
              }
              description={
                <div>
                  <div>
                    <strong>楼层号：</strong>
                    {floor.floorNo}
                  </div>
                  <div>
                    <strong>楼宇：</strong>
                    {floor.buildingName || floor.buildingCode}
                  </div>
                  {floor.area != null && (
                    <div>
                      <strong>面积：</strong>
                      {floor.area}m²
                    </div>
                  )}
                </div>
              }
            />
          </Card>
        </Col>
      ))}
    </Row>
  );
};
