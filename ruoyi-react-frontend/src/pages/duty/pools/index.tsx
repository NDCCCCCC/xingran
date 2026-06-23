import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
import { Button, Form, Input, Select, Table, Modal, InputNumber, Space, Tag, App, Card, Statistic, Row, Col } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import { formatDateTime } from '@/utils/datetime';
import { createSorter } from '@/utils/tableHelpers';
import { usePagination } from '@/hooks/usePagination';
import { useDeptTree } from '@/hooks/useDeptTree';
import { DepartmentTreeSelect } from '@/components/shared';
import {
  getDutyPoolList,
  getDutyPoolStatistics,
  createDutyPool,
  updateDutyPool,
  deleteDutyPool,
  getUserList,
  type DutyPool,
  type DutyPoolCreateRequest,
  type DutyPoolUpdateRequest,
  type SimpleUser,
} from '@/lib/dutyApi';
import { isFormValidationError } from '@/utils/errorHandler';

const { Option } = Select;

const DutyPoolPage: FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<DutyPool[]>([]);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序状态（field 对应后端 dutyPoolAllowedSortFields 白名单 key）
  const [sortField, setSortField] = useState<string>('');
  const [sortOrder, setSortOrder] = useState<'ascend' | 'descend' | null>(null);
  // 列级 sortOrder：只对当前排序列返回方向，其余 undefined（受控高亮）
  const getColumnSortOrder = (field: string): 'ascend' | 'descend' | undefined =>
    sortField === field ? (sortOrder ?? undefined) : undefined;

  const [modalVisible, setModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<DutyPool | null>(null);

  // 统计数据
  const [stats, setStats] = useState({ total: 0, enabled: 0, disabled: 0, totalMembers: 0 });

  // 统一错误处理
  const handleApiError = (error: unknown, defaultMessage: string) => {
    if (error && typeof error === 'object' && 'message' in error) {
      message.error(error.message as string);
    } else {
      message.error(defaultMessage);
    }
  };

  const handleSuccess = (msg: string) => {
    message.success(msg);
  };

  // 部门树数据 — 全项目共享 ['dept','tree'] 缓存条目 (D-LOCKED: 单一数据源 useDeptTree)
  // 仅用于 DepartmentTreeSelect 的 departments prop(展示部门树)。用户筛选改走后端 recursiveDeptId。
  const { data: depts = [] } = useDeptTree();
  // 值班成员候选列表:选部门后用 recursiveDeptId 后端查询该部门+子部门用户。
  // 不再预加载全量用户做前端 filter——getUserList 的 pageSize 被 system 模块 MaxPageSize=100
  // 钳制(见 memory stat-cards-from-list-length-capped-at-100),预加载只能拿到前 100 个用户,
  // 深层部门(科技创新部/基础运维科)的人员排在 100 之后会被漏掉,导致选部门后成员下拉空。
  const [filteredUsers, setFilteredUsers] = useState<SimpleUser[]>([]);
  const [memberUsersLoading, setMemberUsersLoading] = useState(false);

  // 选部门后加载该部门(+子部门)的启用用户。后端按 sys_dept.ancestors 递归查询,
  // 不受全局 MaxPageSize=100 钳制(单部门用户通常 < 100)。与 workstations/loadUserOptions 同模式。
  const loadMemberUsers = useCallback(async (deptId?: string) => {
    if (!deptId) {
      setFilteredUsers([]);
      return;
    }
    setMemberUsersLoading(true);
    try {
      const result = await getUserList({ status: 0, recursiveDeptId: deptId });
      setFilteredUsers(result.data?.list || []);
    } catch (error) {
      console.error('获取值班成员失败:', error);
      setFilteredUsers([]);
    } finally {
      setMemberUsersLoading(false);
    }
  }, []);

  const fetchList = async (page?: number, pageSize?: number, sortCol?: string, sortAsc?: boolean) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const result = await getDutyPoolList({
        current: page ?? paginationProps.current,
        pageSize: pageSize ?? paginationProps.pageSize,
        poolName: values.poolName,
        status: values.status,
        deptId: values.deptId,
        // 服务端排序透传（避坑：详见 memory server-sort-loadfunc-param-drop）
        ...(sortCol ? { orderByColumn: sortCol, isAsc: sortAsc } : {}),
      });
      setDataSource(result.data?.list ?? []);
      setTotal(result.data?.total ?? 0);
    } catch (error) {
      handleApiError(error, '获取值班池列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取统计数据（专用端点 COUNT 聚合，不受分页上限影响）。
  // 旧实现用当前页 list(~10 条)算统计，多页时严重偏小且 total 卡在 pageSize。
  const fetchStats = async () => {
    try {
      const result = await getDutyPoolStatistics();
      setStats({
        total: result.data?.total ?? 0,
        enabled: result.data?.enabled ?? 0,
        disabled: result.data?.disabled ?? 0,
        totalMembers: result.data?.totalMembers ?? 0,
      });
    } catch (error) {
      console.error('获取值班池统计失败:', error);
    }
  };

  // 部门树数据由顶层 useDeptTree() 提供,值班成员按部门加载由 loadMemberUsers 处理(选部门时触发)

  useEffect(() => {
    fetchList(1, paginationProps.pageSize);
    fetchStats();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSearch = () => {
    fetchList(1, paginationProps.pageSize);
  };

  const handleReset = () => {
    form.resetFields();
    fetchList(1, paginationProps.pageSize);
  };

  const handleAdd = () => {
    setEditingRecord(null);
    editForm.resetFields();
    editForm.setFieldsValue({ dailyCount: 1, memberIds: [] });
    setFilteredUsers([]); // 重置成员候选(未选部门)
    setModalVisible(true);
  };

  const handleEdit = (record: DutyPool) => {
    setEditingRecord(record);
    setModalVisible(true);
    // 后端按部门+子部门加载成员(避免前端 filter 被 MaxPageSize=100 钳制)
    loadMemberUsers(record.deptId);
    editForm.setFieldsValue({
      poolName: record.poolName,
      deptId: record.deptId,
      description: record.description,
      dailyCount: record.dailyCount,
      memberIds: record.members?.map((m) => m.userId) || [],
    });
  };

  const handleModalOpenChange = (open: boolean) => {
    if (open && !editingRecord) {
      // 新增时重置筛选
      setFilteredUsers([]);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteDutyPool(id);
      handleSuccess('删除成功');
      fetchList(paginationProps.current, paginationProps.pageSize);
      fetchStats();
    } catch (error) {
      handleApiError(error, '删除失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingRecord) {
        const updateData: DutyPoolUpdateRequest = {
          poolName: values.poolName,
          deptId: values.deptId,
          description: values.description,
          dailyCount: values.dailyCount,
          memberIds: values.memberIds,
        };
        await updateDutyPool(editingRecord.id, updateData);
        handleSuccess('更新成功');
      } else {
        const createData: DutyPoolCreateRequest = {
          poolName: values.poolName,
          deptId: values.deptId,
          description: values.description,
          dailyCount: values.dailyCount,
          memberIds: values.memberIds,
        };
        await createDutyPool(createData);
        handleSuccess('创建成功');
      }
      setModalVisible(false);
      fetchList(paginationProps.current, paginationProps.pageSize);
      fetchStats();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        // 表单验证错误
        return;
      }
      handleApiError(error, editingRecord ? '更新失败' : '创建失败');
    }
  };

  const columns: ColumnsType<DutyPool> = [
    {
      title: '序号',
      key: 'index',
      width: 60,
      render: (_: unknown, __: unknown, index: number) => ((paginationProps.current ?? 1) - 1) * (paginationProps.pageSize ?? 10) + index + 1,
    },
    {
      title: '值班池名称',
      dataIndex: 'poolName',
      key: 'poolName',
      width: 150,
      sorter: true,
      sortOrder: getColumnSortOrder('poolName'),
    },
    {
      title: '所属部门',
      dataIndex: 'deptId',
      key: 'deptId',
      width: 120,
      sorter: true,
      sortOrder: getColumnSortOrder('deptId'),
      render: (_: string, record: DutyPool) => {
        return record.department?.deptName || '-';
      },
    },
    {
      title: '成员数量',
      key: 'memberCount',
      width: 100,
      sorter: createSorter<DutyPool>('memberCount', 'number'),
      render: (_: unknown, record: DutyPool) => record.members?.length || 0,
    },
    {
      title: '每日值班人数',
      dataIndex: 'dailyCount',
      key: 'dailyCount',
      width: 120,
      sorter: createSorter<DutyPool>('dailyCount', 'number'),
      render: (count: number) => `${count} 人/天`,
    },
    {
      title: '成员列表',
      key: 'members',
      width: 300,
      render: (_: unknown, record: DutyPool) => {
        const members = record.members || [];
        if (members.length === 0) return '-';
        return (
          <Space size={[0, 4]} wrap>
            {members.map((m, i) => (
              <Tag key={m.id} color={i < record.dailyCount ? 'blue' : 'default'}>
                {m.user?.nickname || m.user?.username}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      sorter: true,
      sortOrder: getColumnSortOrder('status'),
      render: (status: number) => (
        <Tag color={status === 0 ? 'green' : 'red'}>{status === 0 ? '启用' : '停用'}</Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 200,
      ellipsis: true,
      sorter: createSorter<DutyPool>('description', 'string'),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      sorter: true,
      sortOrder: getColumnSortOrder('createdAt'),
      render: (date: string) => formatDateTime(date, 'YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      fixed: 'right',
      render: (_: unknown, record: DutyPool) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            icon={<DeleteOutlined />}
            style={{ color: '#ff4d4f' }}
            onClick={() => {
              Modal.confirm({
                title: '确定要删除吗？',
                okText: '确定',
                cancelText: '取消',
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            }}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <Card title={null} className="mb-4">
        <Row gutter={16}>
          <Col span={6}>
            <Statistic title="总值班池" value={stats.total} prefix={<TeamOutlined />} />
          </Col>
          <Col span={6}>
            <Statistic title="启用中" value={stats.enabled} styles={{ content: { color: '#3f8600' } }} />
          </Col>
          <Col span={6}>
            <Statistic title="已停用" value={stats.disabled} styles={{ content: { color: '#cf1322' } }} />
          </Col>
          <Col span={6}>
            <Statistic title="总成员数" value={stats.totalMembers} prefix={<UserOutlined />} />
          </Col>
        </Row>
      </Card>

      <Card title="值班池管理">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="poolName" label="值班池名称">
              <Input placeholder="请输入值班池名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="deptId" label="所属部门">
              <DepartmentTreeSelect
                departments={depts}
                placeholder="请选择部门"
                className="user-form-input"
                style={{ width: 200 }}
              />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 100 }}>
                <Option value={0}>启用</Option>
                <Option value={1}>停用</Option>
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
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增值班池
            </Button>
          </Space>
        </div>

        <Table
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={paginationProps}
          onChange={(pagination, _filters, sorter) => {
            const current = pagination.current ?? 1;
            const pageSize = pagination.pageSize ?? 10;
            setCurrent(current);
            setPageSize(pageSize);
            // 排序：用 local const 持有新值传 fetchList，规避 React 18 setState 异步时序
            // （fetchList 不在 useMemo 依赖链，setState 后读 state 仍为旧值——commit 7ab1189 同类坑）
            const field = sorter && !Array.isArray(sorter) ? (sorter.field as string) || '' : '';
            const order = sorter && !Array.isArray(sorter) ? (sorter.order ?? null) : null;
            setSortField(field);
            setSortOrder(order);
            fetchList(current, pageSize, field, order === 'ascend');
          }}
        />
      </Card>

      <Modal
        title={editingRecord ? '编辑值班池' : '新增值班池'}
        open={modalVisible}
        onOk={handleModalOk}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setModalVisible(false)}
        width={600}
        destroyOnHidden
      >
        <Form form={editForm} layout="vertical" preserve={false}>
          <Form.Item
            name="poolName"
            label="值班池名称"
            rules={[{ required: true, message: '请输入值班池名称' }]}
          >
            <Input placeholder="请输入值班池名称" />
          </Form.Item>

          <Form.Item name="deptId" label="所属部门">
            <DepartmentTreeSelect
              departments={depts}
              onChange={(value) => {
                // 选部门后后端加载该部门+子部门的启用用户(避免前端 filter 被 MaxPageSize=100 钳制)
                loadMemberUsers(value);
                // 清空已选择的成员（因为部门已改变）
                editForm.setFieldValue('memberIds', []);
              }}
            />
          </Form.Item>

          <Form.Item
            name="dailyCount"
            label="每日值班人数"
            rules={[{ required: true, message: '请输入每日值班人数' }]}
          >
            <InputNumber min={1} max={10} placeholder="请输入每日值班人数" style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="memberIds"
            label="值班成员"
            rules={[{ required: true, message: '请选择值班成员' }]}
          >
            <Select
              mode="multiple"
              placeholder={filteredUsers.length === 0 ? '请先选择所属部门' : '请选择值班成员'}
              optionLabelProp="label"
              loading={memberUsersLoading}
              disabled={filteredUsers.length === 0 || memberUsersLoading}
            >
              {filteredUsers.map((user) => (
                <Option key={user.id} value={user.id} label={user.nickname || user.username}>
                  {user.username} - {user.nickname} {user.deptName ? `(${user.deptName})` : ''}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DutyPoolPage;

