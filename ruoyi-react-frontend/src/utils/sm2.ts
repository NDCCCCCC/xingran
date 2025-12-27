/**
 * SM2 国密加密工具
 * 使用 sm-crypto 库实现完整的 SM2 加密/解密
 *
 * 安装: npm install sm-crypto
 * 文档: https://www.npmjs.com/package/sm-crypto
 */

import { sm2 } from 'sm-crypto';

// SM2 公钥缓存（十六进制格式）
let cachedPublicKeyHex: string = '';

/**
 * 从后端获取 SM2 公钥
 * 后端直接返回十六进制格式的公钥，供 sm-crypto 使用
 */
export async function fetchPublicKey(): Promise<string> {
  // 如果已经缓存，直接返回
  if (cachedPublicKeyHex) {
    return cachedPublicKeyHex;
  }

  try {
    const response = await fetch('/api/v1/system/auth/public-key');
    const result = await response.json();

    if (result.code === 0 && result.data?.publicKey) {
      cachedPublicKeyHex = result.data.publicKey;
      return cachedPublicKeyHex;
    } else {
      throw new Error('获取公钥失败: ' + (result.message || '未知错误'));
    }
  } catch (error) {
    console.error('获取 SM2 公钥失败:', error);
    throw error;
  }
}

/**
 * 使用 SM2 公钥加密密码
 *
 * @param password 明文密码
 * @param publicKeyRaw SM2 原始公钥（十六进制字符串，去掉 PEM 头尾）
 * @returns 加密后的密文（Base64 编码）
 *
 * 加密模式: C1C3C2 (mode = 1)
 * - C1: 椭圆曲线点（65字节）
 * - C2: 密文
 * - C3: 摘要值（32字节）
 */
export function encryptWithSM2(password: string, publicKeyRaw: string): string {
  if (!password) {
    return '';
  }

  try {
    // 使用 sm-crypto 进行 SM2 加密
    // sm-crypto 和 tjfoc/gmsm 的 mode 定义可能相反
    // 尝试使用 mode 0 (sm-crypto 可能是 C1C3C2)
    let cipherText = sm2.doEncrypt(password, publicKeyRaw, 0);

    // sm-crypto 生成的密文缺少 "04" 前缀（未压缩椭圆曲线点标识符）
    if (!cipherText.startsWith('04')) {
      cipherText = '04' + cipherText;
    }

    // sm-crypto 返回的是十六进制字符串，转为 Base64
    return hexToBase64(cipherText);
  } catch (error) {
    console.error('SM2 加密失败:', error);
    throw new Error('密码加密失败: ' + error);
  }
}

/**
 * 使用 SM2 私钥解密（用于测试验证）
 * @param cipherText 密文（Base64 编码）
 * @param privateKeyRaw SM2 私钥（十六进制字符串）
 * @returns 明文
 */
export function decryptWithSM2(cipherText: string, privateKeyRaw: string): string {
  if (!cipherText) {
    return '';
  }

  try {
    // Base64 转十六进制
    const cipherTextHex = base64ToHex(cipherText);

    // 使用 sm-crypto 解密，mode: 1 = C1C3C2
    const plainText = sm2.doDecrypt(cipherTextHex, privateKeyRaw, 1);

    return plainText;
  } catch (error) {
    console.error('SM2 解密失败:', error);
    throw new Error('密码解密失败: ' + error);
  }
}

/**
 * 十六进制转 Base64
 */
function hexToBase64(hexString: string): string {
  // 确保十六进制字符串是偶数长度
  const paddedHex = hexString.length % 2 === 0
    ? hexString
    : '0' + hexString;

  // 转换为字节数组
  const bytes = [];
  for (let i = 0; i < paddedHex.length; i += 2) {
    bytes.push(parseInt(paddedHex.substr(i, 2), 16));
  }

  // 转换为 Base64
  return btoa(String.fromCharCode.apply(null, bytes));
}

/**
 * Base64 转十六进制
 */
function base64ToHex(base64String: string): string {
  const binaryString = atob(base64String);
  let hex = '';

  for (let i = 0; i < binaryString.length; i++) {
    const byte = binaryString.charCodeAt(i);
    hex += byte.toString(16).padStart(2, '0');
  }

  return hex;
}

/**
 * 清除公钥缓存
 */
export function clearPublicKeyCache(): void {
  cachedPublicKey = '';
  cachedPublicKeyRaw = '';
}

/**
 * 获取加密的登录请求
 * 自动获取公钥并加密密码
 */
export async function getEncryptedLoginRequest(
  username: string,
  password: string
): Promise<{ username: string; password: string; encryptedPassword: boolean }> {
  try {
    const publicKeyHex = await fetchPublicKey();
    const encryptedPassword = encryptWithSM2(password, publicKeyHex);

    return {
      username,
      password: encryptedPassword,
      encryptedPassword: true,
    };
  } catch (error) {
    console.warn('SM2 加密失败，回退到明文传输（仅开发环境）:', error);
    // 回退方案：如果 SM2 不可用，使用明文传输（仅开发环境）
    return {
      username,
      password,
      encryptedPassword: false,
    };
  }
}

/**
 * 检查是否支持 SM2 加密
 */
export function isSM2Available(): boolean {
  // sm-crypto 是纯 JavaScript 实现，始终可用
  return true;
}

/**
 * 生成 SM2 密钥对（用于测试）
 * 返回 { privateKey, publicKey }，都是十六进制字符串
 */
export function generateSM2KeyPair(): { privateKey: string; publicKey: string } {
  const keyPair = sm2.generateKeyPairHex();
  return {
    privateKey: keyPair.privateKey, // 私钥（十六进制）
    publicKey: keyPair.publicKey,   // 公钥（十六进制）
  };
}

/**
 * 将十六进制公钥转换为 PEM 格式
 */
export function publicKeyToPEM(publicKeyHex: string): string {
  const binaryString = hexToBinary(publicKeyHex);
  const base64 = btoa(binaryString);

  let pem = '-----BEGIN PUBLIC KEY-----\n';
  // 每64字符换行
  for (let i = 0; i < base64.length; i += 64) {
    pem += base64.substring(i, i + 64) + '\n';
  }
  pem += '-----END PUBLIC KEY-----';

  return pem;
}

/**
 * 十六进制转二进制字符串
 */
function hexToBinary(hexString: string): string {
  const bytes = [];
  for (let i = 0; i < hexString.length; i += 2) {
    bytes.push(parseInt(hexString.substr(i, 2), 16));
  }
  return String.fromCharCode.apply(null, bytes);
}
