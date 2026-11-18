/**
 * src/lib/systemAnchorApi.ts 单元测试
 *
 * 断言：
 * - previewRoleAnchor / previewPostAnchor / previewUserPermission 全走 `@/lib/api`
 *   的 wrapped `post`（禁止裸 axios），路径正确，返回 result.data 归一化结果；
 * - 无效 ID（空串 / "undefined" / "null"）本地直接抛错，不发请求；
 * - post 失败/响应为空时错误向上抛（由调用方降级，本模块不吞异常）；
 * - normalizeAnchorPreview 覆盖全部分支：
 *   items 优先过滤 !resolved / unresolved 兜底 / 二者皆无 / 字段缺失 / 空响应 /
 *   显式计数字段优先 / resolvedCount 负值钳制到 0 / warnings 数组守卫。
 *
 * ⚠️ vi.mock 工厂会被 hoist，不能引用顶层普通 const；用 vi.hoisted() 定义 spy
 * 后再在工厂中引用（vitest 官方约定）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { AnchorPreviewItem } from "@/types/system";

const { postMock } = vi.hoisted(() => ({
  postMock: vi.fn(async (_url: string, _data?: unknown) => ({ data: undefined as unknown })),
}));

vi.mock("@/lib/api", () => ({
  // 只暴露 post：被测模块若偷用裸 axios / 其它 verb 会在 import 阶段直接报错。
  post: postMock,
}));

import { post } from "@/lib/api";
import {
  normalizeAnchorPreview,
  previewRoleAnchor,
  previewPostAnchor,
  previewUserPermission,
} from "../systemAnchorApi";

const item = (id: string, resolved: boolean): AnchorPreviewItem => ({
  userId: id,
  username: `user-${id}`,
  resolved,
});

type PostResult = { data?: unknown };

/** 让下一次 post 调用解析为给定 data（包成 BaseResponse.data 形态） */
function mockPostData(data: unknown): void {
  postMock.mockResolvedValueOnce({ data } as PostResult);
}

describe("lib/systemAnchorApi — normalizeAnchorPreview（纯函数全分支）", () => {
  it("items 优先：按 !resolved 过滤出未命中项，忽略 unresolved 兜底字段", () => {
    const r = normalizeAnchorPreview({
      mode: 2,
      param: "550e8400-e29b-41d4-a716-446655440000",
      items: [item("a", true), item("b", false), item("c", false)],
      unresolved: [item("z", false)], // 应被 items 覆盖，不采用
      warnings: ["w1"],
    });
    expect(r.mode).toBe(2);
    expect(r.param).toBe("550e8400-e29b-41d4-a716-446655440000");
    expect(r.items).toHaveLength(3);
    expect(r.unresolved.map((u) => u.userId)).toEqual(["b", "c"]);
    expect(r.warnings).toEqual(["w1"]);
    // 未显式给计数时以 items 过滤结果交叉校验
    expect(r.totalCount).toBe(3);
    expect(r.unresolvedCount).toBe(2);
    expect(r.resolvedCount).toBe(1);
  });

  it("items 全 resolved：unresolved 为空数组，resolvedCount = totalCount", () => {
    const r = normalizeAnchorPreview({
      mode: 3,
      items: [item("a", true), item("b", true)],
    });
    expect(r.unresolved).toEqual([]);
    expect(r.totalCount).toBe(2);
    expect(r.resolvedCount).toBe(2);
    expect(r.unresolvedCount).toBe(0);
  });

  it("无 items 时用 unresolved 兜底，totalCount = 未命中数", () => {
    const r = normalizeAnchorPreview({
      mode: 5,
      param: "4",
      unresolved: [item("x", false), item("y", false)],
    });
    expect(r.items).toBeUndefined();
    expect(r.unresolved.map((u) => u.userId)).toEqual(["x", "y"]);
    expect(r.totalCount).toBe(2);
    expect(r.unresolvedCount).toBe(2);
    expect(r.resolvedCount).toBe(0);
  });

  it("items 与 unresolved 皆无：items/unresolved 缺省，计数为 0", () => {
    const r = normalizeAnchorPreview({ mode: 0 });
    expect(r.items).toBeUndefined();
    expect(r.unresolved).toEqual([]);
    expect(r.totalCount).toBe(0);
    expect(r.unresolvedCount).toBe(0);
    expect(r.resolvedCount).toBe(0);
  });

  it("空响应 / null / undefined：全部字段落到安全默认值", () => {
    for (const raw of [undefined, null, {}]) {
      const r = normalizeAnchorPreview(raw);
      expect(r).toEqual({
        mode: 0,
        param: "",
        resolvedCount: 0,
        unresolvedCount: 0,
        totalCount: 0,
        items: undefined,
        unresolved: [],
        warnings: [],
      });
    }
  });

  it("字段缺失 / 类型异常：mode/param/count/warnings 走兜底", () => {
    const r = normalizeAnchorPreview({
      mode: "bad",
      param: null,
      totalCount: "10",
      unresolvedCount: undefined,
      resolvedCount: NaN,
      warnings: "not-array",
      items: "not-array",
      unresolved: "not-array",
    });
    expect(r.mode).toBe(0);
    expect(r.param).toBe("");
    expect(r.totalCount).toBe(0); // items 非数组 → items?.length 兜底为 0
    expect(r.unresolvedCount).toBe(0);
    // resolvedCount 是 number（NaN），按实现原样透传（调用方不应依赖 NaN 场景）
    expect(r.resolvedCount).toBeNaN();
    expect(r.items).toBeUndefined();
    expect(r.unresolved).toEqual([]);
    expect(r.warnings).toEqual([]);
  });

  it("显式计数字段优先于 items 过滤结果", () => {
    const r = normalizeAnchorPreview({
      mode: 2,
      items: [item("a", true), item("b", false)],
      totalCount: 100,
      unresolvedCount: 10,
      resolvedCount: 90,
    });
    expect(r.totalCount).toBe(100);
    expect(r.unresolvedCount).toBe(10);
    expect(r.resolvedCount).toBe(90);
  });

  it("resolvedCount 兜底时用 Math.max(0, total - unresolved) 防止负数", () => {
    const r = normalizeAnchorPreview({
      mode: 2,
      items: [item("a", false)],
      // 无显式计数：total=1, unresolved=1 → 0
    });
    expect(r.resolvedCount).toBe(0);
    const withExplicit = normalizeAnchorPreview({
      mode: 2,
      items: [item("a", true)],
      totalCount: 3,
      unresolvedCount: 10, // 3 - 10 < 0 → 钳制 0
    });
    expect(withExplicit.resolvedCount).toBe(0);
  });

  it("param=0 等假值保留（?? 而非 ||）", () => {
    const r = normalizeAnchorPreview({ param: 0 });
    expect(r.param).toBe(0);
  });
});

describe("lib/systemAnchorApi — previewRoleAnchor", () => {
  beforeEach(() => postMock.mockReset());

  it("走 wrapped post(/system/roles/:id/anchor-preview, {}) 并归一化 result.data", async () => {
    mockPostData({
      mode: 2,
      items: [item("a", true), item("b", false)],
      resolvedCount: 1,
      unresolvedCount: 1,
      totalCount: 2,
    });
    const r = await previewRoleAnchor("role-1");
    expect(post).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith("/system/roles/role-1/anchor-preview", {});
    expect(r).toMatchObject({ mode: 2, totalCount: 2, resolvedCount: 1 });
    expect(r.unresolved.map((u) => u.userId)).toEqual(["b"]);
    expect(r).not.toHaveProperty("code");
  });

  it("空响应数据（undefined）也返回安全默认值，不抛错", async () => {
    mockPostData(undefined);
    const r = await previewRoleAnchor("role-1");
    expect(r.mode).toBe(0);
    expect(r.unresolved).toEqual([]);
  });

  it("无效 ID（空串/'undefined'/'null'）本地抛错且不发请求", async () => {
    for (const bad of ["", "undefined", "null"]) {
      await expect(previewRoleAnchor(bad)).rejects.toThrow("无效的角色 ID");
    }
    expect(postMock).not.toHaveBeenCalled();
  });

  it("post 抛错时原样向上抛，不吞异常", async () => {
    postMock.mockRejectedValueOnce(new Error("network down"));
    await expect(previewRoleAnchor("role-1")).rejects.toThrow("network down");
  });
});

describe("lib/systemAnchorApi — previewPostAnchor", () => {
  beforeEach(() => postMock.mockReset());

  it("走 wrapped post(/system/posts/:id/anchor-preview, {}) 并归一化", async () => {
    mockPostData({
      mode: 5,
      param: "3",
      unresolved: [item("u", false)],
      resolvedCount: 4,
      unresolvedCount: 1,
      totalCount: 5,
    });
    const r = await previewPostAnchor("post-1");
    expect(post).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith("/system/posts/post-1/anchor-preview", {});
    expect(r).toMatchObject({ mode: 5, param: "3", totalCount: 5 });
    expect(r.unresolved.map((u) => u.userId)).toEqual(["u"]);
  });

  it("无效 ID（空串/'undefined'/'null'）本地抛错且不发请求", async () => {
    for (const bad of ["", "undefined", "null"]) {
      await expect(previewPostAnchor(bad)).rejects.toThrow("无效的岗位 ID");
    }
    expect(postMock).not.toHaveBeenCalled();
  });

  it("post 抛错时原样向上抛", async () => {
    postMock.mockRejectedValueOnce(new Error("500"));
    await expect(previewPostAnchor("post-1")).rejects.toThrow("500");
  });
});

describe("lib/systemAnchorApi — previewUserPermission", () => {
  beforeEach(() => postMock.mockReset());

  it("走 wrapped post(/system/users/permission-preview) 传表单值（缺省补空）", async () => {
    mockPostData({
      dataScope: 1,
      dataScopeLabel: "全部",
      all: true,
      anchorSources: [],
      visibleDeptIds: [],
      visibleDeptNames: [],
      notes: [],
      permissions: [],
      menuIds: [],
      deptIds: [],
      allDepts: true,
      roleNames: [],
      postNames: [],
    });
    const r = await previewUserPermission({});
    expect(post).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith("/system/users/permission-preview", {
      deptId: "",
      roleIds: [],
      postIds: [],
    });
    expect(r.dataScope).toBe(1);
  });

  it("完整传参时原样发送 deptId/roleIds/postIds", async () => {
    mockPostData({ dataScope: 2 });
    await previewUserPermission({
      deptId: "dept-1",
      roleIds: ["r1", "r2"],
      postIds: ["p1"],
    });
    expect(post).toHaveBeenCalledWith("/system/users/permission-preview", {
      deptId: "dept-1",
      roleIds: ["r1", "r2"],
      postIds: ["p1"],
    });
  });

  it("后端字段为 null（JSON null）时归一化为数组，避免面板 .length 白屏", async () => {
    mockPostData({
      dataScope: 3,
      dataScopeLabel: "自定义",
      all: false,
      permissions: null,
      menuIds: null,
      deptIds: null,
      visibleDeptIds: null,
      visibleDeptNames: null,
      anchorSources: null,
      notes: null,
      roleNames: null,
      postNames: null,
    });
    const r = await previewUserPermission({ deptId: "d" });
    for (const key of [
      "permissions",
      "menuIds",
      "deptIds",
      "visibleDeptIds",
      "visibleDeptNames",
      "anchorSources",
      "notes",
      "roleNames",
      "postNames",
    ] as const) {
      expect(Array.isArray(r[key]), `${key} 应为数组`).toBe(true);
      expect(r[key]).toEqual([]);
    }
    expect(r.dataScope).toBe(3);
  });

  it("字段缺失（undefined）同样归一化为数组，保留其余字段", async () => {
    mockPostData({ dataScope: 4, all: false, dataScopeLabel: "本部门", extra: "kept" });
    const r = await previewUserPermission({ roleIds: ["r1"] });
    expect(r.permissions).toEqual([]);
    expect(r.menuIds).toEqual([]);
    expect(r.anchorSources).toEqual([]);
    expect(r).toMatchObject({ dataScope: 4, extra: "kept" });
  });

  it("响应 data 为空（undefined/null）时抛错", async () => {
    for (const data of [undefined, null]) {
      mockPostData(data);
      await expect(previewUserPermission({})).rejects.toThrow("权限预览响应为空");
    }
  });

  it("post 抛错时原样向上抛，不吞异常", async () => {
    postMock.mockRejectedValueOnce(new Error("network down"));
    await expect(previewUserPermission({})).rejects.toThrow("network down");
  });
});
