/**
 * SM2 国密加密工具
 * 使用 sm-crypto 库实现完整的 SM2 加密/解密
 *
 * 安装: npm install sm-crypto
 * 文档: https://www.npmjs.com/package/sm-crypto
 */

import { get } from "@/lib/api";
import { hexToBase64, base64ToHex } from "./encoding";

// SM2 模块接口定义（与 sm-crypto 实际签名匹配）
interface SM2Module {
	doEncrypt: (msg: string, publicKey: string, cipherMode: 0 | 1) => string;
	doDecrypt: (encryptData: string, privateKey: string, cipherMode: 0 | 1) => string;
	doSignature: (msg: string, privateKey: string, hash?: boolean) => string;
	doVerifySignature: (msg: string, signHex: string, publicKey: string, hash?: boolean) => boolean;
	generateKeyPairHex: () => { publicKey: string; privateKey: string };
	verifyPublicKey: (publicKey: string) => boolean;
	comparePublicKeyHex: (publicKey1: string, publicKey2: string) => number;
}

// SM2 公钥缓存（十六进制格式）
let cachedPublicKeyHex: string = "";
// 防止较早发起、较晚返回的请求覆盖更新后的公钥缓存
let publicKeyCacheGeneration = 0;

// sm2 模块缓存
let sm2Module: SM2Module | null = null;

// 获取 sm2 对象（使用动态导入）
async function getSM2(): Promise<SM2Module> {
  if (sm2Module) {
    return sm2Module;
  }

  try {
    const smCrypto = await import("sm-crypto");
    sm2Module = smCrypto.sm2 as SM2Module;
    return sm2Module;
  } catch (error) {
    console.error("[SM2] 加载 sm-crypto 失败:", error);
    throw new Error("无法加载 sm-crypto 包，请确保已安装: npm install sm-crypto");
  }
}

/**
 * 从后端获取 SM2 公钥
 * 后端直接返回十六进制格式的公钥，供 sm-crypto 使用
 * @param forceRefresh 是否强制刷新缓存
 */
export async function fetchPublicKey(forceRefresh = false): Promise<string> {
  if (forceRefresh) {
    publicKeyCacheGeneration += 1;
    cachedPublicKeyHex = "";
  } else if (cachedPublicKeyHex) {
    return cachedPublicKeyHex;
  }

  const requestGeneration = publicKeyCacheGeneration;
  const result = await get<{ publicKey: string }>("/system/auth/public-key");

  if (result.code === 0 && result.data?.publicKey) {
    const publicKeyHex = result.data.publicKey;
    if (requestGeneration === publicKeyCacheGeneration) {
      cachedPublicKeyHex = publicKeyHex;
    }
    return publicKeyHex;
  }

  throw new Error("获取公钥失败: " + (result.message || "未知错误"));
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
export async function encryptWithSM2(password: string, publicKeyRaw: string): Promise<string> {
  if (!password) {
    return "";
  }

  const sm2 = await getSM2();
  const cipherText = sm2.doEncrypt(password, publicKeyRaw, 1);
  return hexToBase64(cipherText);
}

/**
 * 使用 SM2 私钥解密（用于测试验证）
 * @param cipherText 密文（Base64 编码）
 * @param privateKeyRaw SM2 私钥（十六进制字符串）
 * @returns 明文
 */
export async function decryptWithSM2(cipherText: string, privateKeyRaw: string): Promise<string> {
  if (!cipherText) {
    return "";
  }

  const cipherTextHex = base64ToHex(cipherText);
  const sm2 = await getSM2();

  return sm2.doDecrypt(cipherTextHex, privateKeyRaw, 1);
}

/**
 * 清除公钥缓存
 */
export function clearPublicKeyCache(): void {
  publicKeyCacheGeneration += 1;
  cachedPublicKeyHex = "";
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
    const encryptedPassword = await encryptWithSM2(password, publicKeyHex);

    return {
      username,
      password: encryptedPassword,
      encryptedPassword: true,
    };
  } catch (error) {
    // P1-S2: 生产环境绝不回退明文 — 密码明文传输是高危。
    // 仅开发环境允许回退以便调试(与 api.ts 请求加密回退逻辑一致)。
    if (import.meta.env.MODE === "production") {
      console.error("[SM2] 生产环境加密失败,拒绝明文回退:", error);
      throw new Error("密码加密失败,请检查网络或联系管理员");
    }
    console.warn("SM2 加密失败，回退到明文传输（仅开发环境）:", error);
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
export async function isSM2Available(): Promise<boolean> {
  try {
    await getSM2();
    return true;
  } catch {
    return false;
  }
}

/**
 * 生成 SM2 密钥对（用于测试）
 * 返回 { privateKey, publicKey }，都是十六进制字符串
 */
export async function generateSM2KeyPair(): Promise<{ privateKey: string; publicKey: string }> {
  const sm2 = await getSM2();
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
  const binaryString = hexToBytes(publicKeyHex);
  const base64 = btoa(binaryString);

  const lines: string[] = ["-----BEGIN PUBLIC KEY-----"];
  for (let i = 0; i < base64.length; i += 64) {
    lines.push(base64.substring(i, i + 64));
  }
  lines.push("-----END OF PUBLIC KEY-----");

  return lines.join("\n");
}

/**
 * 十六进制转字节字符串
 */
function hexToBytes(hexString: string): string {
  const bytes: string[] = [];
  for (let i = 0; i < hexString.length; i += 2) {
    bytes.push(String.fromCharCode(parseInt(hexString.substr(i, 2), 16)));
  }
  return bytes.join("");
}
