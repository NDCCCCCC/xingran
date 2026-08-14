import { useState, useEffect } from "react";
import type { FC } from "react";
import { App, Form, Input, Button, Card, Alert } from "antd";
import { UserOutlined, LockOutlined } from "@ant-design/icons";
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
 * 兼容多种来源：
 * - 普通 Error（message 字段）—— 例如 api.ts 登录短路 reject 出来的 new Error(backendMessage)
 * - axios 错误（response.data.message / response.data.msg）—— 例如 403 账号锁定、验证码错误
 */
function extractLoginErrorMessage(error: unknown): string {
  if (!error) return "登录失败，请重试";
  // axios 风格错误
  const anyError = error as any;
  const respData = anyError?.response?.data;
  if (respData && typeof respData === "object") {
    if (typeof respData.message === "string" && respData.message) return respData.message;
    if (typeof respData.msg === "string" && respData.msg) return respData.msg;
  }
  // 普通 Error
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
        console.error("加载验证码配置失败:", error);
      }
    };
    loadCaptchaConfig();
  }, []);

  // 执行登录请求
  const performLogin = async (loginData: LoginRequest) => {
    setLoading(true);
    // 每次发起登录前清除上一次的内联错误提示
    setLoginError("");
    try {
      await login(loginData);
      // 登录成功后获取用户菜单和权限
      await Promise.all([fetchMenus(), fetchPermissions()]);
      message.success("登录成功");
      navigate(DASHBOARD);
    } catch (error) {
      console.error("登录失败:", error);
      // 提取后端错误信息，显示为表单内联提示
      setLoginError(extractLoginErrorMessage(error));
      // 登录失败后刷新验证码
      if (captchaEnabled !== "disabled") {
        setCaptchaId(""); // 清空验证码ID，触发组件重新加载
      }
    } finally {
      setLoading(false);
    }
  };

  const handleFinish = async (values: LoginRequest) => {
    // 登录页可能已长时间停留，提交前先同步加密开关、SM2 公钥和验证码类型。
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

    // 使用本次预检得到的验证码类型，不依赖页面挂载时的旧状态。
    if (preflight.captchaEnabled !== "disabled") {
      if (preflight.captchaEnabled === "slider") {
        // 滑动验证码：先显示模态框，验证成功后再提交
        setPendingLoginData(loginData);
        setCaptchaModalVisible(true);
        setLoading(false);
        return;
      } else {
        // 数字验证码：直接在表单中输入验证码
        if (!captchaValue) {
          message.warning("请输入验证码");
          setLoading(false);
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
        captcha: data.verified ? "verified" : data.captcha,
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

  const handleCaptchaError = async (error: string) => {
    if (error !== "CAPTCHA_TYPE_MISMATCH") {
      return;
    }

    // 验证码类型在页面停留期间发生变化时局部同步，不丢失用户已填写的账号密码。
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
    <div
      className="login-container"
      style={{
        background: "var(--theme-bg-secondary)",
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        transition: "background var(--theme-transition-slow)",
      }}
    >
      <Card
        className="login-card"
        style={{
          background: "var(--theme-bg-surface)",
          border: "1px solid var(--theme-border-primary)",
          boxShadow: "var(--theme-shadow-xl)",
          borderRadius: "var(--theme-radius-xl)",
          transition: "all var(--theme-transition-base)",
        }}
      >
        <div className="text-center mb-8">
          <h1 className="text-3xl font-bold" style={{ color: "var(--theme-text-primary)" }}>
            星苒
          </h1>
          <p className="mt-2" style={{ color: "var(--theme-text-secondary)" }}>
            光启万物，荫庇四方
          </p>
        </div>

        {/* 登录失败内联提示 —— 仅当存在错误时显示 */}
        {loginError && (
          <Alert
            type="error"
            showIcon
            closable
            title={loginError}
            onClose={() => setLoginError("")}
            style={{ marginBottom: 16 }}
          />
        )}

        <Form form={form} name="login" size="large" onFinish={handleFinish} autoComplete="off">
          <Form.Item name="username" rules={[{ required: true, message: "请输入用户名" }]}>
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>

          <Form.Item name="password" rules={[{ required: true, message: "请输入密码" }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>

          {/* 验证码 - 只显示数字验证码 */}
          {captchaEnabled === "normal" && (
            <Form.Item>
              <TextCaptcha
                value={captchaValue}
                onChange={handleCaptchaChange}
                onError={handleCaptchaError}
              />
            </Form.Item>
          )}

          <Form.Item>
            <Button type="primary" htmlType="submit" className="w-full" loading={loading}>
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
        onError={handleCaptchaError}
      />
    </div>
  );
};

export default Login;
