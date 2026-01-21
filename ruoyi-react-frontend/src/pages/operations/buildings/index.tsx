/**
 * 楼宇管理页面
 */

import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import {
  Table, Button, Space, Modal, Form, Input, InputNumber, Select, TreeSelect, Popconfirm,
  Tag, Card, Row, Col, Statistic, Alert, Radio,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined,
  CheckCircleOutlined, StopOutlined, AppstoreOutlined, TableOutlined,
  ImportOutlined, ExportOutlined,
} from '@ant-design/icons';
import type { Building } from '@/types/operations';
import { buildingApi } from '@/lib/opsApi';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { createStatusColumn, createDateTimeColumn } from '@/utils/tableHelpers';
import ActionButtons from '@/components/shared/ActionButtons';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';
import { post } from '@/lib/api';
import type { BaseResponse, PageResponse } from '@/types';

const { Option } = Select;
const { TextArea } = Input;

interface DepartmentOption {
  id: string;
  deptName: string;
  children?: DepartmentOption[];
}

type ViewMode = 'table' | 'card';

const BuildingManagement: FC = () => {
  const [statistics, setStatistics] = useState({ total: 0, active: 0, inactive: 0 });
  const [viewMode, setViewMode] = useState<ViewMode>('table');
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [departments, setDepartments] = useState<DepartmentOption[]>([]);
  const [loadingDepts, setLoadingDepts] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 加载部门列表
  const loadDepartments = useCallback(async () => {
    setLoadingDepts(true);
    try {
      // 后端返回的是树形结构数组
      const result = await post<BaseResponse<DepartmentOption[]>>('/system/departments/list', {});
      // 转换为 TreeSelect 需要的格式：title 和 value
      const convertToTreeData = (nodes: any[]): DepartmentOption[] => {
        return nodes.map(node => ({
          id: node.id,
          deptName: node.deptName,
          children: node.children?.length > 0 ? convertToTreeData(node.children) : undefined,
        }));
      };
      setDepartments(convertToTreeData(result.data || []));
    } catch (error) {
      handleApiError(error, '加载部门列表', false);
    } finally {
      setLoadingDepts(false);
    }
  }, []);

  const {
    loading, data: buildings, total, selectedRowKeys,
    searchForm, editForm: buildingForm, editModalVisible: modalVisible,
    editingItem: editingBuilding, setSelectedRowKeys, setEditModalVisible: setModalVisible,
    setEditingItem: setEditingBuilding, handleSearch,
    handleReset, handleAdd, handleEdit, handleModalClose, loadData: loadBuildings, resetSelection,
  } = useTableManager<any>(
    async (params) => {
      const result = await buildingApi.list(params) as { data?: { list: any[]; total: number } };
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      externalPagination: {
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        setCurrent,
        setPageSize,
        setTotal,
      },
    }
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await buildingApi.list({ current: 1, pageSize: 1000 }) as { data?: { list: any[] } };
      const allBuildings = result.data?.list || [];
      setStatistics({
        total: allBuildings.length,
        active: allBuildings.filter((b: any) => b.status === 0).length,
        inactive: allBuildings.filter((b: any) => b.status === 1).length,
      });
    } catch (error) { handleApiError(error, '加载统计数据', false); }
  }, []);

  useEffect(() => {
    loadBuildings();
    loadStatistics();
    loadDepartments();
  }, [paginationProps.current, paginationProps.pageSize]);

  const handleSave = async () => {
    try {
      const values = await buildingForm.validateFields();
      if (editingBuilding) {
        await buildingApi.update(editingBuilding.id, values);
        handleSuccess('更新');
      } else {
        await buildingApi.create(values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadBuildings();
      loadStatistics();
    } catch (error: any) { if (error.errorFields) return; handleApiError(error, '操作'); }
  };

  const handleDelete = async (id: string) => {
    try { await buildingApi.delete(id); handleSuccess('删除'); loadBuildings(); loadStatistics(); }
    catch (error) { handleApiError(error, '删除'); }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;
    try { await buildingApi.batch('delete', { ids: selectedRowKeys }); handleSuccess('批量删除'); resetSelection(); loadBuildings(); loadStatistics(); }
    catch (error) { handleApiError(error, '批量删除'); }
  };

  const openModal = (record?: any) => {
    if (record) { handleEdit(record); buildingForm.setFieldsValue(record); }
    else { handleAdd(); buildingForm.setFieldsValue({ status: 0 }); }
  };

  const handleImportSuccess = () => { loadBuildings(); loadStatistics(); setImportVisible(false); };

  // 扁平化部门数据用于查找
  const deptMap = new Map<string, string>();
  const flattenDeptsToMap = (nodes: DepartmentOption[]) => {
    nodes.forEach(node => {
      deptMap.set(node.id, node.deptName);
      if (node.children?.length > 0) {
        flattenDeptsToMap(node.children);
      }
    });
  };
  flattenDeptsToMap(departments);

  // 获取所属机构名称
  const getOrgName = (orgId?: string) => {
    if (!orgId) return '-';
    return deptMap.get(orgId) || '-';
  };

  const columns: ColumnsType<Building> = [
    { title: '楼宇名称', dataIndex: 'name', key: 'name', width: 150 },
    { title: '所属机构', dataIndex: 'orgId', key: 'orgName', width: 150, render: (orgId) => getOrgName(orgId) },
    { title: '地址', dataIndex: 'address', key: 'address', ellipsis: true },
    { title: '楼层数', dataIndex: 'totalFloors', key: 'totalFloors', width: 100, render: (v) => v || 0 },
    createStatusColumn('status', { width: 100 }),
    { title: '描述', dataIndex: 'remark', key: 'remark', ellipsis: true },
    createDateTimeColumn('createdAt', { width: 180 }),
    {
      title: '操作', key: 'action',
      render: (_, record) => {
        const actions = [
          { key: 'edit', label: '编辑', icon: <EditOutlined />, onClick: () => openModal(record) },
          {
            key: 'delete', label: '删除', icon: <DeleteOutlined />, danger: true,
            render: () => (
              <Popconfirm title="确定要删除这个楼宇吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
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
    if (buildings.length === 0) return <div style={{ textAlign: 'center', padding: '40px', color: '#999' }}>暂无数据</div>;
    return (
      <Row gutter={[16, 16]}>
        {buildings.map((building) => (
          <Col key={building.id} xs={24} sm={12} md={8} lg={6}>
            <Card hoverable actions={[
              <EditOutlined key="edit" onClick={() => openModal(building)} />,
              <Popconfirm key="delete" title="确定要删除这个楼宇吗？" onConfirm={() => handleDelete(building.id)} okText="确定" cancelText="取消">
                <DeleteOutlined style={{ color: '#ff4d4f' }} />
              </Popconfirm>,
            ]}>
              <Card.Meta
                title={
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span>{building.name}</span>
                    <Tag color={building.status === 0 ? 'success' : 'error'}>{building.status === 0 ? '正常' : '停用'}</Tag>
                  </div>
                }
                description={
                  <div>
                    {building.address && <div><strong>地址：</strong>{building.address}</div>}
                    <div><strong>所属机构：</strong>{getOrgName(building.orgId)}</div>
                    <div><strong>楼层：</strong>{building.totalFloors || 0}层</div>
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
          <Col span={8}><Card><Statistic title="总楼宇数" value={statistics.total} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={8}><Card><Statistic title="正常楼宇" value={statistics.active} styles={{ content: { color: '#3f8600' } }} prefix={<CheckCircleOutlined />} /></Card></Col>
          <Col span={8}><Card><Statistic title="停用楼宇" value={statistics.inactive} styles={{ content: { color: '#cf1322' } }} prefix={<StopOutlined />} /></Card></Col>
        </Row>
      )}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="name" label="楼宇名称"><Input placeholder="请输入楼宇名称" allowClear style={{ width: 150 }} /></Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 120 }}><Option value={0}>正常</Option><Option value={1}>停用</Option></Select>
            </Form.Item>
            <Form.Item>
              <Space><Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>搜索</Button><Button onClick={handleReset}>重置</Button><Button icon={<ReloadOutlined />} onClick={() => { loadBuildings(); loadStatistics(); }}>刷新</Button></Space>
            </Form.Item>
          </Form>
          <Space>
            <Radio.Group value={viewMode} onChange={(e) => setViewMode(e.target.value)} buttonStyle="solid">
              <Radio.Button value="table"><TableOutlined /> 表格</Radio.Button><Radio.Button value="card"><AppstoreOutlined /> 卡片</Radio.Button>
            </Radio.Group>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>导入</Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>导出</Button>
            {selectedRowKeys.length > 0 && <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>批量删除 ({selectedRowKeys.length})</Button>}
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增楼宇</Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && <Alert title={<span>已选择 <strong>{selectedRowKeys.length}</strong> 个楼宇，<Button type="link" size="small" onClick={() => setSelectedRowKeys([])} style={{ padding: 0 }}>取消选择</Button></span>} type="info" showIcon style={{ marginTop: 12 }} />}
      </Card>
      <Card>
        {viewMode === 'table' ? (
          <Table rowSelection={{ selectedRowKeys, onChange: setSelectedRowKeys }} columns={columns} dataSource={buildings} loading={loading} rowKey="id" pagination={paginationProps} />
        ) : renderCardView()}
      </Card>
      <Modal title={editingBuilding ? '编辑楼宇' : '新增楼宇'} open={modalVisible} onOk={handleSave} onCancel={() => { setModalVisible(false); buildingForm.resetFields(); setEditingBuilding(null); }} width={600}>
        <Form form={buildingForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
          <Form.Item name="name" label="楼宇名称" rules={[{ required: true, message: '请输入楼宇名称' }]}><Input placeholder="请输入楼宇名称" /></Form.Item>
          <Form.Item name="orgId" label="所属机构">
            <TreeSelect
              placeholder="请选择所属机构"
              loading={loadingDepts}
              allowClear
              showSearch
              treeDefaultExpandAll
              treeNodeFilterProp="title"
              fieldNames={{ label: 'deptName', value: 'id', children: 'children' }}
              treeData={departments}
            />
          </Form.Item>
          <Form.Item name="address" label="地址"><Input placeholder="请输入地址" /></Form.Item>
          <Form.Item name="totalFloors" label="楼层数">
            <InputNumber min={0} disabled style={{ width: '100%' }} placeholder="根据创建的楼层自动计算" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true, message: '请选择状态' }]}><Select placeholder="请选择状态"><Option value={0}>正常</Option><Option value={1}>停用</Option></Select></Form.Item>
          <Form.Item name="remark" label="描述"><TextArea rows={3} placeholder="请输入描述" /></Form.Item>
        </Form>
      </Modal>
      <ExcelImport entityType="building" entityName="楼宇" visible={importVisible} onClose={() => setImportVisible(false)} onImportSuccess={handleImportSuccess} />
      <ExcelExport entityType="building" entityName="楼宇" visible={exportVisible} onClose={() => setExportVisible(false)} />
    </div>
  );
};

export default BuildingManagement;
