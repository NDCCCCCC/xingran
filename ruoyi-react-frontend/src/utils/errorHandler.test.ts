/**
 * ErrorHandler 简化测试
 */

import { describe, it, expect } from 'vitest';
import { HttpErrorType } from '@/utils/errorHandler';

describe('HttpErrorType Enum', () => {
  it('should have correct values for HTTP status codes', () => {
    expect(HttpErrorType.BAD_REQUEST).toBe(400);
    expect(HttpErrorType.UNAUTHORIZED).toBe(401);
    expect(HttpErrorType.FORBIDDEN).toBe(403);
    expect(HttpErrorType.NOT_FOUND).toBe(404);
    expect(HttpErrorType.INTERNAL_ERROR).toBe(500);
  });
});
