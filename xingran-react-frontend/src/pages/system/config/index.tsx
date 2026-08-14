import type { FC } from "react";
import { useState, useCallback, useEffect, useMemo } from "react";
import {
  App,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  Alert,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  StopOutlined,
} from "@ant-design/icons";
import type { Config } from "@/types";
import { formatDateTime } from "@/utils/datetime";
import { post } from "@/lib/api";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import ActionButtons from "@/components/shared/ActionButtons";
import { refreshEncryptionConfig } from "@/lib/api";
import { createSorterMeta } from "@/utils/tableHelpers";

const { Option } = Select;
const { TextArea } = Input;

const ConfigManagement: FC = () => {
  const { message } = App.useApp();
  const [statistics, setStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 对应后端 configAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Config>("configName"),
      createSorterMeta<Config>("configKey"),
      createSorterMeta<Config>("configType"),
      createSorterMeta<Config>("createdAt", "date"),
    ],
    []
  );

  const {
    loading,
    data: configs,
    total: _total,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem: editingConfig,
    loadData: loadConfigs,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    getColumnSortOrder,
    handleTableChange,
  } = useTableManager<Config>(
    async (params): Promise<{ list: Config[]; total: number }> => {
      const values = searchForm.getFieldsValue() as Record<string, unknown>;
      const requestParams = {
        current: params.current || paginationProps.current,
        pageSize: params.pageSize || paginationProps.pageSize,
        ...values,
        // 服务端排序透传：useTableManager 经 params 携带 orderByColumn/isAsc，
        // 此前只取 current/pageSize + values 导致排序参数被丢弃。
        ...(params.orderByColumn
          ? { orderByColumn: params.orderByColumn, isAsc: params.isAsc }
          : {}),
      };

      const result = await post<{ list: Config[]; total: number }>(
        "/system/configs/list",
        requestParams
      );
      return result.data || { list: [], total: 0 };
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

  // 加载统计数据(专用 COUNT 端点,不受 MaxPageSize=100 钳制)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<{ total: number; active: number; inactive: number }>(
        "/system/configs/statistics"
      );
      setStatistics(result.data ?? { total: 0, active: 0, inactive: 0 });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  useEffect(() => {
    // 确保 paginationProps 已完全初始化后再加载数据
    if (paginationProps.current && paginationProps.pageSize) {
      loadConfigs();
      loadStatistics();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [paginationProps.current, paginationProps.pageSize]);

  // 创建参数配置
  const handleCreate = async () => {
    try {
      const values = (await editForm.validateFields()) as Record<string, any>;
      if (editingConfig) {
        // F-17 Phase 31: 后端 UpdateConfigRequest 现已接受 configKey 字段(可选)。
        // 显式传 configKey 让后端校验"系统内置参数键不可改",获得二次确认保护。
        // isSystem 仍移除(只读字段,不参与更新)。
        const { isSystem: _isSystem, ...updateValues } = values;
        await post(`/system/configs/${editingConfig.id}/update`, updateValues);

        // 任何配置更新后都刷新加密配置（避免前后端加密开关状态脱同步 — 见 .planning/debug/login-400-bad-request.md）
        // 此前只在 configKey === 'sys.request.encryption.enabled' 时刷新，
        // 但表单的 configKey 字段可编辑 + Vite HMR 可能重置模块，导致 5 月修复在生产链路中失效。
        try {
          await refreshEncryptionConfig();
        } catch (refreshError) {
          // refreshEncryptionConfig 内部已 try/catch 吞错，这里是兜底
          console.error("刷新加密配置失败:", refreshError);
        }
        handleSuccess("更新");
      } else {
        await post("/system/configs", values);
        handleSuccess("创建");
      }
      handleModalClose();
      loadConfigs();
      loadStatistics();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return; // 表单验证错误
      }
      handleApiError(error, "操作");
    }
  };

  // 删除参数配置
  const handleDelete = async (id: string, isSystem: number) => {
    if (isSystem === 1) {
      message.warning("系统内置参数不能删除");
      return;
    }
    try {
      await post(`/system/configs/${id}/delete`, {});
      handleSuccess("删除");
      loadConfigs();
      loadStatistics();
    } catch (error) {
      handleApiError(error, "删除");
    }
  };

  // 批量删除参数配置
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的数据");
      return;
    }
    try {
      await post("/system/configs/batch-delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      setSelectedRowKeys([]);
      loadConfigs();
      loadStatistics();
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  };

  // 刷新缓存
  const handleRefreshCache = async () => {
    try {
      await post("/system/configs/refresh-cache", {});
      handleSuccess("刷新缓存");
    } catch (error) {
      handleApiError(error, "刷新缓存");
    }
  };

  // 表格列
  const columns: ColumnsType<Config> = [
    {
      title: "参数名称",
      dataIndex: "configName",
      key: "configName",
      width: 160,
      minWidth: 140,
      sorter: true,
      sortOrder: getColumnSortOrder("configName"),
    },
    {
      title: "参数键名",
      dataIndex: "configKey",
      key: "configKey",
      width: 160,
      minWidth: 140,
      sorter: true,
      sortOrder: getColumnSortOrder("configKey"),
    },
    {
      title: "参数键值",
      dataIndex: "configValue",
      key: "configValue",
      width: 180,
      minWidth: 150,
      ellipsis: true,
    },
    {
      title: "系统内置",
      dataIndex: "isSystem",
      key: "isSystem",
      width: 100,
      minWidth: 90,
      align: "center" as const,
      render: (isSystem: number) => (
        <Tag color={isSystem === 1 ? "blue" : "default"}>{isSystem === 1 ? "是" : "否"}</Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "configType",
      key: "configType",
      width: 80,
      minWidth: 70,
      align: "center" as const,
      sorter: true,
      sortOrder: getColumnSortOrder("configType"),
      render: (configType: string) => (
        <Tag color={configType === "Y" ? "success" : "default"}>
          {configType === "Y" ? "是" : "否"}
        </Tag>
      ),
    },
    {
      title: "备注",
      dataIndex: "remark",
      key: "remark",
      width: 150,
      minWidth: 120,
      ellipsis: true,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
      render: (date: string) => formatDateTime(date),
    },
    {
      title: "操作",
      key: "action",
      width: 150,
      minWidth: 130,
      fixed: "right" as const,
      render: (_, record) => {
        const actions = [
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
            disabled: record.isSystem === 1,
            render: () => (
              <Popconfirm
                title="确认删除?"
                onConfirm={() => handleDelete(record.id, record.isSystem)}
              >
                <Button
                  type="link"
                  icon={<DeleteOutlined />}
                  style={{ color: "var(--theme-error, #ff4d4f)" }}
                  size="small"
                  disabled={record.isSystem === 1}
                >
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <div>
      {/* 统计卡片 - 只在总数大于10时显示 */}
      {statistics.total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Card>
              <Statistic
                title="总配置数"
                value={statistics.total}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="启用配置"
                value={statistics.active}
                styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="禁用配置"
                value={statistics.inactive}
                styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
                prefix={<StopOutlined />}
              />
            </Card>
          </Col>
        </Row>
      )}

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "flex-start",
            flexWrap: "wrap",
            gap: "16px",
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="configName" label="参数名称">
              <Input
                placeholder="请输入参数名称"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="configKey" label="参数键名">
              <Input
                placeholder="请输入参数键名"
                allowClear
                className="user-form-input"
                style={{ width: 150 }}
              />
            </Form.Item>
            <Form.Item name="configType" label="状态">
              <Select
                placeholder="请选择状态"
                allowClear
                className="user-form-input"
                style={{ width: 120 }}
                onSearch={() => {}}
              >
                <Option value="Y">是</Option>
                <Option value="N">否</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button
                  icon={<ReloadOutlined />}
                  onClick={() => {
                    loadConfigs();
                    loadStatistics();
                  }}
                >
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {selectedRowKeys.length > 0 && (
              <Button
                icon={<DeleteOutlined />}
                style={{ color: "var(--theme-error, #ff4d4f)" }}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增配置
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleRefreshCache}>
              刷新缓存
            </Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && (
          <Alert
            message={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个参数配置，
                <Button
                  type="link"
                  size="small"
                  onClick={() => setSelectedRowKeys([])}
                  style={{ padding: 0 }}
                >
                  取消选择
                </Button>
              </span>
            }
            type="info"
            showIcon
            style={{ marginTop: 12 }}
          />
        )}
      </Card>

      {/* 数据表格 */}
      <Card>
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
            getCheckboxProps: (record) => ({
              disabled: record.isSystem === 1,
            }),
          }}
          columns={columns}
          dataSource={configs}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
          onChange={handleTableChange}
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={editingConfig ? "编辑参数配置" : "新增参数配置"}
        open={editModalVisible}
        onOk={handleCreate}
        onCancel={handleModalClose}
        width={600}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="configName"
            label="参数名称"
            rules={[{ required: true, message: "请输入参数名称" }]}
          >
            <Input placeholder="请输入参数名称" className="user-form-input" />
          </Form.Item>
          <Form.Item
            name="configKey"
            label="参数键名"
            rules={[{ required: true, message: "请输入参数键名" }]}
            extra="建议格式：sys.user.initPassword"
          >
            <Input
              placeholder="请输入参数键名"
              disabled={!!editingConfig && editingConfig.isSystem === 1}
              className="user-form-input"
            />
          </Form.Item>
          <Form.Item
            name="configValue"
            label="参数键值"
            rules={[{ required: true, message: "请输入参数键值" }]}
          >
            <Input placeholder="请输入参数键值" className="user-form-input" />
          </Form.Item>
          <Form.Item name="configType" label="状态" rules={[{ required: true }]}>
            <Select className="user-form-input" onSearch={() => {}}>
              <Option value="Y">是</Option>
              <Option value="N">否</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="isSystem"
            label="系统内置"
            rules={[{ required: true }]}
            extra="系统内置参数不可删除"
          >
            <Select disabled={!!editingConfig} className="user-form-input" onSearch={() => {}}>
              <Option value={0}>否</Option>
              <Option value={1}>是</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ConfigManagement;
