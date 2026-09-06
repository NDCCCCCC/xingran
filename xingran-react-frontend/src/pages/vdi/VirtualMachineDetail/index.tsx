import React, { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { Tabs, Button, Space, Tag, Card, Descriptions, App } from "antd";
import { ArrowLeftOutlined, ReloadOutlined } from "@ant-design/icons";
import { vmApi } from "@/lib/vdiApi";
import type { VirtualMachine } from "@/types/vdi";

const VirtualMachineDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const [vm, setVM] = useState<VirtualMachine | null>(null);
  const [activeTab, setActiveTab] = useState("overview");

  // 加载虚拟机详情
  const loadVMDetail = async () => {
    try {
      const result = await vmApi.get(id!);
      if (result.data) {
        setVM(result.data);
      }
    } catch (_error) {
      message.error("加载虚拟机详情失败");
    }
  };

  useEffect(() => {
    if (id) {
      loadVMDetail();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  if (!vm) {
    return <div>加载中...</div>;
  }

  return (
    <Card>
      <Space orientation="vertical" size="large" style={{ width: "100%" }}>
        {/* 头部 */}
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate("/vdi/vm")}>
            返回列表
          </Button>
          <Button icon={<ReloadOutlined />} onClick={loadVMDetail}>
            刷新
          </Button>
        </Space>

        {/* 标签页 */}
        <Tabs activeKey={activeTab} onChange={(key) => setActiveTab(key)}>
          {/* 概览标签页 */}
          <Tabs.TabPane tab="概览" key="overview">
            <Descriptions title="虚拟机信息" bordered column={2}>
              <Descriptions.Item label="虚拟机 ID">{vm.vm_id}</Descriptions.Item>
              <Descriptions.Item label="名称">{vm.name}</Descriptions.Item>
              <Descriptions.Item label="电源状态">
                <Tag
                  color={
                    vm.power_state === "in_use"
                      ? "success"
                      : vm.power_state === "stopped"
                        ? "error"
                        : "warning"
                  }
                >
                  {vm.power_state === "in_use"
                    ? "运行中"
                    : vm.power_state === "stopped"
                      ? "已关机"
                      : "已休眠"}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="IP 地址">{vm.ip_address || "-"}</Descriptions.Item>
              <Descriptions.Item label="操作系统">{vm.os_type || "-"}</Descriptions.Item>
              <Descriptions.Item label="资源规格">
                {vm.cpu_number || 0}核 / {vm.memory || 0}MB / {vm.disk || 0}GB
              </Descriptions.Item>
              <Descriptions.Item label="绑定用户">{vm.bound_user_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="最后同步">
                {vm.last_sync_at ? new Date(vm.last_sync_at).toLocaleString("zh-CN") : "-"}
              </Descriptions.Item>
            </Descriptions>
          </Tabs.TabPane>

          {/* 操作记录标签页 */}
          <Tabs.TabPane tab="操作记录" key="operations">
            <div
              style={{
                padding: "20px",
                textAlign: "center",
                color: "var(--theme-text-tertiary, #999)",
              }}
            >
              操作记录功能（未来实现）
            </div>
          </Tabs.TabPane>

          {/* 监控标签页 */}
          <Tabs.TabPane tab="监控" key="monitor">
            <div
              style={{
                padding: "20px",
                textAlign: "center",
                color: "var(--theme-text-tertiary, #999)",
              }}
            >
              监控数据功能（未来实现）
            </div>
          </Tabs.TabPane>
        </Tabs>
      </Space>
    </Card>
  );
};

export default VirtualMachineDetail;
