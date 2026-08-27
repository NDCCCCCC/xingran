/**
 * Phase 84 D-03 — "@/lib/api" 端点工厂 mock(plan 0 落地)
 *
 * 形态契约(D-03):
 *   - createApiMock(endpoint) 返回 ApiMockHandle{ post,get,put,del,endpoint }
 *     · endpoint 为该端点专属 spy(url 命中时被调用)
 *     · post/get/put/del 为进程级通用回退 spy(url 未命中任何已注册端点时按
 *       HTTP verb 路由到这里,多个端点工厂共享同一组通用 spy)
 *   - 全部为原生 vi.fn(),支持 .mockResolvedValue / .mockRejectedValue /
 *     .mockImplementationOnce 链式配置
 *   - 不引入 MSW(零新依赖纪律);upload / postFormData / postLongRequest /
 *     getTyped 等别名一律并入对应 verb 的路由(vitest 无法按签名区分,
 *     组件层测试以 url 断言为主)
 *
 * ⚠ 重要使用纪律(模块级拦截生效条件):
 *   vi.mock 只能影响"尚未被加载"的模块 —— 测试文件里被测组件的静态 import
 *   先于测试体执行,因此在函数体内调用 vi.mock 对已缓存的 "@/lib/api"
 *   无效(Vitest 模块图约定)。所以本 harness 采用「静态注册一次 + 动态
 *   登记多端点」模式,这是唯一的可靠用法:
 *
 *   ```ts
 *   // 测试文件顶层(必须在所有 import 之前,vitest 自动 hoist):
 *   vi.mock("@/lib/api", async () => {
 *     const { createApiTestingModule } = await import("@/test/utils/createApiMock");
 *     return createApiTestingModule();
 *   });
 *
 *   // 之后在任意测试体内(组件 import 已完成也能用):
 *   const api = createApiMock("/ops/workstation/list");
 *   api.endpoint.mockResolvedValue({ code: 0, data: { list: [] } });
 *
 *   // 或批量登记:
 *   const handles = mockApiBatch([
 *     { endpoint: "/network/port/binding", response: { code: 0, data: {} } },
 *     { endpoint: "/system/dept/tree" },
 *   ]);
 *   ```
 */
import { vi, type Mock } from "vitest";

/** 单端点 mock 句柄:endpoint 命中 spy + 四个 HTTP verb 通用回退 spy */
export interface ApiMockHandle {
  /** 未命中任何端点时的 POST 回退 spy(全局共享) */
  post: Mock;
  /** 未命中任何端点时的 GET 回退 spy(全局共享) */
  get: Mock;
  /** 未命中任何端点时的 PUT 回退 spy(全局共享) */
  put: Mock;
  /** 未命中任何端点时的 DELETE 回退 spy(全局共享) */
  del: Mock;
  /** 该端点专属 spy:url 命中时收到的第一个参数是原始 url */
  endpoint: Mock;
}

/** mockApiBatch 批量注册项 */
export interface ApiMockHandler {
  endpoint: string;
  /** 注册后立即 mockResolvedValue 到该端点 spy(可选) */
  response?: unknown;
}

type ApiVerb = "get" | "post" | "put" | "del";

/**
 * 端点路由表:url -> 端点专属 handle。
 * 进程级单例,让 createApiMock / mockApiBatch 可在任何时机增量登记,
 * 且互相不覆盖(vi.mock 工厂每次被调用都会读同一张表)。
 */
const endpointRegistry = new Map<string, ApiMockHandle>();

const genericSpies: Record<ApiVerb, Mock> = {
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
};

/**
 * 路由核心:createApiTestingModule 导出的每个 verb 都经由这里分发。
 * url 命中任一已注册端点 -> endpoint spy;否则 -> 通用 verb spy 回退。
 */
function routeCall(verb: ApiVerb, args: unknown[]): unknown {
  const [url] = args;
  const hit = url == null ? undefined : endpointRegistry.get(String(url));
  if (hit) {
    return hit.endpoint(...args);
  }
  return genericSpies[verb](...args);
}

function makeHandle(): ApiMockHandle {
  return {
    post: genericSpies.post,
    get: genericSpies.get,
    put: genericSpies.put,
    del: genericSpies.del,
    endpoint: vi.fn(),
  };
}

/**
 * 单端点 mock 工厂(D-03)。同一 endpoint 重复调用会重建该端点 spy
 * (旧 spy 引用作废);post/get/put/del 永远指向共享的通用回退 spy。
 *
 * 必须配合测试文件顶层的 vi.mock("@/lib/api", ...) 静态安装才能拦截真实
 * 组件依赖(见文件头「重要使用纪律」)。
 */
export function createApiMock(endpoint: string): ApiMockHandle {
  const handle = makeHandle();
  endpointRegistry.set(endpoint, handle);
  return handle;
}

/**
 * 批量端点注册(D-03 可选 helper)。带 response 的项会立即把该端点
 * 配置成 resolved 值;返回 { [endpoint]: handle } 映射便于逐个断言。
 */
export function mockApiBatch(handlers: Array<ApiMockHandler>): Record<string, ApiMockHandle> {
  const handles: Record<string, ApiMockHandle> = {};
  for (const { endpoint, response } of handlers) {
    const handle = createApiMock(endpoint);
    if (response !== undefined) {
      handle.endpoint.mockResolvedValue(response);
    }
    handles[endpoint] = handle;
  }
  return handles;
}

/**
 * 清空全部端点与通用 spy 的调用记录及实现(vi.fn mockClear + mockReset 语义)
 * 供 afterEach 重置,避免跨用例断言串扰。
 */
export function resetApiMocks(): void {
  endpointRegistry.clear();
  genericSpies.get.mockReset();
  genericSpies.post.mockReset();
  genericSpies.put.mockReset();
  genericSpies.del.mockReset();
}

/**
 * 生成可直接作为 vi.mock("@/lib/api", ...) 工厂返回值的模块替身。
 *
 * 覆盖面:
 *   - 具名导出:get/post/put/del/upload/postFormData/postLongRequest +
 *     getTyped/postTyped/putTyped/patchTyped/deleteTyped 别名 +
 *     initEncryptionConfig / refreshEncryptionConfig 加密引导 stub
 *   - default 导出(api 实例本身):同 verb 路由的轻量对象(含 interceptors
 *     空 hook,避免注入器链式访问抛错)
 */
export function createApiTestingModule(): Record<string, unknown> {
  const byVerb =
    (verb: ApiVerb) =>
    (...args: unknown[]) =>
      routeCall(verb, args);

  const getSpy = (...args: unknown[]) => routeCall("get", args);
  const postSpy = (...args: unknown[]) => routeCall("post", args);
  const putSpy = (...args: unknown[]) => routeCall("put", args);
  const delSpy = (...args: unknown[]) => routeCall("del", args);

  const apiInstanceStub = {
    get: getSpy,
    post: postSpy,
    put: putSpy,
    delete: delSpy,
    del: delSpy,
    patch: postSpy,
    request: vi.fn(),
    interceptors: {
      request: { use: vi.fn(), eject: vi.fn() },
      response: { use: vi.fn(), eject: vi.fn() },
    },
    defaults: { headers: { common: {} } },
  };

  return {
    default: apiInstanceStub,
    api: apiInstanceStub,
    initEncryptionConfig: vi.fn(async () => {}),
    refreshEncryptionConfig: vi.fn(async () => true),
    get: getSpy,
    post: postSpy,
    put: putSpy,
    del: delSpy,
    upload: (...args: unknown[]) => routeCall("post", args),
    postFormData: (...args: unknown[]) => routeCall("post", args),
    postLongRequest: (...args: unknown[]) => routeCall("post", args),
    getTyped: byVerb("get"),
    postTyped: byVerb("post"),
    putTyped: byVerb("put"),
    patchTyped: (...args: unknown[]) => routeCall("post", args),
    deleteTyped: byVerb("del"),
  };
}
