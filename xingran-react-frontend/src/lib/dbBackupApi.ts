/**
 * 数据备份与恢复 API（quick 261110-tbq CONTRACT.md 第 1/5 节前端签名）。
 *
 * 所有调用一律走 @/lib/api 的包装函数（返回 Promise<BaseResponse<T>>，业务数据在 .data），
 * 不直接引用 api 实例、不手动判 code===0；下载唯一权威为 @/lib/download 的 downloadFile。
 */
import { get, post, postFormData } from "@/lib/api";
import { downloadFile } from "@/lib/download";
import type { PageResponse, PageParams } from "@/types/base";
import type { BackupRecord, BackupStatistics, Capabilities, Plan, TaskView } from "@/types/backup";

export const dbBackupApi = {
  /** 备份记录分页：POST /system/db-backup/list */
  async list(params: PageParams): Promise<PageResponse<BackupRecord>> {
    const result = await post<PageResponse<BackupRecord>>("/system/db-backup/list", params);
    return result.data ?? { list: [], total: 0, current: 1, pageSize: 10 };
  },

  /** 能力预检：GET /system/db-backup/capabilities（pg_dump/pg_restore 是否可用） */
  async capabilities(): Promise<Capabilities> {
    const result = await get<Capabilities>("/system/db-backup/capabilities");
    return (
      result.data ?? {
        dbType: "",
        pgDumpAvailable: false,
        pgRestoreAvailable: false,
        pgDumpPath: "",
        pgRestorePath: "",
        hint: "能力预检数据为空",
      }
    );
  },

  /** 统计卡：POST /system/db-backup/statistics（后端为 map，键名做兼容读取） */
  async statistics(): Promise<BackupStatistics> {
    const result = await post<BackupStatistics>("/system/db-backup/statistics");
    const data = result.data ?? {};
    const num = (...keys: string[]): number | undefined => {
      for (const key of keys) {
        const value = data[key];
        if (typeof value === "number" && Number.isFinite(value)) return value;
      }
      return undefined;
    };
    return {
      ...data,
      total: num("total", "totalCount", "count"),
      success: num("success", "successCount", "ready"),
      // 占用空间：兼容 usedSpace(bytes) / used_space / totalSize / spaceUsed
      usedSpace: num("usedSpace", "used_space", "totalSize", "total_size", "spaceUsed"),
    };
  },

  /** 异步任务状态轮询：GET /system/db-backup/tasks/:id */
  async task(id: string): Promise<TaskView> {
    const result = await get<TaskView>(`/system/db-backup/tasks/${id}`);
    // 包装函数已在 code!==0 时 reject；此处只兜底异常空体
    return result.data as TaskView;
  },

  /** 触发导出：POST /system/db-backup/export */
  async export(): Promise<{ taskId: string }> {
    const result = await post<{ taskId: string }>("/system/db-backup/export");
    return result.data ?? { taskId: "" };
  },

  /** 上传备份 zip 并触发恢复：multipart POST /system/db-backup/restore，字段名 file */
  async restore(file: File): Promise<{ taskId: string }> {
    const formData = new FormData();
    formData.append("file", file);
    const result = await postFormData<{ taskId: string }>("/system/db-backup/restore", formData);
    return result.data ?? { taskId: "" };
  },

  /** 删除备份记录：POST /system/db-backup/:id/delete */
  async remove(id: string): Promise<void> {
    await post(`/system/db-backup/${id}/delete`);
  },

  /** 下载备份 zip：GET /system/db-backup/:id/download（二进制流，不套响应包装） */
  download(id: string, fileName: string): Promise<string> {
    return downloadFile(`/system/db-backup/${id}/download`, fileName);
  },

  /** 读取备份计划：GET /system/db-backup/plan */
  async getPlan(): Promise<Plan> {
    const result = await get<Plan>("/system/db-backup/plan");
    return result.data as Plan;
  },

  /**
   * 保存备份计划：POST /system/db-backup/plan
   *
   * 敏感字段（oss.accessKey / oss.secretKey / ftp.password）约定：
   * - 回填展示为 "******"；提交时若值仍为 "******"，前端改写为 ""（空串）；
   * - 与后端约定：敏感字段收到 "" 表示「保留原值、不覆盖」（后端 SavePlan 亦做保留）。
   * 这样既不把脱敏占位串写进配置，也不会误清空真实凭据。
   */
  async savePlan(p: Plan): Promise<void> {
    await post("/system/db-backup/plan", p);
  },
};

export default dbBackupApi;
