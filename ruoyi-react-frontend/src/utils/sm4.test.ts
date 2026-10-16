/**
 * SM4 国密工具测试（D-08: 真实 sm-crypto 算法 + 确定性测试向量，加密层零 mock）
 *
 * 仅 mock @/lib/api 的 get（fetchSM4KeyForPassword 的网络依赖），
 * 加解密全部走真实 sm-crypto。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api", () => ({
  get: mockGet,
}));

import { sm4 as sm4Raw } from "sm-crypto";
import {
  encryptSM4CBC,
  decryptSM4CBC,
  encryptRequestBody,
  decryptRequestBody,
  encryptPasswordWithSM4,
  isSM4Available,
  generateSM4Key,
  generateIV,
  generateSM4KeyBytes,
  generateSessionKey,
  fetchSM4KeyForPassword,
  hexToBase64,
  base64ToHex,
} from "./sm4";

const KEY = "0123456789abcdeffedcba9876543210";
const IV = "abcdef98765432100123456789abcdef";

describe("SM4-CBC 加解密（真实算法）", () => {
  it("中文 + JSON 明文加解密往返", async () => {
    const plain = 'hello 国密 {"用户":"测试"}';
    const cipher = await encryptSM4CBC(plain, KEY, IV);
    expect(cipher).toMatch(/^[0-9a-f]+$/);
    expect(cipher).not.toContain(plain);
    expect(await decryptSM4CBC(cipher, KEY, IV)).toBe(plain);
  });

  it("空明文 / 空密文返回空字符串", async () => {
    expect(await encryptSM4CBC("", KEY, IV)).toBe("");
    expect(await decryptSM4CBC("", KEY, IV)).toBe("");
  });

  it("密文篡改后解密抛错（padding is invalid, T-83-02-02）", async () => {
    const cipher = await encryptSM4CBC("tamper-me-请篡改我", KEY, IV);
    const tampered = cipher.slice(0, -1) + (cipher.endsWith("0") ? "1" : "0");
    await expect(decryptSM4CBC(tampered, KEY, IV)).rejects.toThrow();
  });

  it("密钥错误时无法还原明文", async () => {
    const cipher = await encryptSM4CBC("right-key-secret", KEY, IV);
    const wrongKey = "ffffffffffffffffffffffffffffffff";
    await expect(decryptSM4CBC(cipher, wrongKey, IV)).rejects.toThrow();
  });

  it("同一明文相同 key/iv 密文确定（可用于固定样本回归）", async () => {
    const c1 = await encryptSM4CBC("deterministic", KEY, IV);
    const c2 = await encryptSM4CBC("deterministic", KEY, IV);
    expect(c1).toBe(c2);
  });
});

describe("SM4-ECB 密码加密（AD 域控登录）", () => {
  it("ECB 模式往返（用真实 sm-crypto 解密验证）", async () => {
    const password = "P@ssw0rd-密码";
    const cipherHex = await encryptPasswordWithSM4(password, KEY);
    expect(cipherHex).toMatch(/^[0-9a-f]+$/);
    expect(sm4Raw.decrypt(cipherHex, KEY, { mode: "ecb" })).toBe(password);
  });

  it("空密码返回空字符串", async () => {
    expect(await encryptPasswordWithSM4("", KEY)).toBe("");
  });
});

describe("请求体加解密", () => {
  it("JSON 对象加密/解密往返（嵌套结构）", async () => {
    const data = { username: "admin", 嵌套: { list: [1, 2, 3], ok: true } };
    const cipher = await encryptRequestBody(data, KEY, IV);
    expect(cipher).toMatch(/^[0-9a-f]+$/);
    expect(await decryptRequestBody(cipher, KEY, IV)).toEqual(data);
  });

  it("篡改的请求体密文解密抛错（JSON 解析失败或 padding 报错）", async () => {
    const cipher = await encryptRequestBody({ secret: "value" }, KEY, IV);
    const tampered = cipher.slice(0, -2) + (cipher.endsWith("00") ? "11" : "00");
    await expect(decryptRequestBody(tampered, KEY, IV)).rejects.toThrow();
  });
});

describe("可用性与密钥生成", () => {
  it("isSM4Available 返回 true（sm-crypto 已安装）", async () => {
    await expect(isSM4Available()).resolves.toBe(true);
  });

  it("generateSM4Key / generateIV / generateSessionKey 均为 32 位 hex 且随机", () => {
    for (const key of [generateSM4Key(), generateIV(), generateSessionKey()]) {
      expect(key).toMatch(/^[0-9a-f]{32}$/);
    }
    expect(generateSM4Key()).not.toBe(generateSM4Key());
    expect(generateIV()).not.toBe(generateIV());
  });

  it("generateSM4KeyBytes 返回 16 字符原始字节串", () => {
    const bytes = generateSM4KeyBytes();
    expect(bytes).toHaveLength(16);
    for (const ch of bytes) {
      expect(ch.charCodeAt(0)).toBeLessThanOrEqual(255);
    }
  });
});

describe("fetchSM4KeyForPassword（mock 网络）", () => {
  beforeEach(() => {
    mockGet.mockReset();
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("code=0 且返回 keyHex 时返回密钥", async () => {
    mockGet.mockResolvedValueOnce({
      code: 0,
      data: { encryptedKey: "enc", keyHex: KEY },
    });
    await expect(fetchSM4KeyForPassword()).resolves.toBe(KEY);
    expect(mockGet).toHaveBeenCalledWith("/system/auth/sm4-key");
  });

  it("code=0 但无 keyHex 时抛响应格式错误", async () => {
    mockGet.mockResolvedValueOnce({ code: 0, data: { encryptedKey: "enc" } });
    await expect(fetchSM4KeyForPassword()).rejects.toThrow("响应格式不正确");
  });

  it("code!=0 时抛业务错误消息", async () => {
    mockGet.mockResolvedValueOnce({ code: 1007, message: "token 无效" });
    await expect(fetchSM4KeyForPassword()).rejects.toThrow("获取SM4密钥失败: token 无效");
  });

  it("网络异常时记录日志并重抛", async () => {
    mockGet.mockRejectedValueOnce(new Error("network down"));
    await expect(fetchSM4KeyForPassword()).rejects.toThrow("network down");
    expect(console.error).toHaveBeenCalled();
  });
});

describe("向后兼容的编码函数重导出", () => {
  it("hexToBase64 / base64ToHex 从 sm4 模块可用", () => {
    expect(hexToBase64("00ff")).toBe("AP8=");
    expect(base64ToHex("AP8=")).toBe("00ff");
  });
});
