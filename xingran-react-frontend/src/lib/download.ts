/**
 * blob 文件下载链（Phase 94 D-04 单一权威）
 *
 * 自 opsApi.ts:310-369 原样迁入：blobAxios 实例（5min 超时）+ 异步 token
 * 注入拦截器 + 文件名提取 + 浏览器触发 + GET 下载；并新增 downloadFilePost
 * （POST 变体），归一 excelApi.export / asset excel export /
 * rpaApi.downloadReport 三处 POST-blob 内联重复。
 *
 * 本文件不 import 任何 *Api.ts（临时双份由 94-02 迁出）。
 */

import axios from "axios";
import type { AxiosInstance, AxiosResponse, InternalAxiosRequestConfig } from "axios";
import { getAccessToken } from "@/utils/authHelpers";

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
  const token = await getAccessToken();
  if (token && config.headers) {
    config.headers.set("Authorization", `Bearer ${token}`);
  }
  return config;
});

// 从响应头提取文件名
export function extractFilenameFromBlobResponse(
  response: AxiosResponse<Blob>,
  defaultFilename: string
): string {
  const contentDisposition: string | undefined = response.headers["content-disposition"];
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

// 通用文件下载函数(GET)
export async function downloadFile(url: string, filename: string): Promise<void> {
  const response = await blobAxios.get<Blob>(url, { responseType: "blob" });

  if (response.status < 200 || response.status >= 300) {
    throw new Error(`下载失败: ${filename}`);
  }

  triggerBrowserDownload(response.data, filename);
}

// 通用文件下载函数(POST) — excelApi.export / asset excel export /
// rpaApi.downloadReport 三处 POST-blob 内联重复的归一目标（D-04）
export async function downloadFilePost(
  url: string,
  body: unknown,
  defaultFilename: string
): Promise<void> {
  const response = await blobAxios.post<Blob>(url, body, { responseType: "blob" });

  if (response.status < 200 || response.status >= 300) {
    throw new Error(`下载失败: ${defaultFilename}`);
  }

  const filename = extractFilenameFromBlobResponse(response, defaultFilename);
  triggerBrowserDownload(response.data, filename);
}
