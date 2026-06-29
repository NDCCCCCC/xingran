/**
 * 简易 debounce — 替代 lodash.debounce(本项目未安装 lodash)。
 *
 * 仅保留 leading/trailing 选项,API 与 lodash 一致便于未来切换。
 * 用于 Select/AutoComplete 的 onSearch 远程搜索防抖,默认 300ms。
 *
 * @example
 *   const debounced = debounce((keyword: string) => fetch(keyword), 300);
 *   input.addEventListener('input', e => debounced(e.target.value));
 */

export interface DebounceOptions {
  /** 在 wait 期间首次调用立即执行 */
  leading?: boolean;
  /** 在 wait 结束后调用最后一次(默认 true) */
  trailing?: boolean;
}

export function debounce<T extends (...args: never[]) => void>(
  fn: T,
  wait = 300,
  options: DebounceOptions = { trailing: true },
): (...args: Parameters<T>) => void {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let lastArgs: Parameters<T> | null = null;

  return function debounced(this: unknown, ...args: Parameters<T>) {
    lastArgs = args;

    if (options.leading && timer === null) {
      fn.apply(this, args);
    }

    if (timer !== null) {
      clearTimeout(timer);
    }

    timer = setTimeout(() => {
      timer = null;
      if (options.trailing && lastArgs !== null) {
        fn.apply(this, lastArgs);
        lastArgs = null;
      }
    }, wait);
  };
}