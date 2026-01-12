import React, { useState, useEffect } from 'react';
import { Form, Input, Button, Card, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/authStore';
import { useMenuStore } from '@/store/menuStore';
import { TextCaptcha, SliderCaptcha } from '@/components/captcha';
import { getCaptchaConfig } from '@/services/captcha';
import type { LoginRequest } from '@/types';
import type { CaptchaEnabled } from '@/types/captcha';
import './login.css';

const Login: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const { login } = useAuthStore();
  const { fetchMenus } = useMenuStore();
  const [captchaEnabled, setCaptchaEnabled] = useState<CaptchaEnabled>('disabled');
  const [captchaValue, setCaptchaValue] = useState('');
  const [captchaId, setCaptchaId] = useState('');
  const [sliderVerified, setSliderVerified] = useState(false);

  // 加载验证码配置
  useEffect(() => {
    const loadCaptchaConfig = async () => {
      try {
        const config = await getCaptchaConfig();
        console.log('[DEBUG] 获取到的验证码配置:', config);
        setCaptchaEnabled(config.enabled);
      } catch (error) {
        // 配置加载失败，默认不显示验证码
        console.error('加载验证码配置失败:', error);
      }
    };
    loadCaptchaConfig();
  }, []);

  const handleFinish = async (values: LoginRequest) => {
    console.log('[DEBUG] 登录提交 - captchaEnabled:', captchaEnabled);
    console.log('[DEBUG] 验证码值:', captchaValue, 'ID:', captchaId, 'sliderVerified:', sliderVerified);

    setLoading(true);
    try {
      const loginData: LoginRequest = {
        username: values.username,
        password: values.password,
      };

      // 如果启用了验证码，添加验证码参数
      if (captchaEnabled !== 'disabled') {
        if (captchaEnabled === 'slider') {
          if (!sliderVerified) {
            message.warning('请先完成验证码验证');
            setLoading(false);
            return;
          }
          // 滑动验证码验证通过后，token 和 captchaId 由组件传递
          loginData.captcha = 'verified';
          loginData.captchaId = captchaId;
        } else {
          if (!captchaValue) {
            message.warning('请输入验证码');
            setLoading(false);
            return;
          }
          loginData.captcha = captchaValue;
          loginData.captchaId = captchaId;
        }
      }

      console.log('[DEBUG] 准备登录，数据:', loginData);
      await login(loginData);
      // 登录成功后获取用户菜单
      await fetchMenus();
      message.success('登录成功');
      navigate('/dashboard');
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

  const handleCaptchaChange = (value: string, id: string) => {
    setCaptchaValue(value);
    setCaptchaId(id);
  };

  const handleSliderVerified = (token: string, id: string) => {
    setCaptchaId(id);
    setSliderVerified(true);
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

          {/* 验证码 */}
          {captchaEnabled !== 'disabled' && (
            <Form.Item>
              {captchaEnabled === 'normal' ? (
                <TextCaptcha
                  value={captchaValue}
                  onChange={handleCaptchaChange}
                  onError={handleCaptchaError}
                />
              ) : (
                <SliderCaptcha
                  onVerified={handleSliderVerified}
                  onError={handleCaptchaError}
                />
              )}
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
    </div>
  );
};

export default Login;
