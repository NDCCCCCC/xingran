/**
 * blob 文件下载链（Phase 94 D-04 单一权威）
 *
 * 自 opsApi.ts:310-369 原样迁入：blobAxios 实例（5min 超时）+ 异步 token
 * 注入拦截器 + 文件名提取 + 浏览器触发 + GET 下载；并新增 downloadFilePost
 * （POST 变体），归一 excelApi.export / asset excel export 等处 POST-blob
 * 内联重复。Phase 100（V130R-12/D-100-7）起为全站唯一下载权威：
 * networkApi 的 exportMACHistory/batchExport 亦收敛为薄壳。
 *
 * D-100-7：downloadFile/downloadFilePost 同挂 200+application/json 错误体
 * 检测（content-type 强判据，替换 networkApi 旧 size<1024 弱嗅探）——后端把
 * 错误体伪装成下载响应时 throw（message 透传），不再存成伪 .xlsx。
 *
 * 本文件不 import 任何 *Api.ts（临时双份由 94-02 迁出）。
 */

import axios from "axios";
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from "axios";
import { getAccessToken } from "@/utils/authHelpers";
import { getTokenManager } from "@/store/authStore";

// 专用于文件下载的 axios 实例：
// - 复用与主 API 客户端相同的 baseURL（去掉硬编码 /api/v1/ 前缀）
// - 不挂载响应拦截器，因为响应是 Blob 二进制流，无法 JSON 解析
// - 请求拦截器只做 Token 注入，行为与其他 CRUD 保持一致
// - 工位导出 1643 工位 + ~6000 行设备数据 → xlsx ~5-10 MB → 默认 30s timeout 易中招
//   (尤其 dev 环境 Vite proxy 转发增加额外间接,网络抖动会被放大)
//   改 5min 给足缓冲;普通 CRUD 不走 blobAxios 不会受影响
export const blobAxios: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || "/api/v1",
  timeout: 300000,
});

blobAxios.interceptors.request.use(async (config: InternalAxiosRequestConfig) => {
  // Fast path: check cached token synchronously before async call
  const tokenManager = getTokenManager();
  const hasToken = tokenManager.isAuthenticated();

  let token: string | null = null;
  if (hasToken) {
    // Token is cached synchronously; getAccessToken() is fast path (sync check + optional refresh)
    try {
      token = await getAccessToken();
    } catch {
      // Token unavailable, skip header injection
    }
  }
  // else: no token yet, skip header injection without awaiting

  if (token && config.headers) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }
  return config;
});

// 从响应头提取文件名（headers 宽松读取：无头 mock/响应回退默认文件名）
export function extractFilenameFromBlobResponse(
  response: AxiosResponse<Blob>,
  defaultFilename: string
): string {
  const contentDisposition: string | undefined = response.headers?.["content-disposition"];
  if (!contentDisposition) {
    return defaultFilename;
  }

  const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
  if (match && match[1]) {
    return decodeURIComponent(match[1].replace(/['"]/g, ""));
  }

  return defaultFilename;
}

// 触发浏览器下载 Blob
export function triggerBrowserDownload(blob: Blob, filename: string): void {
  const blobUrl = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = blobUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(blobUrl);
  document.body.removeChild(a);
}

// D-100-7：200 + application/json = 后端把错误体伪装成下载响应。
// content-type 强判据（协议级信号）优先于文件名提取；错误体 message
// 字段透传（API Response Format envelope），非法 JSON 回退默认文案。
async function throwIfJsonErrorResponse(
  response: AxiosResponse<Blob>,
  defaultFilename: string
): Promise<void> {
  const contentType = String(response.headers?.["content-type"] ?? "");
  if (!contentType.includes("application/json")) {
    return;
  }
  let message = `下载失败: ${defaultFilename}`;
  try {
    const text = await response.data.text();
    const parsed = JSON.parse(text) as { message?: string };
    if (parsed?.message) {
      message = parsed.message;
    }
  } catch {
    // 非法 JSON 保留默认文案
  }
  throw new Error(message);
}

// 通用文件下载函数(GET) — 返回实际使用的 filename
export async function downloadFile(url: string, filename: string): Promise<string> {
  const response = await blobAxios.get<Blob>(url, { responseType: "blob" });

  if (response.status < 200 || response.status >= 300) {
    throw new Error(`下载失败: ${filename}`);
  }

  await throwIfJsonErrorResponse(response, filename);

  const actualFilename = extractFilenameFromBlobResponse(response, filename);
  triggerBrowserDownload(response.data, actualFilename);
  return actualFilename;
}

// 通用文件下载函数(POST) — excelApi.export / asset excel export 等
// POST-blob 内联重复的归一目标（D-04）；Phase 100 D-100-6 起 networkApi
// batchExport 亦为薄壳消费者。返回实际使用的 filename。
export async function downloadFilePost(
  url: string,
  body: unknown,
  defaultFilename: string
): Promise<string> {
  const response = await blobAxios.post<Blob>(url, body, { responseType: "blob" });

  if (response.status < 200 || response.status >= 300) {
    throw new Error(`下载失败: ${defaultFilename}`);
  }

  await throwIfJsonErrorResponse(response, defaultFilename);

  const filename = extractFilenameFromBlobResponse(response, defaultFilename);
  triggerBrowserDownload(response.data, filename);
  return filename;
}
