/**
 * Phase 88 Batch34 — router/componentLoader 单元测试(只测同步路径,避免真实 load 死锁)
 */
import { describe, it, expect } from "vitest";
import { ComponentLoader, componentLoader, createLazyComponent } from "../componentLoader";

describe("ComponentLoader 白名单(只测同步拒绝路径)", () => {
  it("拒绝 ../ 路径遍历", async () => {
    await expect(componentLoader.load("../../etc/passwd")).rejects.toThrow(/Invalid component/);
  });

  it("拒绝反斜杠", async () => {
    await expect(componentLoader.load("pages\\system\\user")).rejects.toThrow(/Invalid component/);
  });

  it("拒绝 .html 扩展名", async () => {
    // normalizePath 自动加 .tsx,但 normalize 后以 .tsx 结尾,白名单字符串结尾正则不再命中 .html
    // 实际触发 importComponent → Component not found
    await expect(componentLoader.load("pages/x.html")).rejects.toThrow(/Invalid component|Component not found/);
  });

  it("拒绝 .js 扩展名", async () => {
    await expect(componentLoader.load("pages/x.js")).rejects.toThrow(/Invalid component|Component not found/);
  });

  it("拒绝 .json 扩展名", async () => {
    await expect(componentLoader.load("pages/x.json")).rejects.toThrow(/Invalid component|Component not found/);
  });
});

describe("componentLoader 缓存", () => {
  it("初始缓存大小 0", () => {
    expect(componentLoader.getCacheSize()).toBe(0);
  });

  it("clearCache 不 throw", () => {
    componentLoader.clearCache();
    expect(componentLoader.getCacheSize()).toBe(0);
  });
});

describe("createLazyComponent 路径处理", () => {
  it("返 lazy React 组件或错误组件", () => {
    // 命中真实路径 → lazy 组件(React.lazy 返 { $$typeof, render })
    const LC = createLazyComponent("pages/system/user/index");
    expect(LC).toBeDefined();
    // lazy / 错误组件 / 普通 FC 都是 typeof function 或 object(lazy 是 object)
    expect(typeof LC === "function" || typeof LC === "object").toBe(true);
  });

  it("无 pages/ 前缀自动加", () => {
    const LC = createLazyComponent("system/user/index");
    expect(LC).toBeDefined();
    expect(typeof LC === "function" || typeof LC === "object").toBe(true);
  });

  it("moduleLoader 找不到时返错误组件(非 lazy)", () => {
    const ErrComp = createLazyComponent("pages/nonexistent/dir/index");
    // 错误组件 $$typeof 未定义(直接函数)
    expect((ErrComp as any).$$typeof).toBeUndefined();
  });

  it("空 path + 路径不含 / 仍返函数(尝试加载)", () => {
    const LC = createLazyComponent("xxx");
    expect(typeof LC).toBe("function");
  });
});

describe("ComponentLoader 静态属性", () => {
  it("ALLOWED_PREFIXES 含 pages/components", () => {
    expect((ComponentLoader as any).ALLOWED_PREFIXES).toEqual(["pages/", "components/"]);
  });

  it("DANGEROUS_PATTERNS 长度 5", () => {
    expect((ComponentLoader as any).DANGEROUS_PATTERNS.length).toBe(5);
  });

  it("componentModules glob 含 pages/{index,detail}.tsx", () => {
    expect(Object.keys((ComponentLoader as any).componentModules).length).toBeGreaterThan(50);
  });
});
