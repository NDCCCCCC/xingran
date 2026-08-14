import { post } from "@/lib/api";
import type { PageParams } from "@/types/base";
import type { PageData, DedicatedLine } from "@/types/operations";

export const dedicatedLineApi = {
  // 列表查询 - post 返回 Promise<BaseResponse<T>>，T 即 PageData<DedicatedLine>
  list: (params: PageParams) => post<PageData<DedicatedLine>>("/ops/dedicated-lines/list", params),

  // 创建
  create: (data: Omit<DedicatedLine, "id" | "createdAt" | "updatedAt">) =>
    post("/ops/dedicated-lines", data),

  // 更新
  update: (id: string, data: Partial<DedicatedLine>) =>
    post(`/ops/dedicated-lines/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/dedicated-lines/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) => post("/ops/dedicated-lines/batch", { ids, action: "delete" }),

  // 导出
  export: (params: PageParams) => post("/ops/dedicated-lines/export", params),

  // 导入
  import: (file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return post("/ops/dedicated-lines/import", formData);
  },

  // 下载模板
  downloadTemplate: () => post("/ops/dedicated-lines/template", {}),
};
