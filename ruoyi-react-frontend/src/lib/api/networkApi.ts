import axios, { type AxiosInstance } from 'axios';
import { post } from '../api';
import { getAccessToken } from '@/utils/authHelpers';

/**
 * 专用于 Blob/文件下载的 axios 实例(本文件内私有)
 * - 绕过 src/lib/api.ts 响应拦截器(后者会解包 BaseResponse envelope,
 *   导致 xlsx bytes 被错误转 JSON 对象)
 * - 与 Phase 33 M2 opsApi.excelApi.export 模式对齐(F-PATH-06)
 * - 自动注入 Authorization 头(从 TokenManager / getAccessToken)
 * - 移除硬编码 /api/v1/ 前缀,改用环境变量 VITE_API_BASE_URL
 */
const blobAxios: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://10.62.10.33:9000/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

blobAxios.interceptors.request.use(async (config) => {
  try {
    const token = await getAccessToken();
    if (token && config.headers) {
      config.headers.set('Authorization', `Bearer ${token}`);
    }
  } catch (e) {
    // Token 获取失败不阻断,让后端 401 路径处理
    console.warn('[networkApi.blobAxios] 获取 token 失败:', e);
  }
  return config;
});

interface TrajectoryQueryParams {
  mac: string;
  start_time: string; // ISO timestamp
  end_time: string;   // ISO timestamp
}

interface TrajectoryNode {
  mac: string;
  deviceId: string;
  deviceName: string;
  interface: string;
  vlanId?: number;
  startTime: string;
  endTime: string;
  duration: number;
  eventType: 'appeared' | 'disappeared' | 'moved' | 'vlan_changed';
}

interface TrajectoryResponse {
  trajectory: TrajectoryNode[];
}

/**
 * 查询MAC地址轨迹
 * @param params 查询参数（MAC地址 + 时间范围）
 * @returns 轨迹节点数组
 */
export const queryMACTrajectory = async (
  params: TrajectoryQueryParams
): Promise<TrajectoryNode[]> => {
  const result = await post<TrajectoryResponse>(
    '/network/history/trajectory',
    params
  );
  return result.data!.trajectory;
};

/**
 * MAC 历史查询参数(Phase 14-01 引入)
 * - 端点: POST /network/history/list (D-01 锁定)
 * - 字段对应后端 mac_history_query_service.go 的 MACHistoryQueryResult
 */
export interface MACHistoryQueryParams {
  current: number;
  pageSize: number;
  mac?: string;
  deviceId?: string;
  interfaceName?: string;
  vlanId?: number;
  eventType?: string;
  status?: number;
  startTime?: string;
  endTime?: string;
}

/**
 * MAC 历史记录(单条)
 * - 字段对应后端 services.MACHistoryRecord (mac_history_query_service.go:41-52)
 */
export interface MACHistoryRecord {
  id: string;
  macAddress: string;
  deviceId: string;
  deviceNameSnapshot: string;
  interfaceName: string;
  vlanId?: number;
  eventType: 'appeared' | 'disappeared' | 'moved' | 'vlan_changed';
  firstSeen: string;
  lastSeen: string;
  collectedAt: string;
  status: number;
}

/**
 * MAC 历史分页结果
 */
export interface MACHistoryPageResult {
  list: MACHistoryRecord[];
  total: number;
  current: number;
  pageSize: number;
}

/**
 * 查询 MAC 历史记录(列表页主数据源)
 * - 端点: POST /network/history/list (D-01 锁定)
 * - 响应: 标准分页结构 list/total/current/pageSize
 */
export const queryMACHistory = async (
  params: MACHistoryQueryParams
): Promise<MACHistoryPageResult> => {
  const result = await post<MACHistoryPageResult>(
    '/network/history/list',
    params
  );
  return result.data!;
};

/**
 * 拉取某 MAC 在时间范围内的事件序列(供 MACEventsTimeline 组件使用)
 * - 内部复用 /network/history/list 端点,pageSize 设为 100 一次性拉满
 * - 返回按 firstSeen 倒序的事件列表
 */
export const getMACEvents = async (
  mac: string,
  startTime: string,
  endTime: string
): Promise<MACHistoryRecord[]> => {
  const result = await post<MACHistoryPageResult>('/network/history/list', {
    mac,
    startTime,
    endTime,
    current: 1,
    pageSize: 100,
  });
  const list = result.data?.list ?? [];
  const total = result.data?.total ?? 0;
  if (total > list.length) {
    console.warn(`[getMACEvents] 事件被截断:total=${total}, returned=${list.length}。前端 pageSize=100 上限。考虑添加 sort 扩展或扩大分页。`);
  }
  return list.slice().sort(
    (a, b) =>
      new Date(b.firstSeen).getTime() - new Date(a.firstSeen).getTime()
  );
};

/**
 * 导出 MAC 历史数据为 Excel (14-fix-02 重写, F-PATH-07 重构)
 * - 走本地 blobAxios 实例(无 BaseResponse 解包拦截器),确保 xlsx bytes 正确返回为 Blob
 * - 移除硬编码 /api/v1/ 前缀,改用 VITE_API_BASE_URL(经 baseURL 自动拼接)
 * - 与 Phase 33 M2 opsApi.excelApi.export / F-PATH-06 模式对齐
 * - 支持 current(当前查询) 和 all(全量) 两种导出范围
 * - 返回 { blob, filename };filename 优先取 Content-Disposition header,
 *   缺失时回退 mac_history_<scope>_<ts>.xlsx
 */
export const exportMACHistory = async (
  params: MACHistoryQueryParams,
  exportScope: 'current' | 'all' = 'current'
): Promise<{ blob: Blob; filename: string }> => {
  const queryParams: Record<string, string> = { format: 'xlsx', exportScope };
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') queryParams[k] = String(v);
  });
  const response = await blobAxios.get<Blob>('/network/history/list', {
    params: queryParams,
    responseType: 'blob',
  });
  const blob = response.data as unknown as Blob;
  // CR-01 错误反序列化:若 blob 实际是 JSON 错误体,后端在异常时仍可能返回 application/json
  if (blob && blob.size < 1024 && blob.type && blob.type.includes('json')) {
    const text = await blob.text();
    try {
      const errBody = JSON.parse(text) as { message?: string; msg?: string };
      throw new Error(errBody.message || errBody.msg || '导出失败');
    } catch (e) {
      if (e instanceof Error && e.message !== '导出失败') throw e;
      // parse 失败说明不是 JSON,降级为通用错误
      throw new Error(`导出失败:${response.status}`);
    }
  }
  const contentDisposition = (response.headers as Record<string, string>)['content-disposition']
    || (response.headers as Record<string, string>)['Content-Disposition'];
  let filename = `mac_history_${exportScope}_${Date.now()}.xlsx`;
  if (contentDisposition) {
    const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
    if (match && match[1]) filename = decodeURIComponent(match[1].replace(/['"]/g, ''));
  }
  return { blob, filename };
};

/**
 * 触发浏览器下载(从 Blob + filename)
 * - 私有工具函数,供 batchExport 使用
 * - 抽取后 9 个调用方不再需要重复 a/link/click/revokeObjectURL 模板
 */
const triggerBrowserDownload = (blob: Blob, filename: string): void => {
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(a);
};

/**
 * 批量导出网络模块数据 (F-PATH-08 重构)
 * - 走本地 blobAxios 实例(无 BaseResponse 解包拦截器),确保 zip bytes 正确返回为 Blob
 * - 移除硬编码 /api/v1/ 前缀,改用 VITE_API_BASE_URL(经 baseURL 自动拼接)
 * - 9 个网络模块页面(devices/ports/mac/backups/command/credentials/
 *   discoveries/executions/templates)共用此函数
 * - 内部完成:HTTP POST → filename 提取(Content-Disposition) → 触发浏览器下载
 * @param entityTypes 实体类型列表(如 ['devices', 'ports'])
 * @param filters 过滤条件(键值对对象,序列化为 JSON body)
 * @param fallbackFilename 下载头缺失时使用的默认文件名
 * @returns 实际下载使用的 filename
 */
export const batchExport = async (
  entityTypes: string[],
  filters: Record<string, unknown> = {},
  fallbackFilename = `网络管理_批量导出_${Date.now()}.zip`
): Promise<string> => {
  const response = await blobAxios.post<Blob>(
    '/network/batch-export',
    { entityTypes, filters },
    { responseType: 'blob' }
  );
  const blob = response.data as unknown as Blob;
  // CR-01 错误反序列化:若 blob 实际是 JSON 错误体,后端在异常时仍可能返回 application/json
  if (blob && blob.size < 1024 && blob.type && blob.type.includes('json')) {
    const text = await blob.text();
    try {
      const errBody = JSON.parse(text) as { message?: string; msg?: string };
      throw new Error(errBody.message || errBody.msg || '导出失败');
    } catch (e) {
      if (e instanceof Error && e.message !== '导出失败') throw e;
      // parse 失败说明不是 JSON,降级为通用错误
      throw new Error(`导出失败:${response.status}`);
    }
  }
  const contentDisposition = (response.headers as Record<string, string>)['content-disposition']
    || (response.headers as Record<string, string>)['Content-Disposition'];
  let filename = fallbackFilename;
  if (contentDisposition) {
    const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
    if (match && match[1]) filename = decodeURIComponent(match[1].replace(/['"]/g, ''));
  }
  triggerBrowserDownload(blob, filename);
  return filename;
};

export default {
  queryMACTrajectory,
  queryMACHistory,
  getMACEvents,
  exportMACHistory,
  batchExport,
};
