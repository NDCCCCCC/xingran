import React, { useState } from 'react';
import { Form, Input, Button, Card, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useMenuStore } from '@/store/menuStore';
import type { LoginRequest } from '@/types';
import './login.css';

const Login: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const { login } = useAuthStore();
  const { fetchMenus } = useMenuStore();

  const handleFinish = async (values: LoginRequest) => {
    setLoading(true);
    try {
      await login(values);
      // 登录成功后获取用户菜单
      await fetchMenus();
      message.success('登录成功');
      navigate('/dashboard');
    } catch (error) {
      console.error('登录失败:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <Card className="login-card">
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold text-gray-800">RuoYi Next</h1>
          <p className="text-gray-600 mt-2">企业级权限管理系统</p>
        </div>

        <Form
          form={form}
          name="login"
          size="large"
          onFinish={handleFinish}
          autoComplete="off"
        >
          <Form.Item
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder="用户名"
            />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder="密码"
            />
          </Form.Item>

          {/* <Form.Item>
            <div className="flex justify-between">
              <Form.Item name="remember" valuePropName="checked" noStyle>
                <Checkbox>记住密码</Checkbox>
              </Form.Item>
              <a href="#" className="text-primary-600">
                忘记密码？
              </a>
            </div>
          </Form.Item> */}

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              className="w-full"
              loading={loading}
            >
              登录
            </Button>
          </Form.Item>
        </Form>

        <div className="text-center text-sm text-gray-500">
          <p>默认账号：admin / admin123</p>
        </div>
      </Card>
    </div>
  );
};

export default Login;