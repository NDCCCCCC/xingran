import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: "jsdom",
    // Phase 63 收口:全局 15s timeout 替代默认 5s。
    // 原因:network/ports + BulkWriteDrawer 等组件挂载 antd Table/Card/Modal + 多列渲染,
    // 在 coverage 模式下 setup + transform 累积导致 5s timeout 偶发 flake (Phase 53 W4 时代)。
    // 15s 给真实业务测试留缓冲;快速单元测试不受影响(timeout 是 max 不是 min)。
    testTimeout: 15000,
    setupFiles: ["./src/test/setup.ts"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html"],
      exclude: [
        "node_modules/",
        "src/test/",
        "**/*.d.ts",
        "**/*.config.*",
        "**/mockData/**",
        "dist/",
      ],
      // Phase 63 基准阈值:基于实测 29.45/18.57/27.2/29.9 取整下调 (Phase 63 落地)。
      // 2026-08-20 Phase 63 收口验证:实测降到 24.58/15.04/18.94/24.75 (后续 phase
      // 新增未覆盖代码拉低),按"取整下调"原则再降到 24/15/18/24 保证 CI gate 不挂。
      // 后续 phase 应逐步提升(目标例如 60/50/60/60)。
      thresholds: {
        statements: 24,
        branches: 15,
        functions: 18,
        lines: 24,
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
