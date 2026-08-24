/**
 * security.ts 安全工具测试 (Phase 83-03)
 *
 * 覆盖:XSS 检测(containsXSS)与转义(escapeHtml)、对象递归清理(sanitizeObject)、
 * 数据完整性哈希(generateHash/verifyDataIntegrity)、SecureStorage 存取与篡改防护、
 * URL 同源校验(isSecureUrl)、CSP 配置、安全随机串、secureLog 环境分支。
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  SecureStorage,
  containsXSS,
  escapeHtml,
  generateHash,
  generateSecureRandom,
  getCSPConfig,
  isSecureUrl,
  sanitizeObject,
  secureLog,
  verifyDataIntegrity,
} from "./security";

describe("containsXSS", () => {
  it("识别各类 XSS 载荷", () => {
    expect(containsXSS("<script>alert(1)</script>")).toBe(true);
    expect(containsXSS("javascript:void(0)")).toBe(true);
    expect(containsXSS("onclick=alert(1)")).toBe(true);
    expect(containsXSS("onload = evil()")).toBe(true);
    expect(containsXSS("<iframe src=x>")).toBe(true);
    expect(containsXSS("<embed src=x>")).toBe(true);
    expect(containsXSS("<object data=x>")).toBe(true);
  });

  it("普通文本(含中文)不误报", () => {
    expect(containsXSS("hello world 你好世界")).toBe(false);
    expect(containsXSS("user@example.com")).toBe(false);
    expect(containsXSS("<b>bold</b> 不含事件属性")).toBe(false);
  });
});

describe("escapeHtml", () => {
  it("转义全部 HTML 特殊字符(& 先转义,后续替换不二次转义)", () => {
    expect(escapeHtml("<a href=\"x\">&'<'</a>")).toBe(
      "&lt;a href=&quot;x&quot;&gt;&amp;&#039;&lt;&#039;&lt;/a&gt;"
    );
  });

  it("非字符串输入按 String 透传", () => {
    expect(escapeHtml(123)).toBe("123");
    expect(escapeHtml(null)).toBe("null");
    expect(escapeHtml(undefined)).toBe("undefined");
  });
});

describe("sanitizeObject", () => {
  it("递归转义对象/嵌套对象/数组中的 XSS 字段,安全字段原样保留", () => {
    const input = {
      name: "<img onerror=alert(1)>",
      safe: "hello",
      num: 42,
      nested: { script: "<script>bad()</script>", ok: "fine" },
      arr: ["<script>a</script>", "plain", { deep: "<iframe>" }],
    };
    const result = sanitizeObject(input);

    expect(result.name).toBe(escapeHtml("<img onerror=alert(1)>"));
    expect(result.safe).toBe("hello");
    expect(result.num).toBe(42);
    expect(result.nested.script).toBe(escapeHtml("<script>bad()</script>"));
    expect(result.nested.ok).toBe("fine");
    expect(result.arr[0]).toBe(escapeHtml("<script>a</script>"));
    expect(result.arr[1]).toBe("plain");
    expect((result.arr[2] as { deep: string }).deep).toBe(escapeHtml("<iframe>"));
  });

  it("不修改原始对象(浅拷贝)", () => {
    const input = { evil: "<script>evil()</script>" };
    const result = sanitizeObject(input);
    expect(result.evil).toBe(escapeHtml("<script>evil()</script>"));
    expect(input.evil).toBe("<script>evil()</script>");
  });
});

describe("generateHash / verifyDataIntegrity", () => {
  it("SHA-256 哈希确定且区分数据", async () => {
    const h1 = await generateHash({ a: 1 });
    const h2 = await generateHash({ a: 1 });
    const h3 = await generateHash({ a: 2 });

    expect(h1).toBe(h2);
    expect(h1).not.toBe(h3);
    expect(h1).toMatch(/^[0-9a-f]+$/);
  });

  it("verifyDataIntegrity 校验通过与失败", async () => {
    const hash = await generateHash({ data: "x" });
    await expect(verifyDataIntegrity({ data: "x" }, hash)).resolves.toBe(true);
    await expect(verifyDataIntegrity({ data: "tampered" }, hash)).resolves.toBe(false);
  });

  it("crypto.subtle 不可用时回退简单哈希(仍可校验)", async () => {
    const subtle = crypto.subtle;
    Object.defineProperty(crypto, "subtle", { configurable: true, value: undefined });
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    try {
      const fallback = await generateHash({ a: 1 });
      expect(fallback).toMatch(/^[0-9a-f]+$/);
      expect(fallback.length).toBeLessThanOrEqual(8);
      expect(warnSpy).toHaveBeenCalledTimes(1);
      await expect(verifyDataIntegrity({ a: 1 }, fallback)).resolves.toBe(true);
    } finally {
      Object.defineProperty(crypto, "subtle", { configurable: true, value: subtle });
      warnSpy.mockRestore();
    }
  });
});

describe("SecureStorage", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("setItem/getItem 往返一致", async () => {
    await SecureStorage.setItem("k1", { role: "admin" });
    const value = await SecureStorage.getItem<{ role: string }>("k1");
    expect(value).toEqual({ role: "admin" });
  });

  it("读取不存在的键返回 null", async () => {
    expect(await SecureStorage.getItem<unknown>("missing")).toBeNull();
  });

  it("数据被篡改(哈希不匹配)时删除并返回 null", async () => {
    await SecureStorage.setItem("k2", { trusted: true });

    const stored = JSON.parse(localStorage.getItem("k2") as string) as {
      data: unknown;
      hash: string;
    };
    stored.data = { trusted: false }; // 篡改载荷,保留原哈希
    localStorage.setItem("k2", JSON.stringify(stored));

    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    try {
      expect(await SecureStorage.getItem("k2")).toBeNull();
      expect(localStorage.getItem("k2")).toBeNull(); // 已删除
      expect(warnSpy).toHaveBeenCalledTimes(1);
    } finally {
      warnSpy.mockRestore();
    }
  });

  it("损坏的 JSON 返回 null 而非抛错", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    localStorage.setItem("k3", "{not-json");
    try {
      expect(await SecureStorage.getItem("k3")).toBeNull();
      expect(errorSpy).toHaveBeenCalledTimes(1);
    } finally {
      errorSpy.mockRestore();
    }
  });

  it("removeItem / clear", async () => {
    await SecureStorage.setItem("k4", 1);
    SecureStorage.removeItem("k4");
    expect(await SecureStorage.getItem("k4")).toBeNull();

    await SecureStorage.setItem("k5", 2);
    SecureStorage.clear();
    expect(localStorage.getItem("k5")).toBeNull();
  });
});

describe("isSecureUrl", () => {
  it("同源与相对 URL 放行", () => {
    expect(isSecureUrl("/system/users")).toBe(true);
    expect(isSecureUrl(window.location.origin + "/dashboard")).toBe(true);
  });

  it("白名单域名精确匹配或后缀匹配放行", () => {
    expect(isSecureUrl("https://api.example.com/x", ["https://api.example.com"])).toBe(true);
    expect(isSecureUrl("https://sub.example.com/x", ["example.com"])).toBe(true);
  });

  it("非白名单跨域与非法 URL 拒绝", () => {
    expect(isSecureUrl("https://evil.com/x", ["https://api.example.com"])).toBe(false);
    expect(isSecureUrl("http://exa mple.com", [])).toBe(false);
  });
});

describe("getCSPConfig", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("DEV 环境(vitest 默认 DEV=true)返回宽松策略(HMR 需要 unsafe-eval)", () => {
    const csp = getCSPConfig();
    expect(csp).toContain("default-src 'self'");
    expect(csp).toContain("unsafe-eval");
    expect(csp).not.toContain("object-src 'none'");
  });

  it("非 DEV 环境返回生产严格策略", () => {
    vi.stubEnv("DEV", false);
    const csp = getCSPConfig();
    expect(csp).toContain("default-src 'self'");
    expect(csp).toContain("object-src 'none'");
    expect(csp).toContain("upgrade-insecure-requests");
    expect(csp).not.toContain("unsafe-eval");
  });
});

describe("generateSecureRandom", () => {
  it("生成指定长度的字母数字串", () => {
    const s16 = generateSecureRandom();
    expect(s16).toHaveLength(16);
    expect(s16).toMatch(/^[A-Za-z0-9]+$/);

    const s32 = generateSecureRandom(32);
    expect(s32).toHaveLength(32);
  });
});

describe("secureLog", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
  });

  it("非生产环境完整输出(带/不带 data)", () => {
    const infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    secureLog("info", "with-data", { k: 1 });
    expect(infoSpy).toHaveBeenCalledWith("[info] with-data", { k: 1 });

    secureLog("error", "no-data");
    expect(errorSpy).toHaveBeenCalledWith("[error] no-data");
  });

  it("生产环境只输出 error 且不带 data", () => {
    vi.stubEnv("PROD", true);
    const infoSpy = vi.spyOn(console, "info").mockImplementation(() => {});
    const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    secureLog("info", "hidden", { secret: "x" });
    secureLog("warn", "hidden");
    secureLog("error", "kept", { secret: "x" });

    expect(infoSpy).not.toHaveBeenCalled();
    expect(warnSpy).not.toHaveBeenCalled();
    expect(errorSpy).toHaveBeenCalledWith("[error] kept");
  });
});
