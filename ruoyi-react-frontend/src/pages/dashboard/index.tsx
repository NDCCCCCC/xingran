import type { FC } from 'react';
import { Row, Col, Card, Statistic, Typography } from 'antd';
import { UserOutlined, TeamOutlined, FileTextOutlined, MonitorOutlined } from '@ant-design/icons';
import { useAuthStore } from '@/store/authStore';

const { Text } = Typography;

const Dashboard: FC = () => {
  const { user } = useAuthStore();

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">
        欢迎回来，{user?.nickname || user?.username}！
      </h1>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="在线用户"
              value={1128}
              prefix={<UserOutlined />}
              styles={{ content: { color: '#3f8600' } }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="今日访问"
              value={2156}
              prefix={<MonitorOutlined />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="消息通知"
              value={12}
              prefix={<FileTextOutlined />}
              styles={{ content: { color: '#722ed1' } }}
            />
          </Card>
        </Col>

        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="待办事项"
              value={5}
              prefix={<TeamOutlined />}
              styles={{ content: { color: '#eb2f96' } }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} className="mt-6">
        <Col xs={24} lg={12}>
          <Card title="系统信息">
            <div className="space-y-3">
              <div className="flex justify-between">
                <Text type="secondary">系统版本</Text>
                <Text>RuoYi Next v1.0.0</Text>
              </div>
              <div className="flex justify-between">
                <Text type="secondary">运行环境</Text>
                <Text>Go 1.23.2 + PostgreSQL 18.1</Text>
              </div>
              <div className="flex justify-between">
                <Text type="secondary">服务器地址</Text>
                <Text>localhost:8080</Text>
              </div>
              <div className="flex justify-between">
                <Text type="secondary">操作系统</Text>
                <Text>Linux</Text>
              </div>
            </div>
          </Card>
        </Col>

        <Col xs={24} lg={12}>
          <Card title="快速操作">
            <div className="grid grid-cols-2 gap-4">
              <button className="p-4 bg-blue-50 text-blue-600 rounded-lg hover:bg-blue-100 transition-colors">
                <div className="text-2xl mb-2">👥</div>
                <div>用户管理</div>
              </button>
              <button className="p-4 bg-green-50 text-green-600 rounded-lg hover:bg-green-100 transition-colors">
                <div className="text-2xl mb-2">🔐</div>
                <div>角色管理</div>
              </button>
              <button className="p-4 bg-purple-50 text-purple-600 rounded-lg hover:bg-purple-100 transition-colors">
                <div className="text-2xl mb-2">📋</div>
                <div>菜单管理</div>
              </button>
              <button className="p-4 bg-orange-50 text-orange-600 rounded-lg hover:bg-orange-100 transition-colors">
                <div className="text-2xl mb-2">🏢</div>
                <div>部门管理</div>
              </button>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;