/**
 * 统一的编码转换工具函数
 * 用于 SM2/SM4 加密中的十六进制、Base64、字节转换
 */

/**
 * 十六进制字符串转 Base64
 * @param hexString 十六进制字符串
 * @returns Base64 编码的字符串
 */
export function hexToBase64(hexString: string): string {
  // 确保十六进制字符串是偶数长度
  const paddedHex = hexString.length % 2 === 0
    ? hexString
    : "0" + hexString;

  // 转换为字节数组
  const bytes = new Uint8Array(paddedHex.length / 2);
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = parseInt(paddedHex.substr(i * 2, 2), 16);
  }

  // 转换为 Base64
  const binaryString = String.fromCharCode(...bytes);
  return btoa(binaryString);
}

/**
 * Base64 转十六进制字符串
 * @param base64String Base64 编码的字符串
 * @returns 十六进制字符串
 */
export function base64ToHex(base64String: string): string {
  const binaryString = atob(base64String);
  const hex: string[] = [];

  for (let i = 0; i < binaryString.length; i++) {
    const byte = binaryString.charCodeAt(i);
    hex.push(byte.toString(16).padStart(2, "0"));
  }

  return hex.join("");
}

/**
 * 字节字符串转十六进制字符串
 * @param byteString 原始字节字符串
 * @returns 十六进制字符串
 */
export function bytesToHex(byteString: string): string {
  const hex: string[] = [];
  for (let i = 0; i < byteString.length; i++) {
    const byte = byteString.charCodeAt(i);
    hex.push(byte.toString(16).padStart(2, "0"));
  }
  return hex.join("");
}

/**
 * 十六进制字符串转字节字符串
 * @param hexString 十六进制字符串
 * @returns 原始字节字符串
 */
export function hexToBytes(hexString: string): string {
  const bytes: string[] = [];
  for (let i = 0; i < hexString.length; i += 2) {
    bytes.push(String.fromCharCode(parseInt(hexString.substr(i, 2), 16)));
  }
  return bytes.join("");
}

/**
 * ArrayBuffer 转十六进制字符串
 * @param buffer ArrayBuffer
 * @returns 十六进制字符串
 */
export function arrayBufferToHex(buffer: ArrayBuffer): string {
  return Array.from(new Uint8Array(buffer))
    .map(b => b.toString(16).padStart(2, "0"))
    .join("");
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
 * 生成随机字节
 * @param length 字节长度
 * @returns Uint8Array
 */
export function generateRandomBytes(length: number): Uint8Array {
  const array = new Uint8Array(length);
  crypto.getRandomValues(array);
  return array;
}

/**
 * 生成随机十六进制字符串
 * @param length 字节长度（返回的十六进制字符串长度为 length * 2）
 * @returns 十六进制字符串
 */
export function generateRandomHex(length: number): string {
  const bytes = generateRandomBytes(length);
  return Array.from(bytes)
    .map(b => b.toString(16).padStart(2, "0"))
    .join("");
}
