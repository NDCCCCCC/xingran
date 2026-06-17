import { useState, useEffect, useRef, useCallback } from "react";
import type { FC } from "react";
import { Button, App } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import { getCaptcha, verifySliderCaptcha } from "@/services/captcha";
import type { CaptchaResponse } from "@/types/captcha";
import "./SliderCaptcha.css";

interface SliderCaptchaProps {
  onVerified: (token: string, captchaId: string) => void;
  onError?: (error: string) => void;
  active?: boolean; // 是否激活（模态框已打开）
}

const SliderCaptcha: FC<SliderCaptchaProps> = ({ onVerified, onError, active = true }) => {
  // 使用 antd v5 App context 的 message 实例，确保能消费 ConfigProvider 的动态主题
  const { message } = App.useApp();
  const [captchaData, setCaptchaData] = useState<CaptchaResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [currentX, setCurrentX] = useState(0);
  const [verified, setVerified] = useState(false);
  const sliderRef = useRef<HTMLDivElement>(null);
  const dragStateRef = useRef({ currentX: 0 }); // 用 ref 来保存拖动过程中的位置

  // 加载验证码 - 使用 useCallback 保持引用稳定
  const loadCaptcha = useCallback(async () => {
    setLoading(true);
    setVerified(false);
    setCurrentX(0);
    dragStateRef.current = { currentX: 0 };
    try {
      const data = await getCaptcha();
      // 如果返回空对象或没有 captchaType，说明验证码未启用
      if (!data || !data.captchaType) {
        return; // 静默返回，不显示验证码
      }
      if (data.captchaType === "slider") {
        setCaptchaData(data);
      } else {
        onError?.("CAPTCHA_TYPE_MISMATCH");
      }
    } catch {
      message.error("加载验证码失败");
    } finally {
      setLoading(false);
    }
  }, [onError]);

  // 只在组件激活（模态框打开）时加载验证码
  useEffect(() => {
    if (active) {
      loadCaptcha();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active]);

  const handleMouseDown = useCallback(() => {
    if (verified) return;
    setDragging(true);
  }, [verified]);

  const handleMouseUp = useCallback(async () => {
    setDragging(false);
    const x = dragStateRef.current.currentX;

    if (!captchaData || !captchaData.token) {
      console.error("[SliderCaptcha] 验证数据不完整");
      return;
    }

    // 验证滑动位置
    try {
      console.log("[SliderCaptcha] 开始验证滑动位置:", { x, captchaId: captchaData.captchaId, token: captchaData.token });
      const result = await verifySliderCaptcha({
        captchaId: captchaData.captchaId,
        xPos: Math.round(x), // 取整
        token: captchaData.token, // 使用后端返回的 token
      });

      console.log("[SliderCaptcha] 验证响应:", result);
      if (result.success) {
        console.log("[SliderCaptcha] 验证成功，调用onVerified");
        setVerified(true);
        message.success("验证成功");
        onVerified(result.token, captchaData.captchaId);
      } else {
        console.log("[SliderCaptcha] 验证失败（success=false）");
        message.error("验证失败，请重试");
        loadCaptcha();
      }
    } catch (error) {
      console.log("[SliderCaptcha] 验证异常（抛出错误）:", error);
      console.error("[SliderCaptcha] 验证失败:", error);
      message.error("验证失败");
      loadCaptcha();
    }
  }, [captchaData, onVerified, loadCaptcha]);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!sliderRef.current) return;

    const rect = sliderRef.current.getBoundingClientRect();
    let x = e.clientX - rect.left;
    const maxX = 240; // 轨道宽度 - 滑块宽度
    x = Math.max(0, Math.min(x, maxX));

    dragStateRef.current.currentX = x;
    setCurrentX(x); // 更新 UI
  }, []);

  // 手动管理全局事件监听
  useEffect(() => {
    if (!dragging) return;

    // 定义事件监听器类型
    const mouseMoveListener = handleMouseMove;
    const mouseUpListener = handleMouseUp;

    document.addEventListener("mousemove", mouseMoveListener);
    document.addEventListener("mouseup", mouseUpListener);

    // 清理函数
    return () => {
      document.removeEventListener("mousemove", mouseMoveListener);
      document.removeEventListener("mouseup", mouseUpListener);
    };
  }, [dragging, handleMouseMove, handleMouseUp]);

  return (
    <div className="slider-captcha-container">
      {captchaData && (
        <>
          <div className="slider-canvas-wrapper" ref={sliderRef}>
            {captchaData.sliderImg && (
              <img
                src={captchaData.sliderImg}
                alt="滑动验证码"
                className="slider-bg"
                onClick={loadCaptcha}
                title="点击刷新验证码"
              />
            )}
            {captchaData.pieceImg && (
              <img
                src={captchaData.pieceImg}
                alt="拼图块"
                className="slider-piece"
                style={{ left: currentX, top: captchaData.yPos }}
                draggable={false}
              />
            )}
            {verified && (
              <div className="slider-success-overlay">
                <span className="success-icon">✓</span>
                <span>验证成功</span>
              </div>
            )}
            {/* 刷新按钮放在右上角 */}
            <Button
              type="text"
              icon={<ReloadOutlined spin={loading} />}
              onClick={loadCaptcha}
              className="refresh-btn-overlay"
              size="small"
              title="刷新验证码"
            />
          </div>
          <div className="slider-track">
            <div
              className={`slider-button ${dragging ? "dragging" : ""} ${verified ? "verified" : ""}`}
              onMouseDown={handleMouseDown}
              style={{ left: currentX }}
            >
              {verified ? "✓" : ">>"}
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export default SliderCaptcha;
