/**
 * 登录请求加密 —— Vitest 冒烟（源码契约锁）。
 *
 * 来源：原 `tests/e2e/login-encryption.spec.ts` 是一个
 * `import '@playwright/test'`、依赖未安装、CI 从不执行的孤儿 spec（2026-11-12 移除）。
 * 为不丢失它想守护的价值，把核心断言迁移到**真实被 CI 执行**的位置：
 * `.github/workflows/ci.yml` 在 `xingran-react-frontend/` 下跑
 * `npm run test:coverage -- --shard=N/4`，会扫描到本文件。
 *
 * 性质说明（诚实标注，勿夸大）：本测试是**静态契约锁**——读取
 * `src/lib/api.ts` 请求拦截器源码，锁定加密头、加密体字段结构、SM2/SM4 引导、
 * 方法/黑名单判定与「解密失败」重放分支的关键字面量；它不替代浏览器级 e2e，
 * 但能防止重构时静默删掉 `X-Request-Encrypted` / 加密体字段 / 防重放字段，
 * 且无需 Playwright、无需起服务、零新依赖、稳定可重复。
 *
 * 真正的浏览器级登录加密验证的接线条件见 `tests/e2e/README.md`。
 */
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import path from "node:path";

const API_SOURCE = readFileSync(path.resolve(process.cwd(), "src", "lib", "api.ts"), "utf8");

describe("login 请求加密契约（原孤儿 e2e spec 的价值迁移）", () => {
  it("加密开关只作用于 POST/PUT/PATCH 且受黑名单约束", () => {
    // shouldEncryptRequest 的方法白名单
    expect(API_SOURCE).toMatch(/\["POST",\s*"PUT",\s*"PATCH"\]/);
    // 黑名单含公钥/验证码端点（原 spec 依赖 captcha 但 captcha 不该被加密）
    expect(API_SOURCE).toContain('"/system/auth/public-key"');
    expect(API_SOURCE).toContain('"/system/auth/captcha"');
    expect(API_SOURCE).toMatch(/ENCRYPTION_BLACKLIST\.some/);
  });

  it("请求头带 X-Request-Encrypted: true", () => {
    expect(API_SOURCE).toContain('config.headers.set("X-Request-Encrypted", "true")');
  });

  it("加密体为 { encrypted, data, sm4Key, iv, timestamp, nonce } 六字段结构", () => {
    expect(API_SOURCE).toContain("encrypted: true");
    expect(API_SOURCE).toContain("data: hexToBase64(encryptedDataHex)");
    expect(API_SOURCE).toContain("sm4Key: encryptedSM4Key");
    expect(API_SOURCE).toContain("iv: hexToBase64(ivHex)");
    expect(API_SOURCE).toContain("timestamp: Math.floor(Date.now() / 1000)");
    expect(API_SOURCE).toContain("nonce: generateNonce()");
  });

  it("SM4 加密请求体 + SM2 加密会话密钥，且保留明文用于解密失败重放", () => {
    expect(API_SOURCE).toContain("encryptRequestBody(config.data, sm4KeyHex, ivHex)");
    expect(API_SOURCE).toContain("encryptWithSM2(sm4KeyHex, publicKey)");
    // 明文备份（后端换公钥后重放需恢复明文）
    expect(API_SOURCE).toContain("config.__originalPlainData = config.data");
  });

  it("后端返回「解密」错误时清理公钥缓存并重放（原 spec 的报错用例契约）", () => {
    expect(API_SOURCE).toMatch(/message\.includes\("解密"\)/);
  });

  it("加密配置引导走 rawAxios 且失败时 fail-secure 默认开启", () => {
    expect(API_SOURCE).toContain('"/system/auth/encryption-config"');
    expect(API_SOURCE).toContain("ENABLE_REQUEST_ENCRYPTION = true"); // fail secure
  });
});
