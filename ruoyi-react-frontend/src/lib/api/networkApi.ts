import { post } from './api';

interface TrajectoryQueryParams {
  mac: string;
  start_time: string; // ISO timestamp
  end_time: string;   // ISO timestamp
}

interface TrajectoryNode {
  mac: string;
  device_name: string;
  port_name: string;
  vlan_id: number;
  first_seen: string;
  last_seen: string;
  duration_seconds: number;
  event_type: 'appeared' | 'disappeared' | 'moved' | 'vlan_changed';
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
  return result.data.trajectory;
};

export default {
  queryMACTrajectory,
};
