/**
 * Phase 67 T2 — 配置备份页还原/版本/对比/内容四弹窗组件
 *
 * 由 index.tsx :737-1136 纯搬迁（versionColumns 随版本列表抽屉迁入）。
 * 68.1-T4：版本列表「确认恢复到此版本?」裸 Modal.confirm → confirmDelete §1.9 变体。
 */
import {
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Alert,
} from "antd";
// Phase 71 (W4)：裸 Select → 契约组件（children 形态改写 options / 复合子节点保留写法）
import { AppSelect } from "@/components/ui";
import type { ColumnsType } from "antd/es/table";
import type { RefObject } from "react";
import {
  DownloadOutlined,
  DiffOutlined,
  EyeOutlined,
  FolderOpenOutlined,
  FileTextOutlined,
} from "@ant-design/icons";
import type { ConfigBackup } from "@/types";
import type { ConfigRestoreTask, DeviceBackupGroup, DiffLine, DiffResult } from "../types";
import ActionButtons from "@/components/shared/ActionButtons";
import { confirmDelete, useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { formatDateTime } from "@/utils/datetime";

// 形态分类器⑤专用：还原版本下拉为「children + 复合子节点（Space/Tag/时间）」，
// 按 W1/W2 口径保留 children 写法（不改写为 options），故仍需 antd Select.Option。
// 注意：本文件已无裸 <Select> JSX（P65 selector 按 JSXOpeningElement 名匹配，
// Select.Option 为成员表达式，不在命中面），Select 仅用于取 Option。
const { Option } = Select;

// 备份类型选项（随 versionColumns 迁入）
const BACKUP_TYPE_OPTIONS = [
  { label: "自动备份", value: "auto" },
  { label: "手动备份", value: "manual" },
];

export interface RestoreModalsProps {
  /** 当前选中设备分组（还原 Modal / 版本抽屉 / 对比 Modal 共用） */
  selectedDeviceGroup: DeviceBackupGroup | null;
  setSelectedDeviceGroup: (group: DeviceBackupGroup | null) => void;
  // 还原选择 Modal
  restoreModalVisible: boolean;
  closeRestoreModal: () => void;
  restoreTaskId: string | null;
  restoreTask: ConfigRestoreTask | null;
  restoreResult: {
    sentLines?: number;
    totalLines?: number;
    hashMatched?: boolean;
    failedLine?: string;
  } | null;
  selectedRestoreBackup: ConfigBackup | null;
  openRestoreModal: (backup: ConfigBackup) => void;
  handleConfirmRestore: () => void;
  // 版本列表抽屉
  versionListDrawerVisible: boolean;
  closeVersionListDrawer: () => void;
  handleOpenDiffModal: (
    group: DeviceBackupGroup,
    backup1?: ConfigBackup,
    backup2?: ConfigBackup
  ) => void;
  handleRestore: (backup: ConfigBackup) => void;
  handleViewContent: (backup: ConfigBackup) => void;
  // 差异对比 Modal
  diffModalVisible: boolean;
  closeDiffModal: () => void;
  compareBackup1: ConfigBackup | null;
  compareBackup2: ConfigBackup | null;
  openDiffModal: (backup1: ConfigBackup, backup2: ConfigBackup) => void;
  diffResult: DiffResult | null;
  renderDiffLine: (line: DiffLine, side: "left" | "right", index: number) => React.ReactNode;
  leftScrollRef: RefObject<HTMLDivElement | null>;
  rightScrollRef: RefObject<HTMLDivElement | null>;
  handleLeftScroll: (e: React.UIEvent<HTMLDivElement>) => void;
  handleRightScroll: (e: React.UIEvent<HTMLDivElement>) => void;
  // 配置内容抽屉
  selectedBackup: ConfigBackup | null;
  contentDrawerVisible: boolean;
  closeContentDrawer: () => void;
  backupContent: string;
}

export default function RestoreModals(props: RestoreModalsProps) {
  const {
    selectedDeviceGroup,
    setSelectedDeviceGroup,
    restoreModalVisible,
    closeRestoreModal,
    restoreTaskId,
    restoreTask,
    restoreResult,
    selectedRestoreBackup,
    openRestoreModal,
    handleConfirmRestore,
    versionListDrawerVisible,
    closeVersionListDrawer,
    handleOpenDiffModal,
    handleRestore,
    handleViewContent,
    diffModalVisible,
    closeDiffModal,
    compareBackup1,
    compareBackup2,
    openDiffModal,
    diffResult,
    renderDiffLine,
    leftScrollRef,
    rightScrollRef,
    handleLeftScroll,
    handleRightScroll,
    selectedBackup,
    contentDrawerVisible,
    closeContentDrawer,
    backupContent,
  } = props;

  // Phase 86 W3（RBAC-02）：本组件内的写入口 = 「恢复到指定版本」（版本列表抽屉行操作）
  // 与还原 Modal 的「确认还原」页脚按钮 —— 两者都是**还原（恢复配置）**动作，
  // 归 `networkBackup.restore`（决策变量名 `permRestore`，**不得**叫 permDelete：
  // 本页无删除入口，误挂 `.delete` 会造成「持 restore 无 delete 被误拒」+
  // 「持 delete 无 restore 可越权还原」的双向错误）。
  const permGate = useActionPermission();
  const permRestore = permGate.decide(PERMISSION_CODES.networkBackup.restore);

  // 版本列表表格列（原 index.tsx :478-541，操作列含 B 类豁免确认「确认恢复到此版本?」原文保留）
  const versionColumns: ColumnsType<ConfigBackup> = [
    { title: "版本号", dataIndex: "version", key: "version", width: 80 },
    {
      title: "备份类型",
      dataIndex: "backupType",
      key: "backupType",
      width: 100,
      render: (backupType: string) => {
        const option = BACKUP_TYPE_OPTIONS.find((o) => o.value === backupType);
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
              // 68.1-T4 §1.9 非删除破坏性变体：恢复到指定版本
              // Phase 86 W3：`confirmDelete` 只是二次确认 helper，语义 = **还原**（非删除）
              confirmDelete({
                title: "确认恢复到此版本",
                target: `把设备当前配置恢复到版本 ${record.version}`,
                actionLabel: "恢复",
                actionContext: "恢复配置",
                contentText: "将{target}，操作不可恢复，是否继续？",
                successText: "已恢复配置",
                onDelete: async () => {
                  await handleRestore(record);
                },
              });
            },
            // Phase 86 W3：还原 → networkBackup.restore（**不是** 删除类码）
            permission: PERMISSION_CODES.networkBackup.restore,
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <>
      {/* 还原选择模态框（原 :737-903） */}
      <Modal
        title={
          <Space>
            <DownloadOutlined />
            <span>选择还原版本 - {selectedDeviceGroup?.deviceName}</span>
          </Space>
        }
        mask={{ closable: false }}
        keyboard={false}
        open={restoreModalVisible}
        onCancel={() => {
          closeRestoreModal();
          setSelectedDeviceGroup(null);
        }}
        footer={
          restoreTaskId
            ? [
                <Button
                  key="close"
                  onClick={() => {
                    closeRestoreModal();
                    setSelectedDeviceGroup(null);
                  }}
                >
                  关闭
                </Button>,
              ]
            : [
                <Button
                  key="cancel"
                  onClick={() => {
                    closeRestoreModal();
                    setSelectedDeviceGroup(null);
                  }}
                >
                  取消
                </Button>,
                // Phase 86 W3：还原 Modal 的「确认还原」= 还原确认（networkBackup.restore）
                // ready 无码 → 不渲染；未就绪 → 渲染但 disabled + 提示。
                permRestore.visible && (
                  <Button
                    key="confirm"
                    type="primary"
                    danger
                    onClick={handleConfirmRestore}
                    disabled={permRestore.disabled}
                    title={permRestore.hint}
                  >
                    确认还原
                  </Button>
                ),
              ]
        }
        width={800}
      >
        {restoreTaskId ? (
          // Phase 93: 异步恢复任务进度/结果（useRestoreTask 轮询驱动，D-19）
          <Card size="small" title="恢复任务进度">
            {restoreTask?.status === "failed" ? (
              <Alert
                type="error"
                showIcon
                title="恢复失败"
                description={
                  <div>
                    {restoreTask.errorMessage && <div>错误信息：{restoreTask.errorMessage}</div>}
                    {restoreResult?.failedLine && <div>失败行：{restoreResult.failedLine}</div>}
                  </div>
                }
              />
            ) : restoreTask?.status === "success" ? (
              <Alert
                type="success"
                showIcon
                title="恢复成功"
                description={
                  <div>
                    <div>
                      配置已下发 {restoreResult?.sentLines ?? restoreTask.sentLines ?? "-"} /{" "}
                      {restoreResult?.totalLines ?? restoreTask.totalLines ?? "-"} 行
                    </div>
                    {restoreResult?.hashMatched === false && (
                      <div style={{ color: "var(--theme-warning)" }}>
                        回读配置与备份存在差异，已记录警告
                      </div>
                    )}
                  </div>
                }
              />
            ) : (
              <div style={{ textAlign: "center", padding: "24px 0" }}>
                <Spin />
                <div style={{ marginTop: 12 }}>
                  {restoreTask?.status === "running" ? "配置下发中..." : "排队等待中..."}
                </div>
              </div>
            )}
          </Card>
        ) : (
          selectedDeviceGroup && (
            <div>
              <Descriptions column={2} style={{ marginBottom: 16 }} bordered size="small">
                <Descriptions.Item label="设备名称">
                  {selectedDeviceGroup.deviceName}
                </Descriptions.Item>
                <Descriptions.Item label="IP地址">
                  {selectedDeviceGroup.ipAddress}
                </Descriptions.Item>
                <Descriptions.Item label="备份总数">
                  {selectedDeviceGroup.backupCount} 份
                </Descriptions.Item>
                <Descriptions.Item label="已选版本">
                  <Tag color="blue">版本 {selectedRestoreBackup?.version}</Tag>
                </Descriptions.Item>
              </Descriptions>

              <div style={{ marginBottom: 16 }}>
                <Space>
                  <span>选择要还原的版本：</span>
                  {/* 形态分类器⑤：children + 复合子节点（Space/Tag），保留写法；
                      原 Select 无 showSearch（antd 单选默认关），显式 showSearch={false}
                      避免 AppSelect 在版本数 ≥8 时自动开搜索 + 复合 label 过滤失效 */}
                  <AppSelect
                    style={{ width: 400 }}
                    value={selectedRestoreBackup?.id}
                    onChange={(value) => {
                      const backup = selectedDeviceGroup.backups.find((b) => b.id === value);
                      if (backup) {
                        setSelectedDeviceGroup(selectedDeviceGroup);
                        openRestoreModal(backup);
                      }
                    }}
                    placeholder="请选择要还原的版本"
                    onSearch={() => {}}
                    showSearch={false}
                    optionFilterProp="label"
                  >
                    {selectedDeviceGroup.backups.map((backup) => (
                      <Option key={backup.id} value={backup.id}>
                        <Space>
                          <span>版本 {backup.version}</span>
                          <Tag color={backup.backupType === "auto" ? "blue" : "green"}>
                            {backup.backupType === "auto" ? "自动" : "手动"}
                          </Tag>
                          <span style={{ color: "var(--theme-text-tertiary)" }}>
                            {formatDateTime(backup.createdAt)}
                          </span>
                        </Space>
                      </Option>
                    ))}
                  </AppSelect>
                </Space>
              </div>

              {selectedRestoreBackup && (
                <Card size="small" title="选定版本详情">
                  <Descriptions column={2} size="small">
                    <Descriptions.Item label="版本号">
                      {selectedRestoreBackup.version}
                    </Descriptions.Item>
                    <Descriptions.Item label="备份类型">
                      <Tag color={selectedRestoreBackup.backupType === "auto" ? "blue" : "green"}>
                        {selectedRestoreBackup.backupType === "auto" ? "自动备份" : "手动备份"}
                      </Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="文件大小">
                      {(selectedRestoreBackup.backupSize / 1024).toFixed(2)} KB
                    </Descriptions.Item>
                    <Descriptions.Item label="创建时间">
                      {formatDateTime(selectedRestoreBackup.createdAt)}
                    </Descriptions.Item>
                    <Descriptions.Item label="创建人">
                      {selectedRestoreBackup.createdBy}
                    </Descriptions.Item>
                    <Descriptions.Item label="变更原因" span={2}>
                      {selectedRestoreBackup.changeReason || "-"}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>
              )}
            </div>
          )
        )}
      </Modal>

      {/* 版本列表抽屉（原 :905-961） */}
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
        onClose={() => {
          closeVersionListDrawer();
          setSelectedDeviceGroup(null);
        }}
        extra={
          selectedDeviceGroup &&
          selectedDeviceGroup.backups.length >= 2 && (
            <Button
              aria-label="对比备份"
              title="对比备份"
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
              <Descriptions.Item label="设备名称">
                {selectedDeviceGroup.deviceName}
              </Descriptions.Item>
              <Descriptions.Item label="IP地址">{selectedDeviceGroup.ipAddress}</Descriptions.Item>
              <Descriptions.Item label="备份总数">
                {selectedDeviceGroup.backupCount} 份
              </Descriptions.Item>
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

      {/* 差异对比模态框（原 :963-1104） */}
      <Modal
        title="配置差异对比"
        mask={{ closable: false }}
        keyboard={false}
        open={diffModalVisible}
        onCancel={closeDiffModal}
        footer={[
          <Button key="close" onClick={closeDiffModal}>
            关闭
          </Button>,
        ]}
        width={1400}
      >
        <div style={{ marginBottom: 16 }}>
          <DiffSelectRow
            selectedDeviceGroup={selectedDeviceGroup}
            compareBackup1={compareBackup1}
            compareBackup2={compareBackup2}
            openDiffModal={openDiffModal}
          />
        </div>

        {diffResult && (
          <div
            style={{
              height: 600,
              border: "1px solid var(--theme-border-primary)",
              borderRadius: 4,
              overflow: "hidden",
            }}
          >
            {/* 版本信息栏 */}
            <div style={{ display: "flex", borderBottom: "1px solid var(--theme-border-primary)" }}>
              <div
                style={{
                  flex: 1,
                  padding: "8px 16px",
                  background: "var(--theme-bg-secondary)",
                  borderRight: "1px solid var(--theme-border-primary)",
                  fontSize: 13,
                  fontWeight: 500,
                  color: "var(--theme-text-primary)",
                }}
              >
                {diffResult.oldVersion}
              </div>
              <div
                style={{
                  flex: 1,
                  padding: "8px 16px",
                  background: "var(--theme-bg-secondary)",
                  fontSize: 13,
                  fontWeight: 500,
                  color: "var(--theme-text-primary)",
                }}
              >
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
                  background: "var(--theme-bg-elevated)",
                  borderRight: "1px solid var(--theme-border-primary)",
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
                  background: "var(--theme-bg-elevated)",
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

      {/* 配置内容抽屉（原 :1106-1136） */}
      <Drawer
        title={
          <Space>
            <FileTextOutlined />
            <span>
              配置内容 - {selectedBackup?.deviceName || ""} (版本 {selectedBackup?.version || "-"})
            </span>
          </Space>
        }
        placement="right"
        size="large"
        open={contentDrawerVisible}
        onClose={() => {
          closeContentDrawer();
        }}
      >
        <pre
          style={{
            whiteSpace: "pre-wrap",
            wordBreak: "break-all",
            background: "var(--theme-bg-secondary)",
            padding: 16,
            borderRadius: 4,
            maxHeight: "calc(100vh - 100px)",
            overflow: "auto",
          }}
        >
          {backupContent}
        </pre>
      </Drawer>
    </>
  );
}

// 差异对比 Modal 顶部的双版本选择行（原 :977-1027，因依赖 compareBackup1/2 提为子组件）
function DiffSelectRow(props: {
  selectedDeviceGroup: DeviceBackupGroup | null;
  compareBackup1: ConfigBackup | null;
  compareBackup2: ConfigBackup | null;
  openDiffModal: (backup1: ConfigBackup, backup2: ConfigBackup) => void;
}) {
  const { selectedDeviceGroup, compareBackup1, compareBackup2, openDiffModal } = props;
  return (
    <Row gutter={16}>
      <Col span={10}>
        {/* 形态分类器②：children 文本改写 options；原无 showSearch（单选默认关）→ 显式禁搜 */}
        <AppSelect
          placeholder="选择第一个版本"
          style={{ width: "100%" }}
          value={compareBackup1?.id}
          onChange={(value) => {
            if (selectedDeviceGroup) {
              const backup = selectedDeviceGroup.backups.find((b) => b.id === value);
              if (backup && compareBackup2) {
                openDiffModal(backup, compareBackup2);
              }
            }
          }}
          onSearch={() => {}}
          showSearch={false}
          options={selectedDeviceGroup?.backups.map((backup) => ({
            value: backup.id,
            label: `版本 ${backup.version} - ${formatDateTime(backup.createdAt)}`,
          }))}
        />
      </Col>
      <Col span={4} style={{ textAlign: "center", lineHeight: "32px" }}>
        VS
      </Col>
      <Col span={10}>
        {/* 形态分类器②：children 文本改写 options；显式禁搜 */}
        <AppSelect
          placeholder="选择第二个版本"
          style={{ width: "100%" }}
          value={compareBackup2?.id}
          onChange={(value) => {
            if (selectedDeviceGroup) {
              const backup = selectedDeviceGroup.backups.find((b) => b.id === value);
              if (backup && compareBackup1) {
                openDiffModal(compareBackup1, backup);
              }
            }
          }}
          onSearch={() => {}}
          showSearch={false}
          options={selectedDeviceGroup?.backups.map((backup) => ({
            value: backup.id,
            label: `版本 ${backup.version} - ${formatDateTime(backup.createdAt)}`,
          }))}
        />
      </Col>
    </Row>
  );
}

export { DiffSelectRow };
