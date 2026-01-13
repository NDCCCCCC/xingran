import { useState, useEffect, useRef } from 'react';
import type { FC, UIEvent } from 'react';
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
  Descriptions,
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
  FolderOpenOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import type { NetworkDevice, ConfigBackup } from '@/types';
import { post } from '@/lib/api';
import ActionButtons from '@/components/shared/ActionButtons';

// 差异行类型
interface DiffLine {
  type: 'same' | 'removed' | 'added' | 'empty';
  content: string;
  lineNum?: number;
}

// 差异对比结果
interface DiffResult {
  leftContent: string;
  rightContent: string;
  leftLines: DiffLine[];
  rightLines: DiffLine[];
  oldVersion: string;
  newVersion: string;
}

// 设备备份分组
interface DeviceBackupGroup {
  deviceId: string;
  deviceName: string;
  ipAddress: string;
  backups: ConfigBackup[];
  latestBackup: ConfigBackup;
  backupCount: number;
  autoCount: number;
  manualCount: number;
}

const { Option } = Select;
const { TextArea } = Input;

const ConfigBackupPage: FC = () => {
  const [devices, setDevices] = useState<NetworkDevice[]>([]);
  const [deviceGroups, setDeviceGroups] = useState<DeviceBackupGroup[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [backupModalVisible, setBackupModalVisible] = useState(false);
  const [diffModalVisible, setDiffModalVisible] = useState(false);
  const [versionListDrawerVisible, setVersionListDrawerVisible] = useState(false);
  const [restoreModalVisible, setRestoreModalVisible] = useState(false);
  const [contentDrawerVisible, setContentDrawerVisible] = useState(false);
  const [backupForm] = Form.useForm();
  const [searchForm] = Form.useForm();
  const [selectedBackup, setSelectedBackup] = useState<ConfigBackup | null>(null);
  const [selectedRestoreBackup, setSelectedRestoreBackup] = useState<ConfigBackup | null>(null);
  const [selectedDeviceGroup, setSelectedDeviceGroup] = useState<DeviceBackupGroup | null>(null);
  const [compareBackup1, setCompareBackup1] = useState<ConfigBackup | null>(null);
  const [compareBackup2, setCompareBackup2] = useState<ConfigBackup | null>(null);
  const [diffResult, setDiffResult] = useState<DiffResult | null>(null);
  const [backupContent, setBackupContent] = useState('');

  // 同步滚动的 refs
  const leftScrollRef = useRef<HTMLDivElement>(null);
  const rightScrollRef = useRef<HTMLDivElement>(null);
  const isLeftScrolling = useRef(false);
  const isRightScrolling = useRef(false);

  // 统计数据
  const [statistics, setStatistics] = useState({
    total: 0,
    auto: 0,
    manual: 0,
    devices: 0,
  });

  // 同步滚动处理
  const handleLeftScroll = (e: UIEvent<HTMLDivElement>) => {
    if (isLeftScrolling.current) return;
    isRightScrolling.current = true;
    const target = e.target as HTMLDivElement;
    if (rightScrollRef.current) {
      rightScrollRef.current.scrollTop = target.scrollTop;
    }
    setTimeout(() => {
      isRightScrolling.current = false;
    }, 0);
  };

  const handleRightScroll = (e: UIEvent<HTMLDivElement>) => {
    if (isRightScrolling.current) return;
    isLeftScrolling.current = true;
    const target = e.target as HTMLDivElement;
    if (leftScrollRef.current) {
      leftScrollRef.current.scrollTop = target.scrollTop;
    }
    setTimeout(() => {
      isLeftScrolling.current = false;
    }, 0);
  };

  // 计算两个文本的差异（使用简化的 LCS 算法）
  const computeDiff = (content1: string, content2: string): DiffResult => {
    const lines1 = content1.split('\n');
    const lines2 = content2.split('\n');

    // 使用动态规划计算最长公共子序列
    const m = lines1.length;
    const n = lines2.length;
    const dp: number[][] = Array(m + 1).fill(0).map(() => Array(n + 1).fill(0));

    for (let i = 1; i <= m; i++) {
      for (let j = 1; j <= n; j++) {
        if (lines1[i - 1] === lines2[j - 1]) {
          dp[i][j] = dp[i - 1][j - 1] + 1;
        } else {
          dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
        }
      }
    }

    // 回溯生成差异
    const leftLines: DiffLine[] = [];
    const rightLines: DiffLine[] = [];
    let i = m, j = n;

    while (i > 0 || j > 0) {
      if (i > 0 && j > 0 && lines1[i - 1] === lines2[j - 1]) {
        // 相同的行
        leftLines.unshift({ type: 'same', content: lines1[i - 1], lineNum: i });
        rightLines.unshift({ type: 'same', content: lines2[j - 1], lineNum: j });
        i--;
        j--;
      } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
        // 新增的行
        leftLines.unshift({ type: 'empty', content: '', lineNum: undefined });
        rightLines.unshift({ type: 'added', content: lines2[j - 1], lineNum: j });
        j--;
      } else {
        // 删除的行
        leftLines.unshift({ type: 'removed', content: lines1[i - 1], lineNum: i });
        rightLines.unshift({ type: 'empty', content: '', lineNum: undefined });
        i--;
      }
    }

    return {
      leftContent: content1,
      rightContent: content2,
      leftLines,
      rightLines,
      oldVersion: '',
      newVersion: '',
    };
  };

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

  // 将备份列表按设备分组
  const groupBackupsByDevice = (backups: ConfigBackup[]): DeviceBackupGroup[] => {
    const groupMap = new Map<string, ConfigBackup[]>();

    // 按设备ID分组
    backups.forEach(backup => {
      if (!groupMap.has(backup.deviceId)) {
        groupMap.set(backup.deviceId, []);
      }
      groupMap.get(backup.deviceId)!.push(backup);
    });

    // 转换为分组数组
    const groups: DeviceBackupGroup[] = [];
    groupMap.forEach((deviceBackups, deviceId) => {
      // 按创建时间倒序排序
      deviceBackups.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());

      const latestBackup = deviceBackups[0];
      const autoCount = deviceBackups.filter(b => b.backupType === 'auto').length;
      const manualCount = deviceBackups.filter(b => b.backupType === 'manual').length;

      groups.push({
        deviceId,
        deviceName: latestBackup.deviceName,
        ipAddress: latestBackup.ipAddress,
        backups: deviceBackups,
        latestBackup,
        backupCount: deviceBackups.length,
        autoCount,
        manualCount,
      });
    });

    return groups;
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
      const backups = result.data.list || [];
      const groups = groupBackupsByDevice(backups);

      setDeviceGroups(groups);
      setTotal(result.data.total || 0);

      // 更新统计数据
      updateStatistics(backups);
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
    try {
      await post(`/network/backups/${backup.id}/restore`, {});
      message.success('恢复成功');
      setRestoreModalVisible(false);
      setSelectedRestoreBackup(null);
      loadBackups();
    } catch (error) {
      message.error('恢复失败');
    }
  };

  // 打开还原选择模态框
  const handleOpenRestoreModal = (group: DeviceBackupGroup) => {
    setSelectedDeviceGroup(group);
    setSelectedRestoreBackup(group.latestBackup);
    setRestoreModalVisible(true);
  };

  // 确认还原
  const handleConfirmRestore = () => {
    if (!selectedRestoreBackup) {
      message.warning('请选择要还原的版本');
      return;
    }
    Modal.confirm({
      title: '确认恢复配置',
      content: `确定要将 ${selectedDeviceGroup?.deviceName} 恢复到版本 ${selectedRestoreBackup.version} 吗？此操作将会覆盖设备当前配置！`,
      okText: '确认',
      cancelText: '取消',
      okType: 'danger',
      onOk: () => handleRestore(selectedRestoreBackup),
    });
  };

  // 打开版本列表抽屉
  const handleOpenVersionList = (group: DeviceBackupGroup) => {
    setSelectedDeviceGroup(group);
    setVersionListDrawerVisible(true);
  };

  // 打开对比对话框
  const handleOpenDiffModal = (group: DeviceBackupGroup, backup1?: ConfigBackup, backup2?: ConfigBackup) => {
    setSelectedDeviceGroup(group);
    if (group.backups.length < 2) {
      message.info('该设备只有一份备份，无法对比');
      return;
    }
    // 默认：左边是次新版，右边是最新版
    setCompareBackup1(backup1 || group.backups[1]);
    setCompareBackup2(backup2 || group.backups[0]);
    setDiffModalVisible(true);
  };

  // 执行差异对比
  const executeDiff = async () => {
    if (!compareBackup1 || !compareBackup2) {
      message.warning('请选择两个备份版本进行对比');
      return;
    }
    if (compareBackup1.id === compareBackup2.id) {
      message.warning('请选择不同的版本进行对比');
      return;
    }
    try {
      // 获取两个版本的完整内容
      const [result1, result2] = await Promise.all([
        post(`/network/backups/${compareBackup1.id}`, {}),
        post(`/network/backups/${compareBackup2.id}`, {}),
      ]);

      const content1 = result1.data.content || '';
      const content2 = result2.data.content || '';

      // 计算差异
      const diffResult = computeDiff(content1, content2);
      diffResult.oldVersion = `版本 ${compareBackup1.version} (${compareBackup1.createdAt})`;
      diffResult.newVersion = `版本 ${compareBackup2.version} (${compareBackup2.createdAt})`;

      setDiffResult(diffResult);
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

  // 格式化日期时间
  const formatDateTime = (dateStr: string) => {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    const year = date.getFullYear();
    const month = String(date.getMonth() + 1).padStart(2, '0');
    const day = String(date.getDate()).padStart(2, '0');
    const hours = String(date.getHours()).padStart(2, '0');
    const minutes = String(date.getMinutes()).padStart(2, '0');
    const seconds = String(date.getSeconds()).padStart(2, '0');
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
  };

  // 设备列表表格列（按设备分组）
  const deviceColumns: ColumnsType<DeviceBackupGroup> = [
    {
      title: '设备名称',
      dataIndex: 'deviceName',
      key: 'deviceName',
      width: 200,
    },
    {
      title: 'IP地址',
      dataIndex: 'ipAddress',
      key: 'ipAddress',
      width: 140,
    },
    {
      title: '备份总数',
      dataIndex: 'backupCount',
      key: 'backupCount',
      width: 100,
      render: (count: number) => (
        <Tag color="blue" icon={<DatabaseOutlined />}>{count} 份</Tag>
      ),
    },
    {
      title: '自动/手动',
      key: 'backupTypeCount',
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tag color="blue" icon={<ClockCircleOutlined />}>{record.autoCount}</Tag>
          <Tag color="green" icon={<SaveOutlined />}>{record.manualCount}</Tag>
        </Space>
      ),
    },
    {
      title: '最新版本',
      dataIndex: 'latestBackup',
      key: 'latestVersion',
      width: 80,
      render: (backup: ConfigBackup) => <Tag>{backup.version}</Tag>,
    },
    {
      title: '最新备份时间',
      dataIndex: 'latestBackup',
      key: 'latestBackupTime',
      width: 150,
      render: (backup: ConfigBackup) => formatDateTime(backup.createdAt),
    },
    {
      title: '最新备份原因',
      dataIndex: 'latestBackup',
      key: 'latestReason',
      width: 150,
      ellipsis: true,
      render: (backup: ConfigBackup) => backup.changeReason || '-',
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 100,
      render: (_, record) => {
        const actions = [
          {
            key: 'restore',
            label: '还原',
            icon: <DownloadOutlined />,
            onClick: () => handleOpenRestoreModal(record),
          },
          {
            key: 'versions',
            label: '版本列表',
            icon: <HistoryOutlined />,
            onClick: () => handleOpenVersionList(record),
          },
          {
            key: 'diff',
            label: '版本对比',
            icon: <DiffOutlined />,
            onClick: () => handleOpenDiffModal(record),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  // 版本列表表格列
  const versionColumns: ColumnsType<ConfigBackup> = [
    { title: '版本号', dataIndex: 'version', key: 'version', width: 80 },
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
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: formatDateTime,
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      fixed: 'right',
      render: (_, record) => {
        const actions = [
          {
            key: 'view',
            label: '查看',
            icon: <EyeOutlined />,
            onClick: () => handleViewContent(record),
          },
          {
            key: 'restore',
            label: '恢复',
            icon: <DownloadOutlined />,
            render: () => (
              <Popconfirm title="确认恢复到此版本?" onConfirm={() => handleRestore(record)}>
                <Button type="link" icon={<DownloadOutlined />} size="small">
                  恢复
                </Button>
              </Popconfirm>
            ),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
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
              <Input placeholder="请输入设备名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="backupType" label="备份类型">
              <Select placeholder="请选择备份类型" allowClear className="user-form-input" style={{ width: 120 }}>
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

      {/* 设备列表表格 */}
      <Card>
        <Table
          columns={deviceColumns}
          dataSource={deviceGroups}
          loading={loading}
          rowKey="deviceId"
          scroll={{ x: 1200 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 个设备`,
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

      {/* 还原选择模态框 */}
      <Modal
        title={
          <Space>
            <DownloadOutlined />
            <span>选择还原版本 - {selectedDeviceGroup?.deviceName}</span>
          </Space>
        }
        open={restoreModalVisible}
        onCancel={() => {
          setRestoreModalVisible(false);
          setSelectedRestoreBackup(null);
          setSelectedDeviceGroup(null);
        }}
        footer={[
          <Button key="cancel" onClick={() => {
            setRestoreModalVisible(false);
            setSelectedRestoreBackup(null);
            setSelectedDeviceGroup(null);
          }}>
            取消
          </Button>,
          <Button key="confirm" type="primary" danger onClick={handleConfirmRestore}>
            确认还原
          </Button>,
        ]}
        width={800}
      >
        {selectedDeviceGroup && (
          <div>
            <Descriptions column={2} style={{ marginBottom: 16 }} bordered size="small">
              <Descriptions.Item label="设备名称">{selectedDeviceGroup.deviceName}</Descriptions.Item>
              <Descriptions.Item label="IP地址">{selectedDeviceGroup.ipAddress}</Descriptions.Item>
              <Descriptions.Item label="备份总数">{selectedDeviceGroup.backupCount} 份</Descriptions.Item>
              <Descriptions.Item label="已选版本">
                <Tag color="blue">版本 {selectedRestoreBackup?.version}</Tag>
              </Descriptions.Item>
            </Descriptions>

            <div style={{ marginBottom: 16 }}>
              <Space>
                <span>选择要还原的版本：</span>
                <Select
                  style={{ width: 400 }}
                  value={selectedRestoreBackup?.id}
                  onChange={(value) => {
                    const backup = selectedDeviceGroup.backups.find(b => b.id === value);
                    setSelectedRestoreBackup(backup || null);
                  }}
                  placeholder="请选择要还原的版本"
                >
                  {selectedDeviceGroup.backups.map(backup => (
                    <Option key={backup.id} value={backup.id}>
                      <Space>
                        <span>版本 {backup.version}</span>
                        <Tag color={backup.backupType === 'auto' ? 'blue' : 'green'} size="small">
                          {backup.backupType === 'auto' ? '自动' : '手动'}
                        </Tag>
                        <span style={{ color: '#999' }}>{formatDateTime(backup.createdAt)}</span>
                      </Space>
                    </Option>
                  ))}
                </Select>
              </Space>
            </div>

            {selectedRestoreBackup && (
              <Card size="small" title="选定版本详情">
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="版本号">{selectedRestoreBackup.version}</Descriptions.Item>
                  <Descriptions.Item label="备份类型">
                    <Tag color={selectedRestoreBackup.backupType === 'auto' ? 'blue' : 'green'}>
                      {selectedRestoreBackup.backupType === 'auto' ? '自动备份' : '手动备份'}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="文件大小">
                    {(selectedRestoreBackup.backupSize / 1024).toFixed(2)} KB
                  </Descriptions.Item>
                  <Descriptions.Item label="创建时间">{formatDateTime(selectedRestoreBackup.createdAt)}</Descriptions.Item>
                  <Descriptions.Item label="创建人">{selectedRestoreBackup.createdBy}</Descriptions.Item>
                  <Descriptions.Item label="变更原因" span={2}>
                    {selectedRestoreBackup.changeReason || '-'}
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            )}
          </div>
        )}
      </Modal>

      {/* 版本列表抽屉 */}
      <Drawer
        title={
          <Space>
            <FolderOpenOutlined />
            <span>{selectedDeviceGroup?.deviceName} - 配置版本列表</span>
            <Tag color="blue">{selectedDeviceGroup?.backupCount} 个版本</Tag>
          </Space>
        }
        placement="right"
        width={1000}
        open={versionListDrawerVisible}
        onClose={() => setVersionListDrawerVisible(false)}
        extra={
          selectedDeviceGroup && selectedDeviceGroup.backups.length >= 2 && (
            <Button
              type="primary"
              icon={<DiffOutlined />}
              onClick={() => {
                setVersionListDrawerVisible(false);
                handleOpenDiffModal(selectedDeviceGroup);
              }}
            >
              版本对比
            </Button>
          )
        }
      >
        {selectedDeviceGroup && (
          <div>
            <Descriptions column={3} style={{ marginBottom: 16 }} bordered size="small">
              <Descriptions.Item label="设备名称">{selectedDeviceGroup.deviceName}</Descriptions.Item>
              <Descriptions.Item label="IP地址">{selectedDeviceGroup.ipAddress}</Descriptions.Item>
              <Descriptions.Item label="备份总数">{selectedDeviceGroup.backupCount} 份</Descriptions.Item>
            </Descriptions>

            <Table
              columns={versionColumns}
              dataSource={selectedDeviceGroup.backups}
              rowKey="id"
              pagination={false}
              size="small"
              scroll={{ x: 1000 }}
            />
          </div>
        )}
      </Drawer>

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
        width={1400}
      >
        <div style={{ marginBottom: 16 }}>
          <Row gutter={16}>
            <Col span={10}>
              <Select
                placeholder="选择第一个版本"
                style={{ width: '100%' }}
                value={compareBackup1?.id}
                onChange={(value) => {
                  if (selectedDeviceGroup) {
                    const backup = selectedDeviceGroup.backups.find(b => b.id === value);
                    setCompareBackup1(backup || null);
                  }
                }}
              >
                {selectedDeviceGroup?.backups.map(backup => (
                  <Option key={backup.id} value={backup.id}>
                    版本 {backup.version} - {formatDateTime(backup.createdAt)}
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
                  if (selectedDeviceGroup) {
                    const backup = selectedDeviceGroup.backups.find(b => b.id === value);
                    setCompareBackup2(backup || null);
                  }
                }}
              >
                {selectedDeviceGroup?.backups.map(backup => (
                  <Option key={backup.id} value={backup.id}>
                    版本 {backup.version} - {formatDateTime(backup.createdAt)}
                  </Option>
                ))}
              </Select>
            </Col>
          </Row>
        </div>

        {diffResult && (
          <div style={{ height: 600, border: '1px solid #d9d9d9', borderRadius: 4, overflow: 'hidden' }}>
            {/* 版本信息栏 */}
            <div style={{ display: 'flex', borderBottom: '1px solid #d9d9d9' }}>
              <div style={{
                flex: 1,
                padding: '8px 16px',
                background: '#f5f5f5',
                borderRight: '1px solid #d9d9d9',
                fontSize: 13,
                fontWeight: 500,
                color: '#262626',
              }}>
                {diffResult.oldVersion}
              </div>
              <div style={{
                flex: 1,
                padding: '8px 16px',
                background: '#f5f5f5',
                fontSize: 13,
                fontWeight: 500,
                color: '#262626',
              }}>
                {diffResult.newVersion}
              </div>
            </div>

            {/* 差异对比内容区 */}
            <div style={{ display: 'flex', height: 'calc(100% - 41px)' }}>
              {/* 左侧：旧版本 */}
              <div
                ref={leftScrollRef}
                onScroll={handleLeftScroll}
                style={{
                  flex: 1,
                  overflow: 'auto',
                  background: '#ffffff',
                  borderRight: '1px solid #d9d9d9',
                  fontFamily: 'Consolas, "Courier New", monospace',
                  fontSize: 13,
                  lineHeight: '22px',
                }}
              >
                {diffResult.leftLines.map((line, index) => {
                  const lineNum = line.lineNum !== undefined ? String(line.lineNum).padStart(4, ' ') : '    ';
                  return (
                    <div
                      key={`left-${index}`}
                      style={{
                        display: 'flex',
                        backgroundColor: (() => {
                          if (line.type === 'removed') return '#ffced0';
                          if (line.type === 'same') {
                            const prevLine = index > 0 ? diffResult.leftLines[index - 1] : null;
                            const nextLine = index < diffResult.leftLines.length - 1 ? diffResult.leftLines[index + 1] : null;
                            if ((prevLine?.type === 'removed' || nextLine?.type === 'removed')) {
                              return '#f5f5f5';
                            }
                          }
                          return 'transparent';
                        })(),
                      }}
                    >
                      <div style={{
                        width: 50,
                        padding: '0 8px',
                        textAlign: 'right',
                        color: line.type === 'removed' ? '#c92a2a' : '#8c8c8c',
                        userSelect: 'none',
                        flexShrink: 0,
                        borderRight: '1px solid #e8e8e8',
                      }}>
                        {lineNum}
                      </div>
                      <div style={{
                        padding: '0 12px',
                        whiteSpace: 'pre',
                        overflowX: 'auto',
                        color: line.type === 'removed' ? '#c92a2a' : '#262626',
                        flex: 1,
                      }}>
                        {line.content}
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* 右侧：新版本 */}
              <div
                ref={rightScrollRef}
                onScroll={handleRightScroll}
                style={{
                  flex: 1,
                  overflow: 'auto',
                  background: '#ffffff',
                  fontFamily: 'Consolas, "Courier New", monospace',
                  fontSize: 13,
                  lineHeight: '22px',
                }}
              >
                {diffResult.rightLines.map((line, index) => {
                  const lineNum = line.lineNum !== undefined ? String(line.lineNum).padStart(4, ' ') : '    ';
                  return (
                    <div
                      key={`right-${index}`}
                      style={{
                        display: 'flex',
                        backgroundColor: (() => {
                          if (line.type === 'added') return '#d4edda';
                          if (line.type === 'same') {
                            const prevLine = index > 0 ? diffResult.rightLines[index - 1] : null;
                            const nextLine = index < diffResult.rightLines.length - 1 ? diffResult.rightLines[index + 1] : null;
                            if ((prevLine?.type === 'added' || nextLine?.type === 'added')) {
                              return '#f5f5f5';
                            }
                          }
                          return 'transparent';
                        })(),
                      }}
                    >
                      <div style={{
                        width: 50,
                        padding: '0 8px',
                        textAlign: 'right',
                        color: line.type === 'added' ? '#2b7a41' : '#8c8c8c',
                        userSelect: 'none',
                        flexShrink: 0,
                        borderRight: '1px solid #e8e8e8',
                      }}>
                        {lineNum}
                      </div>
                      <div style={{
                        padding: '0 12px',
                        whiteSpace: 'pre',
                        overflowX: 'auto',
                        color: line.type === 'added' ? '#2b7a41' : '#262626',
                        flex: 1,
                      }}>
                        {line.content}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}
      </Modal>

      {/* 配置内容抽屉 */}
      <Drawer
        title={
          <Space>
            <FileTextOutlined />
            <span>配置内容 - {selectedBackup?.deviceName || ''} (版本 {selectedBackup?.version || '-'})</span>
          </Space>
        }
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
