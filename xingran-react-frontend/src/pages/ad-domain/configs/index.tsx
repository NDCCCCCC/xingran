import { useState } from "react";
// Phase 71 (W5)：裸 Input → 契约组件
import { AppInput } from "@/components/ui";
import { Button, Table, Modal, Form, InputNumber, Switch, Space, message, Card, Tag } from "antd";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  ReloadOutlined,
  EyeOutlined,
} from "@ant-design/icons";
import { Drawer, Tabs } from "antd";
import { useLocation } from "react-router-dom";
import AccountPoolTab from "./AccountPoolTab";
import ReconciliationTab from "./ReconciliationTab";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import type { ColumnsType } from "antd/es/table";
import {
  createADConfig,
  updateADConfig,
  deleteADConfig,
  testADConnection,
  syncADData,
  type ADSyncResult,
} from "@/lib/adDomainApi";
import ActionButtons from "@/components/shared/ActionButtons";
import { confirmDelete } from "@/components/shared/confirmDelete";
// ⚠️ 直接引模块路径（不走 @/components/shared 桶文件）：桶文件 re-export 了
// FloorPlanEditor → @react-three/fiber → three.js，引入它会让本页的动态 import
// （既有测试的 `await import(...)`）显著变慢并触发超时（实测）。
// useActionPermission 本身零依赖，直接引更轻。
import { useActionPermission } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { formatDateTime } from "@/utils/datetime";
import { useADConfigs } from "@/hooks/useADConfigs";
import type { ADConfig } from "@/lib/adDomainApi";

import type { FC } from "react";

const ADConfigPage: FC = () => {
  const [editForm] = Form.useForm();
  const location = useLocation();

  // Phase 59 / GUARD-02：页面级「配置列表 / 对账」Tab，activeTab 会话内持久化
  // （keyPrefix=当前路径，切回本页时停留在离开时的 Tab；与 settings-page 同款）
  const [activeTab, setActiveTab] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "activeTab",
    defaultValue: "configs",
  });

  // 使用共享的 AD 配置 Hook（fetchConfigs 接受排序参数，供服务端排序使用）
  const { configs, loading, fetchConfigs } = useADConfigs({
    enabledOnly: false, // 配置页显示所有配置（包括禁用状态）
    autoSelectFirst: false, // 不自动选择
  });

  const [modalVisible, setModalVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [editingConfig, setEditingConfig] = useState<ADConfig | null>(null);

  // Phase 36: 详情 Drawer（含服务账号池 Tab）
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [detailConfig, setDetailConfig] = useState<ADConfig | null>(null);

  const [testingConfig, setTestingConfig] = useState<string | null>(null);
  const [syncingConfig, setSyncingConfig] = useState<string | null>(null);
  const [syncProgress, setSyncProgress] = useState<{ [key: string]: ADSyncResult }>({});

  // 排序状态
  const [_orderByColumn, setOrderByColumn] = useState<string>("createdAt");
  const [_isAsc, setIsAsc] = useState<boolean>(false);

  // Phase 86 W4（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」
  //    （空权限集与「权限还没加载完」在数据上不可区分）。
  const permGate = useActionPermission();
  // 工具栏「新增AD配置」→ 后端 POST /configs（RequireStrictPermissions ops:ad:config:add）
  const permAdd = permGate.decide(PERMISSION_CODES.opsADConfig.add);
  // 编辑类（详情 Drawer 的「编辑此配置」+ 弹窗表单的三个 Switch）→ ops:ad:config:edit
  const permEdit = permGate.decide(PERMISSION_CODES.opsADConfig.edit);
  // ⚠️ 行操作「删除 / 测试连接 / 同步数据」不取 decision 对象 —— 它们是 ActionButtons 的
  // action，走组件内建的 `permission:` 字段三态门控（ready 无权限 → 该 action 不渲染；
  // 未就绪 → 渲染但 disabled + 提示），页面层不重复 decide。
  /**
   * 弹窗表单内的三个 Switch（使用SSL / 使用TLS / 启用自动同步）是**表单字段**，
   * 随「新增AD配置 / 编辑AD配置」弹窗提交生效 → **随弹窗上下文归所属写动作**：
   * 新增弹窗 → opsADConfig.add；编辑弹窗 → opsADConfig.edit。
   *
   * 口径沿用 86-01 system/menu 页的同一裁定（见其 permFormStatus）：
   * 若一律归 edit，则「有 add 无 edit」的用户新增配置时看不到 SSL/TLS/自动同步字段
   * → 提交时这三个字段缺失 → **功能退化**（新建的配置被迫走明文 + 不自动同步）。
   */
  const permFormWrite = editingConfig ? permEdit : permAdd;

  // 统一错误处理
  const handleApiError = (error: unknown, defaultMessage: string) => {
    if (error && typeof error === "object" && "message" in error) {
      message.error(error.message as string);
    } else {
      message.error(defaultMessage);
    }
  };

  const handleSuccess = (msg: string) => {
    message.success(msg);
  };

  const handleCreate = () => {
    setEditingConfig(null);
    setModalVisible(true);
    // 等待 Modal 打开后再设置表单值（在 Modal 的 afterOpenChange 中处理）
    // 默认值将在 Modal 打开后通过 editForm.setFieldsValue 设置
  };

  // Modal 打开后的回调
  const handleModalOpenChange = (open: boolean) => {
    if (open && editingConfig) {
      // 编辑模式：设置表单值（因为 destroyOnHide 会销毁 Form）
      editForm.setFieldsValue({
        ...editingConfig,
      });
    } else if (open && !editingConfig) {
      // 新增模式：设置默认值
      editForm.setFieldsValue({
        serverPort: 389,
        useSsl: false,
        useTls: false,
        syncEnabled: true,
        syncInterval: 3600,
      });
    }
  };

  const handleEdit = (config: ADConfig) => {
    setEditingConfig(config);
    // 账号管理通过详情 Drawer 的「服务账号池」Tab
    editForm.setFieldsValue({ ...config });
    setModalVisible(true);
  };

  // Phase 36: 打开详情 Drawer（含基本信息 + 服务账号池 Tab）
  const handleDetail = (config: ADConfig) => {
    setDetailConfig(config);
    setDetailDrawerVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await editForm.validateFields();

      if (editingConfig) {
        const updateData = { ...values };
        await updateADConfig(editingConfig.id, updateData);
        handleSuccess("更新成功");
      } else {
        await createADConfig(values);
        handleSuccess("创建成功");
      }

      setModalVisible(false);
      fetchConfigs();
    } catch (error) {
      handleApiError(error, "操作失败");
    }
  };

  const handleTest = async (configId: string) => {
    setTestingConfig(configId);
    try {
      await testADConnection(configId);
      handleSuccess("连接测试成功");
    } catch (error) {
      handleApiError(error, "连接测试失败");
    } finally {
      setTestingConfig(null);
    }
  };

  const handleSync = async (configId: string) => {
    setSyncingConfig(configId);
    try {
      const res = await syncADData(configId, "full");
      if (res.code === 0 && res.data?.result) {
        const result = res.data.result;
        handleSuccess(
          `同步成功: OU=${result.ouCount}, 用户组=${result.groupCount}, 用户=${result.userCount}`
        );
        setSyncProgress((prev) => ({ ...prev, [configId]: result }));
        fetchConfigs();
      }
    } catch (error) {
      handleApiError(error, "同步失败");
    } finally {
      setSyncingConfig(null);
    }
  };

  const handleDelete = async (configId: string) => {
    try {
      await deleteADConfig(configId);
      handleSuccess("删除成功");
      fetchConfigs();
    } catch (error) {
      handleApiError(error, "删除失败");
    }
  };

  const columns: ColumnsType<ADConfig> = [
    {
      title: "配置名称",
      dataIndex: "configName",
      key: "configName",
      width: 160,
      minWidth: 140,
      sorter: true,
    },
    {
      title: "服务器地址",
      dataIndex: "serverAddress",
      key: "serverAddress",
      width: 200,
      minWidth: 180,
      render: (text: string, record: ADConfig) => `${text}:${record.serverPort}`,
    },
    {
      title: "域名",
      dataIndex: "domainName",
      key: "domainName",
      width: 150,
      minWidth: 120,
      sorter: true,
    },
    {
      title: "基础DN",
      dataIndex: "baseDn",
      key: "baseDn",
      width: 200,
      minWidth: 180,
      ellipsis: true,
    },
    {
      title: "安全连接",
      key: "security",
      width: 140,
      minWidth: 120,
      render: (_: unknown, record: ADConfig) => (
        <Space>
          {record.useSsl && <Tag color="blue">SSL</Tag>}
          {record.useTls && <Tag color="green">TLS</Tag>}
          {!record.useSsl && !record.useTls && <Tag>Plain</Tag>}
        </Space>
      ),
    },
    {
      title: "自动同步",
      key: "syncEnabled",
      width: 140,
      minWidth: 120,
      render: (_: unknown, record: ADConfig) => (
        <Tag color={record.syncEnabled ? "success" : "default"}>
          {record.syncEnabled ? `是 (${record.syncInterval}s)` : "否"}
        </Tag>
      ),
    },
    {
      title: "最后同步",
      dataIndex: "lastSyncAt",
      key: "lastSyncAt",
      width: 170,
      minWidth: 160,
      render: (text: string | null) => formatDateTime(text),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      minWidth: 70,
      align: "center" as const,
      sorter: true,
      render: (status: number) => (
        <Tag color={status === 0 ? "success" : "default"}>{status === 0 ? "启用" : "停用"}</Tag>
      ),
    },
    {
      title: "同步结果",
      key: "syncResult",
      width: 150,
      minWidth: 130,
      render: (_: unknown, record: ADConfig) => {
        const result = syncProgress[record.id];
        if (!result) return "-";
        return (
          <Space orientation="vertical" size="small">
            <span>OU: {result.ouCount}</span>
            <span>用户组: {result.groupCount}</span>
            <span>用户: {result.userCount}</span>
          </Space>
        );
      },
    },
    {
      title: "操作",
      key: "action",
      width: 250,
      minWidth: 220,
      fixed: "right" as const,
      render: (_: unknown, record: ADConfig) => {
        const actions = [
          {
            key: "test",
            label: "测试连接",
            icon: <CheckCircleOutlined />,
            onClick: () => handleTest(record.id),
            loading: testingConfig === record.id,
            // Phase 86 W4：测试连接是写动作（触发 AD 域控连接测试）→ 挂 ops:ad:config:test
            permission: PERMISSION_CODES.opsADConfig.test,
          },
          {
            key: "sync",
            label: "同步数据",
            icon: <SyncOutlined />,
            onClick: () => handleSync(record.id),
            loading: syncingConfig === record.id,
            // Phase 86 W4：同步数据是写动作（拉取 AD 数据落库）→ 挂 ops:ad:config:sync
            permission: PERMISSION_CODES.opsADConfig.sync,
          },
          {
            key: "detail",
            label: "详情",
            icon: <EyeOutlined />,
            onClick: () => handleDetail(record),
            // 读入口（打开详情 Drawer，只读展示）→ **不门控**
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              confirmDelete({
                target: `AD 配置 ${record.configName}`,
                onDelete: () => handleDelete(record.id),
              });
            },
            // Phase 86 W4：删除是写动作 → 挂 ops:ad:config:delete。
            // 本行的触发器是 ActionButtons 的 action（内建三态门控位），
            // ready 无权限 → 该 action 不渲染；未就绪 → 渲染但 disabled + 提示。
            // confirmDelete.ts 本身零改动（它只是二次确认 helper，不承担权限判定）。
            permission: PERMISSION_CODES.opsADConfig.delete,
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <Card>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: "configs",
            label: "配置列表",
            children: (
              <>
                <div style={{ marginBottom: 16 }}>
                  <Space>
                    {/* Phase 86 W4：新增AD配置 → ops:ad:config:add（写入口 → 条件渲染 + disabled + 提示） */}
                    {permAdd.visible && (
                      <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={handleCreate}
                        disabled={permAdd.disabled}
                        title={permAdd.hint}
                      >
                        新增AD配置
                      </Button>
                    )}
                    {/* 刷新是读入口（重新拉列表，不落库）→ 不门控 */}
                    <Button icon={<ReloadOutlined />} onClick={() => fetchConfigs()}>
                      刷新
                    </Button>
                  </Space>
                </div>

                <Table
                  columns={columns}
                  dataSource={configs}
                  loading={loading}
                  rowKey="id"
                  pagination={false}
                  scroll={{ x: "max-content" }}
                  onChange={(pagination, filters, sorter) => {
                    // 用 local const 持有新值传 fetchConfigs，规避 React 18 setState 异步时序
                    // （fetchConfigs 不在 useMemo 依赖链，setState 后读 state 仍为旧值——commit 7ab1189 同类坑）
                    const s = sorter && !Array.isArray(sorter) ? sorter : null;
                    const col = s && s.field ? String(s.field) : "";
                    const asc = s ? s.order === "ascend" : false;
                    setOrderByColumn(col || "createdAt");
                    setIsAsc(col ? asc : false);
                    // fetchConfigs(sortColumn?, sortAsc?)：col 为空时传 undefined 走后端默认排序
                    fetchConfigs(col || undefined, col ? asc : undefined);
                  }}
                />

                <Modal
                  title={editingConfig ? "编辑AD配置" : "新增AD配置"}
                  open={modalVisible}
                  onOk={async () => {
                    setSubmitting(true);
                    try {
                      await handleSubmit();
                    } finally {
                      setSubmitting(false);
                    }
                  }}
                  cancelButtonProps={{ disabled: submitting }}
                  onCancel={() => setModalVisible(false)}
                  afterOpenChange={handleModalOpenChange}
                  width={600}
                  destroyOnHidden
                >
                  <Form form={editForm} layout="vertical" preserve={false}>
                    <Form.Item
                      label="配置名称"
                      name="configName"
                      rules={[{ required: true, message: "请输入配置名称" }]}
                    >
                      <AppInput allowClear placeholder="如: 公司主AD域" />
                    </Form.Item>

                    <Form.Item
                      label="服务器地址"
                      name="serverAddress"
                      rules={[{ required: true, message: "请输入AD服务器地址" }]}
                    >
                      <AppInput allowClear placeholder="如: ad.example.com 或 192.168.1.100" />
                    </Form.Item>

                    <Form.Item
                      label="端口"
                      name="serverPort"
                      rules={[{ required: true, message: "请输入端口" }]}
                    >
                      <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                    </Form.Item>

                    <Form.Item
                      label="域名"
                      name="domainName"
                      rules={[{ required: true, message: "请输入域名" }]}
                    >
                      <AppInput allowClear placeholder="如: example.com" />
                    </Form.Item>

                    <Form.Item
                      label="基础DN"
                      name="baseDn"
                      rules={[{ required: true, message: "请输入基础DN" }]}
                    >
                      <AppInput allowClear placeholder="如: DC=example,DC=com" />
                    </Form.Item>

                    {/* 账号管理改用「服务账号池」（详情 Drawer 的服务账号池 Tab） */}
                    {/* Phase 86 W4：三个 Switch 是弹窗表单字段，随弹窗提交生效 →
              随弹窗上下文归所属写动作（新增 → add / 编辑 → edit）；
              ready 无权限 → 整个 Switch 不渲染（不是只禁用）。 */}
                    {permFormWrite.visible && (
                      <Form.Item label="使用SSL (LDAPS)" name="useSsl" valuePropName="checked">
                        <Switch disabled={permFormWrite.disabled} title={permFormWrite.hint} />
                      </Form.Item>
                    )}

                    {permFormWrite.visible && (
                      <Form.Item label="使用TLS (StartTLS)" name="useTls" valuePropName="checked">
                        <Switch disabled={permFormWrite.disabled} title={permFormWrite.hint} />
                      </Form.Item>
                    )}

                    {permFormWrite.visible && (
                      <Form.Item label="启用自动同步" name="syncEnabled" valuePropName="checked">
                        <Switch disabled={permFormWrite.disabled} title={permFormWrite.hint} />
                      </Form.Item>
                    )}

                    <Form.Item
                      label="同步间隔(秒)"
                      name="syncInterval"
                      initialValue={3600}
                      rules={[{ required: true, message: "请输入同步间隔" }]}
                    >
                      <InputNumber min={60} style={{ width: "100%" }} />
                    </Form.Item>
                  </Form>
                </Modal>

                {/* Phase 36: 详情 Drawer（含「基本信息」+「服务账号池」Tab） */}
                <Drawer
                  title={detailConfig ? `AD 配置详情: ${detailConfig.configName}` : "AD 配置详情"}
                  open={detailDrawerVisible}
                  onClose={() => setDetailDrawerVisible(false)}
                  size={1100}
                  destroyOnHidden
                >
                  {detailConfig && (
                    <Tabs
                      defaultActiveKey="basic"
                      items={[
                        {
                          key: "basic",
                          label: "基本信息",
                          children: (
                            <div>
                              <p>
                                <strong>配置名：</strong>
                                {detailConfig.configName}
                              </p>
                              <p>
                                <strong>服务器：</strong>
                                {detailConfig.serverAddress}:{detailConfig.serverPort}
                              </p>
                              <p>
                                <strong>域名：</strong>
                                {detailConfig.domainName}
                              </p>
                              <p>
                                <strong>BaseDN：</strong>
                                {detailConfig.baseDn}
                              </p>
                              <p>
                                <strong>SSL/TLS：</strong>
                                SSL={String(detailConfig.useSsl)} / TLS=
                                {String(detailConfig.useTls)}
                              </p>
                              <p>
                                <strong>同步：</strong>
                                {detailConfig.syncEnabled ? "启用" : "停用"} / 间隔{" "}
                                {detailConfig.syncInterval}s
                              </p>
                              <p>
                                <strong>状态：</strong>
                                {detailConfig.status === 0 ? "启用" : "停用"}
                              </p>
                              {/*
                      Phase 86 W4（HIGH-1）：详情 Drawer 内的「编辑此配置」是**写入口**
                      —— 它关闭抽屉并打开编辑弹窗（handleEdit），用户随后可提交保存。
                      不门控 ⇒ 无 ops:ad:config:edit 的用户可「详情 → 编辑此配置 → 提交」
                      绕过整页门控。门控点落在按钮本身（条件渲染 + disabled + 提示）。
                      ⚠️ 详情 Drawer **本身不门控**（读入口）。
                    */}
                              {permEdit.visible && (
                                <Button
                                  type="primary"
                                  icon={<EditOutlined />}
                                  style={{ marginTop: 16 }}
                                  disabled={permEdit.disabled}
                                  title={permEdit.hint}
                                  onClick={() => {
                                    setDetailDrawerVisible(false);
                                    handleEdit(detailConfig);
                                  }}
                                >
                                  编辑此配置
                                </Button>
                              )}
                            </div>
                          ),
                        },
                        {
                          key: "accounts",
                          label: "服务账号池",
                          children: <AccountPoolTab configId={detailConfig.id} />,
                        },
                      ]}
                    />
                  )}
                </Drawer>
              </>
            ),
          },
          {
            key: "reconciliation",
            label: "对账",
            children: <ReconciliationTab />,
          },
        ]}
      />
    </Card>
  );
};

export default ADConfigPage;
