/**
 * Phase 66 契约组件 — RemoteSelect（防抖远程搜索下拉，UI-SPEC §4）
 *
 * 用法：
 * ```tsx
 * <RemoteSelect
 *   value={v} onChange={setV}
 *   fetcher={(kw) => post("/system/users/list", { userName: kw }).then(r =>
 *     r.data.list.map(u => ({ label: u.nickName, value: u.userId })))}
 *   initialOptions={cachedUsers}
 *   placeholder="搜索用户"
 * />
 * ```
 *
 * 契约（65 已锁定）：fetcher(keyword)=>Promise<SearchFieldOption[]>；debounceMs 默认
 * 300；initialOptions 无关键字时展示；递增 requestId 竞态守卫（仅最后发起的请求
 * 可写入 options）；loading 内建；fetcher 失败 catch→handleApiError(error, "搜索选项")
 * 并保留上次成功选项；keyword 清空→回 initialOptions 不发请求；无跨实例缓存。
 */
import { useEffect, useRef, useState } from "react";
import type { SelectProps } from "antd";
import { AppSelect } from "@/components/ui/AppSelect";
import { handleApiError } from "@/utils/errorHandler";

export type RemoteSelectOption = { label: string; value: string | number | boolean | null };

export interface RemoteSelectProps
  extends Omit<SelectProps, "options" | "onSearch" | "loading"> {
  /** 远程搜索：keyword → 选项列表（抛错走统一错误出口） */
  fetcher: (keyword: string) => Promise<RemoteSelectOption[]>;
  /** 防抖毫秒数（默认 300） */
  debounceMs?: number;
  /** 初始选项（keyword 为空时展示，不发请求） */
  initialOptions?: RemoteSelectOption[];
}

export function RemoteSelect({
  fetcher,
  debounceMs = 300,
  initialOptions,
  ...rest
}: RemoteSelectProps) {
  const [options, setOptions] = useState<RemoteSelectOption[]>(initialOptions ?? []);
  const [loading, setLoading] = useState(false);
  // 竞态守卫：递增 id，只有最新一次请求可写入 options
  const requestIdRef = useRef(0);
  // 上次成功选项（失败时保留展示）
  const lastOptionsRef = useRef<RemoteSelectOption[]>(initialOptions ?? []);
  const mountedRef = useRef(true);

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const runFetch = (keyword: string) => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    fetcher(keyword)
      .then((opts) => {
        if (!mountedRef.current || requestId !== requestIdRef.current) return;
        lastOptionsRef.current = opts;
        setOptions(opts);
      })
      .catch((error) => {
        // 失败统一出口；保留上次成功选项
        handleApiError(error, "搜索选项");
      })
      .finally(() => {
        if (mountedRef.current && requestId === requestIdRef.current) {
          setLoading(false);
        }
      });
  };

  const handleSearch = (keyword: string) => {
    const trimmed = (keyword ?? "").trim();
    if (timerRef.current) clearTimeout(timerRef.current);
    if (trimmed === "") {
      // keyword 清空：回 initialOptions，不发请求（清空立即处理，不参与防抖）
      requestIdRef.current += 1; // 使在途请求失效
      setLoading(false);
      lastOptionsRef.current = initialOptions ?? [];
      setOptions(initialOptions ?? []);
      return;
    }
    // 防抖调 fetcher（默认 300ms）
    timerRef.current = setTimeout(() => {
      runFetch(trimmed);
    }, debounceMs);
  };

  return <AppSelect options={options} loading={loading} showSearch filterOption={false} onSearch={handleSearch} {...rest} />;
}

export default RemoteSelect;
