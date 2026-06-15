import { post } from '../api';
import { getAccessToken } from '@/utils/authHelpers';

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
  return (result.data?.list ?? []).slice().sort(
    (a, b) =>
      new Date(b.firstSeen).getTime() - new Date(a.firstSeen).getTime()
  );
};

/**
 * 导出 MAC 历史数据为 Excel (14-fix-02 重写)
 * - 走 fetch + getAccessToken + response.blob(),绕过 src/lib/api.ts 响应拦截器
 *   (后者会解包 BaseResponse envelope,导致 xlsx bytes 被错误转 JSON 对象)
 * - 与 Phase 33 M2 opsApi.excelApi.export 模式对齐
 * - 支持 current(当前查询) 和 all(全量) 两种导出范围
 * - 返回 { blob, filename };filename 优先取 Content-Disposition header,
 *   缺失时回退 mac_history_<scope>_<ts>.xlsx
 */
export const exportMACHistory = async (
  params: MACHistoryQueryParams,
  exportScope: 'current' | 'all' = 'current'
): Promise<{ blob: Blob; filename: string }> => {
  const token = await getAccessToken();
  const search = new URLSearchParams();
  Object.entries({ ...params, format: 'xlsx', exportScope }).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== '') search.set(k, String(v));
  });
  const response = await fetch(`/api/v1/network/history/list?${search.toString()}`, {
    method: 'GET',
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!response.ok) {
    throw new Error(`导出失败:HTTP ${response.status}`);
  }
  const blob = await response.blob();
  // CR-01 错误反序列化:若 blob 实际是 JSON 错误体,后端在异常时仍可能返回 application/json
  if (blob.size < 1024 && blob.type.includes('json')) {
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
  const contentDisposition = response.headers.get('content-disposition');
  let filename = `mac_history_${exportScope}_${Date.now()}.xlsx`;
  if (contentDisposition) {
    const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
    if (match && match[1]) filename = decodeURIComponent(match[1].replace(/['"]/g, ''));
  }
  return { blob, filename };
};

export default {
  queryMACTrajectory,
  queryMACHistory,
  getMACEvents,
  exportMACHistory,
};
