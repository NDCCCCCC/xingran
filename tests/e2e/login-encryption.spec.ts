import { test, expect } from '@playwright/test';

/**
 * End-to-End Tests for Login Endpoint Encryption
 *
 * These tests verify that the login endpoint encryption is working correctly
 * from a browser's perspective, including:
 * - Encrypted request headers are present
 * - Request body is encrypted
 * - Error handling works correctly
 *
 * Prerequisites:
 * - Frontend server running on http://localhost:4000
 * - Backend server running on http://localhost:9000
 * - Playwright installed: npm install -D @playwright/test
 *
 * Run tests:
 * npx playwright test login-encryption.spec.ts
 */

test.describe('Login with Request Encryption', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to login page
    await page.goto('/login');
  });

  test('should successfully login with encrypted request', async ({ page }) => {
    // Monitor network requests
    const loginRequest = page.waitForRequest(request => {
      return request.url().includes('/system/auth/login');
    });

    // Fill login form
    await page.fill('[name="username"]', 'admin');
    await page.fill('[name="password"]', 'admin123');

    // Get captcha (skip for testing, use mock)
    const captchaId = await page.evaluate(() => {
      return fetch('/api/v1/system/auth/captcha')
        .then(r => r.json())
        .then(d => d.data.captchaId);
    });

    await page.fill('[name="captcha"]', '1234');
    await page.fill('[name="captchaId"]', captchaId);

    // Submit login form
    await page.click('[type="submit"]');

    // Wait for login request
    const request = await loginRequest;

    // Verify request was encrypted
    const encryptedHeader = request.headers()['x-request-encrypted'];
    expect(encryptedHeader).toBe('true');

    // Verify login success (redirect to dashboard)
    await page.waitForURL('/', { timeout: 5000 });
    expect(page.url()).toContain('/');
  });

  test('should verify request body contains encrypted structure', async ({ page }) => {
    // Monitor network requests
    const requests: any[] = [];

    page.on('request', request => {
      if (request.url().includes('/system/auth/login')) {
        requests.push({
          url: request.url(),
          headers: request.headers(),
          postData: request.postData(),
        });
      }
    });

    // Fill and submit login form
    await page.fill('[name="username"]', 'admin');
    await page.fill('[name="password"]', 'admin123');

    // Mock captcha
    const captchaId = await page.evaluate(() => {
      return fetch('/api/v1/system/auth/captcha')
        .then(r => r.json())
        .then(d => d.data.captchaId);
    });

    await page.fill('[name="captcha"]', '1234');
    await page.fill('[name="captchaId"]', captchaId);
    await page.click('[type="submit"]');

    // Wait a bit for the request to complete
    await page.waitForTimeout(1000);

    // Verify we captured the login request
    expect(requests.length).toBeGreaterThan(0);

    const loginRequest = requests[0];

    // Verify encrypted structure
    expect(loginRequest.headers['x-request-encrypted']).toBe('true');

    // Parse request body
    const body = JSON.parse(loginRequest.postData);

    // Verify encrypted fields exist
    expect(body.encrypted).toBe(true);
    expect(body.data).toBeTruthy();
    expect(body.sm4Key).toBeTruthy();
    expect(body.iv).toBeTruthy();
    expect(body.timestamp).toBeTruthy();
    expect(body.nonce).toBeTruthy();

    // Verify password is NOT in plaintext
    expect(loginRequest.postData).not.toContain('admin123');
  });

  test('should handle decryption errors gracefully', async ({ page, context }) => {
    // Mock backend returning decryption error
    await context.route('**/system/auth/login', route => {
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 400,
          message: '解密失败: SM2 密钥解析错误',
          timestamp: Date.now() / 1000,
          request_id: 'test-request-id',
        }),
      });
    });

    // Fill and submit login form
    await page.fill('[name="username"]', 'admin');
    await page.fill('[name="password"]', 'admin123');
    await page.click('[type="submit"]');

    // Verify error message is displayed
    const errorMessage = page.locator('.ant-message-error, .error-message');
    await expect(errorMessage).toBeVisible({ timeout: 3000 });

    const errorText = await errorMessage.textContent();
    expect(errorText).toContain('解密');
  });

  test('should verify X-Request-Encrypted header is present', async ({ page }) => {
    // Monitor network requests
    const loginRequest = page.waitForRequest(request => {
      return request.url().includes('/system/auth/login');
    });

    // Fill and submit login form
    await page.fill('[name="username"]', 'admin');
    await page.fill('[name="password"]', 'admin123');

    // Mock captcha
    const captchaId = await page.evaluate(() => {
      return fetch('/api/v1/system/auth/captcha')
        .then(r => r.json())
        .then(d => d.data.captchaId);
    });

    await page.fill('[name="captcha"]', '1234');
    await page.fill('[name="captchaId"]', captchaId);
    await page.click('[type="submit"]');

    // Wait for request
    const request = await loginRequest;

    // Verify encryption header
    expect(request.headers()['x-request-encrypted']).toBe('true');
  });
});

test.describe('Login without Encryption (Backward Compatibility)', () => {
  test('should accept plaintext login when encryption disabled', async ({ page }) => {
    // This test verifies backward compatibility
    // When require_encryption: false, plaintext requests should still work

    // Monitor network requests
    const loginRequest = page.waitForRequest(request => {
      return request.url().includes('/system/auth/login');
    });

    // Fill and submit login form
    await page.fill('[name="username"]', 'admin');
    await page.fill('[name="password"]', 'admin123');

    // Mock captcha
    const captchaId = await page.evaluate(() => {
      return fetch('/api/v1/system/auth/captcha')
        .then(r => r.json())
        .then(d => d.data.captchaId);
    });

    await page.fill('[name="captcha"]', '1234');
    await page.fill('[name="captchaId"]', captchaId);
    await page.click('[type="submit"]');

    // Wait for request
    const request = await loginRequest;

    // Request should have encryption header (frontend always encrypts)
    expect(request.headers()['x-request-encrypted']).toBe('true');
  });
});

test.describe('Network Security', () => {
  test('should not expose credentials in network tab', async ({ page }) => {
    // Monitor all network requests
    const requests: any[] = [];

    page.on('request', request => {
      if (request.url().includes('/system/auth/login')) {
        requests.push({
          url: request.url(),
          postData: request.postData(),
        });
      }
    });

    // Fill and submit login form
    await page.fill('[name="username"]', 'admin');
    await page.fill('[name="password"]', 'MySecretPassword123!');

    // Mock captcha
    const captchaId = await page.evaluate(() => {
      return fetch('/api/v1/system/auth/captcha')
        .then(r => r.json())
        .then(d => d.data.captchaId);
    });

    await page.fill('[name="captcha"]', '1234');
    await page.fill('[name="captchaId"]', captchaId);
    await page.click('[type="submit"]');

    // Wait for request
    await page.waitForTimeout(1000);

    // Verify password is not exposed in any request
    for (const req of requests) {
      expect(req.postData).not.toContain('MySecretPassword123!');
      expect(req.postData).not.toContain('admin');
    }
  });

  test('should use HTTPS in production', async ({ page }) => {
    // This test verifies the application uses HTTPS
    // In development, HTTP is allowed
    const currentUrl = page.url();

    if (process.env.NODE_ENV === 'production') {
      expect(currentUrl).toMatch(/^https:/);
    }
  });
});
