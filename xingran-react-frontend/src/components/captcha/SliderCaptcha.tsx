import { useState, useEffect, useRef, useCallback } from "react";
import type { FC, PointerEvent as ReactPointerEvent } from "react";
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

// 滑块按钮显示宽度（.slider-button CSS width: 50px）
const SLIDER_BUTTON_WIDTH = 50;

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
  // 背景图引用：读取 naturalWidth 用于「显示坐标 → 原始图片坐标」的缩放转换
  const bgImgRef = useRef<HTMLImageElement>(null);

  // rAF 节流：避免每次 pointermove 都强制 reflow + setState，减少渲染堆积
  const rafIdRef = useRef<number | null>(null);
  const pendingXRef = useRef<number | null>(null);

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
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [onError]);

  // 只在组件激活（模态框打开）时加载验证码
  useEffect(() => {
    if (active) {
      loadCaptcha();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active]);

  // 组件卸载时清理 rAF
  useEffect(() => {
    return () => {
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, []);

  const handleMouseUp = useCallback(async () => {
    setDragging(false);
    const displayX = dragStateRef.current.currentX;

    if (!captchaData || !captchaData.token) {
      console.error("[SliderCaptcha] 验证数据不完整");
      return;
    }

    // 坐标缩放转换：前端「显示坐标」→ 后端「原始图片坐标」
    // 后端在固定宽度（400px）的原始坐标系生成缺口（pkg/captcha/slider.go），
    // 前端容器为响应式（width:100%），图片被 object-fit 缩放显示，
    // 直接提交显示坐标会造成系统性偏移（后端容差仅 8px）。
    // 用 naturalWidth/clientWidth 还原缩放比，保证提交坐标落在原始坐标系。
    const bgImg = bgImgRef.current;
    const containerWidth = sliderRef.current?.clientWidth ?? 0;
    let xPos: number;
    if (bgImg && bgImg.naturalWidth > 0 && containerWidth > 0) {
      xPos = Math.round(displayX * (bgImg.naturalWidth / containerWidth));
    } else {
      // 图片尚未 decode 完成时的退路（极少见），直接取整显示坐标
      xPos = Math.round(displayX);
    }

    // 验证滑动位置
    try {
      const result = await verifySliderCaptcha({
        captchaId: captchaData.captchaId,
        xPos,
        token: captchaData.token, // 使用后端返回的 token
      });

      if (result.success) {
        setVerified(true);
        message.success("验证成功");
        onVerified(result.token, captchaData.captchaId);
      } else {
        message.error("验证失败，请重试");
        loadCaptcha();
      }
    } catch (error) {
      console.error("[SliderCaptcha] 验证失败:", error);
      message.error("验证失败");
      loadCaptcha();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [captchaData, onVerified, loadCaptcha]);

  // ---- 指针事件处理（Pointer Events + setPointerCapture 根治事件流抢占）----
  // 关键修复：
  // 1. e.preventDefault() 阻止原生图片拖拽（防 dragstart 抢占 mousemove）
  // 2. setPointerCapture 保证拖动期间持续接收 pointermove，免疫鼠标移出窗口/越过 iframe
  // 3. rAF 节流 setState，避免高频 getBoundingClientRect + 重渲染
  const handlePointerDown = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      if (verified) return;
      // 必须在 pointerdown 上 preventDefault，阻止浏览器启动原生图片拖拽
      // （Chrome 在拖动中光标经过 .slider-bg 时会触发 dragstart，使 document 的
      // mousemove 停止派发，导致滑块定住，只有松手 dragend 才恢复 —— 即用户报告的卡顿）
      e.preventDefault();

      // 捕获指针：拖动期间事件持续派发到本元素，不会被其它元素/窗口边界抢占
      try {
        e.currentTarget.setPointerCapture(e.pointerId);
      } catch {
        // setPointerCapture 在极个别浏览器可能抛 InvalidPointerId，忽略不影响主流程
      }

      setDragging(true);
    },
    [verified]
  );

  const handlePointerMove = useCallback((e: ReactPointerEvent<HTMLDivElement>) => {
    if (!dragging || !sliderRef.current) return;

    const rect = sliderRef.current.getBoundingClientRect();
    // 动态计算可拖动范围：容器显示宽度 - 滑块按钮宽度
    // （原硬编码 MAX_X=240 远小于实际容器宽度，导致滑块拖不到最右）
    const maxDisplayX = Math.max(0, rect.width - SLIDER_BUTTON_WIDTH);
    let x = e.clientX - rect.left;
    x = Math.max(0, Math.min(x, maxDisplayX));

    // 写入 ref（验证时读取），rAF 内统一 setState
    dragStateRef.current.currentX = x;
    pendingXRef.current = x;

    if (rafIdRef.current === null) {
      rafIdRef.current = requestAnimationFrame(() => {
        rafIdRef.current = null;
        if (pendingXRef.current !== null) {
          setCurrentX(pendingXRef.current);
          pendingXRef.current = null;
        }
      });
    }
  }, [dragging]);

  const handlePointerUp = useCallback(
    (e: ReactPointerEvent<HTMLDivElement>) => {
      // 释放指针捕获
      try {
        if (e.currentTarget.hasPointerCapture(e.pointerId)) {
          e.currentTarget.releasePointerCapture(e.pointerId);
        }
      } catch {
        // 忽略
      }

      // 立即 flush 最后一帧位置到 state（避免视觉跳变）
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current);
        rafIdRef.current = null;
      }
      if (pendingXRef.current !== null) {
        setCurrentX(pendingXRef.current);
        pendingXRef.current = null;
      }

      handleMouseUp();
    },
    [handleMouseUp]
  );

  return (
    <div className="slider-captcha-container">
      {captchaData && (
        <>
          <div className="slider-canvas-wrapper" ref={sliderRef}>
            {captchaData.sliderImg && (
              <img
                ref={bgImgRef}
                src={captchaData.sliderImg}
                alt="滑动验证码"
                className="slider-bg"
                onClick={loadCaptcha}
                title="点击刷新验证码"
                // 关键修复：禁用原生图片拖拽，防止拖动滑块时光标经过背景图触发 dragstart
                draggable={false}
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
              onPointerDown={handlePointerDown}
              onPointerMove={handlePointerMove}
              onPointerUp={handlePointerUp}
              onPointerCancel={handlePointerUp}
              // 兜底：旧版浏览器无 Pointer Events 时，至少仍可阻止原生拖拽默认行为
              onMouseDown={(e) => e.preventDefault()}
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
