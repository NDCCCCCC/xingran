import { post } from "@/lib/api";
import type { PageParams } from "@/types/base";
import type { PageData, InfoPoint } from "@/types/operations";

export const infoPointApi = {
  // 列表查询 - post 返回 Promise<BaseResponse<T>>，T 即 PageData<InfoPoint>
  list: (params: PageParams) => post<PageData<InfoPoint>>("/ops/info-points/list", params),

  // 创建
  create: (data: Omit<InfoPoint, "id" | "createdAt" | "updatedAt">) =>
    post("/ops/info-points", data),

  // 更新
  update: (id: string, data: Partial<InfoPoint>) => post(`/ops/info-points/${id}/update`, data),

  // 删除
  delete: (id: string) => post(`/ops/info-points/${id}/delete`),

  // 批量删除
  batchDelete: (ids: string[]) => post("/ops/info-points/batch", { ids, action: "delete" }),
};
