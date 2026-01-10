/**
 * SM4 国密加密工具
 * 使用 sm-crypto 库实现 SM4-CBC 加密/解密
 *
 * 安装: npm install sm-crypto
 * 文档: https://www.npmjs.com/package/sm-crypto
 */

// sm4 模块缓存
let sm4Module: any = null;

// 获取 sm4 对象（使用动态导入）
async function getSM4() {
  if (sm4Module) {
    return sm4Module;
  }

  try {
    // 动态导入 sm-crypto 包
    const smCrypto = await import('sm-crypto');
    sm4Module = smCrypto.sm4;
    console.log('[SM4] sm-crypto 包加载成功');
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
  const key = new Uint8Array(16);
  crypto.getRandomValues(key);
  return Array.from(key)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * 生成随机 IV（16字节，32个十六进制字符）
 * SM4-CBC 模式需要每次使用不同的 IV
 * @returns 32字符的十六进制字符串
 */
export function generateIV(): string {
  const iv = new Uint8Array(16);
  crypto.getRandomValues(iv);
  return Array.from(iv)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
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

  try {
    const sm4 = await getSM4();

    // sm-crypto 的 SM4-CBC 加密
    const cipherHex = sm4.encrypt(plaintext, keyHex, {
      mode: 'cbc',
      iv: ivHex,
    });
    return cipherHex;
  } catch (error) {
    console.error('SM4-CBC 加密失败:', error);
    throw new Error('SM4 加密失败: ' + error);
  }
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

  try {
    const sm4 = await getSM4();

    // sm-crypto 的 SM4-CBC 解密
    const plaintext = sm4.decrypt(cipherHex, keyHex, {
      mode: 'cbc',
      iv: ivHex,
    });
    return plaintext;
  } catch (error) {
    console.error('SM4-CBC 解密失败:', error);
    throw new Error('SM4 解密失败: ' + error);
  }
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
export async function encryptRequestBody(data: any, keyHex: string, ivHex: string): Promise<string> {
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
export async function decryptRequestBody(cipherHex: string, keyHex: string, ivHex: string): Promise<any> {
  const jsonStr = await decryptSM4CBC(cipherHex, keyHex, ivHex);
  return JSON.parse(jsonStr);
}

/**
 * 十六进制字符串转 Base64
 * @param hexString 十六进制字符串
 * @returns Base64 编码的字符串
 */
export function hexToBase64(hexString: string): string {
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
  return btoa(String.fromCharCode(...bytes));
}

/**
 * Base64 转十六进制字符串
 * @param base64String Base64 编码的字符串
 * @returns 十六进制字符串
 */
export function base64ToHex(base64String: string): string {
  const binaryString = atob(base64String);
  let hex = '';

  for (let i = 0; i < binaryString.length; i++) {
    const byte = binaryString.charCodeAt(i);
    hex += byte.toString(16).padStart(2, '0');
  }

  return hex;
}

/**
 * 字节字符串转十六进制字符串
 * 用于将原始字节字符串（如 String.fromCharCode(...bytes)）转换为十六进制
 * @param byteString 原始字节字符串
 * @returns 十六进制字符串
 */
export function bytesToHex(byteString: string): string {
  let hex = '';
  for (let i = 0; i < byteString.length; i++) {
    const byte = byteString.charCodeAt(i);
    hex += byte.toString(16).padStart(2, '0');
  }
  return hex;
}

/**
 * ArrayBuffer 转十六进制字符串
 * @param buffer ArrayBuffer
 * @returns 十六进制字符串
 */
export function arrayBufferToHex(buffer: ArrayBuffer): string {
  return Array.from(new Uint8Array(buffer))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * 十六进制字符串转 ArrayBuffer
 * @param hexString 十六进制字符串
 * @returns ArrayBuffer
 */
export function hexToArrayBuffer(hexString: string): ArrayBuffer {
  const bytes = new Uint8Array(hexString.length / 2);
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(hexString.substr(i * 2, 2), 16);
  }
  return bytes.buffer;
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
