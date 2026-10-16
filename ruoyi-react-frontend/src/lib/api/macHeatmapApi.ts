/**
 * MAC 地址端口使用热力图 API 包装
 *
 * Phase 15 PERF-04 (D-16/D-17 锁定):
 *   - 后端数据源严格走 MV-04 (mv_mac_port_daily_count)
 *   - 走 cache-aside (后端 15-03 装饰器)
 *   - 默认 7 天范围 / TopN 100
 */

import { post } from "../api";

export interface HeatmapCell {
  deviceId: string;
  deviceNameSnapshot: string;
  interfaceName: string;
  date: string;
  changeCount: number;
}

export interface HeatmapResult {
  cells: HeatmapCell[];
  topN: number;
  start: string;
  end: string;
  total: number;
  snapshot: string;
}

export interface HeatmapQuery {
  startTime?: string;
  endTime?: string;
  topN?: number;
}

/**
 * 查询端口使用热力图
 * @param params 查询参数 (时间范围 + TopN)
 * @returns HeatmapResult
 */
export const queryMACHeatmap = async (params: HeatmapQuery): Promise<HeatmapResult> => {
  const result = await post<HeatmapResult>("/network/history/heatmap", params);
  return result.data!;
};
