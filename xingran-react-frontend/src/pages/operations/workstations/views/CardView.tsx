/**
 * Workstation Card View
 * 工位卡片视图
 */

import { Row, Col, Card, Tag, Popconfirm } from "antd";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import type { WorkstationOps } from "@/types";
import { getWorkstationTypeText, getWorkstationStatusText, getWorkstationStatusColor } from "../constants";

export interface WorkstationCardViewProps {
  workstations: WorkstationOps[];
  onEdit: (record: WorkstationOps) => void;
  onDelete: (id: string) => void;
}

export function WorkstationCardView({
  workstations,
  onEdit,
  onDelete,
}: WorkstationCardViewProps) {
  if (workstations.length === 0) {
    return <div style={{ textAlign: "center", padding: "40px", color: "var(--theme-text-tertiary, #999)" }}>暂无数据</div>;
  }

  return (
    <Row gutter={[16, 16]}>
      {workstations.map((workstation) => (
        <Col key={workstation.id} xs={24} sm={12} md={8} lg={6}>
          <Card
            hoverable
            actions={[
              <EditOutlined key="edit" onClick={() => onEdit(workstation)} />,
              <Popconfirm
                key="delete"
                title="确定要删除这个工位吗？"
                onConfirm={() => onDelete(workstation.id)}
                okText="确定"
                cancelText="取消"
              >
                <DeleteOutlined style={{ color: "var(--theme-error, #ff4d4f)" }} />
              </Popconfirm>,
            ]}
          >
            <Card.Meta
              title={
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
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
