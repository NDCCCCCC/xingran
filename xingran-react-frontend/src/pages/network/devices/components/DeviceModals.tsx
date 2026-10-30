/**
 * Phase 67-T3: 设备三弹窗（快速创建 / 手动新增·编辑 / 详情）
 * 原 index.tsx :918-1277 纯搬迁——Modal 写法、字段、文案、注释逐字保留。
 * 注：快速创建/编辑两弹窗的「提交中禁用取消按钮」防抖是原页面对 Modal.onOk
 * 的直接控制，FormModal 契约不含此增强；保持原 Modal 形态（纯搬迁不动）。
 */
import { Modal, Form, InputNumber, Row, Col, Card, Divider, Alert, Button, Space } from "antd";
import type { DividerProps } from "antd/es/divider";
import { ApiOutlined, HistoryOutlined } from "@ant-design/icons";
import type { NetworkDevice } from "@/types";
import { DepartmentTreeSelect } from "@/components/shared";
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import type { ProbeResult } from "../hooks/useDeviceModals";
import { STATUS_OPTIONS, VENDOR_OPTIONS } from "../constants";
import { getOptionLabel } from "../utils";

export interface DeviceModalsProps {
  /* 快速创建 */
  quickCreateModalVisible: boolean;
  quickCreateForm: ReturnType<typeof Form.useForm>[0];
  credentials: Array<{ id: string; credentialName: string }>;
  probeResult: ProbeResult | null;
  probing: boolean;
  submitting: boolean;
  departments: React.ComponentProps<typeof DepartmentTreeSelect>["departments"];
  onProbe: () => void;
  onQuickCreateOk: () => Promise<void>;
  onQuickCreateCancel: () => void;
  onClearProbeResult: () => void;
  /* 编辑 */
  modalVisible: boolean;
  editingDevice: NetworkDevice | null;
  editForm: ReturnType<typeof Form.useForm>[0];
  deviceTypeOptions: Array<{ label: string; value: string | number }>;
  onEditOk: () => Promise<void>;
  onEditCancel: () => void;
  /* 详情 */
  detailModalVisible: boolean;
  viewingDevice: NetworkDevice | null;
  canViewMACHistory: boolean;
  navigate: (path: string) => void;
  getDeviceTypeText: (type: string) => string | undefined;
  onCloseDetail: () => void;
}

const DeviceModals: React.FC<DeviceModalsProps> = ({
  quickCreateModalVisible,
  quickCreateForm,
  credentials,
  probeResult,
  probing,
  submitting,
  departments,
  onProbe,
  onQuickCreateOk,
  onQuickCreateCancel,
  onClearProbeResult,
  modalVisible,
  editingDevice,
  editForm,
  deviceTypeOptions,
  onEditOk,
  onEditCancel,
  detailModalVisible,
  viewingDevice,
  canViewMACHistory,
  navigate,
  getDeviceTypeText,
  onCloseDetail,
}) => (
  <>
    {/* 快速创建模态框 */}
    <Modal
      title="快速创建设备"
      mask={{ closable: false }}
      keyboard={false}
      open={quickCreateModalVisible}
      onOk={onQuickCreateOk}
      cancelButtonProps={{ disabled: submitting }}
      onCancel={onQuickCreateCancel}
      width={700}
      okText="创建设备"
    >
      <Alert
        title="快速创建说明"
        description="只需输入 IP 地址和授权凭证，系统会自动使用凭证中配置的 SNMP Communities 探测获取设备信息并创建设备。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Form form={quickCreateForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
        <Form.Item
          name="ipAddress"
          label="IP地址"
          rules={[{ required: true, message: "请输入IP地址" }]}
        >
          <AppInput placeholder="请输入设备IP地址" />
        </Form.Item>
        <Form.Item
          name="credentialId"
          label="授权凭证"
          rules={[{ required: true, message: "请选择授权凭证" }]}
        >
          {/* UUID 炸弹防护：credentialId 为 UUID 外键，原无 allowClear，禁用清空避免 undefined 触发后端校验 400 */}
          <AppSelect
            placeholder="请选择授权凭证"
            onSearch={() => {}}
            allowClear={false}
            options={credentials.map((cred) => ({ value: cred.id, label: cred.credentialName }))}
          />
        </Form.Item>
        <Form.Item label=" ">
          <Col offset={6}>
            <Space>
              <Button
                aria-label="连接探测"
                title="连接探测"
                type="primary"
                icon={<ApiOutlined />}
                onClick={onProbe}
                loading={probing}
              >
                探测设备
              </Button>
              <Button onClick={onClearProbeResult}>清除结果</Button>
            </Space>
          </Col>
        </Form.Item>

        {probeResult && (
          <>
            <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>
              探测结果
            </Divider>
            <Alert title="探测成功" type="success" style={{ marginBottom: 16 }} />
            <Row gutter={16}>
              <Col span={12}>
                <Card size="small" title="设备信息">
                  <p>
                    <strong>设备名称:</strong> {(probeResult.deviceName as string) || "-"}
                  </p>
                  <p>
                    <strong>设备类型:</strong> {(probeResult.deviceType as string) || "-"}
                  </p>
                  <p>
                    <strong>厂商:</strong> {(probeResult.vendor as string) || "-"}
                  </p>
                  <p>
                    <strong>型号:</strong> {(probeResult.model as string) || "-"}
                  </p>
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small" title="系统信息">
                  <p>
                    <strong>SysName:</strong> {(probeResult.sysName as string) || "-"}
                  </p>
                  <p>
                    <strong>SysDescr:</strong>{" "}
                    {(probeResult.sysDescr as string)?.substring(0, 50) || "-"}
                    {(probeResult.sysDescr as string)?.length > 50 ? "..." : ""}
                  </p>
                </Card>
              </Col>
            </Row>
          </>
        )}

        <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>
          其他信息（可选）
        </Divider>
        <Form.Item name="deptId" label="所属部门">
          <DepartmentTreeSelect departments={departments} />
        </Form.Item>
        <Form.Item name="location" label="位置">
          <AppInput placeholder="请输入设备位置" />
        </Form.Item>
        <Form.Item name="description" label="备注">
          <AppTextArea rows={2} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>

    {/* 手动新增/编辑模态框 */}
    <Modal
      title={editingDevice ? "编辑设备" : "手动新增设备"}
      mask={{ closable: false }}
      keyboard={false}
      open={modalVisible}
      onOk={onEditOk}
      cancelButtonProps={{ disabled: submitting }}
      onCancel={onEditCancel}
      width={700}
    >
      <Alert
        title="手动新增说明"
        description="需要手动填写所有设备信息。如果只想输入IP地址自动获取信息，请使用「快速创建」功能。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Form form={editForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
        <Form.Item
          name="deviceName"
          label="设备名称"
          rules={[{ required: true, message: "请输入设备名称" }]}
        >
          <AppInput placeholder="请输入设备名称" />
        </Form.Item>
        <Form.Item
          name="deviceType"
          label="设备类型"
          rules={[{ required: true, message: "请选择设备类型" }]}
        >
          {/* 形态分类器①/②：children 写法改写为 options；4 项枚举 + 固定业务语义 → 显式禁搜 */}
          <AppSelect
            placeholder="请选择设备类型"
            onSearch={() => {}}
            showSearch={false}
            options={deviceTypeOptions}
          />
        </Form.Item>
        <Form.Item name="vendor" label="厂商" rules={[{ required: true, message: "请选择厂商" }]}>
          {/* 形态分类器①/②：children 改写 options；厂商为固定短枚举（5 项）→ 显式禁搜 */}
          <AppSelect
            placeholder="请选择厂商"
            onSearch={() => {}}
            showSearch={false}
            options={VENDOR_OPTIONS}
          />
        </Form.Item>
        <Form.Item name="model" label="型号">
          <AppInput placeholder="请输入设备型号" />
        </Form.Item>
        <Form.Item
          name="ipAddress"
          label="IP地址"
          rules={[{ required: true, message: "请输入IP地址" }]}
        >
          <AppInput placeholder="请输入IP地址" />
        </Form.Item>
        <Form.Item
          name="port"
          label="SSH端口"
          rules={[{ required: true, message: "请输入SSH端口" }]}
        >
          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item
          name="snmpPort"
          label="SNMP端口"
          rules={[{ required: true, message: "请输入SNMP端口" }]}
        >
          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="credentialId" label="授权凭证">
          {/* 形态分类器①/②：children 改写 options；原 allowClear=true 行为保持不变 */}
          <AppSelect
            placeholder="请选择授权凭证"
            allowClear
            onSearch={() => {}}
            options={credentials.map((cred) => ({ value: cred.id, label: cred.credentialName }))}
          />
        </Form.Item>
        <Form.Item name="deptId" label="所属部门">
          <DepartmentTreeSelect departments={departments} />
        </Form.Item>
        <Form.Item name="location" label="位置">
          <AppInput placeholder="请输入位置" />
        </Form.Item>
        <Form.Item name="status" label="状态" rules={[{ required: true }]}>
          {/* 形态分类器①/②：children 改写 options；状态类固定枚举 → 显式禁搜 */}
          <AppSelect onSearch={() => {}} showSearch={false} options={STATUS_OPTIONS} />
        </Form.Item>
        <Form.Item name="description" label="备注">
          <AppTextArea rows={3} placeholder="请输入备注" />
        </Form.Item>
      </Form>
    </Modal>

    {/* 设备详情模态框 */}
    <Modal
      title="设备详细信息"
      mask={{ closable: false }}
      keyboard={false}
      open={detailModalVisible}
      onCancel={onCloseDetail}
      footer={[
        // 14-05b:详情页头部操作区新增 查看 MAC 历史 联动入口(D-16 锁定)
        ...(canViewMACHistory && viewingDevice
          ? [
              <Button
                aria-label="查看 MAC 历史"
                title="查看 MAC 历史"
                key="mac-history"
                icon={<HistoryOutlined />}
                onClick={() => {
                  if (viewingDevice) {
                    navigate(`/network/mac/history?deviceId=${viewingDevice.id}`);
                  }
                }}
              >
                查看 MAC 历史
              </Button>,
            ]
          : []),
        <Button key="close" onClick={onCloseDetail}>
          关闭
        </Button>,
      ]}
      width={800}
    >
      {viewingDevice && (
        <div>
          <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>
            基本信息
          </Divider>
          <Row gutter={16}>
            <Col span={12}>
              <p>
                <strong>设备名称:</strong> {viewingDevice.deviceName}
              </p>
              <p>
                <strong>设备类型:</strong> {getDeviceTypeText(viewingDevice.deviceType) || "-"}
              </p>
              <p>
                <strong>厂商:</strong> {getOptionLabel(VENDOR_OPTIONS, viewingDevice.vendor) || "-"}
              </p>
              <p>
                <strong>型号:</strong> {viewingDevice.model || "-"}
              </p>
              <p>
                <strong>IP地址:</strong> {viewingDevice.ipAddress}
              </p>
              <p>
                <strong>SSH端口:</strong> {viewingDevice.port}
              </p>
              <p>
                <strong>SNMP端口:</strong> {viewingDevice.snmpPort}
              </p>
            </Col>
            <Col span={12}>
              <p>
                <strong>状态:</strong> {getOptionLabel(STATUS_OPTIONS, viewingDevice.status) || "-"}
              </p>
              <p>
                <strong>所属部门:</strong> {viewingDevice.deptName || "-"}
              </p>
              <p>
                <strong>位置:</strong> {viewingDevice.location || "-"}
              </p>
              <p>
                <strong>授权凭证:</strong> {viewingDevice.credentialName || "-"}
              </p>
              <p>
                <strong>最后连接:</strong> {viewingDevice.lastSeenAt || "-"}
              </p>
              <p>
                <strong>创建时间:</strong> {viewingDevice.createdAt}
              </p>
            </Col>
          </Row>

          <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>
            设备详细信息（SSH采集）
          </Divider>
          <Row gutter={16}>
            <Col span={12}>
              <p>
                <strong>型号:</strong> {viewingDevice.model || "-"}
              </p>
              <p>
                <strong>序列号:</strong> {viewingDevice.serialNumber || "-"}
              </p>
            </Col>
            <Col span={12}>
              <p>
                <strong>软件版本:</strong> {viewingDevice.softwareVersion || "-"}
              </p>
              <p>
                <strong>运行时间:</strong> {viewingDevice.uptime || "-"}
              </p>
            </Col>
          </Row>

          {viewingDevice.description && (
            <>
              <Divider orientationMargin={0} orientation={"left" as DividerProps["orientation"]}>
                备注
              </Divider>
              <p>{viewingDevice.description}</p>
            </>
          )}
        </div>
      )}
    </Modal>
  </>
);

export default DeviceModals;
