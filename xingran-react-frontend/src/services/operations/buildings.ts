import { post } from "@/lib/api";
import type { PageParams } from "@/types/base";
import type { PageData, Building } from "@/types/operations";

export const buildingApi = {
  // 列表查询 - post 返回 Promise<BaseResponse<T>>，T 即 PageData<Building>
  list: (params: PageParams) => post<PageData<Building>>("/ops/buildings/list", params),

  // 创建
  create: (data: Omit<Building, "id" | "createdAt" | "updatedAt">) => post("/ops/buildings", data),

  // 更新
  update: (id: string, data: Partial<Building>) => post(`/ops/buildings/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/buildings/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) => post("/ops/buildings/batch", { ids, action: "delete" }),

  // 导出
  export: (params: PageParams) => post("/ops/buildings/export", params),

  // 导入
  import: (file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return post("/ops/buildings/import", formData);
  },

  // 下载模板
  downloadTemplate: () => post("/ops/buildings/template", {}),
};
