/**
 * SM4 加密工具测试
 */

import { describe, it, expect } from 'vitest';
import { generateSM4Key, generateIV, hexToBase64, base64ToHex } from '@/utils/sm4';

describe('SM4 Encryption Utils', () => {
  describe('generateSM4Key', () => {
    it('should generate a 32 character hex string', () => {
      const key = generateSM4Key();
      expect(key).toHaveLength(32);
    });

    it('should generate a valid hexadecimal string', () => {
      const key = generateSM4Key();
      expect(key).toMatch(/^[0-9a-f]{32}$/i);
    });

    it('should generate different keys each time', () => {
      const key1 = generateSM4Key();
      const key2 = generateSM4Key();
      expect(key1).not.toBe(key2);
    });
  });

  describe('generateIV', () => {
    it('should generate a 32 character hex string', () => {
      const iv = generateIV();
      expect(iv).toHaveLength(32);
    });

    it('should generate a valid hexadecimal string', () => {
      const iv = generateIV();
      expect(iv).toMatch(/^[0-9a-f]{32}$/i);
    });

    it('should generate different IVs each time', () => {
      const iv1 = generateIV();
      const iv2 = generateIV();
      expect(iv1).not.toBe(iv2);
    });
  });

  describe('hexToBase64', () => {
    it('should convert hex string to base64', () => {
      const hex = '48656c6c6f20576f726c64'; // "Hello World" in hex
      const base64 = hexToBase64(hex);
      expect(base64).toBe('SGVsbG8gV29ybGQ=');
    });

    it('should handle empty string', () => {
      const base64 = hexToBase64('');
      expect(base64).toBe('');
    });

    it('should handle odd length hex string', () => {
      const hex = '48656c6c6f'; // "Hello" in hex
      const base64 = hexToBase64(hex);
      expect(base64).toBe('SGVsbG8=');
    });
  });

  describe('base64ToHex', () => {
    it('should convert base64 string to hex', () => {
      const base64 = 'SGVsbG8gV29ybGQ='; // "Hello World" in base64
      const hex = base64ToHex(base64);
      expect(hex).toBe('48656c6c6f20576f726c64');
    });

    it('should handle empty string', () => {
      const hex = base64ToHex('');
      expect(hex).toBe('');
    });

    it('should be reversible with hexToBase64', () => {
      const originalHex = '48656c6c6f20576f726c64';
      const base64 = hexToBase64(originalHex);
      const recoveredHex = base64ToHex(base64);
      expect(recoveredHex).toBe(originalHex);
    });
  });
});
