/**
 * NetworkDevice 数据管理 Hook
 */

import { useState, useCallback } from "react";
import { post } from "@/lib/api";
import { handleApiError } from "@/utils/errorHandler";
import { useDeptTree } from "@/hooks/useDeptTree";
import type { AuthCredential, BaseResponse, PageResponse } from "@/types";

export interface DeviceStatistics {
  total: number;
  online: number;
  offline: number;
  unknown: number;
}

export interface UseDeviceDataReturn {
  departments: ReturnType<typeof useDeptTree>["data"];
  credentials: AuthCredential[];
  statistics: DeviceStatistics;
  loadCredentials: () => Promise<void>;
  loadStatistics: () => Promise<void>;
  // 注入式兜底(2026-06-30):编辑回填时若 credentialId 不在 pageSize:50 列表,
  // 调此方法基于 record.credentialName 注入一条临时 Option,避免 Select 显示 raw UUID。
  ensureCredential: (cred: Pick<AuthCredential, "id"> & Partial<AuthCredential>) => void;
}

export function useDeviceData(): UseDeviceDataReturn {
  // 部门树数据源统一收敛到 useDeptTree (Phase 37-04)
  const { data: departments = [] } = useDeptTree();
  const [credentials, setCredentials] = useState<AuthCredential[]>([]);
  const [statistics, setStatistics] = useState<DeviceStatistics>({
    total: 0,
    online: 0,
    offline: 0,
    unknown: 0,
  });

  // 加载统计数据(专用端点 COUNT 聚合,不受分页/筛选影响)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<{
        totalDevices?: number;
        onlineDevices?: number;
        offlineDevices?: number;
        unknownDevices?: number;
      }>("/network/devices/statistics");
      const data = result.data || {};
      setStatistics({
        total: data.totalDevices ?? 0,
        online: data.onlineDevices ?? 0,
        offline: data.offlineDevices ?? 0,
        unknown: data.unknownDevices ?? 0,
      });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  // 加载授权凭证列表
  const loadCredentials = useCallback(async () => {
    try {
      const result = await post<PageResponse<AuthCredential>>("/network/credentials/list", {
        current: 1,
        pageSize: 50,
      });
      setCredentials(result.data?.list || []);
    } catch (error) {
      handleApiError(error, "加载授权凭证列表", false);
    }
  }, []);

  // 注入式兜底(2026-06-30):若凭证 id 已在列表中则保持原引用,否则追加。
  // 必填字段(protocolType/snmpCommunities/snmpVersion/isDefault/createdAt/updatedAt)
  // 用合理默认值占位,Select 渲染仅依赖 id+credentialName,不影响业务提交。
  // setCredentials 列入 deps 以满足 React Compiler 推断(setState 引用稳定)。
  const ensureCredential = useCallback(
    (cred: Pick<AuthCredential, "id"> & Partial<AuthCredential>) => {
      setCredentials((prev) =>
        prev.find((c) => c.id === cred.id)
          ? prev
          : [
              ...prev,
              {
                id: cred.id,
                credentialName: cred.credentialName || "未命名凭证",
                protocolType: cred.protocolType ?? ("ssh" as AuthCredential["protocolType"]),
                username: cred.username ?? "",
                snmpCommunities: cred.snmpCommunities ?? [],
                snmpVersion: cred.snmpVersion ?? ("v2c" as AuthCredential["snmpVersion"]),
                isDefault: cred.isDefault ?? false,
                createdAt: cred.createdAt ?? "",
                updatedAt: cred.updatedAt ?? "",
              },
            ]
      );
    },
    [setCredentials]
  );

  return {
    departments,
    credentials,
    statistics,
    loadCredentials,
    loadStatistics,
    ensureCredential,
  };
}
