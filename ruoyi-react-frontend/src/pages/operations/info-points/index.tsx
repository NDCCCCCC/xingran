/**
 * 信息点管理页面
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Modal, Form, Input, Select, Popconfirm,
  Tag, Card, Row, Col, Statistic, Alert, Radio, Cascader, Tooltip,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, WarningOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined, DotChartOutlined,
} from '@ant-design/icons';
import type { InfoPoint, DictData } from '@/types';
import { infoPointApi, workstationApi, buildingApi, floorApi } from '@/lib/opsApi';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;

type ViewMode = 'table' | 'card';

interface WorkstationOption {
  id: string;
  name: string;
  buildingId?: string;
  buildingName?: string;
  floorId?: string;
  floorName?: string;
}

interface NetworkDeviceOption {
  id: string;
  name: string;
  ipAddress: string;
}

interface DevicePortOption {
  id: string;
  portName: string;
  status: string;
}

interface CascaderOption {
  value: string;
  label: string;
  children?: CascaderOption[];
  isLeaf?: boolean;
}

const InfoPointManagement: FC = () => {
  const [statistics, setStatistics] = useState({ total: 0, normal: 0, fault: 0, disabled: 0 });
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [workstationOptions, setWorkstationOptions] = useState<WorkstationOption[]>([]);
  const [infoPointTypeDict, setInfoPointTypeDict] = useState<DictData[]>([]);
  const [networkDevices, setNetworkDevices] = useState<NetworkDeviceOption[]>([]);
  const [devicePorts, setDevicePorts] = useState<DevicePortOption[]>([]);
  const [selectedDeviceId, setSelectedDeviceId] = useState<string>('');

  // Cascader 懒加载状态
  const [cascaderOptions, setCascaderOptions] = useState<CascaderOption[]>([]);
  const [loadingCascader, setLoadingCascader] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, current, pageSize, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading, data: infoPoints, total, selectedRowKeys,
    searchForm, editForm: infoPointForm, editModalVisible: modalVisible,
    editingItem: editingInfoPoint, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingInfoPoint, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadInfoPoints, resetSelection,
  } = useTableManager<InfoPoint>(
    async (params) => {
      const result = await infoPointApi.list(params);
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      externalPagination: {
        current,
        pageSize,
        setCurrent,
        setPageSize,
        setTotal,
      },
    }
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await infoPointApi.list({ current: 1, pageSize: 1000 });
      const allInfoPoints = result.data?.list || [];
      setStatistics({
        total: allInfoPoints.length,
        normal: allInfoPoints.filter((i: InfoPoint) => i.status === 0).length,
        fault: allInfoPoints.filter((i: InfoPoint) => i.status === 1).length,
        disabled: allInfoPoints.filter((i: InfoPoint) => i.status === 2).length,
      });
    } catch (error) { handleApiError(error, '加载统计数据', false); }
  }, []);

  // ==================== Cascader 懒加载函数 ====================

  // 加载楼宇列表（第一级）
  const loadBuildingsForCascader = useCallback(async () => {
    try {
      const buildingResult = await buildingApi.list({ current: 1, pageSize: 1000 });
      const buildings = buildingResult.data?.list || [];
      return buildings.map((b: any) => ({
        value: b.id,
        label: b.name,
        isLeaf: false, // 表示有子节点需要懒加载
      }));
    } catch (error) {
      handleApiError(error, '加载楼宇列表', false);
      return [];
    }
  }, []);

  // 加载指定楼宇的楼层（第二级）
  const loadFloorsForCascader = useCallback(async (buildingId: string) => {
    try {
      // 修复：使用 buildingId 而不是 buildingCode
      const floorResult = await floorApi.list({ buildingId, current: 1, pageSize: 1000 });
      const floors = floorResult.data?.list || [];
      return floors.map((f: any) => ({
        value: f.id,
        label: f.name || f.floorNo,
        isLeaf: false, // 表示有子节点需要懒加载
        floorNo: f.floorNo,
      }));
    } catch (error) {
      handleApiError(error, '加载楼层列表', false);
      return [];
    }
  }, []);

  // 加载指定楼层的工位（第三级）
  const loadWorkstationsForCascader = useCallback(async (floorId: string, floorNo: string) => {
    try {
      const wsResult = await workstationApi.list({ floorCode: floorNo, current: 1, pageSize: 1000 });
      const workstations = wsResult.data?.list || [];
      return workstations
        .filter((w: any) => w.id)
        .map((w: any) => ({
          value: w.id,
          label: w.name || '未命名工位',
          isLeaf: true,
        }));
    } catch (error) {
      handleApiError(error, '加载工位列表', false);
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
        const floorNo = (targetOption as any).floorNo;
        const workstations = await loadWorkstationsForCascader(floorId, floorNo || floorId);
        targetOption.children = workstations;
      }

      // 触发重新渲染
      setCascaderOptions([...cascaderOptions]);
    } finally {
      setLoadingCascader(false);
    }
  }, [cascaderOptions, loadFloorsForCascader, loadWorkstationsForCascader]);

  // 初始化 Cascader 选项（加载楼宇列表）
  const initCascaderOptions = useCallback(async () => {
    setLoadingCascader(true);
    try {
      const buildings = await loadBuildingsForCascader();
      setCascaderOptions(buildings);
    } finally {
      setLoadingCascader(false);
    }
  }, [loadBuildingsForCascader]);

  // 加载搜索表单的工位选项（按需加载）
  const loadSearchWorkstationOptions = useCallback(async () => {
    if (workstationOptions.length > 0) return; // 已加载则跳过

    try {
      const wsResult = await workstationApi.list({ current: 1, pageSize: 1000 });
      const workstations = wsResult.data?.list || [];
      setWorkstationOptions(
        workstations
          .filter((w: any) => w.id)
          .map((w: any) => ({
            id: w.id,
            name: w.name || '未命名工位',
          }))
      );
    } catch (error) {
      handleApiError(error, '加载工位选项', false);
    }
  }, [workstationOptions.length]);

  // ===========================================================

  const loadInfoPointTypeDict = useCallback(async () => {
    try {
      const result = await post('/system/dicts/data/list', { dictType: 'ops_info_point_type', current: 1, pageSize: 100 }) as { data?: { list: any[] } };
      setInfoPointTypeDict(result.data?.list || []);
    } catch (error) { handleApiError(error, '加载信息点类型字典', false); }
  }, []);

  const loadNetworkDevices = useCallback(async () => {
    try {
      const result = await post('/network/devices/list', { current: 1, pageSize: 1000 }) as { data?: { list: any[] } };
      const devices = result.data?.list || [];
      setNetworkDevices(devices.map((d: any) => ({
        id: d.id,
        name: d.deviceName,
        ipAddress: d.ipAddress,
      })));
    } catch (error) { handleApiError(error, '加载网络设备列表', false); }
  }, []);

  const loadDevicePorts = useCallback(async (deviceId: string) => {
    if (!deviceId) {
      setDevicePorts([]);
      return;
    }
    try {
      const result = await post('/network/ports/list', { deviceId, current: 1, pageSize: 1000 }) as { data?: { list: any[] } };
      const ports = result.data?.list || [];
      setDevicePorts(ports.map((p: any) => ({
        id: p.id,
        portName: p.interfaceName || p.portName || p.name,
        status: p.status,
      })));
    } catch (error) { handleApiError(error, '加载设备端口列表', false); }
  }, []);

  // 使用 ref 存储加载函数，避免函数引用变化导致 useEffect 频繁触发
  const loadFunctionsRef = useRef({
    loadInfoPoints,
    loadStatistics,
    loadInfoPointTypeDict,
    loadNetworkDevices,
  });
  loadFunctionsRef.current = {
    loadInfoPoints,
    loadStatistics,
    loadInfoPointTypeDict,
    loadNetworkDevices,
  };

  useEffect(() => {
    // 使用 setTimeout 避免同步 setState
    setTimeout(() => {
      loadFunctionsRef.current.loadInfoPoints();
      loadFunctionsRef.current.loadStatistics();
      loadFunctionsRef.current.loadInfoPointTypeDict();
      loadFunctionsRef.current.loadNetworkDevices();
    }, 0);
  }, [current, pageSize]);

  const handleSave = async () => {
    try {
      const values = await infoPointForm.validateFields();

      // 处理 Cascader 的 workstationId 值（从数组提取最后一个元素）
      if (Array.isArray(values.workstationId)) {
        values.workstationId = values.workstationId[values.workstationId.length - 1];
      }

      if (editingInfoPoint) {
        await infoPointApi.update(editingInfoPoint.id, values);
        handleSuccess('更新');
      } else {
        await infoPointApi.create(values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadInfoPoints();
      loadStatistics();
    } catch (error: any) { if (error.errorFields) return; handleApiError(error, '操作'); }
  };

  const handleDelete = async (id: string) => {
    try { await infoPointApi.delete(id); handleSuccess('删除'); loadInfoPoints(); loadStatistics(); }
    catch (error) { handleApiError(error, '删除'); }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try { await infoPointApi.batch('delete', { ids: selectedRowKeys }); handleSuccess('批量删除'); resetSelection(); loadInfoPoints(); loadStatistics(); }
    catch (error) { handleApiError(error, '批量删除'); }
  };

  const openModal = async (record?: InfoPoint) => {
    // 初始化 Cascader 选项（加载楼宇列表）
    await initCascaderOptions();

    if (record) {
      handleEdit(record);
      const formValues = { ...record };

      // 处理 workstationId：将单个字符串转换为三级路径数组
      if (formValues.workstationId && typeof formValues.workstationId === 'string') {
        try {
          // 查询工位详细信息以获取所属楼层
          const wsResult = await workstationApi.get(formValues.workstationId);
          const workstation = wsResult.data as any;

          if (workstation && workstation.floorId) {
            // 查询楼层信息以获取所属楼宇
            const floorResult = await floorApi.get(workstation.floorId);
            const floor = floorResult.data as any;

            const buildingId = floor?.buildingId || floor?.buildingCode;
            const floorId = workstation.floorId;
            const workstationId = formValues.workstationId;

            if (buildingId && floorId && workstationId) {
              // 预加载 Cascader 路径数据，确保回显时能找到匹配的节点
              await preloadCascaderPath(buildingId, floorId);

              // 构建三级路径: [buildingId, floorId, workstationId]
              formValues.workstationId = [buildingId, floorId, workstationId] as any;
            }
          }
        } catch (error) {
          console.error('获取工位信息失败:', error);
          formValues.workstationId = undefined;
        }
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
      (floors[floorIndex] as any).children = workstations;

      // 更新楼宇节点的 children（楼层列表）- 使用类型断言
      const updatedOptions = [...cascaderOptions];
      (updatedOptions[buildingIndex] as any).children = floors;

      // 触发重新渲染
      setCascaderOptions(updatedOptions);
    } catch (error) {
      console.error('预加载 Cascader 路径失败:', error);
    }
  };

  const handleImportSuccess = () => { loadInfoPoints(); loadStatistics(); setImportVisible(false); };

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
    const statusMap = { 0: '正常', 1: '故障', 2: '停用' };
    return statusMap[status as keyof typeof statusMap] || '未知';
  };

  // 截断设备名称，超出长度显示省略号
  const truncateDeviceName = (name: string, maxLength = 15) => {
    if (name.length <= maxLength) return name;
    return name.substring(0, maxLength) + '...';
  };

  const columns: ColumnsType<InfoPoint> = [
    { title: '信息点名称', dataIndex: 'name', key: 'name', width: 150 },
    {
      title: '信息点类型',
      dataIndex: 'infoPointType',
      key: 'infoPointType',
      width: 100,
      render: (type) => <Tag>{getInfoPointTypeText(type)}</Tag>,
    },
    {
      title: '所属工位',
      dataIndex: 'workstationName',
      key: 'workstationName',
      width: 150,
      ellipsis: { showTitle: false },
      render: (_, record) => {
        const name = record.workstationName || getWorkstationName(record.workstationId);
        return <Tooltip title={name}>{name || '-'}</Tooltip>;
      },
    },
    { title: '所属设备', dataIndex: 'deviceName', key: 'deviceName', width: 180, ellipsis: { showTitle: false }, render: (v) => v || '-' },
    { title: '所属端口', dataIndex: 'portName', key: 'portName', width: 180, ellipsis: { showTitle: false }, render: (v) => v || '-' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status) => {
        const colors: Record<number, string> = { 0: 'success', 1: 'error', 2: 'default' };
        return <Tag color={colors[status as keyof typeof colors]}>{getStatusText(status)}</Tag>;
      },
    },
    { title: '描述', dataIndex: 'description', key: 'description', width: 200, ellipsis: { showTitle: false }, render: (v) => <Tooltip title={v}>{v || '-'}</Tooltip> },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作', key: 'action',
      render: (_, record) => {
        const actions = [
          { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true,
            render: () => (
              <Popconfirm title="确定要删除这个信息点吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
                <Button type="link" danger icon={<DeleteOutlined />} size="small">删除</Button>
              </Popconfirm>
            ),
          },
        ];
        return <ActionButtons actions={actions} />;
      },
    },
  ];

  const renderCardView = () => {
    if (infoPoints.length === 0) return <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {infoPoints.map((infoPoint) => (
          <Col key={infoPoint.id} xs={24} sm={12} md={8} lg={6}>
            <Card
              hoverable
              actions={[
                <EditOutlined key="edit" onClick={() => openModal(infoPoint)} />,
                <Popconfirm key="delete" title="确定要删除这个信息点吗？" onConfirm={() => handleDelete(infoPoint.id)} okText="确定" cancelText="取消">
                  <DeleteOutlined style={{ color: '#ff4d4f' }} />
                </Popconfirm>,
              ]}
            >
              <Card.Meta
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>{infoPoint.name}</span>
                    <Tag color={infoPoint.status === 0 ? 'success' : infoPoint.status === 1 ? 'error' : 'default'}>{getStatusText(infoPoint.status)}</Tag>
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
    <div>
      {total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}><Card><Statistic title="总信息点数" value={statistics.total} prefix={<DotChartOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="正常" value={statistics.normal} styles={{ content: { color: '#3f8600' } }} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="故障" value={statistics.fault} styles={{ content: { color: '#cf1322' } }} prefix={<WarningOutlined />} /></Card></Col>
          <Col span={6}><Card><Statistic title="停用" value={statistics.disabled} styles={{ content: { color: '#8c8c8c' } }} prefix={<StopOutlined />} /></Card></Col>
        </Row>
      )}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="workstationId" label="所属工位">
              <Select
                placeholder="请选择工位"
                allowClear
                style={{ width: 150 }}
                onDropdownVisibleChange={(open) => {
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
            <Form.Item name="name" label="信息点名称"><Input placeholder="请输入信息点名称" allowClear style={{ width: 150 }} /></Form.Item>
            <Form.Item name="infoPointType" label="信息点类型">
              <Select placeholder="请选择类型" allowClear style={{ width: 120 }}>
                {infoPointTypeDict.map(d => <Option key={d.dictValue} value={d.dictValue}>{d.dictLabel}</Option>)}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 100 }}><Option value={0}>正常</Option><Option value={1}>故障</Option><Option value={2}>停用</Option></Select>
            </Form.Item>
            <Form.Item>
              <Space><Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button><Button onClick={handleReset}>重置</Button><Button icon={<ReloadOutlined />} onClick={() => { loadInfoPoints(); loadStatistics(); }}>刷新</Button></Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button><Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>导入</Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>导出</Button>
            {selectedRowKeys.length > 0 && <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>批量删除 ({selectedRowKeys.length})</Button>}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增信息点</Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && <Alert title={<span>已选择 <strong>{selectedRowKeys.length}</strong> 个信息点，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
      </Card>
      <Card>
        {viewMode === 'table' ? (
          <Table rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }} columns={columns} dataSource={infoPoints} loading={loading} rowKey="id" pagination={paginationProps} />
        ) : renderCardView()}
      </Card>
      <Modal title={editingInfoPoint ? '编辑信息点' : '新增信息点'} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); infoPointForm.resetFields(); setEditingInfoPoint(null); }} width={600}>
        <Form form={infoPointForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item name="workstationId" label="所属工位" rules={[{ required: true, message: '请选择所属工位' }]}>
            <Cascader
              options={cascaderOptions}
              loadData={handleCascaderLoadData}
              loading={loadingCascader}
              placeholder="请先选择楼宇，再选择楼层，最后选择工位"
              changeOnSelect
              showSearch={{ filter: (inputValue, path) => path.some(option => option.label?.toLowerCase().includes(inputValue.toLowerCase())) }}
            />
          </Form.Item>
          <Form.Item name="name" label="信息点名称" rules={[{ required: true, message: '请输入信息点名称' }]}><Input placeholder="请输入信息点名称" /></Form.Item>
          <Form.Item name="infoPointType" label="信息点类型" rules={[{ required: true, message: '请选择信息点类型' }]}>
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
                    const searchText = `${device.name} ${device.ipAddress}`.toLowerCase();
                    return searchText.includes(input.toLowerCase());
                  }}
                  optionLabelProp="label"
                >
                  {networkDevices.map(d => {
                    const fullName = `${d.name} (${d.ipAddress})`;
                    const displayName = `${truncateDeviceName(d.name)} (${d.ipAddress})`;
                    return (
                      <Option key={d.id} value={d.id} label={displayName} name={d.name}>
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
                >
                  {devicePorts.map(p => (
                    <Option key={p.id} value={p.id}>
                      {p.portName}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}><Select placeholder="请选择状态"><Option value={0}>正常</Option><Option value={1}>故障</Option><Option value={2}>停用</Option></Select></Form.Item>
          <Form.Item name="description" label="描述"><TextArea rows={3} placeholder="请输入描述" /></Form.Item>
        </Form>
      </Modal>
      <ExcelImport entityType="infoPoint" entityName="信息点" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
      <ExcelExport entityType="infoPoint" entityName="信息点" visible={exportVisible} onClose={() => setExportVisible(false)} />
    </div>
  );
};

export default InfoPointManagement;
