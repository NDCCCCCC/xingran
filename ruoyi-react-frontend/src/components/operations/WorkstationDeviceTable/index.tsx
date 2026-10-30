/**
 * 工位设备关联子表格组件
 *
 * 用于在工位列表中展示和管理关联的设备
 * 数据源逻辑：手动设备保存到数据库，AD/资产设备实时查询
 */

import React, { useState, useEffect, useCallback } from "react";
import {
  App,
  Table,
  Button,
  Space,
  Tag,
  Popconfirm,
  Modal,
  Form,
  Alert,
  Collapse,
  Tooltip,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  StarOutlined,
  HistoryOutlined,
  CheckCircleOutlined,
} from "@ant-design/icons";
import type { WorkstationDevice, DeviceFormData, Asset } from "@/types";
import { workstationDeviceApi, assetApi, type PhysicalDevicesResult } from "@/lib/opsApi";
import { HealthBadge } from "@/components/reconciliation";
import { confirmDelete } from "@/components/shared/confirmDelete";
// Phase 71 (W3)：裸 Input/Input.TextArea → AppInput/AppTextArea（本文件无 Select）。
import { AppInput, AppTextArea } from "@/components/ui";

export interface WorkstationDeviceTableProps {
  workstationId: string;
  expandable?: boolean;
  onDeviceChange?: () => void;
  // R4 (Phase 45) — 从父级 lift 注入的对账徽标数据(B6 修复 N+1)
  conflictTypeMap?: Map<string, string>;
  onBadgeClick?: (assetId: string, conflictType: string) => void;
}

/**
 * 物理链路设备面板（Phase 80 / DIAG-01）
 *
 * 渲染口径（**Table + Alert 双区，不渲染 Empty**）：
 *   - 有设备 → 渲染 Table；若同时存在 emptyReason（如 no_user_bound），
 *     在表格上方追加 warning Alert——「有数据但用户维度呈现仍受限」必须可见，
 *     否则用户看到设备却不知为何与预期不符。
 *   - 0 台 → 渲染 info Alert，文案取后端 emptyMessage（60-100 字，
 *     用 <Empty> 排版拥挤；本页已有 Alert 先例）。
 *   - warnings 任何组合下逐条可见。
 *   - diagnosis 逐段计数默认不展示，折叠进 Tooltip（排查「有 1 台但少了几台」）。
 *
 * 空因分级（生产实测 75.3% 工位属 no_user_bound，属常态而非异常）：
 *   no_user_bound → info（中性说明，不是错误）
 *   其余空因      → warning（需人工处理）
 */
const PhysicalDevicesPanel: React.FC<{
  devices: WorkstationDevice[];
  diagnosis: PhysicalDevicesResult | null;
  columns: ColumnsType<WorkstationDevice>;
}> = ({ devices, diagnosis, columns }) => {
  const emptyReason = diagnosis?.emptyReason ?? "";
  const emptyMessage = diagnosis?.emptyMessage ?? "";
  const warnings = diagnosis?.warnings ?? [];
  const hasDevices = devices.length > 0;

  // 空因分级：no_user_bound 是常态（工位未绑定使用人），用 info；其余用 warning
  const alertType = emptyReason === "no_user_bound" ? "info" : "warning";
  const alertTitle =
    emptyReason === "no_user_bound" ? "该工位未绑定使用人" : emptyReason ? "物理链路信息缺失" : "";

  const diagnosisTip = diagnosis?.diagnosis
    ? [
        `工位存在: ${diagnosis.diagnosis.workstationExists ? "是" : "否"}`,
        `已绑用户: ${diagnosis.diagnosis.userBound ? "是" : "否"}`,
        `信息点: ${diagnosis.diagnosis.infoPointActive}/${diagnosis.diagnosis.infoPointTotal} 可用`,
        `已绑端口: ${diagnosis.diagnosis.infoPointWithPort}`,
        `端口可解析: ${diagnosis.diagnosis.portResolved}`,
        `设备可解析: ${diagnosis.diagnosis.deviceResolved}`,
        `MAC 命中: ${diagnosis.diagnosis.macMatched}`,
        `资产命中: ${diagnosis.diagnosis.assetMatched}`,
        `域控命中: ${diagnosis.diagnosis.adMatched}`,
      ].join(" · ")
    : "";

  return (
    <div>
      {/* 空因/告警区：有设备时也要展示（根因可见），0 台时替代空白 */}
      {(emptyReason || warnings.length > 0) && (
        <Alert
          type={alertType}
          showIcon
          title={alertTitle}
          description={
            <div style={{ fontSize: 11 }}>
              {emptyMessage && (
                <div style={{ marginBottom: warnings.length > 0 ? 4 : 0 }}>{emptyMessage}</div>
              )}
              {warnings.length > 0 && (
                <ul style={{ margin: 0, paddingLeft: 16 }}>
                  {warnings.map((w, i) => (
                    <li key={i}>{w}</li>
                  ))}
                </ul>
              )}
            </div>
          }
          style={{ marginBottom: 4 }}
          action={
            diagnosisTip ? (
              <Tooltip title={diagnosisTip}>
                <Button type="link" size="small" style={{ fontSize: 11, height: "auto" }}>
                  诊断详情
                </Button>
              </Tooltip>
            ) : undefined
          }
        />
      )}

      {hasDevices ? (
        <Table
          columns={columns}
          dataSource={devices}
          rowKey="id"
          size="small"
          pagination={false}
          showHeader={false}
          bordered={false}
          style={{ fontSize: 11 }}
        />
      ) : (
        !emptyReason &&
        !warnings.length && (
          <div style={{ fontSize: 11, color: "var(--text-secondary)", padding: "4px 0" }}>
            该工位暂无通过 MAC→port→infoPoint 反推到的设备。请确认工位下是否已配置 ops_info_points
            关联到网络设备端口。
          </div>
        )
      )}
    </div>
  );
};

export const WorkstationDeviceTable: React.FC<WorkstationDeviceTableProps> = ({
  workstationId,
  onDeviceChange,
  conflictTypeMap,
  onBadgeClick,
}) => {
  // 分开存储四种来源的设备
  const { message } = App.useApp();
  const [manualDevices, setManualDevices] = useState<WorkstationDevice[]>([]);
  const [adDevices, setADDevices] = useState<WorkstationDevice[]>([]);
  const [assetDevices, setAssetDevices] = useState<WorkstationDevice[]>([]);
  // Phase 45 R5: 物理链路设备(MAC→port→infoPoint→workstation 反推,user-anchored)
  const [physicalDevices, setPhysicalDevices] = useState<WorkstationDevice[]>([]);
  // Phase 80 / DIAG-01: 物理链路空因诊断（emptyReason/emptyMessage/warnings/diagnosis）
  // 空因口径：「未绑定用户」压过「有数据」——未绑 user_id 时即使链路能召回设备，
  // 该面板按用户维度呈现仍无从归属，故 emptyReason=no_user_bound 属正常业务态。
  const [physicalDiagnosis, setPhysicalDiagnosis] = useState<PhysicalDevicesResult | null>(null);

  // 折叠面板展开状态
  const [adExpanded, setADExpanded] = useState(false);
  const [assetExpanded, setAssetExpanded] = useState(false);
  const [physicalExpanded, setPhysicalExpanded] = useState(false);

  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingDevice, setEditingDevice] = useState<WorkstationDevice | null>(null);
  const [autoFilledAsset, setAutoFilledAsset] = useState<Asset | null>(null);
  const [searchingSerial, setSearchingSerial] = useState(false);
  const [form] = Form.useForm();
  // 使用 Form.useWatch 订阅序列号字段，避免在未连接 Form 上调用 getFieldValue
  const deviceSerial = Form.useWatch("deviceSerial", form);

  // 注:手动添加的设备最多 1-2 条,无需分页;只有资产/域控/物理链路子表可能多行,
  //     这三张子表已配置 pagination={false},故此处删除分页状态以缩减组件高度。

  // 并行查询四种来源的设备（手动持久化 + AD/资产/物理链路实时查询）
  const fetchAllDevices = useCallback(async () => {
    setLoading(true);
    try {
      const [manualResult, adResult, assetResult, physicalResult] = await Promise.allSettled([
        workstationDeviceApi.getManual(workstationId),
        workstationDeviceApi.getAD(workstationId).catch(() => ({ data: [] })),
        workstationDeviceApi.getAsset(workstationId).catch(() => ({ data: [] })),
        // Phase 80 / DIAG-01：物理链路改走诊断端点，取回空因诊断信息
        workstationDeviceApi.getPhysicalWithDiagnosis(workstationId),
      ]);

      // 处理手动设备（持久化）
      if (manualResult.status === "fulfilled" && manualResult.value.data) {
        setManualDevices(manualResult.value.data);
      } else {
        setManualDevices([]);
      }

      // 处理 AD 设备（实时查询，失败时返回空数组）
      if (adResult.status === "fulfilled" && adResult.value.data) {
        setADDevices(adResult.value.data);
      } else {
        setADDevices([]);
      }

      // 处理资产设备（实时查询，失败时返回空数组）
      if (assetResult.status === "fulfilled" && assetResult.value.data) {
        setAssetDevices(assetResult.value.data);
      } else {
        setAssetDevices([]);
      }

      // Phase 80 / DIAG-01: 处理物理链路设备（诊断端点；失败时清空设备与诊断）
      if (physicalResult.status === "fulfilled" && physicalResult.value.data) {
        setPhysicalDevices(physicalResult.value.data.devices ?? []);
        setPhysicalDiagnosis(physicalResult.value.data);
      } else {
        setPhysicalDevices([]);
        setPhysicalDiagnosis(null);
      }

      // 任一关键查询失败时给出提示
      if (manualResult.status === "rejected") {
        console.error("[WorkstationDeviceTable] 加载手动设备失败:", manualResult.reason);
        message.error("加载设备失败");
      }
      if (adResult.status === "rejected") {
        console.warn("[WorkstationDeviceTable] 加载域控设备失败:", adResult.reason);
      }
      if (assetResult.status === "rejected") {
        console.warn("[WorkstationDeviceTable] 加载资产设备失败:", assetResult.reason);
      }
      if (physicalResult.status === "rejected") {
        // Phase 80 / DIAG-01: 原实现只 console.warn → 用户看到误导性的「0 台」空态，
        // 改为显式提示（静默失败是排障黑洞）。
        console.warn("[WorkstationDeviceTable] 加载物理链路设备失败:", physicalResult.reason);
        message.warning("物理链路设备查询失败，请稍后重试");
      }
    } catch (error) {
      // 记录错误便于排查（之前的实现吞掉了真实错误，导致"无声失败"）
      console.error("[WorkstationDeviceTable] 加载设备失败:", error);
      message.error("加载设备失败");
      setManualDevices([]);
      setADDevices([]);
      setAssetDevices([]);
      setPhysicalDevices([]);
      setPhysicalDiagnosis(null);
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [workstationId]);

  useEffect(() => {
    fetchAllDevices();
  }, [fetchAllDevices]);

  // 序列号搜索函数
  const handleSerialSearch = async (serial: string) => {
    if (!serial) {
      setAutoFilledAsset(null);
      return;
    }
    setSearchingSerial(true);
    try {
      const result = await assetApi.searchBySerial(serial);
      if (result.code === 0 && result.data) {
        setAutoFilledAsset(result.data);
        // 自动填充表单字段
        form.setFieldsValue({
          deviceName: result.data.devicesn,
          deviceModel: result.data.deviceModelName,
          deviceType: result.data.deviceTypeName,
          macAddress: result.data.mac1,
          responsibleUser: result.data.nowUserName,
        });
      } else {
        setAutoFilledAsset(null);
        // 清空自动填充字段
        form.setFieldsValue({
          deviceName: undefined,
          deviceModel: undefined,
          deviceType: undefined,
          macAddress: undefined,
          responsibleUser: undefined,
        });
      }
    } catch (_error) {
      setAutoFilledAsset(null);
      // 清空自动填充字段
      form.setFieldsValue({
        deviceName: undefined,
        deviceModel: undefined,
        deviceType: undefined,
        macAddress: undefined,
        responsibleUser: undefined,
      });
    } finally {
      setSearchingSerial(false);
    }
  };

  // 添加设备
  const handleAdd = () => {
    setEditingDevice(null);
    form.resetFields();
    setAutoFilledAsset(null);
    setModalVisible(true);
  };

  // 编辑设备
  const handleEdit = (device: WorkstationDevice) => {
    setEditingDevice(device);
    form.setFieldsValue({
      deviceSerial: device.deviceSerial,
      deviceName: device.deviceName,
      deviceModel: device.deviceModel,
      deviceType: device.deviceType,
      macAddress: device.macAddress,
      ipAddress: device.ipAddress,
      responsibleUser: device.responsibleUser,
      description: device.description,
    });
    setModalVisible(true);
  };

  // 删除设备
  // 68.1-T3：删除确认改走 confirmDelete（Modal 强确认），失败上抛由 helper 统一反馈
  const handleDelete = async (id: string) => {
    await workstationDeviceApi.delete(id);
    fetchAllDevices();
    onDeviceChange?.();
  };

  // 设置主设备（手动设备直接设置，AD/资产设备弹窗确认）
  const handleSetPrimary = (device: WorkstationDevice) => {
    const isManual = !device.id.startsWith("ad-") && !device.id.startsWith("asset-");

    if (isManual) {
      // 手动设备直接设置主设备
      setPrimaryDirect(device.id);
    } else {
      // AD/资产设备弹窗确认
      // eslint-disable-next-line no-restricted-syntax -- 非删除语义：设置主设备是正向指派（可再次更改），UI-SPEC §1.1 判定规则排除
      Modal.confirm({
        title: "设置主设备",
        content: "是否将此设备信息同步到数据库？同步后可手动编辑。",
        onOk: async () => {
          try {
            await workstationDeviceApi.setPrimaryAndSave(device.id, {
              workstationId,
              deviceSerial: device.deviceSerial || "",
              deviceName: device.deviceName || "",
              deviceModel: device.deviceModel ?? undefined,
              deviceType: device.deviceType ?? undefined,
              macAddress: device.macAddress ?? undefined,
              ipAddress: device.ipAddress ?? undefined,
              responsibleUser: device.responsibleUser ?? undefined,
            });
            message.success("设置成功");
            fetchAllDevices();
            onDeviceChange?.();
          } catch (error) {
            console.error("[WorkstationDeviceTable] 设置主设备失败:", error);
            message.error("设置失败");
          }
        },
      });
    }
  };

  // 直接设置主设备
  const setPrimaryDirect = async (id: string) => {
    try {
      await workstationDeviceApi.setPrimary(id);
      message.success("设置成功");
      fetchAllDevices();
      onDeviceChange?.();
    } catch (_error) {
      message.error("设置失败");
    }
  };

  // 提交表单
  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const data: DeviceFormData = {
        workstationId,
        deviceSerial: values.deviceSerial,
        deviceName: values.deviceName,
        deviceModel: values.deviceModel,
        deviceType: values.deviceType,
        macAddress: values.macAddress,
        ipAddress: values.ipAddress,
        responsibleUser: values.responsibleUser,
        description: values.description,
      };

      if (editingDevice) {
        await workstationDeviceApi.update(editingDevice.id, data);
        message.success("更新成功");
      } else {
        await workstationDeviceApi.addManual(data);
        message.success("添加成功");
      }

      setModalVisible(false);
      fetchAllDevices();
      onDeviceChange?.();
    } catch (_error) {
      message.error(editingDevice ? "更新失败" : "添加失败");
    }
  };

  // 通用表格列定义(紧凑布局:列宽收窄,字号 11)
  // showConfidence: 仅 R5 物理链路子表传 true,显示置信度徽标 + 历史时间提示
  const createColumns = (canEdit: boolean, showConfidence = false) => [
    {
      title: "设备名称",
      dataIndex: "deviceName",
      key: "deviceName",
      width: 120,
      ellipsis: true,
    },
    {
      title: "序列号",
      dataIndex: "deviceSerial",
      key: "deviceSerial",
      width: 100,
      ellipsis: true,
    },
    {
      title: "型号",
      dataIndex: "deviceModel",
      key: "deviceModel",
      width: 80,
      ellipsis: true,
    },
    {
      title: "MAC地址",
      dataIndex: "macAddress",
      key: "macAddress",
      width: 110,
      ellipsis: true,
    },
    {
      title: "IP地址",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 100,
      ellipsis: true,
    },
    {
      title: "责任人",
      dataIndex: "responsibleUser",
      key: "responsibleUser",
      width: 80,
      ellipsis: true,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 50,
      render: (status: number) => (
        <Tag
          color={status === 0 ? "success" : "error"}
          style={{ margin: 0, fontSize: 10, padding: "0 4px", lineHeight: "16px" }}
        >
          {status === 0 ? "正常" : "停用"}
        </Tag>
      ),
    },
    // R5 (2026-07-02) — 置信度列(仅物理链路子表显示):实测=绿色Tag,历史关联=橙色Tag+Tooltip
    ...(showConfidence
      ? [
          {
            title: "置信度",
            key: "confidence",
            width: 90,
            render: (_: unknown, record: WorkstationDevice) => {
              // 非物理链路或未分级:不渲染(物理链路以外的数据 confidence 为 undefined)
              if (record.confidence === undefined || record.confidence === null) {
                return null;
              }
              if (record.confidence >= 1.0) {
                // 实测 MAC 命中:绿色 "实测"
                return (
                  <Tag
                    icon={<CheckCircleOutlined />}
                    color="success"
                    style={{ margin: 0, fontSize: 10, padding: "0 4px", lineHeight: "16px" }}
                  >
                    实测
                  </Tag>
                );
              }
              // 仅历史 MAC 命中:橙色 "历史关联" + Tooltip 显示最后上线时间
              const tip = record.historyLastSeen
                ? `最后上线时间: ${record.historyLastSeen}`
                : "设备已离线,基于历史 MAC 关联";
              return (
                <Tooltip title={tip}>
                  <Tag
                    icon={<HistoryOutlined />}
                    color="warning"
                    style={{
                      margin: 0,
                      fontSize: 10,
                      padding: "0 4px",
                      lineHeight: "16px",
                      cursor: "help",
                    }}
                  >
                    历史关联
                  </Tag>
                </Tooltip>
              );
            },
          },
        ]
      : []),
    // R4 (Phase 45) — 对账健康列(行内徽标,从父级 lift 的 conflictTypeMap 取)
    {
      title: "对账健康",
      key: "reconciliation",
      width: 80,
      render: (_: unknown, record: WorkstationDevice) => {
        const aid = record.assetId;
        const ct = (aid && conflictTypeMap?.get(aid)) || null;
        return (
          <HealthBadge
            assetId={aid ?? ""}
            conflictType={ct}
            onClick={(id, t) => onBadgeClick?.(id, t)}
          />
        );
      },
    },
    {
      title: "主设备",
      dataIndex: "isPrimary",
      key: "isPrimary",
      width: 50,
      align: "center" as const,
      render: (isPrimary: boolean) =>
        isPrimary ? <StarOutlined style={{ color: "var(--theme-warning)", fontSize: 11 }} /> : null,
    },
    {
      title: "操作",
      key: "action",
      width: canEdit ? 110 : 80,
      render: (_: unknown, record: WorkstationDevice) => (
        <Space size={2}>
          {canEdit && (
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
              style={{ padding: 0, fontSize: 11, height: "auto" }}
            >
              编辑
            </Button>
          )}
          {canEdit && (
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              style={{ padding: 0, fontSize: 11, height: "auto" }}
              onClick={() =>
                confirmDelete({
                  target: `该设备绑定关系（${record.deviceName || record.id}）`,
                  onDelete: () => handleDelete(record.id),
                })
              }
            >
              删除
            </Button>
          )}
          {!record.isPrimary && (
            // eslint-disable-next-line no-restricted-syntax -- 非删除语义：设为主设备为正向指派，非危险删除，保留 Popconfirm 轻确认
            <Popconfirm
              title="设为主设备？"
              onConfirm={() => handleSetPrimary(record)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="link"
                size="small"
                icon={<StarOutlined />}
                style={{ padding: 0, fontSize: 11, height: "auto" }}
              >
                主设备
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  // 注:删除 currentPageData 分页切片,直接渲染 manualDevices(最多 1-2 条无需分页)

  return (
    <div style={{ padding: "4px 8px", backgroundColor: "var(--theme-bg-secondary)" }}>
      <Space style={{ marginBottom: 4 }} size="small">
        <Button type="primary" size="small" icon={<PlusOutlined />} onClick={handleAdd}>
          手动添加
        </Button>
      </Space>

      {/* 手动添加的设备 */}
      <Table
        // title={() => <span style={{ fontWeight: 600, fontSize: 13 }}>手动添加的设备</span>}
        columns={createColumns(true)}
        dataSource={manualDevices}
        loading={loading}
        rowKey="id"
        size="small"
        pagination={false}
        bordered={false}
        style={{ fontSize: 11 }}
      />

      {/* AD设备（折叠面板） */}
      {adDevices.length > 0 && (
        <Collapse
          style={{ marginTop: 4, backgroundColor: "transparent" }}
          activeKey={adExpanded ? ["ad"] : []}
          onChange={(keys) => setADExpanded(keys.includes("ad"))}
          size="small"
          bordered={false}
          items={[
            {
              key: "ad",
              label: <span style={{ fontSize: 12 }}>域控设备（{adDevices.length}台）</span>,
              children: (
                <Table
                  columns={createColumns(false)}
                  dataSource={adDevices}
                  rowKey="id"
                  size="small"
                  pagination={false}
                  showHeader={false}
                  bordered={false}
                  style={{ fontSize: 11 }}
                />
              ),
            },
          ]}
        />
      )}

      {/* 资产设备（折叠面板） */}
      {assetDevices.length > 0 && (
        <Collapse
          style={{ marginTop: 2, backgroundColor: "transparent" }}
          activeKey={assetExpanded ? ["asset"] : []}
          onChange={(keys) => setAssetExpanded(keys.includes("asset"))}
          size="small"
          bordered={false}
          items={[
            {
              key: "asset",
              label: <span style={{ fontSize: 12 }}>资产设备（{assetDevices.length}台）</span>,
              children: (
                <Table
                  columns={createColumns(false)}
                  dataSource={assetDevices}
                  rowKey="id"
                  size="small"
                  pagination={false}
                  showHeader={false}
                  bordered={false}
                  style={{ fontSize: 11 }}
                />
              ),
            },
          ]}
        />
      )}

      {/* Phase 45 R5: 物理链路设备（折叠面板） — MAC→port→infoPoint→workstation 反推 */}
      {/* Phase 45 R5: 物理链路设备(0 行也显示,跟 AD/资产保持一致便于排错) */}
      <Collapse
        style={{ marginTop: 2, backgroundColor: "transparent" }}
        activeKey={physicalExpanded ? ["physical"] : []}
        onChange={(keys) => setPhysicalExpanded(keys.includes("physical"))}
        size="small"
        bordered={false}
        items={[
          {
            key: "physical",
            label: <span style={{ fontSize: 12 }}>物理链路设备（{physicalDevices.length}台）</span>,
            children: (
              <PhysicalDevicesPanel
                devices={physicalDevices}
                diagnosis={physicalDiagnosis}
                columns={createColumns(false, true)}
              />
            ),
          },
        ]}
      />

      <Modal
        title={editingDevice ? "编辑设备" : "添加设备"}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        destroyOnHidden
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="deviceSerial"
            label="序列号"
            rules={[{ required: true, message: "请输入序列号" }]}
            extra={
              searchingSerial
                ? "正在查询资产信息..."
                : autoFilledAsset
                  ? "已从资产系统自动匹配"
                  : "未在资产系统中找到"
            }
          >
            <AppInput
              placeholder="请输入设备序列号"
              onBlur={(e) => handleSerialSearch(e.target.value)}
              disabled={!!editingDevice}
            />
          </Form.Item>

          {autoFilledAsset && (
            <div
              style={{
                marginBottom: 16,
                padding: 12,
                background: "var(--theme-bg-secondary)",
                borderRadius: 4,
              }}
            >
              <div style={{ marginBottom: 8, fontWeight: "bold" }}>自动匹配的设备信息：</div>
              <div>设备名称: {autoFilledAsset.devicesn || "-"}</div>
              <div>设备型号: {autoFilledAsset.deviceModelName || "-"}</div>
              <div>设备类型: {autoFilledAsset.deviceTypeName || "-"}</div>
              <div>MAC地址: {autoFilledAsset.mac1 || "-"}</div>
              <div>责任人: {autoFilledAsset.nowUserName || "-"}</div>
            </div>
          )}

          {!autoFilledAsset && deviceSerial && (
            <Alert
              title="未找到资产信息"
              description="该序列号在资产系统中不存在，您仍可以添加设备，但需要手动填写设备信息"
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />
          )}

          <Form.Item name="description" label="备注">
            <AppTextArea placeholder="请输入备注（可选）" rows={2} />
          </Form.Item>

          <Form.Item
            name="ipAddress"
            label="IP地址"
            rules={[
              {
                pattern: /^(\d{1,3}\.){3}\d{1,3}$/,
                message: "请输入有效的IPv4地址",
              },
            ]}
          >
            <AppInput placeholder="请输入IP地址（可选，如 192.168.1.10）" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default WorkstationDeviceTable;
