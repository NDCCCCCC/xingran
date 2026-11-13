/// <reference types="vite/client" />

/**
 * 构建时常量类型声明。
 *
 * __APP_VERSION__ 由 xingran-react-frontend/vite.config.ts 的 `define` 在编译期
 * 替换为 JSON 字符串字面量（来源 APP_VERSION / VITE_APP_VERSION 环境变量，
 * 缺省 "dev"）。本文件只做类型声明，不产生任何运行时代码，也不读取 package.json。
 */
declare const __APP_VERSION__: string;
