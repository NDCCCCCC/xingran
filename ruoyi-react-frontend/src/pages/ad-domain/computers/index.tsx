import { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Select,
  Input,
  Space,
  Button,
  Tag,
  Form,
  message,
  Descriptions,
  Modal,
} from 'antd';
import {
  ReloadOutlined,
  DesktopOutlined,
  CheckCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  getADComputerList,
  getADComputerDetail,
  getADOUTree,
  type ADComputerDetail,
  type ADOUNode,
} from '@/lib/adDomainApi';
import { useADConfigs } from '@/hooks/useADConfigs';
import type { FC } from 'react';
import { usePagination } from '@/hooks/usePagination';

const ADComputerPage: FC = () => {
  const [form] = Form.useForm();

  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  const [computers, setComputers] = useState<ADComputerDetail[]>([]);
  const [loading, setLoading] = useState(false);
  const { paginationProps, setCurrent, setTotal } = usePagination();

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedComputer, setSelectedComputer] = useState<ADComputerDetail | null>(null);

  const fetchOUTree = async () => {
    if (!selectedConfig) return;
    try {
      const res = await getADOUTree(selectedConfig);
      if (res.code === 0) {
        setOUTree(res.data ?? []);
      }
    } catch (error) {
      console.error('获取OU树失败', error);
    }
  };

  useEffect(() => {
    if (selectedConfig) {
      fetchOUTree();
      fetchComputers();
    }
  }, [selectedConfig, paginationProps.current, paginationProps.pageSize]);

  const fetchComputers = async () => {
    if (!selectedConfig) return;

    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const res = await getADComputerList({
        configId: selectedConfig,
        ouDn: values.ouDn,
        computerName: values.computerName,
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
      });
      if (res.code === 0) {
        setComputers(res.data?.list ?? []);
        setTotal(res.data?.total ?? 0);
      }
    } catch {
      message.error('获取电脑设备列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = () => {
    setCurrent(1);
    fetchComputers();
  };

  const handleViewDetail = async (computer: ADComputerDetail) => {
    if (!selectedConfig) return;
    try {
      const res = await getADComputerDetail(selectedConfig, computer.distinguishedName);
      if (res.code === 0) {
        setSelectedComputer(res.data);
        setDetailModalVisible(true);
      }
    } catch (error) {
      message.error('获取电脑设备详情失败');
    }
  };

  // 构建扁平化的OU选项列表
  const buildOUOptions = (nodes: ADOUNode[], level = 0): Array<{ value: string; label: string }> => {
    const options: Array<{ value: string; label: string }> = [];
    const prefix = '　'.repeat(level);

    for (const node of nodes) {
      options.push({
        value: node.dn,
        label: `${prefix}${node.name}`,
      });
      if (node.children && node.children.length > 0) {
        options.push(...buildOUOptions(node.children, level + 1));
      }
    }
    return options;
  };

  const columns: ColumnsType<ADComputerDetail> = [
    {
      title: '计算机名',
      dataIndex: 'computerName',
      key: 'computerName',
      fixed: 'left',
      width: 180,
    },
    {
      title: '最后登录用户',
      dataIndex: 'lastLogonUser',
      key: 'lastLogonUser',
      width: 150,
      render: (text: string) => text || '-',
    },
    {
      title: 'IP地址',
      dataIndex: 'ipAddress',
      key: 'ipAddress',
      width: 140,
      render: (text: string) => text || '-',
    },
    {
      title: 'MAC地址',
      dataIndex: 'macAddress',
      key: 'macAddress',
      width: 160,
      render: (text: string) => text || '-',
    },
    {
      title: '操作系统',
      dataIndex: 'operatingSystem',
      key: 'operatingSystem',
      width: 200,
      ellipsis: true,
      render: (text: string) => text || '-',
    },
    {
      title: 'CPU',
      dataIndex: 'cpuModel',
      key: 'cpuModel',
      width: 180,
      ellipsis: true,
      render: (text: string) => text || '-',
    },
    {
      title: '架构',
      dataIndex: 'architecture',
      key: 'architecture',
      width: 100,
      render: (text: string) => text || '-',
    },
    {
      title: '内存',
      dataIndex: 'memoryCapacity',
      key: 'memoryCapacity',
      width: 100,
      render: (text: string) => text || '-',
    },
    {
      title: '硬盘',
      dataIndex: 'hardDiskCapacity',
      key: 'hardDiskCapacity',
      width: 100,
      render: (text: string) => text || '-',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: number) => (
        <Tag
          icon={status === 0 ? <CheckCircleOutlined /> : <WarningOutlined />}
          color={status === 0 ? 'success' : 'default'}
        >
          {status === 0 ? '在线' : '离线'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 100,
      render: (_: any, record: ADComputerDetail) => (
        <Button
          type="link"
          size="small"
          onClick={() => handleViewDetail(record)}
        >
          详情
        </Button>
      ),
    },
  ];

  return (
    <Card>
      <Space orientation="vertical" style={{ width: '100%' }} size="large">
        {/* 搜索栏 */}
        <Form form={form} layout="inline">
          <Form.Item label="AD配置">
            <Select
              style={{ width: 200 }}
              value={selectedConfig}
              onChange={setSelectedConfig}
              options={configs.map((c) => ({ label: c.configName, value: c.id }))}
            />
          </Form.Item>
          <Form.Item name="ouDn">
            <Select
              placeholder="所属OU"
              allowClear
              style={{ width: 200 }}
              options={buildOUOptions(ouTree)}
            />
          </Form.Item>
          <Form.Item name="computerName">
            <Input placeholder="计算机名称" allowClear style={{ width: 180 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={fetchComputers}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>

        {/* 电脑设备列表 */}
        <Table
          columns={columns}
          dataSource={computers}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1800 }}
          pagination={paginationProps}
        />
      </Space>

      {/* 详情弹窗 */}
      <Modal
        title={
          <Space>
            <DesktopOutlined />
            <span>电脑设备详情 - {selectedComputer?.computerName}</span>
          </Space>
        }
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>,
        ]}
        width={800}
      >
        {selectedComputer && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="计算机名" span={2}>
              {selectedComputer.computerName}
            </Descriptions.Item>
            <Descriptions.Item label="最后登录用户" span={1}>
              {selectedComputer.lastLogonUser || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="状态" span={1}>
              <Tag
                icon={selectedComputer.status === 0 ? <CheckCircleOutlined /> : <WarningOutlined />}
                color={selectedComputer.status === 0 ? 'success' : 'default'}
              >
                {selectedComputer.status === 0 ? '在线' : '离线'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="IP地址" span={1}>
              {selectedComputer.ipAddress || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="MAC地址" span={1}>
              {selectedComputer.macAddress || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="操作系统" span={2}>
              {selectedComputer.operatingSystem || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="系统版本" span={2}>
              {selectedComputer.osVersion || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="CPU型号" span={2}>
              {selectedComputer.cpuModel || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="架构" span={1}>
              {selectedComputer.architecture || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="内存容量" span={1}>
              {selectedComputer.memoryCapacity || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="硬盘容量" span={1}>
              {selectedComputer.hardDiskCapacity || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="序列号" span={1}>
              {selectedComputer.serialNumber || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="管理者" span={1}>
              {selectedComputer.managedBy || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="最后登录时间" span={1}>
              {selectedComputer.lastLogon ? new Date(selectedComputer.lastLogon).toLocaleString('zh-CN') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="最后上线时间" span={1}>
              {selectedComputer.lastOnlineTime ? new Date(selectedComputer.lastOnlineTime).toLocaleString('zh-CN') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="登录次数" span={1}>
              {selectedComputer.logonCount}
            </Descriptions.Item>
            <Descriptions.Item label="DN" span={2}>
              <span style={{ fontSize: '12px', fontFamily: 'monospace' }}>
                {selectedComputer.distinguishedName}
              </span>
            </Descriptions.Item>
            {selectedComputer.originalDescription && (
              <Descriptions.Item label="原始描述" span={2}>
                <span style={{ fontSize: '12px', fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
                  {selectedComputer.originalDescription}
                </span>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Modal>
    </Card>
  );
};

export default ADComputerPage;
