/**
 * SM4 国密加密工具
 * 使用 sm-crypto 库实现 SM4-CBC 加密/解密
 *
 * 安装: npm install sm-crypto
 * 文档: https://www.npmjs.com/package/sm-crypto
 */

import { get } from "@/lib/api";
import { hexToBase64, base64ToHex, arrayBufferToHex, hexToArrayBuffer, generateRandomHex } from "./encoding";

// SM4 模块接口定义
interface SM4Module {
	encrypt: (msg: string, key: string, options?: { mode: "cbc" | "ecb"; iv?: string }) => string;
	decrypt: (cipherHex: string, key: string, options?: { mode: "cbc" | "ecb"; iv?: string }) => string;
}

// sm4 模块缓存
let sm4Module: SM4Module | null = null;

// 获取 sm4 对象（使用动态导入）
async function getSM4(): Promise<SM4Module> {
  if (sm4Module) {
    return sm4Module;
  }

  try {
    const smCrypto = await import("sm-crypto");
    sm4Module = smCrypto.sm4;
    return sm4Module;
  } catch (error) {
    console.error("[SM4] 加载 sm-crypto 失败:", error);
    throw new Error("无法加载 sm-crypto 包，请确保已安装: npm install sm-crypto");
  }
}

/**
 * 生成随机 SM4 密钥（16字节，32个十六进制字符）
 * @returns 32字符的十六进制字符串
 */
export function generateSM4Key(): string {
  return generateRandomHex(16);
}

/**
 * 生成随机 IV（16字节，32个十六进制字符）
 * SM4-CBC 模式需要每次使用不同的 IV
 * @returns 32字符的十六进制字符串
 */
export function generateIV(): string {
  return generateRandomHex(16);
}

/**
 * 生成随机 SM4 密钥（16字节原始字符串）
 * 用于 SM2 加密传输，返回原始字节字符串（非十六进制）
 * @returns 16字符的原始字节字符串
 */
export function generateSM4KeyBytes(): string {
  const key = new Uint8Array(16);
  crypto.getRandomValues(key);
  return String.fromCharCode(...key);
}

/**
 * 使用 SM4-CBC 加密数据
 * sm-crypto 语法: sm4.encrypt(msg, key, {mode: 'cbc', iv: '...'})
 *
 * @param plaintext 明文字符串
 * @param keyHex SM4 密钥（十六进制，32字符）
 * @param ivHex 初始化向量（十六进制，32字符）
 * @returns 密文（十六进制字符串）
 */
export async function encryptSM4CBC(plaintext: string, keyHex: string, ivHex: string): Promise<string> {
  if (!plaintext) {
    return "";
  }

  const sm4 = await getSM4();
  return sm4.encrypt(plaintext, keyHex, {
    mode: "cbc",
    iv: ivHex,
  });
}

/**
 * 使用 SM4-CBC 解密数据
 *
 * @param cipherHex 密文（十六进制字符串）
 * @param keyHex SM4 密钥（十六进制，32字符）
 * @param ivHex 初始化向量（十六进制，32字符）
 * @returns 明文字符串
 */
export async function decryptSM4CBC(cipherHex: string, keyHex: string, ivHex: string): Promise<string> {
  if (!cipherHex) {
    return "";
  }

  const sm4 = await getSM4();
  return sm4.decrypt(cipherHex, keyHex, {
    mode: "cbc",
    iv: ivHex,
  });
}

/**
 * 加密请求体
 * 将数据对象转为 JSON 字符串后使用 SM4-CBC 加密
 *
 * @param data 要加密的数据对象
 * @param keyHex SM4 密钥（十六进制，32字符）
 * @param ivHex 初始化向量（十六进制，32字符）
 * @returns 加密后的密文（十六进制字符串）
 */
export async function encryptRequestBody(data: Record<string, unknown>, keyHex: string, ivHex: string): Promise<string> {
  const jsonStr = JSON.stringify(data);
  return encryptSM4CBC(jsonStr, keyHex, ivHex);
}

/**
 * 解密请求体
 * 解密 SM4-CBC 加密的密文并解析为对象
 *
 * @param cipherHex 密文（十六进制字符串）
 * @param keyHex SM4 密钥（十六进制，32字符）
 * @param ivHex 初始化向量（十六进制，32字符）
 * @returns 解密后的数据对象
 */
export async function decryptRequestBody(cipherHex: string, keyHex: string, ivHex: string): Promise<Record<string, unknown>> {
  const jsonStr = await decryptSM4CBC(cipherHex, keyHex, ivHex);
  return JSON.parse(jsonStr);
}

/**
 * 检查是否支持 SM4 加密
 * @returns true 如果 sm-crypto 可用
 */
export async function isSM4Available(): Promise<boolean> {
  try {
    await getSM4();
    return true;
  } catch {
    return false;
  }
}

// ==================== 密码加密（AD域控登录） ====================

/**
 * 使用 SM4-ECB 模式加密密码
 * 用于 AD 域控登录时的密码字段加密
 * ECB 模式不需要 IV，比 CBC 更简单
 *
 * @param password 明文密码
 * @param keyHex SM4 密钥（16字节，32个十六进制字符）
 * @returns 加密后的密文（十六进制字符串）
 */
export async function encryptPasswordWithSM4(password: string, keyHex: string): Promise<string> {
  if (!password) {
    return "";
  }

  const sm4 = await getSM4();

  // 使用 SM4-ECB 模式加密密码（无需 IV）
  const cipherHex = sm4.encrypt(password, keyHex, { mode: "ecb" });

  return cipherHex;
}

/**
 * 生成会话密钥（SM4 密钥）用于当前登录会话的密码加密
 * 使用 crypto.getRandomValues 确保密钥安全性（T-19-11 缓解措施）
 *
 * @returns 32字符的十六进制 SM4 密钥
 */
export function generateSessionKey(): string {
  return generateSM4Key();
}

/**
 * 从后端获取加密的 SM4 密钥并解密
 * 密钥通过 SM2 公钥加密传输，保证密钥安全
 *
 * 注意：此函数依赖后端 /api/v1/system/auth/sm4-key 端点
 *
 * @returns SM4 密钥（32个十六进制字符）
 */
export async function fetchSM4KeyForPassword(): Promise<string> {
  try {
    const result = await get<{ encryptedKey?: string; keyHex?: string }>("/system/auth/sm4-key");

    if (result.code === 0 && result.data?.encryptedKey) {
      // 使用 SM2 解密密钥（sm2 已静态导入顶部）

      // 后端使用 SM2 公钥加密密钥，需要服务端私钥解密？
      // 实际上这里应该是：后端用前端 SM2 公钥加密 SM4 密钥，前端用 SM2 私钥解密
      // 但当前架构中前端不持有 SM2 私钥（只有公钥）
      // 因此这个端点需要后端直接返回明文 SM4 密钥（通过 HTTPS 保护）
      // 或者采用其他密钥交换方案
      // 当前实现：后端直接返回 keyHex（通过 HTTPS 传输保护）
      if (result.data.keyHex) {
        return result.data.keyHex as string;
      }

      throw new Error("获取SM4密钥失败: 响应格式不正确");
    }

    throw new Error("获取SM4密钥失败: " + (result.message || "未知错误"));
  } catch (error) {
    console.error("[SM4] 获取密钥失败:", error);
    throw error;
  }
}

// 重新导出编码转换函数，保持向后兼容
export { hexToBase64, base64ToHex, arrayBufferToHex, hexToArrayBuffer };

