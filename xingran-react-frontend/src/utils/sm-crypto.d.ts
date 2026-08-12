/**
 * sm-crypto 类型声明
 * 解决 CommonJS 到 ES 模块的导入问题
 */

declare module "sm-crypto" {
  export const sm2: {
    doEncrypt(msg: string, publicKey: string, cipherMode: number): string;
    doDecrypt: (encryptedData: string, privateKey: string, cipherMode: number) => string;
    generateKeyPairHex: () => { privateKey: string; publicKey: string };
    compressPublicKeyHex: (publicKey: string) => string;
    comparePublicKeyHex: (publicKey1: string, publicKey2: string) => number;
    doSignature: (data: string, privateKey: string, hash: boolean) => string;
    doVerifySignature: (data: string, sign: string, publicKey: string, hash: boolean) => boolean;
    getPublicKeyFromPrivateKey: (privateKey: string) => string;
    getPoint: () => any;
    verifyPublicKey: (publicKey: string) => boolean;
  };

  export const sm3: {
    (msg: string): string;
  };

  export const sm4: {
    encrypt: (msg: string, key: string, options?: { mode?: "ecb" | "cbc"; iv?: string }) => string;
    decrypt: (encryptedData: string, key: string, options?: { mode?: "ecb" | "cbc"; iv?: string }) => string;
  };

  const sm2: {
    doEncrypt(msg: string, publicKey: string, cipherMode: number): string;
    doDecrypt: (encryptedData: string, privateKey: string, cipherMode: number) => string;
    generateKeyPairHex: () => { privateKey: string; publicKey: string };
    compressPublicKeyHex: (publicKey: string) => string;
    comparePublicKeyHex: (publicKey1: string, publicKey2: string) => number;
    doSignature: (data: string, privateKey: string, hash: boolean) => string;
    doVerifySignature: (data: string, sign: string, publicKey: string, hash: boolean) => boolean;
    getPublicKeyFromPrivateKey: (privateKey: string) => string;
    getPoint: () => any;
    verifyPublicKey: (publicKey: string) => boolean;
  };

  const sm3: {
    (msg: string): string;
  };

  const sm4: {
    encrypt: (msg: string, key: string, options?: { mode?: "ecb" | "cbc"; iv?: string }) => string;
    decrypt: (encryptedData: string, key: string, options?: { mode?: "ecb" | "cbc"; iv?: string }) => string;
  };

  export default sm2;
}

export {};
