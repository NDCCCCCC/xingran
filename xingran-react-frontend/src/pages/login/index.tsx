import { useState, useEffect } from "react";
import type { FC } from "react";
import { App, Form, Input, Button, Alert } from "antd";
import {
  UserOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  ApartmentOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/store/authStore";
import { useMenuStore } from "@/store/menuStore";
import { TextCaptcha } from "@/components/captcha";
import CaptchaModal, { type CaptchaSuccessData } from "@/components/captcha/CaptchaModal";
import { getCaptchaConfig } from "@/services/captcha";
import { submitLoginPreflight } from "@/lib/loginPreflight";
import { DASHBOARD } from "@/constants/routes";
import type { LoginRequest } from "@/types";
import type { CaptchaEnabled } from "@/types/captcha";
import "./login.css";

/**
 * 从登录失败错误中提取用户可读的错误文案。
 */
function extractLoginErrorMessage(error: unknown): string {
  if (!error) return "登录失败，请重试";
  const anyError = error as any;
  const respData = anyError?.response?.data;
  if (respData && typeof respData === "object") {
    if (typeof respData.message === "string" && respData.message) return respData.message;
    if (typeof respData.msg === "string" && respData.msg) return respData.msg;
  }
  if (typeof anyError?.message === "string" && anyError.message) {
    return anyError.message;
  }
  return "登录失败，请重试";
}

const Login: FC = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [loginError, setLoginError] = useState<string>("");
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const { login } = useAuthStore();
  const { fetchMenus, fetchPermissions } = useMenuStore();
  const [captchaEnabled, setCaptchaEnabled] = useState<CaptchaEnabled>("disabled");
  const [captchaValue, setCaptchaValue] = useState("");
  const [captchaId, setCaptchaId] = useState("");
  const [captchaModalVisible, setCaptchaModalVisible] = useState(false);
  const [pendingLoginData, setPendingLoginData] = useState<LoginRequest | null>(null);

  useEffect(() => {
    const loadCaptchaConfig = async () => {
      try {
        const config = await getCaptchaConfig();
        setCaptchaEnabled(config.enabled);
      } catch (error) {
        console.error("加载验证码配置失败:", error);
      }
    };
    loadCaptchaConfig();
  }, []);

  const performLogin = async (loginData: LoginRequest) => {
    setLoading(true);
    setLoginError("");
    try {
      await login(loginData);
      await Promise.all([fetchMenus(), fetchPermissions()]);
      message.success("登录成功");
      navigate(DASHBOARD);
    } catch (error) {
      console.error("登录失败:", error);
      setLoginError(extractLoginErrorMessage(error));
      if (captchaEnabled !== "disabled") {
        setCaptchaId("");
      }
    } finally {
      setLoading(false);
    }
  };

  const handleFinish = async (values: LoginRequest) => {
    setLoginError("");
    setLoading(true);

    const preflight = await submitLoginPreflight();
    if (!preflight.ok) {
      setLoginError(preflight.friendlyMessage);
      setLoading(false);
      return;
    }

    setCaptchaEnabled(preflight.captchaEnabled);

    const loginData: LoginRequest = {
      username: values.username,
      password: values.password,
    };

    if (preflight.captchaEnabled !== "disabled") {
      if (preflight.captchaEnabled === "slider") {
        setPendingLoginData(loginData);
        setCaptchaModalVisible(true);
        setLoading(false);
        return;
      } else {
        if (!captchaValue) {
          message.warning("请输入验证码");
          setLoading(false);
          return;
        }
        loginData.captcha = captchaValue;
        loginData.captchaId = captchaId;
      }
    }

    await performLogin(loginData);
  };

  const handleCaptchaModalSuccess = async (data: CaptchaSuccessData) => {
    setCaptchaModalVisible(false);
    if (pendingLoginData) {
      const loginDataWithCaptcha = {
        ...pendingLoginData,
        captcha: data.verified ? "verified" : data.captcha,
        captchaId: data.captchaId,
      };
      await performLogin(loginDataWithCaptcha);
      setPendingLoginData(null);
    }
  };

  const handleCaptchaModalCancel = () => {
    setCaptchaModalVisible(false);
    setPendingLoginData(null);
  };

  const handleCaptchaChange = (value: string, id: string) => {
    setCaptchaValue(value);
    setCaptchaId(id);
  };

  const handleCaptchaError = async (error: string) => {
    if (error !== "CAPTCHA_TYPE_MISMATCH") return;
    const preflight = await submitLoginPreflight();
    if (!preflight.ok) {
      setLoginError(preflight.friendlyMessage);
      return;
    }
    setCaptchaModalVisible(false);
    setPendingLoginData(null);
    setCaptchaEnabled(preflight.captchaEnabled);
    setCaptchaValue("");
    setCaptchaId("");
    setLoginError("验证码配置已更新，请重新验证");
  };

  return (
    <div className="login-container">
      {/* 左侧品牌面板 - 墨绿琥珀 */}
      <aside className="login-brand" aria-label="品牌信息">
        {/* 装饰几何 */}
        <div className="login-brand__decor login-brand__decor--hex" aria-hidden="true">
          <svg viewBox="0 0 100 100">
            <polygon points="50,3 92,25 92,75 50,97 8,75 8,25" />
          </svg>
        </div>
        <div className="login-brand__decor login-brand__decor--ring" aria-hidden="true">
          <svg viewBox="0 0 100 100">
            <circle cx="50" cy="50" r="46" />
          </svg>
        </div>

        <div className="login-brand__top">
          <div className="login-brand__mark">星</div>
          <span className="login-brand__wordmark">星苒 · XINGRAN</span>
        </div>

        <div className="login-brand__center">
          <span className="login-brand__eyebrow">
            <span className="login-brand__eyebrow-dot" />
            v1.0 · 国密级安全
          </span>

          <h1 className="login-brand__title">
            光启万物
            <span className="login-brand__title-sep">·</span>
            荫庇四方
          </h1>

          <p className="login-brand__tagline">
            以国密算法为基座,融合自动化与 AI 调度,让每一次资源调度都可观测、可审计、可信赖。
          </p>

          <div className="login-brand__features">
            <div className="login-brand__feature">
              <div className="login-brand__feature-icon">
                <SafetyCertificateOutlined />
              </div>
              <div className="login-brand__feature-text">
                <span className="login-brand__feature-title">国密安全</span>
                <span className="login-brand__feature-desc">SM2 / SM3 / SM4 端到端加密</span>
              </div>
            </div>

            <div className="login-brand__feature">
              <div className="login-brand__feature-icon">
                <ApartmentOutlined />
              </div>
              <div className="login-brand__feature-text">
                <span className="login-brand__feature-title">资产智能调度</span>
                <span className="login-brand__feature-desc">楼宇、工位、机房全生命周期管理</span>
              </div>
            </div>

            <div className="login-brand__feature">
              <div className="login-brand__feature-icon">
                <ThunderboltOutlined />
              </div>
              <div className="login-brand__feature-text">
                <span className="login-brand__feature-title">实时监控告警</span>
                <span className="login-brand__feature-desc">多维指标可视化与秒级异常响应</span>
              </div>
            </div>
          </div>
        </div>

        <div className="login-brand__footer">
          <span className="login-brand__badge">SM2</span>
          <span className="login-brand__badge">SM3</span>
          <span className="login-brand__badge">SM4</span>
          <span className="login-brand__copyright">© 2026 XingRan-Next</span>
        </div>
      </aside>

      {/* 右侧登录表单 - 米白 + 琥珀金按钮 */}
      <main className="login-panel">
        <div className="login-form-card">
          <div className="login-header">
            <h2 className="login-header__title">欢迎回来</h2>
            <p className="login-header__subtitle">请使用您的账号登录以访问运维控制台</p>
          </div>

          {loginError && (
            <Alert
              className="login-alert"
              type="error"
              showIcon
              closable
              message={loginError}
              onClose={() => setLoginError("")}
            />
          )}

          <Form
            form={form}
            name="login"
            size="large"
            onFinish={handleFinish}
            autoComplete="off"
            className="login-form"
            layout="vertical"
          >
            <Form.Item
              name="username"
              label={<span className="login-label">账号</span>}
              rules={[{ required: true, message: "请输入用户名" }]}
            >
              <Input prefix={<UserOutlined />} placeholder="用户名" autoComplete="username" />
            </Form.Item>

            <Form.Item
              name="password"
              label={<span className="login-label">密码</span>}
              rules={[{ required: true, message: "请输入密码" }]}
            >
              <Input.Password
                prefix={<LockOutlined />}
                placeholder="密码"
                autoComplete="current-password"
              />
            </Form.Item>

            {captchaEnabled === "normal" && (
              <Form.Item label={<span className="login-label">验证码</span>}>
                <div className="login-captcha-wrapper">
                  <TextCaptcha
                    value={captchaValue}
                    onChange={handleCaptchaChange}
                    onError={handleCaptchaError}
                  />
                </div>
              </Form.Item>
            )}

            <Form.Item style={{ marginBottom: 0, marginTop: 24 }}>
              <Button type="primary" htmlType="submit" loading={loading} className="login-submit">
                登录
              </Button>
            </Form.Item>
          </Form>

          <div className="login-footer">
            <span>使用企业账号登录</span>
            <span>v1.0.0</span>
          </div>
        </div>
      </main>

      <CaptchaModal
        visible={captchaModalVisible}
        captchaType={captchaEnabled}
        onSuccess={handleCaptchaModalSuccess}
        onCancel={handleCaptchaModalCancel}
        onError={handleCaptchaError}
      />
    </div>
  );
};

export default Login;
