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
      // GOV-01 全量口径: Vitest 4 已移除 coverage.all,include 是全量口径唯一开关——
      // 未配置 include 时只报告被 import 的文件(旧口径 24.58% 失真的来源),
      // 584 个 src 文件(含 0 语句文件)全部进入 coverage-final.json,未测文件以 0% 计入。
      // D-10: 白名单排除的单一真相源在下方 exclude 数组,gate 脚本做漂移检测。
      include: ["src/**/*.{ts,tsx}"],
      reporter: ["text", "json", "html"],
      exclude: [
        "src/test/",
        "**/*.d.ts",
        // QUAL-02 白名单 (D-10: 白名单排除的单一真相源在此——gate 脚本 82-02 做
        // 漂移检测, cad 条目出现在 coverage json 中即视为配置漂移失败):
        // 合计 1028/22602 ≈ 4.55% ≤ 5% 上限; 复审条件 (D-11): 目录自身语句覆盖率达
        // ≥70% 即可启动移除, milestone 收口时强制重审。
        // cad-editor: 804 stmts / 8 文件 (重画布低确定性 UI)
        // cad-elements: 224 stmts / 5 文件 (重画布低确定性 UI)
        "src/components/cad-editor/**",
        "src/components/cad-elements/**",
      ],
      // D-16: 原生 thresholds 配置已整段删除——gate 唯一真相源移交外部 bash 脚本
      // (check-frontend-coverage.sh) + .coverage-fe-floors 数据文件;保留旧值会让
      // test:coverage 在口径切换瞬间因全量实测 3.85% < 24 直接失败。
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
