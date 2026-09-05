/**
 * 通用 CRUD 资源工厂（Phase 94 D-01/D-03 单一权威）
 *
 * createResourceApi<T> 自 opsApi.ts 私有 createCrudApi<T> 逐行提升：
 * 8 方法（list/get/create/update/delete/batch/statistics/searchOptions）,
 * 与 react-admin/refine 核心五方法 1:1 对齐;import/export 不进工厂核心。
 *
 * 唯一类型强化（D-02）:create/update 参数由 Partial<T> 替换为
 * Partial<CreatePayload<T>> — 对象字面量误传 id/时间戳编译期报错。
 *
 * 传输层约束:工厂只消费 ./api 的 post（SM2+SM4 加密与 401 刷新重放全在
 * 主实例拦截器）;不 import 任何 *Api.ts 文件（避免反向依赖成环）。
 */

import { post } from "./api";
import type { PageParams, PageResponse } from "@/types";
import type { CreatePayload } from "@/types/apiFactory";

/**
 * 通用下拉选项 — 后端 dropdown-options 端点的响应元素。
 * 与 antd Select 的 options={[{value, label}]} 直接对齐。
 *
 * 后端硬 LIMIT 50;keyword 通过 onSearch 远程查询。
 * 替代反模式: pageSize:1000 + filterOption 客户端 substring 匹配。
 */
export interface DropdownOption {
  value: string;
  label: string;
}

export interface CrudApiConfig {
  basePath: string;
  /** 自定义 dropdown-options 端点路径,默认 "/dropdown-options" */
  dropdownPath?: string;
}

export function createResourceApi<T>(config: CrudApiConfig) {
  const { basePath, dropdownPath = "/dropdown-options" } = config;

  return {
    list: async (params: PageParams & Record<string, unknown>) => {
      return await post<PageResponse<T>>(`${basePath}/list`, params);
    },

    get: async (id: string) => {
      return await post<T>(`${basePath}/${id}`, {});
    },

    create: async (data: Partial<CreatePayload<T>>) => {
      return await post(basePath, data);
    },

    update: async (id: string, data: Partial<CreatePayload<T>>) => {
      return await post(`${basePath}/${id}/update`, data);
    },

    delete: async (id: string) => {
      return await post(`${basePath}/${id}/delete`, {});
    },

    batch: async (action: string, data: Record<string, unknown>) => {
      return await post(`${basePath}/batch`, { action, ...data });
    },

    // 统计(专用 COUNT 端点,可选筛选参数;返回各 status 桶计数 data)
    statistics: async (params: Record<string, unknown> = {}) => {
      const res = await post<Record<string, number>>(`${basePath}/statistics`, params);
      return res.data ?? {};
    },

    /**
     * 下拉数据源远程搜索 — 配合 antd Select 的 showSearch + filterOption={false} + onSearch。
     * 后端硬 LIMIT 50;keyword 通过 onSearch 防抖传入。
     * @example
     *   const [opts, setOpts] = useState<DropdownOption[]>([]);
     *   const debouncedSearch = useMemo(() => debounce(setOpts, 300), []);
     *   <Select showSearch filterOption={false} options={opts}
     *           onSearch={(kw) => workstationApi.searchOptions({ name: kw }).then(setOpts)} />
     */
    searchOptions: async (params: Record<string, unknown> = {}) => {
      const res = await post<DropdownOption[]>(`${basePath}${dropdownPath}`, params);
      return res.data ?? [];
    },
  };
}
