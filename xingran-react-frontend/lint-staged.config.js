/**
 * lint-staged 配置
 *
 * 注意 tsc 任务使用函数形式:lint-staged 会把匹配的文件路径列表追加到命令尾部,
 * `tsc --noEmit <files>` 对子集文件没有意义(类型依赖全项目),会报错。
 * 函数返回纯字符串可让 lint-staged 不追加文件,等价于全量 type-check。
 */
export default {
  "*.{ts,tsx,js,jsx}": ["eslint --fix", "prettier --write"],
  "*.{ts,tsx}": () => "npm run type-check",
  "*.{json,md,yml,yaml}": ["prettier --write"],
};
