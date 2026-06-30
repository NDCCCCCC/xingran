/**
 * 配置备份管理页面 - 重构版本
 *
 * 通过提取 Hooks 和工具函数，将 1084 行的组件拆分为更小的模块：
 * - types.ts: 类型定义
 * - utils.ts: 工具函数 (diff 计算、分组逻辑)
 * - hooks/: 自定义 Hooks (数据管理、差异对比、弹窗管理)
 */

import { useState, useEffect, useMemo, type FC } from "react";
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Card,
  Tag,
  Drawer,
  Row,
  Col,
  Statistic,
  Descriptions,
  App,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  SaveOutlined,
  SearchOutlined,
  ReloadOutlined,
  DownloadOutlined,
  DiffOutlined,
  HistoryOutlined,
  CloudUploadOutlined,
  EyeOutlined,
  DatabaseOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  FileTextOutlined,
  ExportOutlined,
} from "@ant-design/icons";
import type { ConfigBackup } from "@/types";
import type { DeviceBackupGroup, DiffLine } from "./types";
import { useBackupData, useBackupDiff, useBackupModals } from "./hooks";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import ActionButtons from "@/components/shared/ActionButtons";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal } from "@/components/shared";
import { usePagination } from "@/hooks/usePagination";
import { useServerSort } from "@/hooks/useServerSort";
import { createSorterMeta } from "@/utils/tableHelpers";
import { formatDateTime } from "@/utils/datetime";

const { Option } = Select;
const { TextArea } = Input;

// 备份类型选项
const BACKUP_TYPE_OPTIONS = [
  { label: "自动备份", value: "auto" },
  { label: "手动备份", value: "manual" },
];

const ConfigBackupPage: FC = () => {
  const { message } = App.useApp();
  // 表单和状态
  const [backupForm] = Form.useForm();
  const [searchForm] = Form.useForm();
  const [selectedDeviceGroup, setSelectedDeviceGroup] = useState<DeviceBackupGroup | null>(null);

const [batchModalVisible, setBatchModalVisible] = useState(false);
const [batchExporting, setBatchExporting] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 与 columns.dataIndex 对齐(useServerSort 按 sorter.field 匹配)
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<DeviceBackupGroup>("deviceId"),
      createSorterMeta<DeviceBackupGroup>("deviceName"),
      createSorterMeta<DeviceBackupGroup>("createdAt", "date"),
    ],
    []
  );
  const { orderByColumn, isAsc, handleTableChange: handleBackupSortChange, sortOrder: backupSortOrder } = useServerSort<DeviceBackupGroup>({
    sorterMetas,
  });

  // 使用自定义 Hooks
  const {
    devices,
    deviceGroups,
    statistics,
    loading,
    total,
    loadDevices,
    loadBackups,
    loadStatistics,
  } = useBackupData({ current: paginationProps.current ?? 1, pageSize: paginationProps.pageSize ?? 10, searchForm });

  const {
    diffModalVisible,
    diffResult,
    compareBackup1,
    compareBackup2,
    leftScrollRef,
    rightScrollRef,
    openDiffModal,
    closeDiffModal,
    handleLeftScroll,
    handleRightScroll,
  } = useBackupDiff();

  const {
    backupModalVisible,
    restoreModalVisible,
    contentDrawerVisible,
    versionListDrawerVisible,
    selectedBackup,
    selectedRestoreBackup,
    selectedDeviceGroup: modalDeviceGroup,
    backupContent,
    openBackupModal,
    closeBackupModal,
    openRestoreModal,
    closeRestoreModal,
    openContentDrawer,
    closeContentDrawer,
    openVersionListDrawer,
    closeVersionListDrawer,
    handleBackup,
    handleRestore: confirmRestore,
  } = useBackupModals({
    onLoad: () => {
      loadBackups();
      loadStatistics();
    },
  });

  // 初始化加载
  useEffect(() => {
    Promise.all([loadBackups(), loadStatistics()]);
  }, [paginationProps.current, paginationProps.pageSize, loadBackups, loadStatistics]);

  // 打开备份弹窗
  const handleOpenBackupModal = () => {
    loadDevices();
    backupForm.resetFields();
    openBackupModal();
  };

  // 打开还原选择模态框
  const handleOpenRestoreModal = (group: DeviceBackupGroup) => {
    setSelectedDeviceGroup(group);
    openRestoreModal(group.latestBackup);
  };

  // 确认还原
  const handleConfirmRestore = () => {
    if (!selectedRestoreBackup) {
      message.warning("请选择要还原的版本");
      return;
    }
    Modal.confirm({
      title: "确认恢复配置",
      content: `确定要将 ${selectedDeviceGroup?.deviceName} 恢复到版本 ${selectedRestoreBackup.version} 吗？此操作将会覆盖设备当前配置！`,
      okText: "确认",
      cancelText: "取消",
      okType: "danger",
      onOk: () => confirmRestore(),
    });
  };

  // 打开版本列表抽屉
  const handleOpenVersionList = (group: DeviceBackupGroup) => {
    setSelectedDeviceGroup(group);
    openVersionListDrawer(group);
  };

  // 打开对比对话框
  const handleOpenDiffModal = (group: DeviceBackupGroup, backup1?: ConfigBackup, backup2?: ConfigBackup) => {
    setSelectedDeviceGroup(group);
    if (group.backups.length < 2) {
      message.info("该设备只有一份备份，无法对比");
      return;
    }
    // 默认：左边是次新版，右边是最新版
    const b1 = backup1 || group.backups[1];
    const b2 = backup2 || group.backups[0];
    openDiffModal(b1, b2);
  };

  // 查看配置内容
  const handleViewContent = async (backup: ConfigBackup) => {
    openContentDrawer(backup);
  };

  // 恢复配置
  const handleRestore = (backup: ConfigBackup) => {
    Modal.confirm({
      title: "确认恢复配置",
      content: `确定要恢复到版本 ${backup.version} 吗？`,
      okText: "确认",
      cancelText: "取消",
      okType: "danger",
      onOk: async () => {
        try {
          await post(`/network/backups/${backup.id}/restore`, {});
          message.success("恢复成功");
          loadBackups();
        } catch (error) {
          message.error("恢复失败");
        }
      },
    });
  };

  // 批量导出
  const handleBatchExport = async (entityTypes: string[]) => {
    setBatchExporting(true);
    try {
      const filename = await batchExport(entityTypes, {});
      message.success(`批量导出成功，文件: ${filename}`);
      setBatchModalVisible(false);
    } catch (error: any) {
      message.error(`批量导出失败：${error.message}`);
    } finally {
      setBatchExporting(false);
    }
  };

  // 渲染差异行
  const renderDiffLine = (line: DiffLine, side: "left" | "right", index: number) => {
    const lineNum = line.lineNum !== undefined ? String(line.lineNum).padStart(4, " ") : "    ";
    const isRemoved = line.type === "removed";
    const isAdded = line.type === "added";
    const isSame = line.type === "same";
    const isEmpty = line.type === "empty";

    // 计算背景色
    let backgroundColor = "transparent";
    if (isRemoved) backgroundColor = "#ffced0";
    else if (isAdded) backgroundColor = "#d4edda";
    else if (isSame) {
      const lines = side === "left" ? diffResult?.leftLines : diffResult?.rightLines;
      const prevLine = index > 0 ? lines![index - 1] : null;
      const nextLine = index < (lines!.length - 1) ? lines![index + 1] : null;
      if ((prevLine?.type === "removed" || prevLine?.type === "added" ||
           nextLine?.type === "removed" || nextLine?.type === "added")) {
        backgroundColor = "#f5f5f5";
      }
    }

    const textColor = isRemoved ? "#c92a2a" : isAdded ? "#2b7a41" : "var(--theme-text-tertiary, #8c8c8c)";
    const contentColor = isRemoved ? "#c92a2a" : isAdded ? "#2b7a41" : "var(--theme-text-primary, #262626)";

    if (isEmpty) {
      return (
        <div key={`${side}-${index}`} style={{ display: "flex", backgroundColor }}>
          <div style={{
            width: 50,
            padding: "0 8px",
            textAlign: "right",
            color: "var(--theme-text-tertiary, #8c8c8c)",
            userSelect: "none",
            flexShrink: 0,
            borderRight: "1px solid #e8e8e8",
          }}>
            {lineNum}{/* 批量导出 Modal */}

          <BatchExportModal

            visible={batchModalVisible}

            onConfirm={handleBatchExport}

            onCancel={() => setBatchModalVisible(false)}

            loading={batchExporting}

          />


          </div>
          <div style={{
            padding: "0 12px",
            whiteSpace: "pre",
            overflowX: "auto",
            flex: 1,
          }} />
        </div>
      );
    }

    return (
      <div key={`${side}-${index}`} style={{ display: "flex", backgroundColor }}>
        <div style={{
          width: 50,
          padding: "0 8px",
          textAlign: "right",
          color: textColor,
          userSelect: "none",
          flexShrink: 0,
          borderRight: "1px solid #e8e8e8",
        }}>
          {lineNum}
        </div>
        <div style={{
          padding: "0 12px",
          whiteSpace: "pre",
          overflowX: "auto",
          color: contentColor,
          flex: 1,
        }}>
          {line.content}
        </div>
      </div>
    );
  };

  // 设备列表表格列（按设备分组）
  const deviceColumns: ColumnsType<DeviceBackupGroup> = [
    {
      title: "设备名称",
      dataIndex: "deviceName",
      key: "deviceName",
      width: 200,
      sorter: true,
      sortOrder: orderByColumn === "deviceName" ? backupSortOrder : null,
    },
    {
      title: "IP地址",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 140,
    },
    {
      title: "备份总数",
      dataIndex: "backupCount",
      key: "backupCount",
      width: 100,
      render: (count: number) => (
        <Tag color="blue" icon={<DatabaseOutlined />}>{count} 份</Tag>
      ),
    },
    {
      title: "自动/手动",
      key: "backupTypeCount",
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tag color="blue" icon={<ClockCircleOutlined />}>{record.autoCount}</Tag>
          <Tag color="green" icon={<SaveOutlined />}>{record.manualCount}</Tag>
        </Space>
      ),
    },
    {
      title: "最新版本",
      dataIndex: "latestBackup",
      key: "latestVersion",
      width: 80,
      render: (backup: ConfigBackup) => <Tag>{backup.version}</Tag>,
    },
    {
      title: "最新备份时间",
      dataIndex: "latestBackup",
      key: "latestBackupTime",
      width: 150,
      render: (backup: ConfigBackup) => formatDateTime(backup.createdAt),
    },
    {
      title: "最新备份原因",
      dataIndex: "latestBackup",
      key: "latestReason",
      width: 150,
      ellipsis: true,
      render: (backup: ConfigBackup) => backup.changeReason || "-",
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 100,
      render: (_, record) => {
        const actions = [
          {
            key: "restore",
            label: "还原",
            icon: <DownloadOutlined />,
            onClick: () => handleOpenRestoreModal(record),
          },
          {
            key: "versions",
            label: "版本列表",
            icon: <HistoryOutlined />,
            onClick: () => handleOpenVersionList(record),
          },
          {
            key: "diff",
            label: "版本对比",
            icon: <DiffOutlined />,
            onClick: () => handleOpenDiffModal(record),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  // 版本列表表格列
  const versionColumns: ColumnsType<ConfigBackup> = [
    { title: "版本号", dataIndex: "version", key: "version", width: 80 },
    {
      title: "备份类型",
      dataIndex: "backupType",
      key: "backupType",
      width: 100,
      render: (backupType: string) => {
        const option = BACKUP_TYPE_OPTIONS.find(o => o.value === backupType);
        return <Tag color={backupType === "auto" ? "blue" : "green"}>{option?.label}</Tag>;
      },
    },
    {
      title: "文件大小",
      dataIndex: "backupSize",
      key: "backupSize",
      width: 100,
      render: (size: number) => {
        const kb = (size / 1024).toFixed(2);
        return `${kb} KB`;
      },
    },
    { title: "变更原因", dataIndex: "changeReason", key: "changeReason", ellipsis: true },
    { title: "创建人", dataIndex: "createdBy", key: "createdBy", width: 100 },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      render: (value: string | Date | null | undefined) => formatDateTime(value),
    },
    {
      title: "操作",
      key: "action",
      width: 100,
      fixed: "right",
      render: (_, record) => {
        const actions = [
          {
            key: "view",
            label: "查看",
            icon: <EyeOutlined />,
            onClick: () => handleViewContent(record),
          },
          {
            key: "restore",
            label: "恢复",
            icon: <DownloadOutlined />,
            onClick: () => {
              Modal.confirm({
                title: "确认恢复到此版本?",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleRestore(record),
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="备份总数"
              value={statistics.total}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="自动备份"
              value={statistics.auto}
              styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="手动备份"
              value={statistics.manual}
              styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
              prefix={<SaveOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="涉及设备"
              value={statistics.devices}
              prefix={<HistoryOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="deviceName" label="设备名称">
              <Input placeholder="请输入设备名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="backupType" label="备份类型">
              <Select placeholder="请选择备份类型" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
                {BACKUP_TYPE_OPTIONS.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={() => { loadBackups(); loadStatistics(); }}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={() => { loadBackups(); loadStatistics(); }}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<CloudUploadOutlined />} onClick={handleOpenBackupModal}>
              创建备份
            </Button>
            <NetworkExport
              entityType="backups"
              entityName="配置备份"
              filters={Object.fromEntries(
                Object.entries(searchForm.getFieldsValue() as Record<string, unknown>).filter(([, v]) => v !== undefined && v !== null && v !== "")
              )}
              current={paginationProps?.current ?? 1}
              pageSize={paginationProps?.pageSize ?? 10}
            />
          </Space>
        </div>
      </Card>

      {/* 设备列表表格 */}
      <Card>
        <Table
          columns={deviceColumns}
          dataSource={deviceGroups}
          loading={loading}
          rowKey="deviceId"
          scroll={{ x: 1200 }}
          pagination={paginationProps}
          onChange={(pagination, _filters, sorter) => {
            handleBackupSortChange(pagination, _filters, sorter);
            setCurrent(pagination.current ?? 1);
            setPageSize(pagination.pageSize ?? 10);
            const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
            const searchParams: Record<string, unknown> = {
              current: pagination.current ?? 1,
              pageSize: pagination.pageSize ?? 10,
              ...(orderByColumn ? { orderByColumn, isAsc } : {}),
            };
            Object.keys(formValues).forEach(key => {
              const value = formValues[key];
              if (value !== undefined && value !== null && value !== "") {
                searchParams[key] = value;
              }
            });
            loadBackups(searchParams);
          }}
        />
      </Card>

      {/* 创建备份模态框 */}
      <Modal
        title="创建配置备份"
        open={backupModalVisible}
        onOk={() => handleBackup(backupForm)}
        onCancel={() => closeBackupModal(backupForm)}
        width={600}
      >
        <Form form={backupForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="deviceIds" label="选择设备" rules={[{ required: true, message: "请选择设备" }]}>
            <Select
              mode="multiple"
              placeholder="请选择要备份的设备"
              showSearch
              optionFilterProp="children"
             onSearch={() => {}}>
              {devices.map(device => (
                <Option key={device.id} value={device.id}>
                  {device.deviceName} ({device.ipAddress})
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="changeReason" label="变更原因">
            <TextArea rows={3} placeholder="请输入变更原因" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 还原选择模态框 */}
      <Modal
        title={
          <Space>
            <DownloadOutlined />
            <span>选择还原版本 - {selectedDeviceGroup?.deviceName}</span>
          </Space>
        }
        open={restoreModalVisible}
        onCancel={() => {
          closeRestoreModal();
          setSelectedDeviceGroup(null);
        }}
        footer={[
          <Button key="cancel" onClick={() => { closeRestoreModal(); setSelectedDeviceGroup(null); }}>
            取消
          </Button>,
          <Button key="confirm" type="primary" danger onClick={handleConfirmRestore}>
            确认还原
          </Button>,
        ]}
        width={800}
      >
        {selectedDeviceGroup && (
          <div>
            <Descriptions column={2} style={{ marginBottom: 16 }} bordered size="small">
              <Descriptions.Item label="设备名称">{selectedDeviceGroup.deviceName}</Descriptions.Item>
              <Descriptions.Item label="IP地址">{selectedDeviceGroup.ipAddress}</Descriptions.Item>
              <Descriptions.Item label="备份总数">{selectedDeviceGroup.backupCount} 份</Descriptions.Item>
              <Descriptions.Item label="已选版本">
                <Tag color="blue">版本 {selectedRestoreBackup?.version}</Tag>
              </Descriptions.Item>
            </Descriptions>

            <div style={{ marginBottom: 16 }}>
              <Space>
                <span>选择要还原的版本：</span>
                <Select
                  style={{ width: 400 }}
                  value={selectedRestoreBackup?.id}
                  onChange={(value) =>    {
                    const backup = selectedDeviceGroup.backups.find(b => b.id === value);
                    if (backup) {
                      setSelectedDeviceGroup(selectedDeviceGroup);
                      openRestoreModal(backup);
                    }
                  }}
                  placeholder="请选择要还原的版本"
                 onSearch={() => {}}>
                  {selectedDeviceGroup.backups.map(backup => (
                    <Option key={backup.id} value={backup.id}>
                      <Space>
                        <span>版本 {backup.version}</span>
                        <Tag color={backup.backupType === "auto" ? "blue" : "green"}>
                          {backup.backupType === "auto" ? "自动" : "手动"}
                        </Tag>
                        <span style={{ color: "var(--theme-text-tertiary, #999)" }}>{formatDateTime(backup.createdAt)}</span>
                      </Space>
                    </Option>
                  ))}
                </Select>
              </Space>
            </div>

            {selectedRestoreBackup && (
              <Card size="small" title="选定版本详情">
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="版本号">{selectedRestoreBackup.version}</Descriptions.Item>
                  <Descriptions.Item label="备份类型">
                    <Tag color={selectedRestoreBackup.backupType === "auto" ? "blue" : "green"}>
                      {selectedRestoreBackup.backupType === "auto" ? "自动备份" : "手动备份"}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="文件大小">
                    {(selectedRestoreBackup.backupSize / 1024).toFixed(2)} KB
                  </Descriptions.Item>
                  <Descriptions.Item label="创建时间">{formatDateTime(selectedRestoreBackup.createdAt)}</Descriptions.Item>
                  <Descriptions.Item label="创建人">{selectedRestoreBackup.createdBy}</Descriptions.Item>
                  <Descriptions.Item label="变更原因" span={2}>
                    {selectedRestoreBackup.changeReason || "-"}
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            )}
          </div>
        )}
      </Modal>

      {/* 版本列表抽屉 */}
      <Drawer
        title={
          <Space>
            <FolderOpenOutlined />
            <span>{selectedDeviceGroup?.deviceName} - 配置版本列表</span>
            <Tag color="blue">{selectedDeviceGroup?.backupCount} 个版本</Tag>
          </Space>
        }
        placement="right"
        size="large"
        open={versionListDrawerVisible}
        onClose={() => { closeVersionListDrawer(); setSelectedDeviceGroup(null); }}
        extra={
          selectedDeviceGroup && selectedDeviceGroup.backups.length >= 2 && (
            <Button
              type="primary"
              icon={<DiffOutlined />}
              onClick={() => {
                closeVersionListDrawer();
                handleOpenDiffModal(selectedDeviceGroup);
              }}
            >
              版本对比
            </Button>
          )
        }
      >
        {selectedDeviceGroup && (
          <div>
            <Descriptions column={3} style={{ marginBottom: 16 }} bordered size="small">
              <Descriptions.Item label="设备名称">{selectedDeviceGroup.deviceName}</Descriptions.Item>
              <Descriptions.Item label="IP地址">{selectedDeviceGroup.ipAddress}</Descriptions.Item>
              <Descriptions.Item label="备份总数">{selectedDeviceGroup.backupCount} 份</Descriptions.Item>
            </Descriptions>

            <Table
              columns={versionColumns}
              dataSource={selectedDeviceGroup.backups}
              rowKey="id"
              pagination={false}
              size="small"
              scroll={{ x: 1000 }}
            />
          </div>
        )}
      </Drawer>

      {/* 差异对比模态框 */}
      <Modal
        title="配置差异对比"
        open={diffModalVisible}
        onCancel={closeDiffModal}
        footer={[
          <Button key="close" onClick={closeDiffModal}>关闭</Button>,
        ]}
        width={1400}
      >
        <div style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col span={10}>
              <Select
                placeholder="选择第一个版本"
                style={{ width: "100%" }}
                value={compareBackup1?.id}
                onChange={(value) =>    {
                  if (selectedDeviceGroup) {
                    const backup = selectedDeviceGroup.backups.find(b => b.id === value);
                    if (backup && compareBackup2) {
                      openDiffModal(backup, compareBackup2);
                    }
                  }
                }}
               onSearch={() => {}}>
                {selectedDeviceGroup?.backups.map(backup => (
                  <Option key={backup.id} value={backup.id}>
                    版本 {backup.version} - {formatDateTime(backup.createdAt)}
                  </Option>
                ))}
              </Select>
            </Col>
            <Col span={4} style={{ textAlign: "center", lineHeight: "32px" }}>
              VS
            </Col>
            <Col span={10}>
              <Select
                placeholder="选择第二个版本"
                style={{ width: "100%" }}
                value={compareBackup2?.id}
                onChange={(value) =>    {
                  if (selectedDeviceGroup) {
                    const backup = selectedDeviceGroup.backups.find(b => b.id === value);
                    if (backup && compareBackup1) {
                      openDiffModal(compareBackup1, backup);
                    }
                  }
                }}
               onSearch={() => {}}>
                {selectedDeviceGroup?.backups.map(backup => (
                  <Option key={backup.id} value={backup.id}>
                    版本 {backup.version} - {formatDateTime(backup.createdAt)}
                  </Option>
                ))}
              </Select>
            </Col>
          </Row>
        </div>

        {diffResult && (
          <div style={{ height: 600, border: "1px solid #d9d9d9", borderRadius: 4, overflow: "hidden" }}>
            {/* 版本信息栏 */}
            <div style={{ display: "flex", borderBottom: "1px solid #d9d9d9" }}>
              <div style={{
                flex: 1,
                padding: "8px 16px",
                background: "#f5f5f5",
                borderRight: "1px solid #d9d9d9",
                fontSize: 13,
                fontWeight: 500,
                color: "var(--theme-text-primary, #262626)",
              }}>
                {diffResult.oldVersion}
              </div>
              <div style={{
                flex: 1,
                padding: "8px 16px",
                background: "#f5f5f5",
                fontSize: 13,
                fontWeight: 500,
                color: "var(--theme-text-primary, #262626)",
              }}>
                {diffResult.newVersion}
              </div>
            </div>

            {/* 差异对比内容区 */}
            <div style={{ display: "flex", height: "calc(100% - 41px)" }}>
              {/* 左侧：旧版本 */}
              <div
                ref={leftScrollRef}
                onScroll={handleLeftScroll}
                style={{
                  flex: 1,
                  overflow: "auto",
                  background: "#ffffff",
                  borderRight: "1px solid #d9d9d9",
                  fontFamily: 'Consolas, "Courier New", monospace',
                  fontSize: 13,
                  lineHeight: "22px",
                }}
              >
                {diffResult.leftLines.map((line, index) => renderDiffLine(line, "left", index))}
              </div>

              {/* 右侧：新版本 */}
              <div
                ref={rightScrollRef}
                onScroll={handleRightScroll}
                style={{
                  flex: 1,
                  overflow: "auto",
                  background: "#ffffff",
                  fontFamily: 'Consolas, "Courier New", monospace',
                  fontSize: 13,
                  lineHeight: "22px",
                }}
              >
                {diffResult.rightLines.map((line, index) => renderDiffLine(line, "right", index))}
              </div>
            </div>
          </div>
        )}
      </Modal>

      {/* 配置内容抽屉 */}
      <Drawer
        title={
          <Space>
            <FileTextOutlined />
            <span>配置内容 - {selectedBackup?.deviceName || ""} (版本 {selectedBackup?.version || "-"})</span>
          </Space>
        }
        placement="right"
        size="large"
        open={contentDrawerVisible}
        onClose={() => { closeContentDrawer(); }}>
          <pre style={{
            whiteSpace: "pre-wrap",
            wordBreak: "break-all",
            background: "#f5f5f5",
            padding: 16,
            borderRadius: 4,
            maxHeight: "calc(100vh - 100px)",
            overflow: "auto",
          }}>
            {backupContent}
          </pre>
        </Drawer>
    </div>
  );
};

export default ConfigBackupPage;
