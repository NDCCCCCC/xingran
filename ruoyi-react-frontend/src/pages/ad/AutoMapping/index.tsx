import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  message,
  Tag,
  Progress,
  Checkbox,
  Alert
} from 'antd';
import { SendOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { post } from '@/lib/api';

interface Department {
  id: string;
  deptName: string;
  leader?: string;
  phone?: string;
  status: string;
}

interface AutoMappingPreview {
  deptId: string;
  deptName: string;
  adGroupName: string;
  willCreate: boolean;
  reason?: string;
}

const AutoMapping: React.FC = () => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [selectedDepts, setSelectedDepts] = useState<string[]>([]);
  const [previewModal, setPreviewModal] = useState(false);
  const [previewData, setPreviewData] = useState<AutoMappingPreview[]>([]);
  const [mapping, setMapping] = useState(false);
  const [progress, setProgress] = useState(0);

  // 获取可映射的部门列表
  const fetchDepartments = async () => {
    try {
      const result = await post('/api/v1/departments/level/2', {});
      setDepartments((result.data as { list: Department[] }).list || []);
    } catch (error) {
      message.error('获取部门列表失败');
    }
  };

  // 预览自动映射
  const handlePreview = async (deptIds: string[]) => {
    try {
      const result = await post('/api/v1/ad/groups/automap/preview', {
        deptIds
      });
      setPreviewData(result.data as AutoMappingPreview[]);
      setPreviewModal(true);
    } catch (error) {
      message.error('预览失败');
    }
  };

  // 执行自动映射
  const handleAutoMap = async () => {
    setMapping(true);
    setProgress(0);

    try {
      const result = await post('/api/v1/ad/groups/automap', {
        deptIds: selectedDepts.length > 0 ? selectedDepts : undefined
      });

      // 模拟进度
      const interval = setInterval(() => {
        setProgress(prev => {
          if (prev >= 95) {
            clearInterval(interval);
            return 95;
          }
          return prev + 5;
        });
      }, 200);

      message.success(`成功映射 ${(result.data as { count: number }).count} 个部门`);
      setProgress(100);
      clearInterval(interval);
    } catch (error) {
      message.error('自动映射失败');
    } finally {
      setMapping(false);
      setTimeout(() => setProgress(0), 2000);
    }
  };

  useEffect(() => {
    fetchDepartments();
  }, []);

  const columns = [
    {
      title: '部门名称',
      dataIndex: 'deptName',
      key: 'deptName',
    },
    {
      title: '负责人',
      dataIndex: 'leader',
      key: 'leader',
    },
    {
      title: '联系电话',
      dataIndex: 'phone',
      key: 'phone',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === '0' ? 'success' : 'error'}>
          {status === '0' ? '正常' : '停用'}
        </Tag>
      )
    }
  ];

  const previewColumns = [
    {
      title: '部门名称',
      dataIndex: 'deptName',
      key: 'deptName',
    },
    {
      title: 'AD组名称',
      dataIndex: 'adGroupName',
      key: 'adGroupName',
    },
    {
      title: '将创建组',
      dataIndex: 'willCreate',
      key: 'willCreate',
      render: (willCreate: boolean) => (
        <Tag color={willCreate ? 'blue' : 'green'}>
          {willCreate ? '新建' : '已存在'}
        </Tag>
      )
    },
    {
      title: '说明',
      dataIndex: 'reason',
      key: 'reason',
    }
  ];

  return (
    <div>
      <Alert
        message="自动映射规则"
        description="系统将根据部门名称自动匹配AD组（cxhub-{部门名称}命名规则）。例如：'技术部' → 'cxhub-技术部'组"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Card
        title="可映射部门列表"
        extra={
          <Space>
            <Button onClick={() => handlePreview(selectedDepts)} disabled={selectedDepts.length === 0}>
              预览映射
            </Button>
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={handleAutoMap}
              disabled={mapping}
            >
              {mapping ? '映射中...' : '开始映射'}
            </Button>
          </Space>
        }
      >
        {mapping && (
          <Progress percent={progress} status="active" style={{ marginBottom: 16 }} />
        )}

        <Table
          columns={columns}
          dataSource={departments}
          rowKey="id"
          rowSelection={{
            selectedRowKeys: selectedDepts,
            onChange: (keys) => setSelectedDepts(keys as string[])
          }}
          pagination={{
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 个部门`
          }}
        />
      </Card>

      <Modal
        title="自动映射预览"
        open={previewModal}
        onCancel={() => setPreviewModal(false)}
        width={800}
        footer={[
          <Button key="cancel" onClick={() => setPreviewModal(false)}>
            取消
          </Button>,
          <Button
            key="confirm"
            type="primary"
            icon={<CheckCircleOutlined />}
            onClick={() => {
              setPreviewModal(false);
              handleAutoMap();
            }}
          >
            确认映射
          </Button>
        ]}
      >
        <Table
          columns={previewColumns}
          dataSource={previewData}
          rowKey="deptId"
          pagination={false}
          size="small"
        />
      </Modal>
    </div>
  );
};

export default AutoMapping;
