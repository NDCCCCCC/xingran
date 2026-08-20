/**
 * sm-crypto 类型声明 (ambient 形式)
 *
 * 注: **不能** 在这个文件里写 `import` / `export {}` — 一旦如此,
 * TypeScript 会把它当成"模块文件"而不是"全局脚本",里面的
 * `declare module "sm-crypto"` 块变成 augmentation (增强) 而非 new
 * declaration,TS7016 ("Could not find a declaration file for module")
 * 不会消除。本文件必须保持纯 ambient 状态,所有 import/export 都去掉。
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
    decrypt: (
      encryptedData: string,
      key: string,
      options?: { mode?: "ecb" | "cbc"; iv?: string }
    ) => string;
  };

  // PR #3: TS7 严格解析下还要求 default export 形式可用
  // (await import("sm-crypto") 的结果作为对象解构时)
  const _default: { sm2: typeof sm2; sm4: typeof sm4 };
  export default _default;
}
