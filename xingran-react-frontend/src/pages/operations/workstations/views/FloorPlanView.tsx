/**
 * Workstation Floor Plan View
 * 工位平面图视图
 */

import { Card, Space, Tag, Button, Select } from "antd";
import { CloseOutlined, BgColorsOutlined, DesktopOutlined } from "@ant-design/icons";
import type { WorkstationOps } from "@/types";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";
import FloorPlanEditor from "@/components/shared/FloorPlanEditor";
import { getWorkstationTypeText, getWorkstationStatusText, getWorkstationStatusColor } from "../constants";
import type { FloorOption } from "../types";

const { Option } = Select;

export interface WorkstationFloorPlanViewProps {
  selectedFloorForPlan: string;
  floorOptions: FloorOption[];
  floorPlanWorkstations: WorkstationNode[];
  allWorkstations: WorkstationOps[];
  onFloorChange: (floorId: string) => void;
  onPositionUpdate: (items: { id: string; positionX: number; positionY: number; rotation?: number }[]) => Promise<void>;
  onEdit: (workstation: WorkstationNode) => void;
  onCloseFloorPlan: () => void;
}

export function WorkstationFloorPlanView({
  selectedFloorForPlan,
  floorOptions,
  floorPlanWorkstations,
  allWorkstations,
  onFloorChange,
  onPositionUpdate,
  onEdit,
  onCloseFloorPlan,
}: WorkstationFloorPlanViewProps) {
  return (
    <div style={{ display: "flex", gap: 16, height: "calc(100vh - 280px)", minHeight: 600 }}>
      {/* 左侧：平面图编辑器 */}
      <div style={{ flex: 1, display: "flex", flexDirection: "column", minWidth: 0 }}>
        {/* 楼层选择器 */}
        <div
          style={{ marginBottom: 12, padding: "12px", background: "#f5f5f5", borderRadius: "4px", flexShrink: 0 }}
        >
          <Space wrap>
            <span>选择楼层：</span>
            <Select
              style={{ width: 200 }}
              placeholder="请选择要查看的楼层"
              value={selectedFloorForPlan}
              onChange={onFloorChange}
              allowClear
             onSearch={() => {}}>
              {floorOptions.map((f) => (
                <Option key={f.code} value={f.code}>
                  {f.name}
                </Option>
              ))}
            </Select>
            {selectedFloorForPlan && (
              <Tag color="blue">
                当前楼层：{floorOptions.find((f) => f.code === selectedFloorForPlan)?.name}
              </Tag>
            )}
            <Tag color="cyan">工位数：{floorPlanWorkstations.length}</Tag>
            <Button type="default" icon={<CloseOutlined />} onClick={onCloseFloorPlan} size="middle">
              关闭平面图
            </Button>
          </Space>
        </div>
        {/* 平面图编辑器容器 */}
        <div style={{ flex: 1, minHeight: 0 }}>
          {!selectedFloorForPlan ? (
            <div
              style={{
                textAlign: "center",
                padding: "60px",
                color: "var(--theme-text-tertiary, #999)",
                height: "100%",
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <BgColorsOutlined style={{ fontSize: "48px", marginBottom: "16px" }} />
              <div>请选择要查看的楼层平面图</div>
            </div>
          ) : floorPlanWorkstations.length === 0 ? (
            <div
              style={{
                textAlign: "center",
                padding: "60px",
                color: "var(--theme-text-tertiary, #999)",
                height: "100%",
                display: "flex",
                flexDirection: "column",
                alignItems: "center",
                justifyContent: "center",
              }}
            >
              <DesktopOutlined style={{ fontSize: "48px", marginBottom: "16px" }} />
              <div>该楼层暂无工位数据</div>
            </div>
          ) : (
            <FloorPlanEditor
              key={selectedFloorForPlan}
              floorId={selectedFloorForPlan}
              workstations={floorPlanWorkstations}
              onUpdatePosition={onPositionUpdate}
              onEdit={onEdit}
            />
          )}
        </div>
      </div>

      {/* 右侧：工位列表 */}
      <Card
        title="工位列表"
        style={{ width: 320, flexShrink: 0, overflow: "hidden" }}
        bodyStyle={{ padding: 0, height: "calc(100% - 57px)", overflow: "auto" }}
      >
        <div style={{ padding: "8px" }}>
          {floorPlanWorkstations.length === 0 ? (
            <div style={{ textAlign: "center", padding: "20px", color: "var(--theme-text-tertiary, #999)" }}>暂无工位数据</div>
          ) : (
            <Space orientation="vertical" style={{ width: "100%" }} size={[8, 8]}>
              {floorPlanWorkstations.map((ws) => {
                const typeText = getWorkstationTypeText(ws.type);
                const _fullWorkstation = allWorkstations.find((w) => w.id === ws.id);
                return (
                  <Card
                    key={ws.id}
                    size="small"
                    hoverable
                    style={{ cursor: "pointer" }}
                    onClick={() => ws && onEdit(ws)}
                    bodyStyle={{ padding: "12px" }}
                  >
                    <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 8 }}>
                      <span style={{ fontWeight: "bold", fontSize: 14 }}>{ws.name}</span>
                      <Tag color={getWorkstationStatusColor(ws.status)} style={{ margin: 0 }}>
                        {getWorkstationStatusText(ws.status)}
                      </Tag>
                    </div>
                    <div style={{ fontSize: 12, color: "#666", marginBottom: 4 }}>
                      <div>类型：{typeText}</div>
                      <div>
                        位置：({Math.round(ws.x)}, {Math.round(ws.y)})
                      </div>
                    </div>
                  </Card>
                );
              })}
            </Space>
          )}
        </div>
      </Card>
    </div>
  );
}
