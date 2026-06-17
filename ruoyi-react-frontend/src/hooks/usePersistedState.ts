/**
 * 带 sessionStorage 持久化的 useState
 *
 * 用途:列表页中那些不放在 searchForm 里的辅助筛选值(如 `selectedDeptId`、`viewMode`)
 * 也需要跨编辑保存/F5 刷新保留下来。本 hook 提供与 `useState` 完全一致的 API,但写入
 * sessionStorage(会话内保留,关闭标签页丢失),并在 mount 时自动恢复。
 *
 * 设计原则:
 * - 存储 key 由 `keyPrefix + keySuffix` 组成,默认 prefix 由调用方传入(典型用法:`usePersistedState({ keyPrefix: location.pathname, keySuffix: 'selectedDeptId' })`)
 * - JSON 序列化/反序列化带 try/catch 防御,损坏的 storage 数据不会破坏页面渲染
 * - SSR 友好:typeof window 检查 + try/catch,即使在非浏览器环境下也能正常使用
 * - 与 `useTableManager` / `usePagination` 的持久化策略保持一致(`sessionStorage` + 相同前缀命名)
 *
 * API 形式(2026-06 优化):
 * - `usePersistedState<T>(opts)` 只返回 `value`,适合只读筛选(消除大量 setX 未用警告)
 * - `usePersistedStateController<T>(opts)` 返回 `[value, setValue, reset]` tuple,适合需要写入的页面
 */
import { useState, useCallback, useEffect } from "react";
import { TABLE_STATE_PREFIX, sanitizePathForKey } from "@/constants/storage";

export interface UsePersistedStateOptions<T> {
  /** 路径前缀,通常传 `location.pathname` */
  keyPrefix: string;
  /** 槽位名,如 `'selectedDeptId'`、`'viewMode'`,在同一页面内必须唯一 */
  keySuffix: string;
  /** 默认值,storage 中无值或解析失败时使用 */
  defaultValue: T;
  /** 是否使用 sessionStorage(默认 true),false 切到 localStorage(不推荐,跨标签污染) */
  useSessionStorage?: boolean;
  /**
   * 自定义序列化器(可选)。例如需要对值做"清洗"(过滤空值)或自定义结构。
   * 默认 `JSON.stringify` / `JSON.parse`。
   */
  serialize?: (value: T) => string;
  deserialize?: (raw: string) => T;
}

export type UsePersistedStateController<T> = readonly [
  T,
  (next: T | ((prev: T) => T)) => void,
  () => void,
];

/** 安全读取 sessionStorage/localStorage */
function safeGet(useSession: boolean, key: string): string | null {
  try {
    if (typeof window === "undefined") return null;
    return useSession ? window.sessionStorage.getItem(key) : window.localStorage.getItem(key);
  } catch {
    return null;
  }
}

/** 安全写入 sessionStorage/localStorage */
function safeSet(useSession: boolean, key: string, value: string): void {
  try {
    if (typeof window === "undefined") return;
    const store = useSession ? window.sessionStorage : window.localStorage;
    store.setItem(key, value);
  } catch {
    // 配额溢出 / 隐私模式 — 静默吞掉,不阻塞 UI
  }
}

/** 安全删除 */
function safeRemove(useSession: boolean, key: string): void {
  try {
    if (typeof window === "undefined") return;
    const store = useSession ? window.sessionStorage : window.localStorage;
    store.removeItem(key);
  } catch {
    // ignore
  }
}

/** 在 lazy initializer 内部读取 storage 的 helper — 只在第一次 render 调用,避免 ref-in-render 警告 */
function readInitial<T>(
  useSession: boolean,
  storageKey: string,
  defaultValue: T,
  deserialize?: (raw: string) => T,
): T {
  const raw = safeGet(useSession, storageKey);
  if (raw === null) return defaultValue;
  try {
    return deserialize ? deserialize(raw) : (JSON.parse(raw) as T);
  } catch {
    // 损坏的数据 — 清理掉并使用默认值
    safeRemove(useSession, storageKey);
    return defaultValue;
  }
}

/**
 * 内部实现:返回完整 controller tuple。
 * `usePersistedState` 和 `usePersistedStateController` 都基于此。
 */
function usePersistedStateInternal<T>(
  options: UsePersistedStateOptions<T>,
): UsePersistedStateController<T> {
  const {
    keyPrefix,
    keySuffix,
    defaultValue,
    useSessionStorage = true,
    serialize,
    deserialize,
  } = options;

  const storageKey = `${TABLE_STATE_PREFIX}${sanitizePathForKey(keyPrefix)}_${keySuffix}`;

  // 同步获取初始值(只在第一次 render 时执行一次)— useState 的 lazy initializer 不会触发
  // "refs during render" 警告,因为它只在 mount 时执行一次
  const [value, setValueInternal] = useState<T>(() =>
    readInitial(useSessionStorage, storageKey, defaultValue, deserialize)
  );

  // 写入时同时更新 state 和 storage
  const setValue = useCallback(
    (next: T | ((prev: T) => T)) => {
      setValueInternal((prev) => {
        const resolved = typeof next === "function" ? (next as (p: T) => T)(prev) : next;
        const encoded = serialize ? serialize(resolved) : JSON.stringify(resolved);
        safeSet(useSessionStorage, storageKey, encoded);
        return resolved;
      });
    },
    [storageKey, serialize, useSessionStorage]
  );

  // reset = 清除 storage + 恢复默认值
  const reset = useCallback(() => {
    safeRemove(useSessionStorage, storageKey);
    setValueInternal(defaultValue);
  }, [storageKey, defaultValue, useSessionStorage]);

  // 跨标签/跨实例同步:监听 `storage` 事件,其他标签或同一标签的 hook 实例
  // 写入同一 key 时同步更新当前状态(useful for 同一页面多个 usePersistedState 引用同一 key)
  useEffect(() => {
    if (typeof window === "undefined") return;
    const handler = (e: StorageEvent) => {
      if (e.key !== storageKey) return;
      if (e.newValue === null) {
        setValueInternal(defaultValue);
        return;
      }
      try {
        const parsed = deserialize ? deserialize(e.newValue) : (JSON.parse(e.newValue) as T);
        setValueInternal(parsed);
      } catch {
        // ignore corrupted cross-tab update
      }
    };
    window.addEventListener("storage", handler);
    return () => window.removeEventListener("storage", handler);
  }, [storageKey, defaultValue, deserialize]);

  return [value, setValue, reset] as const;
}

/**
 * 只读入口:返回持久化的 value,不带 setter。
 * 大多数筛选场景只读取 value,使用此入口可避免 `setX is assigned a value but never used` 警告。
 */
export function usePersistedState<T>(options: UsePersistedStateOptions<T>): T {
  return usePersistedStateInternal(options)[0];
}

/**
 * 读写入口:返回 [value, setValue, reset] tuple。
 * 需要在用户交互时更新持久化值时使用(setValue / reset 不必都使用,ESLint 视为 useState 风格豁免)。
 */
export function usePersistedStateController<T>(
  options: UsePersistedStateOptions<T>,
): UsePersistedStateController<T> {
  return usePersistedStateInternal(options);
}