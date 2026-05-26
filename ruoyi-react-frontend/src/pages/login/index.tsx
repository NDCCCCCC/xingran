import { useState, useEffect } from 'react';
import type { FC } from 'react';
import { Form, Input, Button, Card, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useMenuStore } from '@/store/menuStore';
import { TextCaptcha } from '@/components/captcha';
import CaptchaModal, { type CaptchaSuccessData } from '@/components/captcha/CaptchaModal';
import { getCaptchaConfig } from '@/services/captcha';
import { DASHBOARD } from '@/constants/routes';
import type { LoginRequest } from '@/types';
import type { CaptchaEnabled } from '@/types/captcha';
import './login.css';

const Login: FC = () => {
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const { login } = useAuthStore();
  const { fetchMenus, fetchPermissions } = useMenuStore();
  const [captchaEnabled, setCaptchaEnabled] = useState<CaptchaEnabled>('disabled');
  const [captchaValue, setCaptchaValue] = useState('');
  const [captchaId, setCaptchaId] = useState('');

  // 验证码模态框状态
  const [captchaModalVisible, setCaptchaModalVisible] = useState(false);

  // 保存待提交的登录数据
  const [pendingLoginData, setPendingLoginData] = useState<LoginRequest | null>(null);

  // 加载验证码配置
  useEffect(() => {
    const loadCaptchaConfig = async () => {
      try {
        const config = await getCaptchaConfig();
        setCaptchaEnabled(config.enabled);
      } catch (error) {
        // 配置加载失败，默认不显示验证码
        console.error('加载验证码配置失败:', error);
      }
    };
    loadCaptchaConfig();
  }, []);

  // 执行登录请求
  const performLogin = async (loginData: LoginRequest) => {
    setLoading(true);
    try {
      await login(loginData);
      // 登录成功后获取用户菜单和权限
      await Promise.all([
        fetchMenus(),
        fetchPermissions(),
      ]);
      message.success('登录成功');
      navigate(DASHBOARD);
    } catch (error) {
      console.error('登录失败:', error);
      // 登录失败后刷新验证码
      if (captchaEnabled !== 'disabled') {
        setCaptchaId(''); // 清空验证码ID，触发组件重新加载
      }
    } finally {
      setLoading(false);
    }
  };

  const handleFinish = async (values: LoginRequest) => {
    const loginData: LoginRequest = {
      username: values.username,
      password: values.password,
    };

    // 如果启用了验证码，根据类型处理
    if (captchaEnabled !== 'disabled') {
      if (captchaEnabled === 'slider') {
        // 滑动验证码：先显示模态框，验证成功后再提交
        setPendingLoginData(loginData);
        setCaptchaModalVisible(true);
        return;
      } else {
        // 数字验证码：直接在表单中输入验证码
        if (!captchaValue) {
          message.warning('请输入验证码');
          return;
        }
        loginData.captcha = captchaValue;
        loginData.captchaId = captchaId;
      }
    }

    // 执行登录
    await performLogin(loginData);
  };

  // 验证码模态框验证成功回调
  const handleCaptchaModalSuccess = async (data: CaptchaSuccessData) => {
    setCaptchaModalVisible(false);

    if (pendingLoginData) {
      // 添加验证码信息
      const loginDataWithCaptcha = {
        ...pendingLoginData,
        captcha: data.verified ? 'verified' : data.captcha,
        captchaId: data.captchaId,
      };

      // 执行登录
      await performLogin(loginDataWithCaptcha);

      // 清空待提交数据
      setPendingLoginData(null);
    }
  };

  // 验证码模态框取消回调
  const handleCaptchaModalCancel = () => {
    setCaptchaModalVisible(false);
    setPendingLoginData(null);
  };

  const handleCaptchaChange = (value: string, id: string) => {
    setCaptchaValue(value);
    setCaptchaId(id);
  };

  const handleCaptchaError = (error: string) => {
    if (error === 'CAPTCHA_TYPE_MISMATCH') {
      // 验证码类型不匹配，重新加载配置
      window.location.reload();
    }
  };

  return (
    <div
      className="login-container"
      style={{
        background: 'var(--theme-bg-secondary)',
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        transition: 'background var(--theme-transition-slow)',
      }}
    >
      <Card
        className="login-card"
        style={{
          background: 'var(--theme-bg-surface)',
          border: '1px solid var(--theme-border-primary)',
          boxShadow: 'var(--theme-shadow-xl)',
          borderRadius: 'var(--theme-radius-xl)',
          transition: 'all var(--theme-transition-base)',
        }}
      >
        <div className="text-center mb-8">
          <h1
            className="text-3xl font-bold"
            style={{ color: 'var(--theme-text-primary)' }}
          >
            产险湖北数智化平台
          </h1>
          <p
            className="mt-2"
            style={{ color: 'var(--theme-text-secondary)' }}
          >
            企业级权限管理系统
          </p>
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

          {/* 验证码 - 只显示数字验证码 */}
          {captchaEnabled === 'normal' && (
            <Form.Item>
              <TextCaptcha
                value={captchaValue}
                onChange={handleCaptchaChange}
                onError={handleCaptchaError}
              />
            </Form.Item>
          )}

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
          {/* <p>默认账号：admin / admin123</p> */}
        </div>
      </Card>

      {/* 验证码模态框 - 用于滑动验证码 */}
      <CaptchaModal
        visible={captchaModalVisible}
        captchaType={captchaEnabled}
        onSuccess={handleCaptchaModalSuccess}
        onCancel={handleCaptchaModalCancel}
      />
    </div>
  );
};

export default Login;
