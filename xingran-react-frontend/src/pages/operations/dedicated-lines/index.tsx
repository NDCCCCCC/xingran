/**
 * 专线管理页面
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import type { FC } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  Table, Button, Space, Modal, Form, Input, InputNumber, Select,
  Tag, Card, Row, Col, Alert, Radio, Layout,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, WarningOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined, LineChartOutlined,
} from "@ant-design/icons";
import type { DedicatedLine } from "@/types";
import { dedicatedLineApi, serverRoomApi } from "@/lib/opsApi";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { useDict } from "@/hooks/useDict";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import ActionButtons from "@/components/shared/ActionButtons";
import ExcelImport from "@/components/shared/ExcelImport";
import ExcelExport from "@/components/shared/ExcelExport";
import { StatisticsCards } from "@/components/operations/StatisticsCards";

const { Option } = Select;
const { TextArea } = Input;
const { Content } = Layout;

interface ServerRoomOption {
  id: string;
  name: string;
  orgId: string;
}

type ViewMode = "table" | "card";

interface Statistics {
  total: number;
  normal: number;
  fault: number;
  disabled: number;
}

const DedicatedLineManagement: FC = () => {
  const location = useLocation();
  const [viewMode, setViewMode] = usePersistedStateController<ViewMode>({
    keyPrefix: location.pathname,
    keySuffix: "viewMode",
    defaultValue: "table",
  });
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});
  const [serverRooms, setServerRooms] = useState<ServerRoomOption[]>([]);
  const [statistics, setStatistics] = useState<Statistics>({ total: 0, normal: 0, fault: 0, disabled: 0 });

  // Wave 5: useDict replaces raw post() calls for both line type + ISP dicts
  const { data: lineTypeDict = [] } = useDict("ops_dedicated_line_type");
  const { data: ispDict = [] } = useDict("ops_isp");

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 加载所有机房列表（不分部门）
  const loadServerRooms = useCallback(async () => {
    try {
      const result = await serverRoomApi.list({ current: 1, pageSize: 50 }) as { data?: { list: Record<string, unknown>[] } };
      const rooms = result.data?.list || [];
      setServerRooms(rooms.map((r: Record<string, unknown>) => ({
        id: r.id as string,
        name: r.name as string,
        orgId: r.orgId as string,
      })));
    } catch (error) {
      handleApiError(error, "加载机房列表", false);
    }
  }, []);

  // 服务端排序:field 必须与 columns 的 dataIndex 一致(useServerSort 按 sorter.field===dataIndex 匹配)
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<DedicatedLine>("name"),
      createSorterMeta<DedicatedLine>("bandwidth"),
      createSorterMeta<DedicatedLine>("status"),
      createSorterMeta<DedicatedLine>("createdAt", "date"),
    ],
    []
  );

  const {
    loading, data: lines, total, selectedRowKeys,
    searchForm, editForm: lineForm, editModalVisible: modalVisible,
    editingItem: editingLine, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingLine, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadLines, resetSelection,
    handleTableChange: handleLineTableChange,
    getColumnSortOrder: getLineColumnSortOrder,
  } = useTableManager<DedicatedLine>(
    async (params) => {
      const result = await dedicatedLineApi.list(params) as { data?: { list: DedicatedLine[]; total: number } };
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

  const loadStatistics = useCallback(async (): Promise<Statistics> => {
    try {
      const stats = await dedicatedLineApi.statistics();
      return { total: stats.total ?? 0, normal: stats.normal ?? 0, fault: stats.fault ?? 0, disabled: stats.disabled ?? 0 };
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
      return { total: 0, normal: 0, fault: 0, disabled: 0 };
    }
  }, []);

  // 统一的数据刷新函数
  const refreshData = useCallback(() => {
    loadLines();
    loadStatistics().then(setStatistics);
  }, [loadLines, loadStatistics]);

  // 初始化加载
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    Promise.all([loadServerRooms(), loadStatistics()])
      .then(([_, stats]) => setStatistics(stats));
    loadLines();
  }, [loadServerRooms, loadStatistics, loadLines]);

  const handleSave = async () => {
    try {
      const values = await lineForm.validateFields() as Partial<DedicatedLine>;
      if (editingLine) {
        await dedicatedLineApi.update(editingLine.id, values);
        handleSuccess("更新");
      } else {
        await dedicatedLineApi.create(values);
        handleSuccess("创建");
      }
      handleModalClose();
      refreshData();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await dedicatedLineApi.delete(id);
      handleSuccess("删除");
      refreshData();
    } catch (error) {
      handleApiError(error, "删除");
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try {
      await dedicatedLineApi.batch("delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      refreshData();
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  };

  const openModal = (record?: DedicatedLine) => {
    if (record) {
      handleEdit(record);
      lineForm.setFieldsValue(record);
    } else {
      handleAdd();
      const defaultType = lineTypeDict.find(d => d.isDefault)?.dictValue || lineTypeDict[0]?.dictValue;
      const defaultIsp = ispDict.find(d => d.isDefault)?.dictValue || ispDict[0]?.dictValue;
      lineForm.setFieldsValue({ status: 0, lineType: defaultType, isp: defaultIsp });
    }
  };

  const handleImportSuccess = () => {
    refreshData();
    setImportVisible(false);
  };

  const getLineTypeText = (type: string) => {
    const dictItem = lineTypeDict.find(d => d.dictValue === type);
    return dictItem?.dictLabel || type;
  };

  const getIspText = (isp: string) => {
    const dictItem = ispDict.find(d => d.dictValue === isp);
    return dictItem?.dictLabel || isp;
  };

  const getStatusText = (status: number) => {
    const statusMap: Record<number, string> = { 0: "正常", 1: "故障", 2: "停用" };
    return statusMap[status] || "未知";
  };

  const columns: ColumnsType<DedicatedLine> = [
    { title: "专线名称", dataIndex: "name", key: "name", width: 150, sorter: true, sortOrder: getLineColumnSortOrder("name") },
    {
      title: "专线类型",
      dataIndex: "lineType",
      key: "lineType",
      width: 120,
      render: (type) => <Tag>{getLineTypeText(type as string)}</Tag>,
    },
    { title: "带宽", dataIndex: "bandwidth", key: "bandwidth", width: 100, sorter: true, sortOrder: getLineColumnSortOrder("bandwidth"), render: (v) => v || "-" },
    { title: "运营商", dataIndex: "isp", key: "isp", width: 120, render: (isp) => <Tag>{getIspText(isp as string)}</Tag> },
    { title: "源机房", dataIndex: "sourceRoomName", key: "sourceRoomName", width: 120, render: (v) => v || "-" },
    { title: "源设备", dataIndex: "sourceDeviceName", key: "sourceDeviceName", width: 120, render: (v) => v || "-" },
    { title: "源端口", dataIndex: "sourcePort", key: "sourcePort", width: 100, render: (v) => v || "-" },
    { title: "源IP", dataIndex: "sourceIpAddress", key: "sourceIpAddress", width: 130, render: (v) => v || "-" },
    { title: "目的机房", dataIndex: "destRoomName", key: "destRoomName", width: 120, render: (v) => v || "-" },
    { title: "目的设备", dataIndex: "destDeviceName", key: "destDeviceName", width: 120, render: (v) => v || "-" },
    { title: "目的端口", dataIndex: "destPort", key: "destPort", width: 100, render: (v) => v || "-" },
    { title: "目的IP", dataIndex: "destIpAddress", key: "destIpAddress", width: 130, render: (v) => v || "-" },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      sorter: true,
      sortOrder: getLineColumnSortOrder("status"),
      render: (status) => {
        const colors: Record<number, string> = { 0: "success", 1: "error", 2: "default" };
        return <Tag color={colors[status as keyof typeof colors]}>{getStatusText(status as number)}</Tag>;
      },
    },
    { title: "月费(元)", dataIndex: "monthlyFee", key: "monthlyFee", width: 100, render: (v) => v ? `¥${v}` : "-" },
    createDateTimeColumn("createdAt", { width: 180, sorter: true, sortOrder: getLineColumnSortOrder("createdAt") }),
    {
      title: "操作", key: "action",
      render: (_, record) => {
        const actions = [
          { key: "edit", label: "编辑", icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: "delete", label: "删除", icon: <DeleteOutlined />, danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除这条专线吗？",
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

  const renderCardView = () => {
    if (lines.length === 0) return <div style={{ textAlign: "center", padding: "40px", color: "var(--theme-text-tertiary, #999)" }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {lines.map((line) => (
          <Col key={line.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(line)} />,
                <DeleteOutlined
                  key="delete"
                  style={{ color: "var(--theme-error, #ff4d4f)" }}
                  onClick={() => {
                    Modal.confirm({
                      title: "确定要删除这条专线吗？",
                      okText: "确定",
                      cancelText: "取消",
                      okButtonProps: { danger: true },
                      onOk: () => handleDelete(line.id),
                    });
                  }}
                />,
              ]}
            >
              <Card.Meta
                title={
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <span>{line.name}</span>
                    <Tag color={line.status === 0 ? "success" : line.status === 1 ? "error" : "default"}>{getStatusText(line.status)}</Tag>
                  </div>
                }
                description={
                  <div>
                    <div><strong>类型：</strong>{getLineTypeText(line.lineType)}</div>
                    <div><strong>运营商：</strong>{getIspText(line.isp)}</div>
                    {line.bandwidth && <div><strong>带宽：</strong>{line.bandwidth}</div>}
                    {line.sourceIpAddress && <div><strong>源IP：</strong>{line.sourceIpAddress}</div>}
                    {line.destIpAddress && <div><strong>目的IP：</strong>{line.destIpAddress}</div>}
                    {line.monthlyFee && <div><strong>月费：</strong>¥{line.monthlyFee}</div>}
                  </div>
                }
              />
            </Card>
          </Col>
        ))}
      </Row>
    );
  };

  return (
    <Layout style={{ background: "#000", minHeight: "calc(100vh - 64px)" }}>
      <Content style={{ background: "#fff", padding: 16 }}>
      <StatisticsCards
        show={total > 10}
        items={[
          { title: "总专线数", value: statistics.total, prefix: <LineChartOutlined /> },
          { title: "正常", value: statistics.normal, styles: { content: { color: "var(--theme-success, #3f8600)" } }, prefix: <CheckCircleOutlined /> },
          { title: "故障", value: statistics.fault, styles: { content: { color: "var(--theme-error, #cf1322)" } }, prefix: <WarningOutlined /> },
          { title: "停用", value: statistics.disabled, styles: { content: { color: "var(--theme-text-tertiary, #8c8c8c)" } }, prefix: <StopOutlined /> },
        ]}
      />
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="name" label="专线名称">
              <Input placeholder="请输入专线名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="sourceRoomId" label="源机房">
              <Select
                placeholder="请选择源机房"
                allowClear
                showSearch
                filterOption={(input, option) =>
                  String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())
                }
                onSearch={() => {}}
                className="user-form-input"
                style={{ width: 150 }}
                options={serverRooms.map(room => ({ label: room.name, value: room.id }))}
              />
            </Form.Item>
            <Form.Item name="destRoomId" label="目的机房">
              <Select
                placeholder="请选择目的机房"
                allowClear
                showSearch
                filterOption={(input, option) =>
                  String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())
                }
                onSearch={() => {}}
                className="user-form-input"
                style={{ width: 150 }}
                options={serverRooms.map(room => ({ label: room.name, value: room.id }))}
              />
            </Form.Item>
            <Form.Item name="lineType" label="专线类型">
              <Select placeholder="请选择类型" allowClear className="user-form-input" style={{ width: 130 }} onSearch={() => {}}>
                {lineTypeDict.map(d => <Option key={d.dictValue} value={d.dictValue}>{d.dictLabel}</Option>)}
              </Select>
            </Form.Item>
            <Form.Item name="isp" label="运营商">
              <Select placeholder="请选择运营商" allowClear className="user-form-input" style={{ width: 120 }} onSearch={() => {}}>
                {ispDict.map(d => <Option key={d.dictValue} value={d.dictValue}>{d.dictLabel}</Option>)}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 100 }} onSearch={() => {}}>
                <Option value={0}>正常</Option>
                <Option value={1}>故障</Option>
                <Option value={2}>停用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={refreshData}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button>
              <Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>导入</Button>
            <Button icon={<ExportOutlined />} onClick={() => {
              const values = searchForm.getFieldsValue() as Record<string, unknown>;
              const currentFilters: Record<string, unknown> = {};
              Object.keys(values).forEach(key => {
                const value = values[key];
                if (value !== undefined && value !== null && value !== "") {
                  currentFilters[key] = value;
                }
              });
              setExportFilters(currentFilters);
              setExportVisible(true);
            }}>导出</Button>
            {selectedRowKeys.length > 0 && <Button icon={<DeleteOutlined />} style={{ color: "var(--theme-error, #ff4d4f)" }} onClick={handleBatchDelete}>批量删除 ({selectedRowKeys.length})</Button>}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增专线</Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && <Alert title={<span>已选择 <strong>{selectedRowKeys.length}</strong> 条专线，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
      </Card>
      <Card>
        {viewMode === "table" ? (
          <Table
            rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
            columns={columns}
            dataSource={lines}
            loading={loading}
            rowKey="id"
            pagination={paginationProps}
            onChange={handleLineTableChange}
          />
        ) : renderCardView()}
      </Card>
      <Modal title={editingLine ? "编辑专线" : "新增专线"} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); lineForm.resetFields(); setEditingLine(null); }} width={700}>
        <Form form={lineForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item name="name" label="专线名称" rules={[{ required: true, message: "请输入专线名称" }]}>
            <Input placeholder="请输入专线名称" />
          </Form.Item>
          <Form.Item name="lineType" label="专线类型" rules={[{ required: true, message: "请选择专线类型" }]}>
            <Select placeholder="请选择专线类型" onSearch={() => {}}>
              {lineTypeDict.map(d => <Option key={d.dictValue} value={d.dictValue}>{d.dictLabel}</Option>)}
            </Select>
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="bandwidth" label="带宽">
                <Input placeholder="如：100M, 1G" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="isp" label="运营商" rules={[{ required: true, message: "请选择运营商" }]}>
                <Select placeholder="请选择运营商" className="user-form-input" onSearch={() => {}}>
                  {ispDict.map(d => <Option key={d.dictValue} value={d.dictValue}>{d.dictLabel}</Option>)}
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="sourceRoomId" label="源机房">
                <Select
                  placeholder="请选择源机房"
                  allowClear
                  showSearch
                  filterOption={(input, option) =>
                    String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())
                  }
                  onSearch={() => {}}
                  className="user-form-input"
                  options={serverRooms.map(room => ({ label: room.name, value: room.id }))}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="sourceDeviceName" label="源设备名称">
                <Input placeholder="请输入源设备名称" className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="sourcePort" label="源端口">
            <Input placeholder="请输入源端口" className="user-form-input" style={{ width: "50%" }} />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="destRoomId" label="目的机房">
                <Select
                  placeholder="请选择目的机房"
                  allowClear
                  showSearch
                  filterOption={(input, option) =>
                    String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())
                  }
                  onSearch={() => {}}
                  className="user-form-input"
                  options={serverRooms.map(room => ({ label: room.name, value: room.id }))}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="destDeviceName" label="目的设备名称">
                <Input placeholder="请输入目的设备名称" className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="destPort" label="目的端口">
            <Input placeholder="请输入目的端口" className="user-form-input" style={{ width: "50%" }} />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="sourceIpAddress" label="源IP地址">
                <Input placeholder="请输入源IP地址" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="sourceSubnetMask" label="源子网掩码">
                <Input placeholder="请输入源子网掩码" className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="destIpAddress" label="目的IP地址">
                <Input placeholder="请输入目的IP地址" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="destSubnetMask" label="目的子网掩码">
                <Input placeholder="请输入目的子网掩码" className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="vlan" label="VLAN">
                <Input placeholder="请输入VLAN" className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="carrierContactName" label="运营商联系人">
                <Input placeholder="请输入联系人" className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="carrierContactPhone" label="联系人电话">
                <Input placeholder="请输入联系电话" className="user-form-input" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="monthlyFee" label="月租费用">
                <InputNumber min={0} precision={2} placeholder="请输入月租费用" style={{ width: "100%" }} className="user-form-input" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label="状态" rules={[{ required: true, message: "请选择状态" }]}>
                <Select placeholder="请选择状态" className="user-form-input" onSearch={() => {}}>
                  <Option value={0}>正常</Option>
                  <Option value={1}>故障</Option>
                  <Option value={2}>停用</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="remark" label="备注">
            <TextArea rows={3} placeholder="请输入备注" className="user-form-input" />
          </Form.Item>
        </Form>
      </Modal>
      <ExcelImport entityType="dedicatedLine" entityName="专线" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
      <ExcelExport entityType="dedicatedLine" entityName="专线" visible={exportVisible} onClose={() => setExportVisible(false)} filters={exportFilters} />
      </Content>
    </Layout>
  );
};

export default DedicatedLineManagement;
