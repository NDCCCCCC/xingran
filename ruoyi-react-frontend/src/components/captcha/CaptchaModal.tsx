/**
 * CaptchaModal 验证码模态框组件
 * 用于在登录时弹出滑动验证码或数字验证码
 */

import { useState, useEffect, type FC } from "react";
import { App, Modal } from "antd";
import { SliderCaptcha, TextCaptcha } from "@/components/captcha";
import type { CaptchaEnabled } from "@/types/captcha";

export interface CaptchaModalProps {
  visible: boolean;
  captchaType: CaptchaEnabled;
  onSuccess: (data: CaptchaSuccessData) => void;
  onCancel: () => void;
  onError?: (error: string) => void;
}

export interface CaptchaSuccessData {
  captchaId: string;
  captcha?: string; // normal验证码的值
  verified?: boolean; // slider验证码的验证状态
}

const CaptchaModal: FC<CaptchaModalProps> = ({
  visible,
  captchaType,
  onSuccess,
  onCancel,
  onError,
}) => {
  const { message } = App.useApp();
  const [captchaValue, setCaptchaValue] = useState("");
  const [captchaId, setCaptchaId] = useState("");
  const [, setSliderVerified] = useState(false);

  // 当模态框打开时，重置验证码状态
  useEffect(() => {
    if (visible) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- intentional reset on visibility change
      setCaptchaValue("");
      setCaptchaId("");
      setSliderVerified(false);
    }
  }, [visible]);

  // 处理验证码验证成功
  const handleCaptchaSuccess = (data: CaptchaSuccessData) => {
    // 延迟关闭模态框，让用户看到"验证成功"提示
    setTimeout(() => {
      onSuccess(data);
    }, 500);
  };

  // 数字验证码：点击确定按钮
  const handleNormalConfirm = () => {
    if (!captchaValue) {
      message.warning("请输入验证码");
      return;
    }

    if (!captchaId) {
      message.error("验证码信息不完整，请重试");
      return;
    }

    handleCaptchaSuccess({
      captchaId,
      captcha: captchaValue,
    });
  };

  // 数字验证码变更回调
  const handleCaptchaChange = (value: string, id: string) => {
    setCaptchaValue(value);
    setCaptchaId(id);
  };

  // 滑动验证码验证成功回调
  const handleSliderVerified = (_token: string, id: string) => {
    setCaptchaId(id);
    setSliderVerified(true);

    handleCaptchaSuccess({
      captchaId: id,
      verified: true,
    });
  };

  // 验证码错误回调
  const handleCaptchaError = (error: string) => {
    console.error("[CaptchaModal] 验证码错误:", error);

    if (error === "CAPTCHA_TYPE_MISMATCH") {
      message.info("验证码配置已变化，正在自动更新");
      if (onError) {
        onError(error);
      } else {
        onCancel();
      }
    }
  };

  // 计算模态框宽度
  const getModalWidth = () => {
    return captchaType === "slider" ? 500 : 400;
  };

  return (
    <Modal
      title="安全验证"
      open={visible}
      onCancel={onCancel}
      footer={null}
      closable={false}
      maskClosable={false}
      width={getModalWidth()}
      centered
      destroyOnHidden
    >
      <div style={{ padding: "16px 0" }}>
        {captchaType === "slider" ? (
          <SliderCaptcha
            active={visible}
            onVerified={handleSliderVerified}
            onError={handleCaptchaError}
          />
        ) : (
          <div>
            <TextCaptcha
              value={captchaValue}
              onChange={handleCaptchaChange}
              onError={handleCaptchaError}
            />
            <div style={{ marginTop: "16px", textAlign: "center" }}>
              <button
                type="button"
                onClick={handleNormalConfirm}
                style={{
                  padding: "8px 24px",
                  background: "var(--theme-primary, #337ab0)",
                  color: "white",
                  border: "none",
                  borderRadius: "4px",
                  cursor: "pointer",
                  fontSize: "14px",
                }}
              >
                确定
              </button>
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
};

export default CaptchaModal;
