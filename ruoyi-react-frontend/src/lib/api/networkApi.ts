import { post } from '../api';

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

export default {
  queryMACTrajectory,
  queryMACHistory,
  getMACEvents,
};
