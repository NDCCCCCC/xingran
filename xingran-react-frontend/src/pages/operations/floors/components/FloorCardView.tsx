/**
 * 楼层卡片视图组件
 */

import type { FC } from "react";
import { Row, Col, Card, Tag, Popconfirm } from "antd";
import { EditOutlined, BgColorsOutlined, DeleteOutlined } from "@ant-design/icons";
import type { Floor } from "@/types";

interface FloorCardViewProps {
  floors: Floor[];
  onEdit: (floor: Floor) => void;
  onEditFloorPlan: (floor: Floor) => void;
  onDelete: (id: string) => void;
}

export const FloorCardView: FC<FloorCardViewProps> = ({
  floors,
  onEdit,
  onEditFloorPlan,
  onDelete,
}) => {
  if (floors.length === 0) {
    return (
      <div
        style={{ textAlign: "center", padding: "40px", color: "var(--theme-text-tertiary, #999)" }}
      >
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
              <EditOutlined key="edit" onClick={() => onEdit(floor)} />,
              <BgColorsOutlined
                key="floorPlan"
                onClick={() => onEditFloorPlan(floor)}
                title="查看平面图"
              />,
              <Popconfirm
                key="delete"
                title="确定要删除这个楼层吗？"
                onConfirm={() => onDelete(floor.id)}
                okText="确定"
                cancelText="取消"
              >
                <DeleteOutlined style={{ color: "var(--theme-error, #ff4d4f)" }} />
              </Popconfirm>,
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
                  {floor.area && (
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
