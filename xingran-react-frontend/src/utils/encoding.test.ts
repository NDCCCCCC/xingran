import { describe, expect, it } from "vitest";
import {
  hexToBase64,
  base64ToHex,
  bytesToHex,
  hexToBytes,
  arrayBufferToHex,
  hexToArrayBuffer,
  generateRandomBytes,
  generateRandomHex,
} from "./encoding";

describe("encoding 工具（D-08 真实算法 + 确定性向量）", () => {
  describe("hexToBase64 / base64ToHex", () => {
    it("已知向量: 00ff ↔ AP8=", () => {
      expect(hexToBase64("00ff")).toBe("AP8=");
      expect(base64ToHex("AP8=")).toBe("00ff");
    });

    it("多字节十六进制往返", () => {
      const hex = "0123456789abcdeffedcba9876543210";
      expect(base64ToHex(hexToBase64(hex))).toBe(hex);
    });

    it("奇数长度 hex 自动前补 0（abc → 0abc → Crw=）", () => {
      expect(hexToBase64("abc")).toBe("Crw=");
      expect(base64ToHex("Crw=")).toBe("0abc");
    });

    it("空字符串往返为空", () => {
      expect(hexToBase64("")).toBe("");
      expect(base64ToHex("")).toBe("");
    });

    it("非 Base64 输入时 atob 抛错", () => {
      expect(() => base64ToHex("###not-base64###")).toThrow();
    });
  });

  describe("bytesToHex / hexToBytes", () => {
    it("全字节域（0x00-0xff）往返", () => {
      const byteString = Array.from({ length: 256 }, (_, i) => String.fromCharCode(i)).join("");
      const hex = bytesToHex(byteString);
      expect(hex).toHaveLength(512);
      expect(hexToBytes(hex)).toBe(byteString);
    });

    it("已知向量: 字节串 0x00 0xff ↔ 00ff", () => {
      const twoBytes = String.fromCharCode(0x00, 0xff);
      expect(bytesToHex(twoBytes)).toBe("00ff");
      expect(hexToBytes("00ff")).toBe(twoBytes);
    });

    it("空字符串往返为空", () => {
      expect(bytesToHex("")).toBe("");
      expect(hexToBytes("")).toBe("");
    });
  });

  describe("arrayBufferToHex / hexToArrayBuffer", () => {
    it("往返保持字节一致", () => {
      const hex = "deadbeefcafebabe";
      const buffer = hexToArrayBuffer(hex);
      expect(buffer).toBeInstanceOf(ArrayBuffer);
      expect(arrayBufferToHex(buffer)).toBe(hex);
    });

    it("空输入往返为空", () => {
      expect(arrayBufferToHex(new ArrayBuffer(0))).toBe("");
      expect(hexToArrayBuffer("")).toBeInstanceOf(ArrayBuffer);
      expect(hexToArrayBuffer("").byteLength).toBe(0);
    });
  });

  describe("generateRandomBytes / generateRandomHex", () => {
    it("generateRandomHex 返回 length*2 个十六进制字符", () => {
      const hex = generateRandomHex(16);
      expect(hex).toHaveLength(32);
      expect(hex).toMatch(/^[0-9a-f]{32}$/);
    });

    it("两次生成的随机 hex 不同", () => {
      expect(generateRandomHex(16)).not.toBe(generateRandomHex(16));
    });

    it("generateRandomBytes 返回指定长度的 Uint8Array", () => {
      const bytes = generateRandomBytes(8);
      expect(bytes).toBeInstanceOf(Uint8Array);
      expect(bytes).toHaveLength(8);
    });
  });
});
