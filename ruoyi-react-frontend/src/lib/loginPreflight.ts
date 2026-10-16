import { refreshEncryptionConfig } from "@/lib/api";
import { getCaptchaConfig } from "@/services/captcha";
import type { CaptchaEnabled } from "@/types/captcha";
import { clearPublicKeyCache, fetchPublicKey } from "@/utils/sm2";

const PREFLIGHT_TIMEOUT_MS = 5000;
const PREFLIGHT_FRIENDLY_ERROR = "登录安全配置已过期，自动更新失败，请检查网络后重试";

export type LoginPreflightResult =
  { ok: true; captchaEnabled: CaptchaEnabled } | { ok: false; friendlyMessage: string };

function withTimeout<T>(promise: Promise<T>, timeoutMs: number): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timer = window.setTimeout(() => {
      reject(new Error(`登录配置刷新超过 ${timeoutMs}ms`));
    }, timeoutMs);

    promise.then(
      (value) => {
        window.clearTimeout(timer);
        resolve(value);
      },
      (error) => {
        window.clearTimeout(timer);
        reject(error);
      }
    );
  });
}

async function refreshPublicKey(): Promise<void> {
  clearPublicKeyCache();
  await fetchPublicKey(true);
}

/**
 * 登录提交前同步所有可能在页面停留期间失效的公开安全配置。
 * 三项请求互不依赖，并发执行以保证总等待时间不超过 5 秒。
 */
export async function submitLoginPreflight(): Promise<LoginPreflightResult> {
  const [encryptionResult, publicKeyResult, captchaResult] = await Promise.allSettled([
    withTimeout(refreshEncryptionConfig(), PREFLIGHT_TIMEOUT_MS),
    withTimeout(refreshPublicKey(), PREFLIGHT_TIMEOUT_MS),
    withTimeout(getCaptchaConfig(), PREFLIGHT_TIMEOUT_MS),
  ]);

  const encryptionFailed =
    encryptionResult.status === "rejected" || encryptionResult.value !== true;
  const publicKeyFailed = publicKeyResult.status === "rejected";
  const captchaFailed = captchaResult.status === "rejected";

  if (encryptionFailed || publicKeyFailed || captchaFailed) {
    console.error("[LoginPreflight] 登录安全配置刷新失败", {
      encryption: encryptionFailed ? "failed" : "ok",
      publicKey: publicKeyFailed ? "failed" : "ok",
      captcha: captchaFailed ? "failed" : "ok",
    });
    return { ok: false, friendlyMessage: PREFLIGHT_FRIENDLY_ERROR };
  }

  return {
    ok: true,
    captchaEnabled: captchaResult.value.enabled,
  };
}
