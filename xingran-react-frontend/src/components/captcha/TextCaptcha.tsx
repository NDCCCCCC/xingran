import { useState, useEffect, useCallback } from "react";
import type { FC, ChangeEvent } from "react";
import { App, Button } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import { getCaptcha } from "@/services/captcha";
import type { CaptchaResponse, CaptchaProps } from "@/types/captcha";
import "./TextCaptcha.css";

const TextCaptcha: FC<CaptchaProps> = ({ value, onChange, onError }) => {
  const { message } = App.useApp();
  const [captchaData, setCaptchaData] = useState<CaptchaResponse | null>(null);
  const [loading, setLoading] = useState(false);

  // 加载验证码
  const loadCaptcha = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getCaptcha();
      // 如果返回空对象或没有 captchaType，说明验证码未启用
      if (!data || !data.captchaType) {
        return; // 静默返回，不显示验证码
      }
      if (data.captchaType === "normal") {
        setCaptchaData(data);
        onChange?.("", data.captchaId);
      } else if (data.captchaType === "slider") {
        // 如果返回的是滑动验证码，通知父组件切换
        onError?.("CAPTCHA_TYPE_MISMATCH");
      }
    } catch (error) {
      console.error("[TextCaptcha] 加载验证码失败:", error);
      message.error("加载验证码失败");
      onError?.(error instanceof Error ? error.message : "加载验证码失败");
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [onChange, onError]);

  // 只在组件挂载时加载验证码
   
  useEffect(() => {
    loadCaptcha();
  }, [loadCaptcha]);

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const val = e.target.value;
    onChange?.(val, captchaData?.captchaId || "");
  };

  return (
    <div className="text-captcha-container">
      <div className="captcha-input-wrapper">
        <input
          type="text"
          value={value || ""}
          onChange={handleChange}
          placeholder="请输入验证码"
          className="captcha-input"
          maxLength={6}
        />
        <div className="captcha-image-wrapper">
          {captchaData?.captchaImg && (
            <img
              src={captchaData.captchaImg}
              alt="验证码"
              className="captcha-image"
              onClick={loadCaptcha}
              title="点击刷新"
            />
          )}
          {!loading && (
            <Button
              type="text"
              icon={<ReloadOutlined />}
              onClick={loadCaptcha}
              className="refresh-btn"
              size="small"
            />
          )}
        </div>
      </div>
    </div>
  );
};

export default TextCaptcha;
