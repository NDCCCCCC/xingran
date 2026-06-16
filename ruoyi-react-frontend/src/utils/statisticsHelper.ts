/**
 * 统计辅助函数
 * 用于处理后端 MaxPageSize 限制导致的统计数据获取问题
 */

import type { DedicatedLine, RoomDevice, ServerRoom, InfoPoint, Building, Floor } from "@/types";
import { dedicatedLineApi, roomDeviceApi, serverRoomApi, infoPointApi, buildingApi, floorApi } from "@/lib/opsApi";

/**
 * 分批获取所有数据并计算统计信息
 * 由于后端 MaxPageSize 限制为 100，需要分批获取所有数据进行统计
 */
async function fetchAllDataForStatistics<T>(
  fetchFn: (params: { current: number; pageSize: number }) => Promise<{ data?: { list: T[]; total?: number } }>
): Promise<T[]> {
  const pageSize = 100;
  let allData: T[] = [];
  let current = 1;
  let hasMore = true;

  while (hasMore) {
    const result = await fetchFn({ current, pageSize });
    const list = result.data?.list || [];
    allData = allData.concat(list);

    // 如果返回的数据少于 pageSize，说明已经获取完所有数据
    if (list.length < pageSize) {
      hasMore = false;
    } else {
      current++;
      // 防止无限循环，最多获取 10000 条
      if (allData.length >= 10000) {
        hasMore = false;
      }
    }
  }

  return allData;
}

/**
 * 专线统计数据
 */
export async function fetchDedicatedLineStatistics() {
  const allLines = await fetchAllDataForStatistics<DedicatedLine>(dedicatedLineApi.list);

  // 优化：单次遍历计算所有统计（js-combine-iterations）
  let normal = 0;
  let fault = 0;
  let disabled = 0;

  for (const line of allLines) {
    if (line.status === 0) normal++;
    else if (line.status === 1) fault++;
    else if (line.status === 2) disabled++;
  }

  return {
    total: allLines.length,
    normal,
    fault,
    disabled,
  };
}

/**
 * 机房设备统计数据
 */
export async function fetchRoomDeviceStatistics() {
  const allDevices = await fetchAllDataForStatistics<RoomDevice>(roomDeviceApi.list);

  // 优化：单次遍历计算所有统计
  let normal = 0;
  let fault = 0;
  let scrapped = 0;

  for (const device of allDevices) {
    if (device.status === 0) normal++;
    else if (device.status === 1) fault++;
    else if (device.status === 2) scrapped++;
  }

  return {
    total: allDevices.length,
    normal,
    fault,
    scrapped,
  };
}

/**
 * 机房统计数据
 */
export async function fetchServerRoomStatistics() {
  const allRooms = await fetchAllDataForStatistics<ServerRoom>(serverRoomApi.list);

  // 优化：单次遍历计算所有统计
  let active = 0;
  let inactive = 0;

  for (const room of allRooms) {
    if (room.status === 0) active++;
    else if (room.status === 1) inactive++;
  }

  return {
    total: allRooms.length,
    active,
    inactive,
  };
}

/**
 * 信息点统计数据
 */
export async function fetchInfoPointStatistics() {
  const allInfoPoints = await fetchAllDataForStatistics<InfoPoint>(infoPointApi.list);

  // 优化：单次遍历计算所有统计
  let normal = 0;
  let fault = 0;
  let disabled = 0;

  for (const infoPoint of allInfoPoints) {
    if (infoPoint.status === 0) normal++;
    else if (infoPoint.status === 1) fault++;
    else if (infoPoint.status === 2) disabled++;
  }

  return {
    total: allInfoPoints.length,
    normal,
    fault,
    disabled,
  };
}

/**
 * 楼宇统计数据（支持筛选条件）
 */
export async function fetchBuildingStatistics(searchParams: Record<string, unknown> = {}) {
  const allBuildings = await fetchAllDataForStatistics<Building>(
    (params) => buildingApi.list({ ...params, ...searchParams }) as Promise<{ data?: { list: Building[] } }>
  );

  // 优化：单次遍历计算所有统计
  let active = 0;
  let inactive = 0;

  for (const building of allBuildings) {
    if (building.status === 0) active++;
    else if (building.status === 1) inactive++;
  }

  return {
    total: allBuildings.length,
    active,
    inactive,
  };
}

/**
 * 楼层统计数据
 */
export async function fetchFloorStatistics() {
  const allFloors = await fetchAllDataForStatistics<Floor>(floorApi.list);

  // 优化：单次遍历计算所有统计
  let active = 0;
  let inactive = 0;

  for (const floor of allFloors) {
    if (floor.status === 0) active++;
    else if (floor.status === 1) inactive++;
  }

  return {
    total: allFloors.length,
    active,
    inactive,
  };
}
