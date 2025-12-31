import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Button, message } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { getCaptcha, verifySliderCaptcha } from '@/services/captcha';
import type { CaptchaResponse } from '@/types/captcha';
import './SliderCaptcha.css';

interface SliderCaptchaProps {
  onVerified: (token: string, captchaId: string) => void;
  onError?: (error: string) => void;
}

const SliderCaptcha: React.FC<SliderCaptchaProps> = ({ onVerified, onError }) => {
  const [captchaData, setCaptchaData] = useState<CaptchaResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [currentX, setCurrentX] = useState(0);
  const [verified, setVerified] = useState(false);
  const sliderRef = useRef<HTMLDivElement>(null);
  const dragStateRef = useRef({ currentX: 0 }); // 用 ref 来保存拖动过程中的位置

  // 加载验证码
  const loadCaptcha = async () => {
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
      if (data.captchaType === 'slider') {
        setCaptchaData(data);
      } else {
        onError?.('CAPTCHA_TYPE_MISMATCH');
      }
    } catch (error) {
      message.error('加载验证码失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCaptcha();
  }, []);

  const handleMouseDown = () => {
    if (verified) return;
    console.log('[SliderCaptcha] handleMouseDown - 开始拖动');
    setDragging(true);
  };

  const handleMouseUp = async () => {
    setDragging(false);
    const x = dragStateRef.current.currentX;

    console.log('[SliderCaptcha] 验证 - currentX:', x, 'state.currentX:', currentX, 'captchaId:', captchaData?.captchaId);

    if (!captchaData || !captchaData.token) {
      console.error('[SliderCaptcha] 验证数据不完整');
      return;
    }

    // 验证滑动位置
    try {
      const result = await verifySliderCaptcha({
        captchaId: captchaData.captchaId,
        xPos: Math.round(x), // 取整
        token: captchaData.token, // 使用后端返回的 token
      });

      if (result.success) {
        setVerified(true);
        message.success('验证成功');
        onVerified(result.token, captchaData.captchaId);
      } else {
        message.error('验证失败，请重试');
        loadCaptcha();
      }
    } catch (error) {
      console.error('[SliderCaptcha] 验证失败:', error);
      message.error('验证失败');
      loadCaptcha();
    }
  };

  const handleMouseMove = (e: MouseEvent) => {
    if (!sliderRef.current) return;

    const rect = sliderRef.current.getBoundingClientRect();
    let x = e.clientX - rect.left;
    const maxX = 240; // 轨道宽度 - 滑块宽度
    x = Math.max(0, Math.min(x, maxX));

    dragStateRef.current.currentX = x;
    setCurrentX(x); // 更新 UI

    console.log('[SliderCaptcha] handleMouseMove - x:', x, 'clientX:', e.clientX, 'rect.left:', rect.left);
  };

  // 手动管理全局事件监听
  useEffect(() => {
    // 拖动开始时添加监听器
    const startDragging = () => {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp as any);
    };

    // 拖动结束时移除监听器
    const stopDragging = () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp as any);
    };

    if (dragging) {
      startDragging();
    }

    // 清理函数
    return () => {
      stopDragging();
    };
  }, [dragging]);

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
              className={`slider-button ${dragging ? 'dragging' : ''} ${verified ? 'verified' : ''}`}
              onMouseDown={handleMouseDown}
              style={{ left: currentX }}
            >
              {verified ? '✓' : '>>'}
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export default SliderCaptcha;
