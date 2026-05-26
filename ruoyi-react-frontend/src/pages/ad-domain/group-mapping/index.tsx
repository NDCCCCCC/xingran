import { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Select,
  Input,
  Space,
  Button,
  Tag,
  Modal,
  Form,
  message,
  Tooltip,
  Popconfirm,
  Descriptions,
} from 'antd';
import {
  ReloadOutlined,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  SyncOutlined,
  ThunderboltOutlined,
  LinkOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  getMappingList,
  createMapping,
  updateMapping,
  deleteMapping,
  autoMapAllDepartments,
  getADGroupList,
  type DeptGroupMapping,
  type MappingListRequest,
  type CreateMappingRequest,
} from '@/lib/adDomainApi';
import { useADConfigs } from '@/hooks/useADConfigs';
import { getDeptTree } from '@/lib/dutyApi';
import type { SimpleDept } from '@/lib/dutyApi';
import { usePagination } from '@/hooks/usePagination';
import type { FC } from 'react';

const GroupMappingPage: FC = () => {
  const [addForm] = Form.useForm();
  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  // Mapping list state
  const [mappings, setMappings] = useState<DeptGroupMapping[]>([]);
  const [loading, setLoading] = useState(false);
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // Department tree for add modal
  const [deptTree, setDeptTree] = useState<SimpleDept[]>([]);
  const [flatDepts, setFlatDepts] = useState<{ id: string; deptName: string }[]>([]);

  // AD groups for add modal
  const [adGroups, setAdGroups] = useState<{ id: string; groupName: string; groupDn: string }[]>([]);

  // Modal state
  const [addModalVisible, setAddModalVisible] = useState(false);
  const [addLoading, setAddLoading] = useState(false);
  const [autoMapLoading, setAutoMapLoading] = useState(false);

  // Search state
  const [searchGroupName, setSearchGroupName] = useState<string>();

  // Flatten department tree for Select component
  const flattenDepts = (nodes: SimpleDept[]): { id: string; deptName: string }[] => {
    const result: { id: string; deptName: string }[] = [];
    const traverse = (items: SimpleDept[], prefix = '') => {
      for (const item of items) {
        result.push({ id: item.id, deptName: prefix + item.deptName });
        if (item.children && item.children.length > 0) {
          traverse(item.children, prefix + item.deptName + ' / ');
        }
      }
    };
    traverse(nodes);
    return result;
  };

  // Fetch department tree
  useEffect(() => {
    const fetchDepts = async () => {
      try {
        const res = await getDeptTree();
        if (res.code === 0 && res.data) {
          setDeptTree(res.data);
          setFlatDepts(flattenDepts(res.data));
        }
      } catch {
        // Ignore errors
      }
    };
    fetchDepts();
  }, []);

  // Fetch mappings
  const fetchMappings = useCallback(async (groupName?: string) => {
    if (!selectedConfig) return;
    setLoading(true);
    try {
      const params: MappingListRequest = {
        adConfigId: selectedConfig,
        groupName: groupName,
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
      };
      const res = await getMappingList(params);
      if (res.code === 0 && res.data) {
        setMappings(res.data.list ?? []);
        setTotal(res.data.total ?? 0);
      }
    } catch {
      message.error('获取映射列表失败');
    } finally {
      setLoading(false);
    }
  }, [selectedConfig, paginationProps.current, paginationProps.pageSize, setTotal]);

  useEffect(() => {
    if (selectedConfig) {
      fetchMappings(searchGroupName);
    }
  }, [selectedConfig, paginationProps.current, paginationProps.pageSize, searchGroupName, fetchMappings]);

  // Fetch AD groups for the add modal
  const fetchADGroups = async () => {
    if (!selectedConfig) return;
    try {
      const res = await getADGroupList({
        configId: selectedConfig,
        current: 1,
        pageSize: 500,
      });
      if (res.code === 0 && res.data) {
        setAdGroups(
          (res.data.list ?? []).map(g => ({
            id: g.id,
            groupName: g.groupName,
            groupDn: g.groupDn,
          }))
        );
      }
    } catch {
      message.error('获取AD组列表失败');
    }
  };

  // Handle search
  const handleSearch = () => {
    const name = document.querySelector<HTMLInputElement>('input[placeholder="搜索组名"]')?.value;
    setCurrent(1);
    setSearchGroupName(name || undefined);
  };

  // Handle add mapping
  const handleAdd = () => {
    if (!selectedConfig) {
      message.warning('请先选择AD配置');
      return;
    }
    fetchADGroups();
    addForm.resetFields();
    setAddModalVisible(true);
  };

  const handleAddSubmit = async () => {
    try {
      const values = await addForm.validateFields();
      setAddLoading(true);
      const req: CreateMappingRequest = {
        deptId: values.deptId,
        adGroupId: values.adGroupId,
        adConfigId: selectedConfig,
        mappingType: 'manual',
        syncEnabled: values.syncEnabled ?? true,
      };
      const res = await createMapping(req);
      if (res.code === 0) {
        message.success('创建映射成功');
        setAddModalVisible(false);
        fetchMappings(searchGroupName);
      } else {
        message.error(res.message || '创建映射失败');
      }
    } catch (err) {
      if (err instanceof Error) {
        message.error(err.message);
      }
    } finally {
      setAddLoading(false);
    }
  };

  // Handle delete mapping
  const handleDelete = async (id: string) => {
    try {
      const res = await deleteMapping(id);
      if (res.code === 0) {
        message.success('删除映射成功');
        fetchMappings(searchGroupName);
      } else {
        message.error(res.message || '删除映射失败');
      }
    } catch {
      message.error('删除映射失败');
    }
  };

  // Handle toggle sync enabled
  const handleToggleSync = async (mapping: DeptGroupMapping) => {
    try {
      const res = await updateMapping(mapping.id, {
        syncEnabled: !mapping.syncEnabled,
      });
      if (res.code === 0) {
        message.success(mapping.syncEnabled ? '已停用同步' : '已启用同步');
        fetchMappings(searchGroupName);
      }
    } catch {
      message.error('更新失败');
    }
  };

  // Handle auto-map all
  const handleAutoMapAll = async () => {
    if (!selectedConfig) {
      message.warning('请先选择AD配置');
      return;
    }
    setAutoMapLoading(true);
    try {
      const res = await autoMapAllDepartments(selectedConfig);
      if (res.code === 0 && res.data) {
        const r = res.data;
        message.success(
          `自动映射完成: 总部门=${r.totalDepts}, 成功=${r.mappedCount}, 失败=${r.failedCount}`
        );
        if (r.failedDepts && r.failedDepts.length > 0) {
          Modal.info({
            title: '部分部门映射失败',
            width: 500,
            content: (
              <div>
                {r.failedDepts.map((d, i) => (
                  <div key={i} style={{ color: '#ff4d4f', marginBottom: 4 }}>{d}</div>
                ))}
              </div>
            ),
          });
        }
        fetchMappings(searchGroupName);
      } else {
        message.error(res.message || '自动映射失败');
      }
    } catch {
      message.error('自动映射失败');
    } finally {
      setAutoMapLoading(false);
    }
  };

  // Mapping type tag
  const renderMappingType = (type: string) => {
    if (type === 'auto') return <Tag color="blue">自动</Tag>;
    return <Tag color="green">手动</Tag>;
  };

  // Mapping status tag
  const renderMappingStatus = (status: string) => {
    if (status === 'active') return <Tag color="success" icon={<CheckCircleOutlined />}>激活</Tag>;
    return <Tag color="default" icon={<CloseCircleOutlined />}>停用</Tag>;
  };

  // Table columns
  const columns: ColumnsType<DeptGroupMapping> = [
    {
      title: '部门',
      key: 'dept',
      width: 200,
      render: (_: unknown, record: DeptGroupMapping) => {
        const deptName = record.Dept?.deptName;
        if (deptName) {
          const flat = flatDepts.find(d => d.id === record.deptId);
          return flat ? flat.deptName : deptName;
        }
        return <Tag>未知</Tag>;
      },
    },
    {
      title: 'AD组名',
      dataIndex: 'groupName',
      key: 'groupName',
      width: 200,
    },
    {
      title: '类型',
      dataIndex: 'mappingType',
      key: 'mappingType',
      width: 80,
      render: renderMappingType,
    },
    {
      title: '状态',
      dataIndex: 'mappingStatus',
      key: 'mappingStatus',
      width: 80,
      render: renderMappingStatus,
    },
    {
      title: '同步',
      key: 'syncEnabled',
      width: 80,
      render: (_: unknown, record: DeptGroupMapping) => (
        <Tag
          color={record.syncEnabled ? 'success' : 'warning'}
          style={{ cursor: 'pointer' }}
          onClick={() => handleToggleSync(record)}
        >
          {record.syncEnabled ? '已启用' : '已停用'}
        </Tag>
      ),
    },
    {
      title: '最后同步',
      dataIndex: 'lastSyncAt',
      key: 'lastSyncAt',
      width: 170,
      render: (val: string) =>
        val ? new Date(val).toLocaleString() : <Tag color="warning">未同步</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (val: string) => (val ? new Date(val).toLocaleString() : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      fixed: 'right',
      render: (_: unknown, record: DeptGroupMapping) => (
        <Space>
          <Popconfirm
            title="确定删除此映射？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Card>
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        {/* Toolbar */}
        <Form layout="inline">
          <Form.Item label="AD配置">
            <Select
              style={{ width: 200 }}
              value={selectedConfig}
              onChange={(val) => {
                setSelectedConfig(val);
                setCurrent(1);
              }}
              options={configs.map(c => ({ label: c.configName, value: c.id }))}
            />
          </Form.Item>
          <Form.Item>
            <Input
              placeholder="搜索组名"
              allowClear
              style={{ width: 160 }}
              onPressEnter={handleSearch}
              onChange={(e) => {
                if (!e.target.value) {
                  setSearchGroupName(undefined);
                  setCurrent(1);
                }
              }}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button
                icon={<ReloadOutlined />}
                onClick={() => fetchMappings(searchGroupName)}
              >
                刷新
              </Button>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={handleAdd}
                disabled={!selectedConfig}
              >
                添加映射
              </Button>
              <Tooltip title="自动为所有二级部门创建映射（匹配 cxhub-{部门名} 组）">
                <Button
                  icon={<ThunderboltOutlined />}
                  onClick={handleAutoMapAll}
                  loading={autoMapLoading}
                  disabled={!selectedConfig}
                >
                  批量自动映射
                </Button>
              </Tooltip>
            </Space>
          </Form.Item>
        </Form>

        {/* Stats summary */}
        {mappings.length > 0 && (
          <Descriptions size="small" bordered column={4}>
            <Descriptions.Item label="总映射数">{paginationProps.total}</Descriptions.Item>
            <Descriptions.Item label="自动映射">
              {mappings.filter(m => m.mappingType === 'auto').length}
            </Descriptions.Item>
            <Descriptions.Item label="手动映射">
              {mappings.filter(m => m.mappingType === 'manual').length}
            </Descriptions.Item>
            <Descriptions.Item label="同步启用">
              {mappings.filter(m => m.syncEnabled).length}
            </Descriptions.Item>
          </Descriptions>
        )}

        {/* Mapping table */}
        <Table
          columns={columns}
          dataSource={mappings}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
          scroll={{ x: 1200 }}
        />
      </Space>

      {/* Add mapping modal */}
      <Modal
        title="添加部门-组映射"
        open={addModalVisible}
        onOk={handleAddSubmit}
        onCancel={() => setAddModalVisible(false)}
        confirmLoading={addLoading}
        okText="确定"
        cancelText="取消"
        width={500}
      >
        <Form form={addForm} layout="vertical">
          <Form.Item
            label="部门"
            name="deptId"
            rules={[{ required: true, message: '请选择部门' }]}
          >
            <Select
              showSearch
              placeholder="选择部门"
              optionFilterProp="children"
              style={{ width: '100%' }}
            >
              {flatDepts.map(d => (
                <Select.Option key={d.id} value={d.id}>
                  {d.deptName}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            label="AD组"
            name="adGroupId"
            rules={[{ required: true, message: '请选择AD组' }]}
          >
            <Select
              showSearch
              placeholder="选择AD组"
              optionFilterProp="children"
              style={{ width: '100%' }}
            >
              {adGroups.map(g => (
                <Select.Option key={g.id} value={g.id}>
                  {g.groupName}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            label="启用同步"
            name="syncEnabled"
            valuePropName="checked"
            initialValue={true}
          >
            <Select>
              <Select.Option value={true}>启用</Select.Option>
              <Select.Option value={false}>停用</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default GroupMappingPage;
