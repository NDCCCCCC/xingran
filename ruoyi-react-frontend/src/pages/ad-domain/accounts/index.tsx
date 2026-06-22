import { useEffect, useState, useCallback } from 'react';
import {
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Input,
  message,
  Popconfirm,
  Card,
  Statistic,
  Row,
  Col,
  Tooltip
} from 'antd';
import {
  PlusOutlined,
  ReloadOutlined,
  UnlockOutlined,
  EditOutlined,
  StopOutlined,
  CheckCircleOutlined,
  DeleteOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { FC } from 'react';
import {
  listADServiceAccounts,
  createADServiceAccount,
  updateADServiceAccount,
  deleteADServiceAccount,
  enableADServiceAccount,
  disableADServiceAccount,
  unlockADServiceAccount,
  getADServiceAccountStats,
  type ADServiceAccount,
  type ADServiceAccountStats,
  getADConfigList,
  type ADConfig
} from '@/lib/adDomainApi';
import { formatDateTime } from '@/utils/datetime';
import { encryptPasswordWithSM4, fetchSM4KeyForPassword } from '@/utils/sm4';

// 状态映射
const STATUS_MAP: Record<number, { color: string; text: string }> = {
  0: { color: 'green', text: '可用' },
  1: { color: 'default', text: '已停用' },
  2: { color: 'orange', text: '熔断中' },
};

const ADAccountPoolPage: FC = () => {
  const [configs, setConfigs] = useState<ADConfig[]>([]);
  const [selectedConfigId, setSelectedConfigId] = useState<string>('');
  const [list, setList] = useState<ADServiceAccount[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [stats, setStats] = useState<ADServiceAccountStats | null>(null);

  // Modal 状态
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editing, setEditing] = useState<ADServiceAccount | null>(null);
  const [form] = Form.useForm();

  // Unlock Modal
  const [unlockModalVisible, setUnlockModalVisible] = useState(false);
  const [unlockingId, setUnlockingId] = useState<string>('');
  const [unlockForm] = Form.useForm();

  // 加载 AD 配置列表
  const loadConfigs = useCallback(async () => {
    try {
      const res = await getADConfigList({ current: 1, pageSize: 100, status: 0 });
      if (res.code === 0 && res.data && res.data.list) {
        setConfigs(res.data.list);
        if (res.data.list.length > 0 && !selectedConfigId) {
          setSelectedConfigId(res.data.list[0].id);
        }
      }
    } catch (e) {
      message.error('加载 AD 配置失败');
    }
  }, [selectedConfigId]);

  // 加载账号列表
  const loadList = useCallback(async () => {
    if (!selectedConfigId) return;
    setLoading(true);
    try {
      const res = await listADServiceAccounts({
        configId: selectedConfigId,
        page,
        pageSize,
      });
      if (res.code === 0 && res.data) {
        setList(res.data.list);
        setTotal(res.data.total);
      } else {
        message.error(res.message || '加载失败');
      }
    } catch (e) {
      message.error('加载账号列表失败');
    } finally {
      setLoading(false);
    }
  }, [selectedConfigId, page, pageSize]);

  // 加载统计
  const loadStats = useCallback(async () => {
    if (!selectedConfigId) return;
    try {
      const res = await getADServiceAccountStats(selectedConfigId);
      if (res.code === 0 && res.data) {
        setStats(res.data);
      }
    } catch (e) {
      // 静默失败，不影响主流程
    }
  }, [selectedConfigId]);

  useEffect(() => {
    loadConfigs();
  }, [loadConfigs]);

  useEffect(() => {
    if (selectedConfigId) {
      loadList();
      loadStats();
    }
  }, [selectedConfigId, loadList, loadStats]);

  // 新增
  const handleCreate = () => {
    setEditing(null);
    form.resetFields();
    setEditModalVisible(true);
  };

  // 编辑
  const handleEdit = (record: ADServiceAccount) => {
    setEditing(record);
    form.setFieldsValue({
      username: record.username,
      remark: record.remark,
    });
    setEditModalVisible(true);
  };

  // 提交（创建/更新）
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      // 密码字段：前端 SM4 加密后再传
      let passwordCiphertext = '';
      if (values.password) {
        try {
          const keyHex = await fetchSM4KeyForPassword();
          passwordCiphertext = await encryptPasswordWithSM4(values.password, keyHex);
        } catch (e) {
          message.error('获取 SM4 密钥失败');
          return;
        }
      }

      if (editing) {
        // 更新
        const updateData: any = { id: editing.id, username: values.username, remark: values.remark };
        if (passwordCiphertext) {
          updateData.passwordCiphertext = passwordCiphertext;
        }
        const res = await updateADServiceAccount(updateData);
        if (res.code === 0) {
          message.success('更新成功');
          setEditModalVisible(false);
          loadList();
          loadStats();
        } else {
          message.error(res.message || '更新失败');
        }
      } else {
        // 新增
        if (!passwordCiphertext) {
          message.error('请输入密码');
          return;
        }
        const res = await createADServiceAccount({
          configId: selectedConfigId,
          username: values.username,
          passwordCiphertext,
          remark: values.remark,
        });
        if (res.code === 0) {
          message.success('创建成功');
          setEditModalVisible(false);
          loadList();
          loadStats();
        } else {
          message.error(res.message || '创建失败');
        }
      }
    } catch (e) {
      // 表单校验失败
    }
  };

  // 删除
  const handleDelete = async (id: string) => {
    try {
      const res = await deleteADServiceAccount(id);
      if (res.code === 0) {
        message.success('删除成功');
        loadList();
        loadStats();
      } else {
        message.error(res.message || '删除失败');
      }
    } catch (e) {
      message.error('删除失败');
    }
  };

  // 启用/停用
  const handleToggleEnabled = async (record: ADServiceAccount) => {
    const action = record.status === 1 ? enableADServiceAccount : disableADServiceAccount;
    try {
      const res = await action(record.id);
      if (res.code === 0) {
        message.success(record.status === 1 ? '已启用' : '已停用');
        loadList();
        loadStats();
      } else {
        message.error(res.message || '操作失败');
      }
    } catch (e) {
      message.error('操作失败');
    }
  };

  // 立即解锁
  const handleUnlockClick = (id: string) => {
    setUnlockingId(id);
    unlockForm.resetFields();
    setUnlockModalVisible(true);
  };

  const handleUnlockSubmit = async () => {
    try {
      const values = await unlockForm.validateFields();
      if (values.reason.length < 10) {
        message.error('解锁原因至少 10 字符');
        return;
      }
      const res = await unlockADServiceAccount({ id: unlockingId, reason: values.reason });
      if (res.code === 0) {
        message.success('解锁成功');
        setUnlockModalVisible(false);
        loadList();
        loadStats();
      } else {
        message.error(res.message || '解锁失败');
      }
    } catch (e) {
      // 校验失败
    }
  };

  // 表格列定义
  const columns: ColumnsType<ADServiceAccount> = [
    {
      title: '账号',
      dataIndex: 'username',
      key: 'username',
      width: 200,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: number) => (
        <Tag color={STATUS_MAP[status]?.color || 'default'}>
          {STATUS_MAP[status]?.text || '未知'}
        </Tag>
      ),
    },
    {
      title: '失败次数',
      dataIndex: 'failureCount',
      key: 'failureCount',
      width: 100,
    },
    {
      title: '上次失败原因',
      dataIndex: 'lastFailureReason',
      key: 'lastFailureReason',
      ellipsis: true,
      render: (text?: string) =>
        text ? (
          <Tooltip title={text}>
            <span>{text.length > 30 ? text.slice(0, 30) + '...' : text}</span>
          </Tooltip>
        ) : (
          '-'
        ),
    },
    {
      title: '上次成功',
      dataIndex: 'lastSuccessAt',
      key: 'lastSuccessAt',
      width: 160,
      render: (t?: string) => (t ? formatDateTime(t) : '从未'),
    },
    {
      title: '熔断到期',
      dataIndex: 'circuitBreakerUntil',
      key: 'circuitBreakerUntil',
      width: 160,
      render: (t?: string) => (t ? formatDateTime(t) : '-'),
    },
    {
      title: '备注',
      dataIndex: 'remark',
      key: 'remark',
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 280,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          {record.status === 2 && (
            <Button
              type="link"
              size="small"
              icon={<UnlockOutlined />}
              onClick={() => handleUnlockClick(record.id)}
            >
              立即解锁
            </Button>
          )}
          <Button
            type="link"
            size="small"
            icon={record.status === 1 ? <CheckCircleOutlined /> : <StopOutlined />}
            onClick={() => handleToggleEnabled(record)}
          >
            {record.status === 1 ? '启用' : '停用'}
          </Button>
          <Popconfirm
            title="确认删除？"
            description="删除后该账号无法恢复。"
            onConfirm={() => handleDelete(record.id)}
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
    <div style={{ padding: 16 }}>
      {/* AD 配置选择 */}
      <Card style={{ marginBottom: 16 }}>
        <Space>
          <span>AD 配置：</span>
          <select
            value={selectedConfigId}
            onChange={(e) => setSelectedConfigId(e.target.value)}
            style={{ minWidth: 240, padding: '4px 8px' }}
          >
            {configs.map((c) => (
              <option key={c.id} value={c.id}>
                {c.configName}
              </option>
            ))}
          </select>
        </Space>
      </Card>

      {/* 统计卡片 */}
      {stats && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={4}>
            <Card>
              <Statistic title="账号总数" value={stats.total} />
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              <Statistic title="可用" value={stats.available} valueStyle={{ color: '#3f8600' }} />
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              <Statistic title="已停用" value={stats.disabled} />
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              <Statistic
                title="熔断中"
                value={stats.circuitBroken}
                valueStyle={{ color: '#cf1322' }}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic title="当前活跃账号" value={stats.currentAccount || '无'} />
            </Card>
          </Col>
        </Row>
      )}

      {/* 列表 */}
      <Card
        title="服务账号池"
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => { loadList(); loadStats(); }}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              新增账号
            </Button>
          </Space>
        }
      >
        <Table
          rowKey="id"
          columns={columns}
          dataSource={list}
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={{
            current: page,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      {/* 新增/编辑 Modal */}
      <Modal
        title={editing ? '编辑账号' : '新增账号'}
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        onOk={handleSubmit}
        destroyOnClose
      >
        <Form form={form} layout="vertical" preserve={false}>
          <Form.Item
            name="username"
            label="账号（UPN 或 sAMAccountName）"
            rules={[{ required: true, message: '请输入账号' }]}
          >
            <Input placeholder="如 svc-01@corp.local" disabled={!!editing} />
          </Form.Item>
          <Form.Item
            name="password"
            label={editing ? '新密码（留空 = 不修改）' : '密码'}
            rules={editing ? [] : [{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder="SM4 加密后传输" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} maxLength={500} showCount />
          </Form.Item>
        </Form>
      </Modal>

      {/* 解锁 Modal */}
      <Modal
        title="立即解锁"
        open={unlockModalVisible}
        onCancel={() => setUnlockModalVisible(false)}
        onOk={handleUnlockSubmit}
        destroyOnClose
      >
        <Form form={unlockForm} layout="vertical" preserve={false}>
          <Form.Item
            name="reason"
            label="解锁原因（至少 10 字符，会记录到操作日志）"
            rules={[
              { required: true, message: '请填写解锁原因' },
              { min: 10, message: '解锁原因至少 10 字符' },
            ]}
          >
            <Input.TextArea
              rows={3}
              maxLength={200}
              showCount
              placeholder="如：AD 域控已解锁该账号，恢复使用"
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ADAccountPoolPage;