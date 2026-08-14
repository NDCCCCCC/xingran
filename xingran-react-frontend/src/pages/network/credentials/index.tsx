import type { FC } from "react";
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  Alert,
  App,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  KeyOutlined,
  ApiOutlined,
} from "@ant-design/icons";
import type { AuthCredential, PageResponse } from "@/types";
import { formatDateTime } from "@/utils/datetime";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import { useTableManager } from "@/hooks/useTableManager";
import { createSorterMeta } from "@/utils/tableHelpers";
import { withErrorHandling } from "@/utils/errorHandler";
import { useState, useEffect, useMemo } from "react";
import ActionButtons from "@/components/shared/ActionButtons";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal } from "@/components/shared";
import { usePagination } from "@/hooks/usePagination";

const { Option } = Select;
const { TextArea } = Input;

const CredentialManagement: FC = () => {
  const { message } = App.useApp();
  const [showPassword, setShowPassword] = useState(false);
  const [showEnablePassword, setShowEnablePassword] = useState(false);
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    ssh: 0,
    telnet: 0,
  });

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 必须与 columns 的 dataIndex 一致
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<AuthCredential>("credentialName"),
      createSorterMeta<AuthCredential>("protocolType"),
    ],
    []
  );

  const {
    loading,
    data: credentials,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem: editingCredential,
    setEditModalVisible,
    setEditingItem,
    loadData: loadCredentials,
    handleSearch,
    handleReset,
    handleRefresh,
    handleAdd,
    handleEdit,
    handleModalClose,
    handleTableChange: handleCredTableChange,
    getColumnSortOrder: getCredColumnSortOrder,
  } = useTableManager<AuthCredential>(
    async (params) => {
      const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
      // 映射搜索字段名到API字段名
      const searchParams: Record<string, unknown> = {
        current: params.current ?? 1,
        pageSize: params.pageSize ?? 10,
      };
      if (formValues.searchCredentialName) {
        searchParams.credentialName = formValues.searchCredentialName;
      }
      if (formValues.searchProtocolType) {
        searchParams.protocolType = formValues.searchProtocolType;
      }
      const result = await post<PageResponse<AuthCredential>>("/network/credentials/list", searchParams);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
      sorterMetas,
    }
  );

  // 加载统计数据(专用端点 COUNT 聚合,不受分页影响)
  const loadStatistics = async () => {
    try {
      const result = await post<{ total: number; ssh: number; telnet: number }>("/network/credentials/statistics");
      setStatistics({
        total: result.data?.total ?? 0,
        ssh: result.data?.ssh ?? 0,
        telnet: result.data?.telnet ?? 0,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  };

  useEffect(() => {
    Promise.all([loadCredentials(), loadStatistics()]);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mount-only fetch; loadCredentials/loadStatistics recreated each render
  }, []);

  // 创建/更新凭证
  const handleCreate = async () => {
    await withErrorHandling(
      async () => {
        const formValues = await editForm.validateFields();
        const values = formValues as Record<string, unknown> & { confirmPassword?: string };
        // 移除确认密码字段
        const { confirmPassword: _confirmPassword, ...data } = values;
        if (editingCredential) {
          await post(`/network/credentials/${editingCredential.id}/update`, data);
          return "更新成功";
        } else {
          await post("/network/credentials", data);
          return "创建成功";
        }
      },
      {
        onSuccess: () => {
          handleModalClose();
          setShowPassword(false);
          setShowEnablePassword(false);
          loadCredentials();
          loadStatistics();
        },
      }
    );
  };

  // 删除凭证
  const handleDelete = async (id: string) => {
    await withErrorHandling(
      async () => {
        await post(`/network/credentials/${id}/delete`, {});
        return "删除成功";
      },
      {
        onSuccess: () => {
          loadCredentials();
          loadStatistics();
        },
      }
    );
  };

  // 批量删除凭证
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的数据");
      return;
    }
    await withErrorHandling(
      async () => {
        await post("/network/credentials/batch-delete", { ids: selectedRowKeys });
        return "批量删除成功";
      },
      {
        onSuccess: () => {
          setSelectedRowKeys([]);
          loadCredentials();
          loadStatistics();
        },
      }
    );
  };

  const handleBatchExport = async (entityTypes: string[]) => {
    setBatchExporting(true);
    try {
      const filename = await batchExport(entityTypes, {}); // 可以根据需要添加筛选条件
      message.success(`批量导出成功，文件: ${filename}`);
      setBatchModalVisible(false);
    } catch (error: any) {
      message.error(`批量导出失败：${error.message}`);
    } finally {
      setBatchExporting(false);
    }
  };

  // 设置为默认凭证
  const handleSetDefault = async (id: string) => {
    await withErrorHandling(
      async () => {
        await post(`/network/credentials/${id}/set-default`, {});
        return "设置成功";
      },
      {
        onSuccess: () => {
          loadCredentials();
          loadStatistics();
        },
      }
    );
  };

  // 打开编辑模态框
  const _openModal = (record?: AuthCredential) => {
    if (record) {
      setEditingItem(record);
      editForm.setFieldsValue({
        ...record,
        // 确保 snmpVersion 有默认值
        snmpVersion: record.snmpVersion || "v2c",
      });
    } else {
      setEditingItem(null);
      editForm.resetFields();
      editForm.setFieldsValue({
        protocolType: "ssh",
        snmpVersion: "v2c",
      });
    }
    setEditModalVisible(true);
    setShowPassword(false);
    setShowEnablePassword(false);
  };

  // 协议类型选项
  const protocolTypeOptions = [
    { label: "SSH", value: "ssh" },
    { label: "Telnet", value: "telnet" },
  ];

  // SNMP版本选项
  const snmpVersionOptions = [
    { label: "v1", value: "v1" },
    { label: "v2c", value: "v2c" },
    { label: "v3", value: "v3" },
  ];

  // 表格列
  const columns: ColumnsType<AuthCredential> = [
    { title: "凭证名称", dataIndex: "credentialName", key: "credentialName", width: 180, sorter: true, sortOrder: getCredColumnSortOrder("credentialName") },
    {
      title: "协议类型",
      dataIndex: "protocolType",
      key: "protocolType",
      width: 100,
      sorter: true,
      sortOrder: getCredColumnSortOrder("protocolType"),
      render: (protocolType: string) => (
        <Tag color="blue">{protocolType?.toUpperCase()}</Tag>
      ),
    },
    { title: "用户名", dataIndex: "username", key: "username", width: 120 },
    {
      title: "密码",
      dataIndex: "password",
      key: "password",
      width: 100,
      render: (password: string) => password ? "******" : "-",
    },
    {
      title: "SNMP Communities",
      dataIndex: "snmpCommunities",
      key: "snmpCommunities",
      width: 200,
      render: (snmpCommunities: string[]) => (
        snmpCommunities && snmpCommunities.length > 0
          ? snmpCommunities.map((c, i) => <Tag key={i}>{c}</Tag>)
          : "-"
      ),
    },
    {
      title: "SNMP版本",
      dataIndex: "snmpVersion",
      key: "snmpVersion",
      width: 100,
      render: (snmpVersion: string) => snmpVersion?.toUpperCase() || "-",
    },
    {
      title: "默认凭证",
      dataIndex: "isDefault",
      key: "isDefault",
      width: 100,
      render: (isDefault: boolean) => (
        isDefault ? <Tag color="gold">默认</Tag> : <Tag>普通</Tag>
      ),
    },
    { title: "备注", dataIndex: "description", key: "description", width: 150, ellipsis: true },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 200,
      render: (_, record) => {
        const actions = [
          {
            key: "set-default",
            label: "设为默认",
            onClick: () => handleSetDefault(record.id),
            hidden: record.isDefault,
          },
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确认删除?",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            },
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];

  // 监听协议类型变化，动态显示特权密码字段
  const protocolType = Form.useWatch("protocolType", editForm);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="凭证总数"
              value={statistics.total}
              prefix={<KeyOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="SSH凭证"
              value={statistics.ssh}
              styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="Telnet凭证"
              value={statistics.telnet}
              styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="searchCredentialName" label="凭证名称">
              <Input placeholder="请输入凭证名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="searchProtocolType" label="协议类型">
              <Select placeholder="请选择协议类型" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
                {protocolTypeOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>新增</Button>
            {selectedRowKeys.length > 0 && (
              <Button icon={<DeleteOutlined />} style={{ color: "var(--theme-error, #ff4d4f)" }} onClick={handleBatchDelete}>
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <NetworkExport
              entityType="credentials"
              entityName="授权凭证"
              filters={Object.fromEntries(
                Object.entries(searchForm.getFieldsValue() as Record<string, unknown>).filter(([, v]) => v !== undefined && v !== null && v !== "")
              )}
              current={paginationProps?.current ?? 1}
              pageSize={paginationProps?.pageSize ?? 10}
            />
          </Space>{/* 批量导出 Modal */}

        <BatchExportModal

          visible={batchModalVisible}

          onConfirm={handleBatchExport}

          onCancel={() => setBatchModalVisible(false)}

          loading={batchExporting}

        />


        </div>
      </Card>

      {/* 凭证表格 */}
      <Card>
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
          columns={columns}
          dataSource={credentials}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1400 }}
          pagination={paginationProps}
          onChange={handleCredTableChange}
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={editingCredential ? "编辑授权凭证" : "新增授权凭证"}
        open={editModalVisible}
        onOk={handleCreate}
        onCancel={() => { handleModalClose(); setShowPassword(false); setShowEnablePassword(false); }}
        width={600}
      >
        <Alert
          message="提示"
          description="请选择协议类型（SSH 或 Telnet），并填写对应的配置信息。一个凭证可以同时包含 SSH/Telnet 配置和 SNMP 配置。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Form form={editForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="credentialName" label="凭证名称" rules={[{ required: true, message: "请输入凭证名称" }]}>
            <Input placeholder="请输入凭证名称" />
          </Form.Item>
          <Form.Item name="protocolType" label="协议类型" rules={[{ required: true, message: "请选择协议类型" }]}>
            <Select placeholder="请选择协议类型 (SSH 或 Telnet)" onSearch={() => {}}>
              {protocolTypeOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>

          {/* SSH/Telnet 字段 - 始终显示 */}
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: "请输入用户名" }]}>
            <Input placeholder="请输入用户名" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[{ required: !editingCredential, message: "请输入密码" }]}
            extra={editingCredential ? "留空表示不修改密码" : ""}
          >
            <Input.Password
              placeholder="请输入密码"
              visibilityToggle={{ visible: showPassword, onVisibleChange: setShowPassword }}
            />
          </Form.Item>
          {protocolType === "ssh" && (
            <Form.Item name="enablePassword" label="特权密码">
              <Input.Password
                placeholder="请输入特权密码（可选）"
                visibilityToggle={{ visible: showEnablePassword, onVisibleChange: setShowEnablePassword }}
              />
            </Form.Item>
          )}

          {/* SNMP 字段 - 可选 */}
          <Form.Item
            name="snmpCommunities"
            label="SNMP Communities"
            extra="用于设备状态检查和自动发现，可留空"
          >
            <Select
              mode="tags"
              placeholder="请输入 SNMP Community，可添加多个"
              tokenSeparators={[","]}
             onSearch={() => {}}>
            </Select>
          </Form.Item>
          <Form.Item name="snmpVersion" label="SNMP版本" rules={[{ required: true }]}>
            <Select onSearch={() => {}}>
              {snmpVersionOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="description" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default CredentialManagement;

