/**
 * 信息点管理页面
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import type { FC } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  Table, Button, Space, Modal, Form, Input, Select,
  Tag, Card, Row, Col, Alert, Radio, Cascader, Tooltip, Layout,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, WarningOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined, DotChartOutlined,
} from "@ant-design/icons";
import type { InfoPoint, WorkstationOps, Floor, Building } from "@/types";
import { infoPointApi, workstationApi, buildingApi, floorApi } from "@/lib/opsApi";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { useSidebarDeptFilter } from "@/hooks/useSidebarDeptFilter";
import { useDict } from "@/hooks/useDict";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { createDateTimeColumn } from "@/utils/tableHelpers";
import ActionButtons from "@/components/shared/ActionButtons";
import ExcelImport from "@/components/shared/ExcelImport";
import ExcelExport from "@/components/shared/ExcelExport";
import { DeptSidebar } from "@/components/operations/DeptSidebar";
import { StatisticsCards } from "@/components/operations/StatisticsCards";
import { post } from "@/lib/api";

const { Option } = Select;
const { TextArea } = Input;
const { Content } = Layout;

type ViewMode = "table" | "card";

interface WorkstationOption {
  id: string;
  name: string;
  buildingId?: string;
  buildingName?: string;
  floorId?: string;
  floorName?: string;
  orgId?: string;
}

interface NetworkDeviceOption {
  id: string;
  deviceName: string;
  ipAddress: string;
}

interface DevicePortOption {
  id: string;
  interfaceName: string;
  adminStatus?: string;
  description?: string;
}

interface CascaderOption {
  value: string;
  label: string;
  children?: CascaderOption[];
  isLeaf?: boolean;
  floorNo?: string;
}

interface Statistics {
  total: number;
  normal: number;
  fault: number;
  disabled: number;
}

const InfoPointManagement: FC = () => {
  const [statistics, setStatistics] = useState<Statistics>({ total: 0, normal: 0, fault: 0, disabled: 0 });
  const location = useLocation();
  const [viewMode, setViewMode] = usePersistedStateController<ViewMode>({
    keyPrefix: location.pathname,
    keySuffix: "viewMode",
    defaultValue: "table",
  });
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [exportFilters, setExportFilters] = useState<Record<string, unknown>>({});
  const [workstationOptions, setWorkstationOptions] = useState<WorkstationOption[]>([]);
  const [networkDevices, setNetworkDevices] = useState<NetworkDeviceOption[]>([]);
  const [devicePorts, setDevicePorts] = useState<DevicePortOption[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>("");

  // Wave 5: useDict replaces raw post() call for ops_info_point_type
  const { data: infoPointTypeDict = [] } = useDict("ops_info_point_type");

  // Cascader 懒加载状态
  const [cascaderOptions, setCascaderOptions] = useState<CascaderOption[]>([]);
  const [loadingCascader, setLoadingCascader] = useState(false);

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();
  const { selectedDeptId, handleDeptSelect } = useSidebarDeptFilter({
    clearFieldNames: ["workstationId"],
  });

  const {
    loading, data: infoPoints, total, selectedRowKeys,
    searchForm, editForm: infoPointForm, editModalVisible: modalVisible,
    editingItem: editingInfoPoint, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingInfoPoint, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadInfoPoints, resetSelection,
  } = useTableManager<InfoPoint>(
    async (params) => {
      const searchParams = selectedDeptId
        ? { ...params, orgId: selectedDeptId }
        : params;
      const result = await infoPointApi.list(searchParams);
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
    }
  );

  const loadStatistics = useCallback(async (): Promise<Statistics> => {
    try {
      return await infoPointApi.statistics();
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
      return { total: 0, normal: 0, fault: 0, disabled: 0 };
    }
  }, []);

  const refreshData = useCallback(() => {
    // 使用 handleSearch 而不是 loadInfoPoints，以保留搜索筛选条件
    handleSearch();
    loadStatistics().then(setStatistics);
  }, [handleSearch, loadStatistics]);

  // ==================== Cascader 懒加载函数 ====================

  const loadBuildingsForCascader = useCallback(async (orgId?: string) => {
    try {
      const params: Record<string, unknown> = { current: 1, pageSize: 1000 };
      if (orgId) {
        params.orgId = orgId;
      }
      const buildingResult = await buildingApi.list(params);
      const buildings = buildingResult.data?.list || [];
      return buildings.map((b: Building) => ({
        value: b.id,
        label: b.name,
        isLeaf: false,
      }));
    } catch (error) {
      handleApiError(error, "加载楼宇列表", false);
      return [];
    }
  }, []);

  const loadFloorsForCascader = useCallback(async (buildingId: string) => {
    try {
      const floorResult = await floorApi.list({ buildingId, current: 1, pageSize: 1000 });
      const floors = floorResult.data?.list || [];
      return floors.map((f: Floor) => ({
        value: f.id,
        label: f.name || f.floorNo || "",
        isLeaf: false,
        floorNo: f.floorNo,
      }));
    } catch (error) {
      handleApiError(error, "加载楼层列表", false);
      return [];
    }
  }, []);

  const loadWorkstationsForCascader = useCallback(async (floorId: string, floorNo: string) => {
    try {
      const wsResult = await workstationApi.list({ floorCode: floorNo, current: 1, pageSize: 1000 });
      const workstations = wsResult.data?.list || [];
      return workstations
        .filter((w: WorkstationOps) => w.id)
        .map((w: WorkstationOps) => ({
          value: w.id,
          label: w.name || "未命名工位",
          isLeaf: true,
        }));
    } catch (error) {
      handleApiError(error, "加载工位列表", false);
      return [];
    }
  }, []);

  // Cascader 懒加载处理函数
  const handleCascaderLoadData = useCallback(async (selectedOptions: CascaderOption[]) => {
    const targetOption = selectedOptions[selectedOptions.length - 1];

    setLoadingCascader(true);

    try {
      if (selectedOptions.length === 1) {
        // 加载楼层（第二级）
        const buildingId = targetOption.value;
        const floors = await loadFloorsForCascader(buildingId);
        targetOption.children = floors;
      } else if (selectedOptions.length === 2) {
        // 加载工位（第三级）
        const floorId = targetOption.value;
        const floorNo = targetOption.floorNo;
        const workstations = await loadWorkstationsForCascader(floorId, floorNo || floorId);
        targetOption.children = workstations;
      }

      // 触发重新渲染
      setCascaderOptions([...cascaderOptions]);
    } finally {
      setLoadingCascader(false);
    }
  }, [cascaderOptions, loadFloorsForCascader, loadWorkstationsForCascader]);

  const initCascaderOptions = useCallback(async () => {
    setLoadingCascader(true);
    try {
      const buildings = await loadBuildingsForCascader(selectedDeptId || undefined);
      setCascaderOptions(buildings);
    } finally {
      setLoadingCascader(false);
    }
  }, [loadBuildingsForCascader, selectedDeptId]);

  const loadSearchWorkstationOptions = useCallback(async () => {
    if (workstationOptions.length > 0) {
      return;
    }

    try {
      const params: Record<string, unknown> = { current: 1, pageSize: 1000 };
      if (selectedDeptId) {
        params.orgId = selectedDeptId;
      }
      const wsResult = await workstationApi.list(params);
      const workstations = wsResult.data?.list ?? [];
      setWorkstationOptions(
        workstations
          .filter((w: WorkstationOps) => w.id)
          .map((w: WorkstationOps) => ({
            id: w.id,
            name: w.name ?? "未命名工位",
          }))
      );
    } catch (error) {
      handleApiError(error, "加载工位选项", false);
    }
  }, [workstationOptions.length, selectedDeptId]);

  // ===========================================================

  const loadNetworkDevices = useCallback(async () => {
    try {
      const result = await post("/network/devices/list", { current: 1, pageSize: 1000 }) as { data?: { list: NetworkDeviceOption[] } };
      setNetworkDevices(result.data?.list || []);
    } catch (error) {
      handleApiError(error, "加载网络设备列表", false);
    }
  }, []);

  const loadDevicePorts = useCallback(async (deviceId: string) => {
    if (!deviceId) {
      setDevicePorts([]);
      return;
    }
    try {
      const result = await post("/network/ports/list", { deviceId, current: 1, pageSize: 1000 }) as { data?: { list: DevicePortOption[] } };
      setDevicePorts(result.data?.list || []);
    } catch (error) {
      handleApiError(error, "加载设备端口列表", false);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    Promise.all([loadStatistics(), loadNetworkDevices()])
      .then(([stats]) => setStatistics(stats));
     
  }, [loadStatistics, loadNetworkDevices]);

  useEffect(() => {
    loadInfoPoints();
    setWorkstationOptions([]);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedDeptId]);

  // 编辑模式：初始化设备ID和端口列表
  useEffect(() => {
    const initializeDeviceAndPorts = async () => {
      if (editingInfoPoint && editingInfoPoint.deviceId) {
        setSelectedDeviceId(editingInfoPoint.deviceId);
        await loadDevicePorts(editingInfoPoint.deviceId);
        // 端口列表加载完成后，重新设置 portId 以确保 Select 显示正确的标签
        if (editingInfoPoint.portId) {
          infoPointForm.setFieldsValue({ portId: editingInfoPoint.portId });
        }
      } else if (!editingInfoPoint) {
        // 新增模式：清空设备ID和端口列表
        setSelectedDeviceId("");
        setDevicePorts([]);
      }
    };
    initializeDeviceAndPorts();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editingInfoPoint]);

  const handleSave = async () => {
    try {
      const values = await infoPointForm.validateFields() as Record<string, unknown>;

      if (Array.isArray(values.workstationId)) {
        values.workstationId = (values.workstationId as unknown[])[values.workstationId.length - 1];
      }

      // 字段映射：前端的 description 映射到后端的 remark
      if (values.description !== undefined) {
        values.remark = values.description;
        delete values.description;
      }

      if (editingInfoPoint) {
        await infoPointApi.update(editingInfoPoint.id, values);
        handleSuccess("更新");
      } else {
        await infoPointApi.create(values);
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
      await infoPointApi.delete(id);
      handleSuccess("删除");
      refreshData();
    } catch (error) {
      handleApiError(error, "删除");
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try {
      await infoPointApi.batch("delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      refreshData();
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  };

  const openModal = async (record?: InfoPoint) => {
    // 初始化 Cascader 选项（加载楼宇列表）
    await initCascaderOptions();

    if (record) {
      handleEdit(record);
      const formValues = { ...record } as Record<string, unknown>;

      // 如果有设备ID，先加载设备端口列表，确保表单回显时能找到匹配的端口
      if (formValues.deviceId) {
        setSelectedDeviceId(formValues.deviceId as string);
        // 直接调用 API 并同步设置状态，确保在 setFieldsValue 之前完成
        try {
          const result = await post("/network/ports/list", { deviceId: formValues.deviceId, current: 1, pageSize: 1000 }) as { data?: { list: DevicePortOption[] } };
          setDevicePorts(result.data?.list || []);
        } catch (error) {
          console.error("加载设备端口列表失败:", error);
          setDevicePorts([]);
        }
      }

      // 处理 workstationId：将单个字符串转换为三级路径数组
      if (formValues.workstationId && typeof formValues.workstationId === "string") {
        try {
          // 查询工位详细信息以获取所属楼层
          const wsResult = await workstationApi.get(formValues.workstationId);
          const workstation = wsResult.data as WorkstationOps | undefined;

          if (workstation && workstation.floorId) {
            // 查询楼层信息以获取所属楼宇
            const floorResult = await floorApi.get(workstation.floorId);
            const floor = floorResult.data as Floor | undefined;

            const buildingId = floor?.buildingId || floor?.buildingCode;
            const floorId = workstation.floorId;
            const workstationId = formValues.workstationId;

            if (buildingId && floorId && workstationId) {
              // 预加载 Cascader 路径数据，确保回显时能找到匹配的节点
              await preloadCascaderPath(buildingId, floorId);

              // 构建三级路径: [buildingId, floorId, workstationId]
              formValues.workstationId = [buildingId, floorId, workstationId];
            }
          }
        } catch (error) {
          console.error("获取工位信息失败:", error);
          formValues.workstationId = undefined;
        }
      }

      // 字段映射：后端的 remark 映射到前端的 description
      if (formValues.remark !== undefined) {
        formValues.description = formValues.remark;
        delete formValues.remark;
      }

      infoPointForm.setFieldsValue(formValues);
    }
    else {
      handleAdd();
      const defaultType = infoPointTypeDict.find(d => d.isDefault)?.dictValue || infoPointTypeDict[0]?.dictValue;
      infoPointForm.setFieldsValue({ status: 0, infoPointType: defaultType });
    }
  };

  // 预加载 Cascader 路径数据（用于编辑回显）
  const preloadCascaderPath = async (buildingId: string, floorId: string) => {
    try {
      // 找到对应的楼宇节点
      const buildingIndex = cascaderOptions.findIndex(b => b.value === buildingId);
      if (buildingIndex === -1) return;

      // 加载该楼宇下的楼层列表
      const floors = await loadFloorsForCascader(buildingId);

      // 找到对应的楼层节点
      const floorIndex = floors.findIndex(f => f.value === floorId);
      if (floorIndex === -1) return;

      const floorNo = floors[floorIndex].floorNo;

      // 加载该楼层下的工位列表
      const workstations = await loadWorkstationsForCascader(floorId, floorNo || floorId);

      // 更新楼层节点的 children（工位列表）- 使用类型断言
      (floors[floorIndex] as CascaderOption & { children?: CascaderOption[] }).children = workstations;

      // 更新楼宇节点的 children（楼层列表）- 使用类型断言
      const updatedOptions = [...cascaderOptions];
      (updatedOptions[buildingIndex] as CascaderOption & { children?: CascaderOption[] }).children = floors;

      // 触发重新渲染
      setCascaderOptions(updatedOptions);
    } catch (error) {
      console.error("预加载 Cascader 路径失败:", error);
    }
  };

  const handleImportSuccess = () => {
    refreshData();
    setImportVisible(false);
  };

  const getInfoPointTypeText = (type: string) => {
    const dictItem = infoPointTypeDict.find(d => d.dictValue === type);
    return dictItem?.dictLabel || type;
  };

  // 创建工位ID到名称的映射
  const workstationMap = useMemo(() => {
    const map = new Map<string, string>();
    workstationOptions.forEach(ws => {
      map.set(ws.id, ws.name);
    });
    return map;
  }, [workstationOptions]);

  const getWorkstationName = (workstationId: string) => {
    return workstationMap.get(workstationId) || workstationId;
  };

  const getStatusText = (status: number) => {
    const statusMap = { 0: "正常", 1: "故障", 2: "停用" };
    return statusMap[status as keyof typeof statusMap] || "未知";
  };

  // 截断设备名称，超出长度显示省略号
  const truncateDeviceName = (name: string | undefined, maxLength = 15) => {
    if (!name || name.length <= maxLength) return name || "";
    return name.substring(0, maxLength) + "...";
  };

  const columns: ColumnsType<InfoPoint> = [
    { title: "信息点名称", dataIndex: "name", key: "name", width: 150 },
    {
      title: "信息点类型",
      dataIndex: "infoPointType",
      key: "infoPointType",
      width: 100,
      render: (type) => <Tag>{getInfoPointTypeText(type)}</Tag>,
    },
    {
      title: "所属楼宇",
      dataIndex: "buildingName",
      key: "buildingName",
      width: 120,
      ellipsis: { showTitle: false },
      render: (text) => text || "-",
    },
    {
      title: "所属楼层",
      dataIndex: "floorName",
      key: "floorName",
      width: 120,
      ellipsis: { showTitle: false },
      render: (text) => text || "-",
    },
    {
      title: "所属工位",
      dataIndex: "workstationName",
      key: "workstationName",
      width: 150,
      ellipsis: { showTitle: false },
      render: (_, record) => {
        const name = record.workstationName || getWorkstationName(record.workstationId);
        return <Tooltip title={name}>{name || "-"}</Tooltip>;
      },
    },
    { title: "所属设备", dataIndex: "deviceName", key: "deviceName", width: 260, ellipsis: { showTitle: false }, render: (v) => v || "-" },
    { title: "所属端口", dataIndex: "portName", key: "portName", width: 140, ellipsis: { showTitle: false }, render: (v) => v || "-" },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      render: (status) => {
        const colors: Record<number, string> = { 0: "success", 1: "error", 2: "default" };
        return <Tag color={colors[status as keyof typeof colors]}>{getStatusText(status)}</Tag>;
      },
    },
    { title: "描述", dataIndex: "remark", key: "remark", width: 200, ellipsis: { showTitle: false }, render: (v) => <Tooltip title={v}>{v || "-"}</Tooltip> },
    createDateTimeColumn("createdAt", { width: 180 }),
    {
      title: "操作", key: "action",
      render: (_, record) => {
        const actions = [
          { key: "edit", label: "编辑", icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: "delete", label: "删除", icon: <DeleteOutlined />, danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除这个信息点吗？",
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
    if (infoPoints.length === 0) return <div style={{ textAlign: "center", padding: "40px", color: "var(--theme-text-tertiary, #999)" }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {infoPoints.map((infoPoint) => (
          <Col key={infoPoint.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(infoPoint)} />,
                <DeleteOutlined
                  key="delete"
                  style={{ color: "var(--theme-error, #ff4d4f)" }}
                  onClick={() => {
                    Modal.confirm({
                      title: "确定要删除这个信息点吗？",
                      okText: "确定",
                      cancelText: "取消",
                      okButtonProps: { danger: true },
                      onOk: () => handleDelete(infoPoint.id),
                    });
                  }}
                />,
              ]}
            >
              <Card.Meta
                title={
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <span>{infoPoint.name}</span>
                    <Tag color={infoPoint.status === 0 ? "success" : infoPoint.status === 1 ? "error" : "default"}>{getStatusText(infoPoint.status)}</Tag>
                  </div>
                }
                description={
                  <div>
                    <div><strong>类型：</strong>{getInfoPointTypeText(infoPoint.infoPointType)}</div>
                    <div><strong>工位：</strong>{infoPoint.workstationName || getWorkstationName(infoPoint.workstationId)}</div>
                    {infoPoint.deviceName && <div><strong>设备：</strong>{infoPoint.deviceName}</div>}
                    {infoPoint.portName && <div><strong>端口：</strong>{infoPoint.portName}</div>}
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
      <DeptSidebar selectedDeptId={selectedDeptId} onSelect={handleDeptSelect} />
      <Content style={{ background: "#fff" }}>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: "总信息点数", value: statistics.total, prefix: <DotChartOutlined /> },
            { title: "正常", value: statistics.normal, styles: { content: { color: "var(--theme-success, #3f8600)" } }, prefix: <CheckCircleOutlined /> },
            { title: "故障", value: statistics.fault, styles: { content: { color: "var(--theme-error, #cf1322)" } }, prefix: <WarningOutlined /> },
            { title: "停用", value: statistics.disabled, styles: { content: { color: "var(--theme-text-tertiary, #8c8c8c)" } }, prefix: <StopOutlined /> },
          ]}
        />
        <Card style={{ marginBottom: 16 }}>
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
            <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
              <Form.Item name="workstationId" label="所属工位">
                <Select
                  placeholder="请先选择部门"
                  allowClear
                  className="user-form-input"
                  style={{ width: 150 }}
                  disabled={!selectedDeptId}
                  onOpenChange={(open) => {
                    if (open) loadSearchWorkstationOptions();
                  }}
                  showSearch
                  filterOption={(input, option) => {
                    const ws = workstationOptions.find(w => w.id === option?.value);
                    return ws?.name.toLowerCase().includes(input.toLowerCase()) ?? false;
                  }}
                >
                  {workstationOptions.filter(w => w.id).map(w => <Option key={w.id} value={w.id}>{w.name}</Option>)}
                </Select>
              </Form.Item>
              <Form.Item name="name" label="信息点名称"><Input placeholder="请输入信息点名称" allowClear className="user-form-input" style={{ width: 150 }} /></Form.Item>
              <Form.Item name="infoPointType" label="信息点类型">
                <Select placeholder="请选择类型" allowClear className="user-form-input" style={{ width: 120 }}>
                  {infoPointTypeDict.map(d => <Option key={d.dictValue} value={d.dictValue}>{d.dictLabel}</Option>)}
                </Select>
              </Form.Item>
              <Form.Item name="status" label="状态">
                <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 100 }}><Option value={0}>正常</Option><Option value={1}>故障</Option><Option value={2}>停用</Option></Select>
              </Form.Item>
              <Form.Item>
                <Space><Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button><Button onClick={handleReset}>重置</Button><Button icon={<ReloadOutlined />} onClick={refreshData}>刷新</Button></Space>
              </Form.Item>
            </Form>
            <Space>
              <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
                <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button><Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button>
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
              <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增信息点</Button>
            </Space>
          </div>
          {selectedRowKeys.length > 0 && <Alert title={<span>已选择 <strong>{selectedRowKeys.length}</strong> 个信息点，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
        </Card>
        <Card>
          {viewMode === "table" ? (
            <Table
              rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }}
              columns={columns}
              dataSource={infoPoints}
              loading={loading}
              rowKey="id"
              pagination={paginationProps}
              onChange={(pagination) => {
                const newPage = pagination.current || 1;
                const newPageSize = pagination.pageSize || 10;
                setCurrent(newPage);
                setPageSize(newPageSize);
                const formValues = searchForm.getFieldsValue() as Record<string, unknown>;
	                const searchParams: Record<string, unknown> = { current: newPage, pageSize: newPageSize };
	                Object.keys(formValues).forEach(key => {
	                  const value = formValues[key];
	                  if (value !== undefined && value !== null && value !== "") {
	                    searchParams[key] = value;
	                  }
	                });
	                loadInfoPoints(searchParams);
              }}
            />
          ) : renderCardView()}
        </Card>
        <Modal title={editingInfoPoint ? "编辑信息点" : "新增信息点"} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); infoPointForm.resetFields(); setEditingInfoPoint(null); }} width={600}>
          <Form form={infoPointForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
            <Form.Item name="workstationId" label="所属工位" rules={[{ required: true, message: "请选择所属工位" }]}>
              <Cascader
                options={cascaderOptions}
                loadData={handleCascaderLoadData}
                loading={loadingCascader}
                placeholder="请先选择楼宇，再选择楼层，最后选择工位"
                changeOnSelect
                showSearch={{ filter: (inputValue, path) => path.some(option => option.label?.toLowerCase().includes(inputValue.toLowerCase())) }}
              />
            </Form.Item>
            <Form.Item name="name" label="信息点名称" rules={[{ required: true, message: "请输入信息点名称" }]}><Input placeholder="请输入信息点名称" /></Form.Item>
            <Form.Item name="infoPointType" label="信息点类型" rules={[{ required: true, message: "请选择信息点类型" }]}>
              <Select placeholder="请选择信息点类型">
                {infoPointTypeDict.map(d => <Option key={d.dictValue} value={d.dictValue}>{d.dictLabel}</Option>)}
              </Select>
            </Form.Item>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="deviceId" label="所属设备">
                  <Select
                    placeholder="请选择网络设备"
                    allowClear
                    showSearch
                    onChange={(value) => {
                      setSelectedDeviceId(value);
                      loadDevicePorts(value);
                    }}
                    filterOption={(input, option) => {
                      const device = networkDevices.find(d => d.id === option?.value);
                      if (!device) return false;
                      const deviceName = device.deviceName || "";
                      const searchText = `${deviceName} ${device.ipAddress}`.toLowerCase();
                      return searchText.includes(input.toLowerCase());
                    }}
                    optionLabelProp="label"
                  >
                    {networkDevices.map(d => {
                      const deviceName = d.deviceName || "未命名设备";
                      const fullName = `${deviceName} (${d.ipAddress})`;
                      const displayName = deviceName.length > 12 ? `${deviceName.substring(0, 12)}... (${d.ipAddress})` : fullName;
                      return (
                        <Option key={d.id} value={d.id} label={deviceName}>
                          <Tooltip title={fullName} placement="right">
                            <span>{displayName}</span>
                          </Tooltip>
                        </Option>
                      );
                    })}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="portId" label="所属端口">
                  <Select
                    placeholder={selectedDeviceId ? "请选择端口" : "请先选择设备"}
                    allowClear
                    disabled={!selectedDeviceId}
                    showSearch
                    filterOption={(input, option) =>
                      (option?.label as string)?.toString().toLowerCase().includes(input.toLowerCase())
                    }
                    optionLabelProp="label"
                  >
                    {devicePorts.map(p => {
                      const displayLabel = p.interfaceName || p.id || "未命名端口";
                      return (
                        <Option key={p.id} value={p.id} label={displayLabel}>
                          {displayLabel}
                        </Option>
                      );
                    })}
                  </Select>
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="status" label="状态" rules={[{ required: true, message: "请选择状态" }]}><Select placeholder="请选择状态"><Option value={0}>正常</Option><Option value={1}>故障</Option><Option value={2}>停用</Option></Select></Form.Item>
            <Form.Item name="description" label="描述"><TextArea rows={3} placeholder="请输入描述" /></Form.Item>
          </Form>
        </Modal>
        <ExcelImport entityType="infoPoint" entityName="信息点" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
        <ExcelExport entityType="infoPoint" entityName="信息点" visible={exportVisible} onClose={() => setExportVisible(false)} filters={exportFilters} />
      </Content>
    </Layout>
  );
};

export default InfoPointManagement;

