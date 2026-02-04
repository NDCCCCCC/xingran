/**
 * SM4 国密加密工具
 * 使用 sm-crypto 库实现 SM4-CBC 加密/解密
 *
 * 安装: npm install sm-crypto
 * 文档: https://www.npmjs.com/package/sm-crypto
 */

import { hexToBase64, base64ToHex, arrayBufferToHex, hexToArrayBuffer, generateRandomHex } from './encoding';

// SM4 模块接口定义
interface SM4Module {
	encrypt: (msg: string, key: string, options?: { mode: 'cbc' | 'ecb'; iv?: string }) => string;
	decrypt: (cipherHex: string, key: string, options?: { mode: 'cbc' | 'ecb'; iv?: string }) => string;
}

// sm4 模块缓存
let sm4Module: SM4Module | null = null;

// 获取 sm4 对象（使用动态导入）
async function getSM4(): Promise<SM4Module> {
  if (sm4Module) {
    return sm4Module;
  }

  try {
    const smCrypto = await import('sm-crypto');
    sm4Module = smCrypto.sm4;
    return sm4Module;
  } catch (error) {
    console.error('[SM4] 加载 sm-crypto 失败:', error);
    throw new Error('无法加载 sm-crypto 包，请确保已安装: npm install sm-crypto');
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
    return '';
  }

  const sm4 = await getSM4();
  return sm4.encrypt(plaintext, keyHex, {
    mode: 'cbc',
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
    return '';
  }

  const sm4 = await getSM4();
  return sm4.decrypt(cipherHex, keyHex, {
    mode: 'cbc',
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

// 重新导出编码转换函数，保持向后兼容
export { hexToBase64, base64ToHex, arrayBufferToHex, hexToArrayBuffer };

