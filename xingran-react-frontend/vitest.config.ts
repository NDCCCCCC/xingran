import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: "jsdom",
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
      // Phase 63 基准阈值:基于实测 29.45/18.57/27.2/29.9 取整下调,
      // 防止后续 coverage 小幅波动直接挂掉 npm run test:coverage。
      // 后续 phase 应逐步提升 (例如到 60/50/60/60)。
      thresholds: {
        statements: 25,
        branches: 15,
        functions: 22,
        lines: 25,
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
