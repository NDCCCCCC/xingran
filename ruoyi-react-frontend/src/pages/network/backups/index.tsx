import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
  Card,
  Tag,
  Drawer,
  Popconfirm,
  Row,
  Col,
  Statistic,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  SaveOutlined,
  SearchOutlined,
  ReloadOutlined,
  DownloadOutlined,
  DiffOutlined,
  HistoryOutlined,
  CloudUploadOutlined,
  EyeOutlined,
  DatabaseOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import type { NetworkDevice, ConfigBackup } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;

const ConfigBackupPage: React.FC = () => {
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [backups, setBackups] = useState<ConfigBackup[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [backupModalVisible, setBackupModalVisible] = useState(false);
  const [diffModalVisible, setDiffModalVisible] = useState(false);
  const [contentDrawerVisible, setContentDrawerVisible] = useState(false);
  const [backupForm] = Form.useForm();
  const [searchForm] = Form.useForm();
  const [selectedBackup, setSelectedBackup] = useState<ConfigBackup | null>(null);
  const [compareBackup1, setCompareBackup1] = useState<ConfigBackup | null>(null);
  const [compareBackup2, setCompareBackup2] = useState<ConfigBackup | null>(null);
  const [diffResult, setDiffResult] = useState('');
  const [backupContent, setBackupContent] = useState('');

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    auto: 0,
    manual: 0,
    devices: 0,
  });

  // 加载设备列表
  const loadDevices = async () => {
    try {
      const result = await post('/network/devices/list', {
        current: 1,
        pageSize: 1000,
      });
      setDevices(result.data.list || []);
    } catch (error) {
      console.error('加载设备列表失败:', error);
    }
  };

  // 加载备份列表
  const loadBackups = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/network/backups/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      setBackups(result.data.list);
      setTotal(result.data.total);
      // 更新统计数据
      updateStatistics(result.data.list || []);
    } catch (error) {
      console.error('加载备份列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载统计数据
  const loadStatistics = async () => {
    try {
      const result = await post('/network/backups/list', { current: 1, pageSize: 10000 });
      const allBackups = result.data.list || [];
      updateStatistics(allBackups);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  // 更新统计数据
  const updateStatistics = (backupList: ConfigBackup[]) => {
    const autoCount = backupList.filter(b => b.backupType === 'auto').length;
    const manualCount = backupList.filter(b => b.backupType === 'manual').length;
    const uniqueDevices = new Set(backupList.map(b => b.deviceId)).size;
    setStatistics({
      total: backupList.length,
      auto: autoCount,
      manual: manualCount,
      devices: uniqueDevices,
    });
  };

  // 创建备份
  const handleBackup = async () => {
    try {
      const values = await backupForm.validateFields();
      await post('/network/backups', values);
      message.success('备份任务已创建');
      setBackupModalVisible(false);
      backupForm.resetFields();
      loadBackups();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('创建备份失败');
    }
  };

  // 查看配置内容
  const handleViewContent = async (backup: ConfigBackup) => {
    try {
      const result = await post(`/network/backups/${backup.id}`, {});
      setBackupContent(result.data.content || '');
      setSelectedBackup(backup);
      setContentDrawerVisible(true);
    } catch (error) {
      message.error('加载配置内容失败');
    }
  };

  // 恢复配置
  const handleRestore = async (backup: ConfigBackup) => {
    Modal.confirm({
      title: '确认恢复配置',
      content: `确定要恢复到版本 ${backup.version} 吗？此操作将会覆盖设备当前配置！`,
      okText: '确认',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        try {
          await post(`/network/backups/${backup.id}/restore`, {});
          message.success('恢复成功');
        } catch (error) {
          message.error('恢复失败');
        }
      },
    });
  };

  // 查看差异
  const handleViewDiff = async (backup: ConfigBackup) => {
    setSelectedBackup(backup);
    // 获取该设备的其他备份版本用于对比
    const deviceBackups = backups.filter(b => b.deviceId === backup.deviceId);
    if (deviceBackups.length < 2) {
      message.info('该设备只有一份备份，无法对比');
      return;
    }
    setCompareBackup1(backup);
    setCompareBackup2(null);
    setDiffModalVisible(true);
  };

  // 执行差异对比
  const executeDiff = async () => {
    if (!compareBackup1 || !compareBackup2) {
      message.warning('请选择两个备份版本进行对比');
      return;
    }
    try {
      const result = await post('/network/backups/diff', {
        backupId1: compareBackup1.id,
        backupId2: compareBackup2.id,
      });
      setDiffResult(result.data.diff || '');
    } catch (error) {
      message.error('对比失败');
    }
  };

  // 打开创建备份模态框
  const openBackupModal = () => {
    loadDevices();
    backupForm.resetFields();
    setBackupModalVisible(true);
  };

  // 备份类型选项
  const backupTypeOptions = [
    { label: '自动备份', value: 'auto' },
    { label: '手动备份', value: 'manual' },
  ];

  // 备份列表表格列
  const backupColumns: ColumnsType<ConfigBackup> = [
    { title: '设备名称', dataIndex: 'deviceName', key: 'deviceName', width: 150 },
    { title: 'IP地址', dataIndex: 'ipAddress', key: 'ipAddress', width: 140 },
    {
      title: '备份类型',
      dataIndex: 'backupType',
      key: 'backupType',
      width: 100,
      render: (backupType: string) => {
        const option = backupTypeOptions.find(o => o.value === backupType);
        return <Tag color={backupType === 'auto' ? 'blue' : 'green'}>{option?.label}</Tag>;
      },
    },
    { title: '版本号', dataIndex: 'version', key: 'version', width: 80 },
    { title: '配置哈希', dataIndex: 'configHash', key: 'configHash', width: 120, ellipsis: true },
    {
      title: '文件大小',
      dataIndex: 'backupSize',
      key: 'backupSize',
      width: 100,
      render: (size: number) => {
        const kb = (size / 1024).toFixed(2);
        return `${kb} KB`;
      },
    },
    { title: '变更原因', dataIndex: 'changeReason', key: 'changeReason', ellipsis: true },
    { title: '创建人', dataIndex: 'createdBy', key: 'createdBy', width: 100 },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 250,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => handleViewContent(record)}>查看</Button>
          <Button type="link" size="small" icon={<DiffOutlined />} onClick={() => handleViewDiff(record)}>对比</Button>
          <Popconfirm title="确认恢复到此版本?" onConfirm={() => handleRestore(record)}>
            <Button type="link" size="small" icon={<DownloadOutlined />}>恢复</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    loadBackups();
    loadStatistics();
  }, [current, pageSize]);

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="备份总数"
              value={statistics.total}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="自动备份"
              value={statistics.auto}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="手动备份"
              value={statistics.manual}
              styles={{ content: { color: '#52c41a' } }}
              prefix={<SaveOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="涉及设备"
              value={statistics.devices}
              prefix={<HistoryOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="deviceName" label="设备名称">
              <Input placeholder="请输入设备名称" allowClear style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="backupType" label="备份类型">
              <Select placeholder="请选择备份类型" allowClear style={{ width: 120 }}>
                {backupTypeOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={() => { loadBackups(); loadStatistics(); }}>查询</Button>
                <Button icon={<ReloadOutlined />} onClick={() => { loadBackups(); loadStatistics(); }}>刷新</Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<CloudUploadOutlined />} onClick={openBackupModal}>
              创建备份
            </Button>
          </Space>
        </div>
      </Card>

      {/* 备份列表表格 */}
      <Card>
        <Table
          columns={backupColumns}
          dataSource={backups}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1600 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, size) => { setCurrent(page); setPageSize(size || 10); },
          }}
        />
      </Card>

      {/* 创建备份模态框 */}
      <Modal
        title="创建配置备份"
        open={backupModalVisible}
        onOk={handleBackup}
        onCancel={() => setBackupModalVisible(false)}
        width={600}
      >
        <Form form={backupForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="deviceIds" label="选择设备" rules={[{ required: true, message: '请选择设备' }]}>
            <Select
              mode="multiple"
              placeholder="请选择要备份的设备"
              showSearch
              optionFilterProp="children"
            >
              {devices.map(device => (
                <Option key={device.id} value={device.id}>
                  {device.deviceName} ({device.ipAddress})
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="changeReason" label="变更原因">
            <TextArea rows={3} placeholder="请输入变更原因" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 差异对比模态框 */}
      <Modal
        title="配置差异对比"
        open={diffModalVisible}
        onCancel={() => setDiffModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDiffModalVisible(false)}>关闭</Button>,
          <Button key="diff" type="primary" onClick={executeDiff} disabled={!compareBackup1 || !compareBackup2}>
            执行对比
          </Button>,
        ]}
        width={1000}
      >
        <div style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col span={10}>
              <Select
                placeholder="选择第一个版本"
                style={{ width: '100%' }}
                value={compareBackup1?.id}
                onChange={(value) => {
                  const backup = backups.find(b => b.id === value);
                  setCompareBackup1(backup || null);
                }}
              >
                {backups
                  .filter(b => selectedBackup && b.deviceId === selectedBackup.deviceId)
                  .map(backup => (
                    <Option key={backup.id} value={backup.id}>
                      版本 {backup.version} - {backup.createdAt}
                    </Option>
                  ))}
              </Select>
            </Col>
            <Col span={4} style={{ textAlign: 'center', lineHeight: '32px' }}>
              VS
            </Col>
            <Col span={10}>
              <Select
                placeholder="选择第二个版本"
                style={{ width: '100%' }}
                value={compareBackup2?.id}
                onChange={(value) => {
                  const backup = backups.find(b => b.id === value);
                  setCompareBackup2(backup || null);
                }}
              >
                {backups
                  .filter(b => selectedBackup && b.deviceId === selectedBackup.deviceId)
                  .map(backup => (
                    <Option key={backup.id} value={backup.id}>
                      版本 {backup.version} - {backup.createdAt}
                    </Option>
                  ))}
              </Select>
            </Col>
          </Row>
        </div>

        {diffResult && (
          <pre style={{
            background: '#f5f5f5',
            padding: 16,
            borderRadius: 4,
            maxHeight: 500,
            overflow: 'auto',
            fontSize: 12,
          }}>
            {diffResult}
          </pre>
        )}
      </Modal>

      {/* 配置内容抽屉 */}
      <Drawer
        title={`配置内容 - ${selectedBackup?.deviceName || ''} (版本 ${selectedBackup?.version || '-'})`}
        placement="right"
        width={800}
        open={contentDrawerVisible}
        onClose={() => setContentDrawerVisible(false)}
      >
        <pre style={{
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-all',
          background: '#f5f5f5',
          padding: 16,
          borderRadius: 4,
          maxHeight: 'calc(100vh - 100px)',
          overflow: 'auto',
        }}>
          {backupContent}
        </pre>
      </Drawer>
    </div>
  );
};

export default ConfigBackupPage;
