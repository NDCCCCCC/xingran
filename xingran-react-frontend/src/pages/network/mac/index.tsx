import type { FC } from "react";
import {
  Table,
  Button,
  Space,
  Form,
  Input,
  InputNumber,
  Select,
  Card,
  Row,
  Col,
  Statistic,
  Tag,
  Layout,
  Drawer,
  Grid,
  App,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  AppleOutlined,
  ApartmentOutlined,
} from "@ant-design/icons";
import type { DeviceMACAddress, NetworkDevice } from "@/types";
import { formatDateTime } from "@/utils/datetime";
import { get, post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import { useTableManager } from "@/hooks/useTableManager";
import { createSorterMeta } from "@/utils/tableHelpers";
import { withErrorHandling } from "@/utils/errorHandler";
import { useState, useEffect, useMemo } from "react";
import { usePagination } from "@/hooks/usePagination";
import { useSidebarDeptFilter } from "@/hooks/useSidebarDeptFilter";
import { useSearchParams } from "react-router-dom";
import dayjs from "dayjs";
import NetworkExport from "@/components/shared/NetworkExport";
import { BatchExportModal } from "@/components/shared";
import { DeptSidebar } from "@/components/operations/DeptSidebar";
import { MACEventsTimeline } from "@/components/network";

const { Option } = Select;
const { Content } = Layout;
const { useBreakpoint } = Grid;

const MACAddressPage: FC = () => {
  const { message } = App.useApp();
  const breakpoint = useBreakpoint();
  const isMobile = !!breakpoint.xs;
  const [searchParams, setSearchParams] = useSearchParams();

  const [devices, setDevices] = useState<NetworkDevice[]>([]);

  // 部门侧栏筛选 hook(2026-06-30):与 workstations/index.tsx 同款。
  // 联动:后端 /network/mac/list 直接支持 deptId(JOIN sys_network_device.dept_id),
  // queryFn 透传 selectedDeptId 即可,无需前端 deviceIds 中间层。
  const { selectedDeptId, handleDeptSelect, setSelectedDeptId } = useSidebarDeptFilter({
    searchForm: undefined,
  });

  // 移动端 dept tree 折叠进 Drawer
  const [deptDrawerOpen, setDeptDrawerOpen] = useState(false);

  // MAC 事件时间线抽屉:点击 MAC 列 → 右侧抽屉显示该 MAC 7d 范围事件
  const [timelineDrawerOpen, setTimelineDrawerOpen] = useState(false);
  const [timelineDrawerMac, setTimelineDrawerMac] = useState<string>("");
  // 抽屉打开瞬间锁定 7d 范围,避免抽屉打开期间用户停留时间过长导致 endTime 漂移
  const timelineDrawerRange = useMemo(() => {
    if (!timelineDrawerOpen) return null;
    return {
      startTime: dayjs().subtract(7, "day").toISOString(),
      endTime: dayjs().toISOString(),
    };
  }, [timelineDrawerOpen]);

  // URL ?deptId=xxx 同步(2026-06-30 quick)
  useEffect(() => {
    const urlDeptId = searchParams.get("deptId");
    if (urlDeptId && urlDeptId !== selectedDeptId) {
      setSelectedDeptId(urlDeptId);
    }
  }, [searchParams]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleDeptChangeWithUrl = (
    selectedKeys: Parameters<
      NonNullable<ReturnType<typeof useSidebarDeptFilter>["handleDeptSelect"]>
    >[0],
    info: Parameters<NonNullable<ReturnType<typeof useSidebarDeptFilter>["handleDeptSelect"]>>[1]
  ) => {
    handleDeptSelect(selectedKeys, info);
    const deptId = selectedKeys.length > 0 ? (selectedKeys[0] as string) : "";
    const next = new URLSearchParams(searchParams);
    if (deptId) next.set("deptId", deptId);
    else next.delete("deptId");
    setSearchParams(next);
  };

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    dynamic: 0,
    static: 0,
    secure: 0,
  });

  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 必须与 columns 的 dataIndex 一致
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<DeviceMACAddress>("macAddress"),
      createSorterMeta<DeviceMACAddress>("interfaceName"),
      createSorterMeta<DeviceMACAddress>("vlanId"),
    ],
    []
  );

  const {
    loading,
    data: macAddresses,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    loadData: loadMACAddresses,
    handleSearch,
    handleReset,
    handleRefresh,
    handleTableChange: handleMacTableChange,
    getColumnSortOrder: getMacColumnSortOrder,
  } = useTableManager<DeviceMACAddress>(
    async (params) => {
      const formValues = searchForm.getFieldsValue();
      const values = formValues as Record<string, unknown>;
      // 2026-06-30: 透传 dept 联动的 deptId。selectedDeptId 是 state,本 queryFn 闭包在
      // 每次 render 同步重建(loadFunctionRef.current = loadFunction),useEffect[selectedDeptId]
      // 在 re-render 后才触发 handleSearch → 此刻读到的 selectedDeptId 已最新,无 stale closure。
      // 后端 JOIN sys_network_device.dept_id 一处过滤,无需前端 deviceIds 中间层。
      const deptIdPayload = selectedDeptId ? { deptId: selectedDeptId } : {};
      const result = (await post("/network/mac/list", {
        current: params.current ?? paginationProps.current ?? 1,
        pageSize: params.pageSize ?? paginationProps.pageSize ?? 10,
        ...deptIdPayload,
        ...values,
      })) as { data: { list: DeviceMACAddress[]; total: number } };
      loadStatistics();
      return result.data;
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

  // 加载统计数据(专用端点 COUNT 聚合,不受分页/筛选影响)
  const loadStatistics = async () => {
    try {
      const result = await get<{
        totalRecords?: number;
        dynamic?: number;
        static?: number;
        secure?: number;
      }>("/network/mac/statistics");
      const data = result.data || {};
      setStatistics({
        total: data.totalRecords ?? 0,
        dynamic: data.dynamic ?? 0,
        static: data.static ?? 0,
        secure: data.secure ?? 0,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  };

  // 加载设备列表(2026-06-30:加 deptId 参数,选部门后重查该部门的设备,填充设备下拉框)
  // 设备下拉框与 MAC 列表是两条独立链路:MAC 列表靠 queryFn 透传 deptId 给后端过滤,
  // 不依赖本函数结果,故无 await 时序要求,可与 handleSearch 并行触发。
  const loadDevices = async (deptId?: string) => {
    try {
      const result = (await post("/network/devices/list", {
        current: 1,
        pageSize: 50,
        ...(deptId ? { deptId } : {}),
      })) as { data?: { list: NetworkDevice[] } };
      const list = result.data?.list || [];
      setDevices(list);
    } catch (error) {
      console.error("加载设备列表失败:", error);
    }
  };

  // 部门变化时重查设备下拉框 + 重查 MAC 列表(2026-06-30)
  // 触发链路:DeptSidebar 选中 → selectedDeptId 变 → 这里:
  //   - handleSearch 读最新 selectedDeptId → queryFn 透传 deptId → 后端 JOIN 过滤
  //   - loadDevices 并行重查设备下拉框(不阻塞 MAC 查询)
  useEffect(() => {
    // 清空 deviceId(原选的可能不属于新部门,避免误筛)
    searchForm.setFieldValue("deviceId", undefined);
    void loadDevices(selectedDeptId || undefined);
    handleSearch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedDeptId]);

  // 批量删除
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请选择要删除的数据");
      return;
    }
    await withErrorHandling(
      async () => {
        await post("/network/mac/batch-delete", { ids: selectedRowKeys });
        return "批量删除成功";
      },
      {
        onSuccess: () => {
          setSelectedRowKeys([]);
          loadMACAddresses();
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

  // MAC类型选项
  const macTypeOptions = [
    { label: "动态", value: "dynamic" },
    { label: "静态", value: "static" },
    { label: "安全", value: "secure" },
  ];

  // 表格列(2026-06-30 quick:移除"设备名称"列,信息密度低)
  const columns: ColumnsType<DeviceMACAddress> = [
    {
      title: "MAC地址",
      dataIndex: "macAddress",
      key: "macAddress",
      width: 160,
      sorter: true,
      sortOrder: getMacColumnSortOrder("macAddress"),
      render: (mac: string) => (
        <Button
          type="link"
          size="small"
          style={{ padding: 0, height: "auto", fontFamily: "monospace" }}
          onClick={() => {
            setTimelineDrawerMac(mac);
            setTimelineDrawerOpen(true);
          }}
        >
          {mac}
        </Button>
      ),
    },
    {
      title: "接口",
      dataIndex: "interfaceName",
      key: "interfaceName",
      width: 150,
      sorter: true,
      sortOrder: getMacColumnSortOrder("interfaceName"),
    },
    {
      title: "VLAN ID",
      dataIndex: "vlanId",
      key: "vlanId",
      width: 100,
      sorter: true,
      sortOrder: getMacColumnSortOrder("vlanId"),
      render: (vlanId: number) => vlanId || "-",
    },
    {
      title: "MAC类型",
      dataIndex: "macType",
      key: "macType",
      width: 100,
      render: (macType: string) => {
        const option = macTypeOptions.find((o) => o.value === macType);
        const color = macType === "dynamic" ? "blue" : macType === "static" ? "green" : "orange";
        return <Tag color={color}>{option?.label || macType}</Tag>;
      },
    },
    {
      title: "采集时间",
      dataIndex: "collectedAt",
      key: "collectedAt",
      width: 180,
      render: (date: string) => formatDateTime(date),
    },
  ];

  return (
    <Layout style={{ background: "transparent" }}>
      {/* 桌面端:左侧部门树常驻(2026-06-30 quick,与 workstations 同款 DeptSidebar) */}
      {!isMobile && (
        <DeptSidebar
          width={240}
          selectedDeptId={selectedDeptId}
          onSelect={handleDeptChangeWithUrl}
        />
      )}
      <Content style={{ background: "#fff", padding: isMobile ? 12 : 24 }}>
        {/* 移动端:Dept tree 折叠进 Drawer,通过按钮触发(与 workstations 同步) */}
        {isMobile && (
          <div style={{ marginBottom: 16 }}>
            <Button icon={<ApartmentOutlined />} onClick={() => setDeptDrawerOpen(true)}>
              {selectedDeptId ? `已选部门:${selectedDeptId.slice(0, 8)}...` : "选择部门"}
            </Button>
            <Drawer
              title="选择部门"
              open={deptDrawerOpen}
              onClose={() => setDeptDrawerOpen(false)}
              width={300}
              styles={{ body: { padding: 0 } }}
            >
              <DeptSidebar
                width={300}
                selectedDeptId={selectedDeptId}
                onSelect={(keys, info) => {
                  handleDeptChangeWithUrl(keys, info);
                  setDeptDrawerOpen(false);
                }}
              />
            </Drawer>
          </div>
        )}
        {/* 统计卡片 */}
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Card>
              <Statistic title="MAC地址总数" value={statistics.total} prefix={<AppleOutlined />} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="动态MAC"
                value={statistics.dynamic}
                styles={{ content: { color: "var(--theme-info, #1890ff)" } }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="静态MAC"
                value={statistics.static}
                styles={{ content: { color: "var(--theme-success, #52c41a)" } }}
              />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic
                title="安全MAC"
                value={statistics.secure}
                styles={{ content: { color: "var(--theme-warning, #faad14)" } }}
              />
            </Card>
          </Col>
        </Row>

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
              <Form.Item name="deviceId" label="设备">
                <Select
                  placeholder={selectedDeptId ? "请选择设备" : "请先选择部门"}
                  disabled={!selectedDeptId}
                  allowClear
                  className="user-form-input"
                  style={{ width: 200 }}
                  showSearch
                  optionFilterProp="deviceName"
                  onSearch={() => {}}
                >
                  {devices.map((device) => (
                    <Option key={device.id} value={device.id}>
                      {device.deviceName}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item name="macAddress" label="MAC地址">
                <Input
                  placeholder="请输入MAC地址"
                  allowClear
                  className="user-form-input"
                  style={{ width: 180 }}
                />
              </Form.Item>
              <Form.Item name="interfaceName" label="接口">
                <Input
                  placeholder="请输入接口名称"
                  allowClear
                  className="user-form-input"
                  style={{ width: 150 }}
                />
              </Form.Item>
              <Form.Item name="vlanId" label="VLAN ID">
                <InputNumber placeholder="VLAN ID" style={{ width: 120 }} />
              </Form.Item>
              <Form.Item name="macType" label="MAC类型">
                <Select
                  placeholder="请选择类型"
                  allowClear
                  className="user-form-input"
                  style={{ width: 120 }}
                  onSearch={() => {}}
                >
                  {macTypeOptions.map((opt) => (
                    <Option key={opt.value} value={opt.value}>
                      {opt.label}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item>
                <Space>
                  <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                    查询
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={handleReset}>
                    重置
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                    刷新
                  </Button>
                </Space>
              </Form.Item>
            </Form>
            <Space>
              <NetworkExport
                entityType="mac"
                entityName="MAC地址"
                filters={(() => {
                  const values = searchForm.getFieldsValue() as Record<string, unknown>;
                  const filtered: Record<string, unknown> = {};
                  Object.keys(values).forEach((key) => {
                    const value = values[key];
                    if (value !== undefined && value !== null && value !== "") {
                      filtered[key] = value;
                    }
                  });
                  return filtered;
                })()}
                current={paginationProps.current}
                pageSize={paginationProps.pageSize}
              />
              <Button
                icon={<DeleteOutlined />}
                onClick={handleBatchDelete}
                disabled={selectedRowKeys.length === 0}
                style={{ color: "var(--theme-error, #ff4d4f)" }}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            </Space>
            {/* 批量导出 Modal */}

            <BatchExportModal
              visible={batchModalVisible}

              onConfirm={handleBatchExport}

              onCancel={() => setBatchModalVisible(false)}

              loading={batchExporting}
            />
          </div>
        </Card>

        {/* MAC地址表格 */}
        <Card>
          <Table
            rowSelection={{
              selectedRowKeys,
              onChange: setSelectedRowKeys,
            }}
            columns={columns}
            dataSource={macAddresses}
            loading={loading}
            rowKey="id"
            scroll={{ x: 1200 }}
            pagination={paginationProps}
            onChange={handleMacTableChange}
          />
        </Card>

        {/* MAC 事件时间线抽屉:点击 MAC 列 → 抽屉打开显示该 MAC 7d 范围内的事件 */}
        <Drawer
          title={timelineDrawerMac ? `MAC 事件时间线 — ${timelineDrawerMac}` : "MAC 事件时间线"}
          open={timelineDrawerOpen}
          onClose={() => setTimelineDrawerOpen(false)}
          width={480}
          destroyOnClose
        >
          {timelineDrawerMac && timelineDrawerRange && (
            <MACEventsTimeline
              mac={timelineDrawerMac}
              startTime={timelineDrawerRange.startTime}
              endTime={timelineDrawerRange.endTime}
            />
          )}
        </Drawer>
      </Content>
    </Layout>
  );
};

export default MACAddressPage;
