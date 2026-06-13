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

export default {
  queryMACTrajectory,
};
