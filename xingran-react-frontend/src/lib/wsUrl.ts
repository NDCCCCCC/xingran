/**
 * WebSocket URL 统一构造。
 *
 * 为什么需要这个模块
 * ------------------
 * 1) 子路径部署：生产经 nginx 以 `/xingran/` 前缀暴露同一端口，WebSocket 握手
 *    必须带同一前缀，否则会落到同端口的其它服务（SillyTavern）上，必然失败。
 *    早期代码硬编码 `${host}/ws/...`，既没有前缀、路径也与后端注册的不一致。
 *
 * 2) 后端 WS 端点统一挂在 `/api/v1/system/ws/` 之下（见 internal/api/router.go
 *    的 `system` → `authorized` → `wsGroup` 注册链），所以 WS 基址应当由 API
 *    基址派生，而不是另写一份常量 —— 两处各写一份必然漂移。
 *
 * 3) 后端目前只注册了 `/notices` 一个 WS 端点；仪表盘频道端点尚未实现，
 *    因此连接会失败。本模块只负责把 URL 拼对，重连策略由调用方决定。
 */

/**
 * API 基址（与 src/lib/api.ts 的 baseURL 保持同一来源）。
 * 每次调用都重新读取 env —— 运行时可被部署配置覆盖，测试也可直接 stub。
 */
function rawApiBase(): string {
  return import.meta.env.VITE_API_BASE_URL || "/api/v1";
}

/**
 * 取 API 基址的路径部分。
 * VITE_API_BASE_URL 可能是绝对 URL（开发环境指向本地后端），也可能是根相对路径。
 */
function apiPath(): string {
  try {
    const url = new URL(rawApiBase(), window.location.origin);
    return url.pathname.replace(/\/+$/, "");
  } catch {
    return rawApiBase().replace(/\/+$/, "");
  }
}

/**
 * 部署子路径前缀：""（根路径部署）或 "/xingran"（子路径部署）。
 * 从 API 路径里剥掉结尾的 /api/vN 得到。
 */
export function deploymentPrefix(): string {
  const path = apiPath();
  const match = path.match(/^(.*?)\/api\/v\d+$/);
  return match ? match[1] : "";
}

/**
 * 构造 WebSocket 绝对 URL。
 *
 * @param path 服务端路径，通常由 apiPath() 派生，例如 "/xingran/api/v1/system/ws/notices"
 */
export function buildWsUrl(path: string): string {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const host = import.meta.env.VITE_WS_HOST || window.location.host;
  const prefix = deploymentPrefix();
  const suffix = path.startsWith("/") ? path : `/${path}`;
  // path 已含部署前缀时不再重复拼接
  const full = suffix.startsWith(prefix) ? suffix : `${prefix}${suffix}`;
  return `${protocol}//${host}${full}`;
}

/**
 * 通知 WebSocket 端点 —— 后端真实注册的路径。
 * 与 src/lib/noticeApi.ts 的推导保持一致。
 */
export function noticeWsUrl(): string {
  return buildWsUrl(`${apiPath()}/system/ws/notices`);
}

/**
 * 仪表盘频道 WebSocket 端点。
 *
 * 注意：后端当前只实现了 /notices，仪表盘频道端点尚未提供 —— 调用方需要
 * 自带重连退避与失败降级，不要无限重连。
 */
export function dashboardWsUrl(channel: string): string {
  return buildWsUrl(`${apiPath()}/system/ws/${channel}`);
}
