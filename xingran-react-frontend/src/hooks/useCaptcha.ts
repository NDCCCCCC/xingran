// 验证码Hook
import { useState, useEffect, useCallback } from "react";
import { getCaptcha, getCaptchaConfig } from "@/services/captcha";
import type { CaptchaConfig, CaptchaResponse } from "@/types/captcha";

export function useCaptcha() {
  const [config, setConfig] = useState<CaptchaConfig | null>(null);
  const [captchaData, setCaptchaData] = useState<CaptchaResponse | null>(null);
  const [loading, setLoading] = useState(false);

  // 加载配置
  const loadConfig = useCallback(async () => {
    try {
      const cfg = await getCaptchaConfig();
      setConfig(cfg);
    } catch (error) {
      console.error("加载验证码配置失败:", error);
    }
  }, []);

  // 加载验证码
  const loadCaptcha = useCallback(async () => {
    setLoading(true);
    try {
      const data = await getCaptcha();
      setCaptchaData(data);
    } catch (error) {
      console.error("加载验证码失败:", error);
      throw error;
    } finally {
      setLoading(false);
    }
  }, []);

  // 验证验证码（用于滑动验证码）
  const verifyCaptcha = useCallback(async (_input: string) => {
    // 对于滑动验证码，验证在组件内部完成
    // 这里只是占位，实际验证逻辑在组件中
    return true;
  }, []);

  // 初始化时加载配置
  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  return {
    config,
    captchaData,
    loading,
    loadConfig,
    loadCaptcha,
    verifyCaptcha,
    isEnabled: config?.enabled !== "disabled",
    captchaType: config?.enabled || "disabled",
  };
}
