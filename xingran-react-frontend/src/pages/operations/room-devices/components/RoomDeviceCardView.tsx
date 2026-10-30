import type { FC } from "react";
import { Row, Col, Card, Tag } from "antd";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import type { RoomDevice } from "@/types";
import { useActionPermission } from "@/components/shared";
import { confirmDelete } from "@/components/shared/confirmDelete";
import { cardActionStyle, getStatusText, getDeviceTypeText } from "../constants";

type ActionDecision = ReturnType<ReturnType<typeof useActionPermission>["decide"]>;

export interface RoomDeviceCardViewProps {
  devices: RoomDevice[];
  permEdit: ActionDecision;
  permDelete: ActionDecision;
  openModal: (record?: RoomDevice) => void;
  handleDelete: (id: string) => Promise<void>;
}

export const RoomDeviceCardView: FC<RoomDeviceCardViewProps> = ({
  devices,
  permEdit,
  permDelete,
  openModal,
  handleDelete,
}) => {
  if (devices.length === 0) {
    return (
      <div style={{ textAlign: "center", padding: 48, color: "var(--theme-text-tertiary)" }}>
        暂无数据
      </div>
    );
  }

  return (
    <Row gutter={[16, 16]}>
      {devices.map((device) => (
        <Col key={device.id} xs={24} sm={12} md={8} lg={6}>
          <Card
            hoverable
            actions={[
              ...(permEdit.visible
                ? [
                    <EditOutlined
                      key="edit"
                      title={permEdit.hint}
                      style={cardActionStyle(permEdit.disabled)}
                      onClick={() => {
                        if (permEdit.disabled) return;
                        openModal(device);
                      }}
                    />,
                  ]
                : []),
              ...(permDelete.visible
                ? [
                    <DeleteOutlined
                      key="delete"
                      title={permDelete.hint}
                      style={{
                        ...cardActionStyle(permDelete.disabled),
                        color: permDelete.disabled
                          ? "var(--theme-text-tertiary)"
                          : "var(--theme-error)",
                      }}
                      onClick={() => {
                        if (permDelete.disabled) return;
                        confirmDelete({
                          target: `机房设备 ${device.name}`,
                          onDelete: () => handleDelete(device.id),
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
                  <span>{device.name}</span>
                  <Tag
                    color={
                      device.status === 0 ? "success" : device.status === 1 ? "error" : "default"
                    }
                  >
                    {getStatusText(device.status)}
                  </Tag>
                </div>
              }
              description={
                <div>
                  <div>
                    <strong>编码：</strong>
                    {device.deviceCode}
                  </div>
                  <div>
                    <strong>机房：</strong>
                    {device.roomName || "-"}
                  </div>
                  <div>
                    <strong>类型：</strong>
                    {getDeviceTypeText(device.deviceType)}
                  </div>
                  {device.vendor && (
                    <div>
                      <strong>厂商：</strong>
                      {device.vendor}
                    </div>
                  )}
                  {device.model && (
                    <div>
                      <strong>型号：</strong>
                      {device.model}
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

export default RoomDeviceCardView;
