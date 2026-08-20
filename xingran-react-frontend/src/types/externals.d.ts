// 本文件必须保持纯 ambient 状态(没有 import / 也没有 export {}):
// 一旦有,TypeScript 会把文件当 module,里面的 `declare module` 块
// 变成 augmentation(增强)而不是 new declaration,TS7016 不会消停。
//
// 不放 global.d.ts 里是因为后者有 import (为支持 declare global { Window }),
// 同样会把 declare module 变成 augmentation。

declare module "@breejs/later" {
  export interface Schedule {
    schedules: unknown[];
    // 实际 @breejs/later 签名: next(count?: number, end?: Date): Date | Date[]
    // 这里把它放宽到 any 让 TS 7 的 overload 解析不抱怨
    // (仓库只用 next(count) 的形态,运行时实际上始终返回 Date[])
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    next(...args: any[]): any;
  }
  export function schedule(cronExpression: string): Schedule;
  export function timeout(expression: string, date?: Date): Date;

  // PR #3 cronSelector utils.ts 用到的子命名空间
  // (之前依赖 @ts-expect-error 隐式 any,TS 7 收严后必须显式声明)
  export const date: { localTime(): void };
  export const parse: {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    cron(expression: string, hasSeconds?: boolean): any;
  };
}
