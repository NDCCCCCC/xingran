import { createRoot } from "react-dom/client";
import "./index.css";
import "./design-system/themes/theme-styles.css";
import App from "./App.tsx";
import { initEncryptionConfig } from "@/lib/api";
import "@/lib/echarts"; // ECharts 按需加载配置

/**
 * 初始化应用
 * 在渲染应用前加载必要的配置
 */
async function initializeApp(): Promise<void> {
  try {
    // 初始化加密配置（从后端动态获取）
    await initEncryptionConfig();
  } catch (error) {
    console.error("[App] 应用初始化失败:", error);
    // 即使初始化失败也继续启动应用（使用默认配置）
  }
}

// 在初始化完成后渲染应用
initializeApp().then(() => {
  createRoot(document.getElementById("root")!).render(
    <App />
  );
}).catch((error) => {
  console.error("[App] 应用启动失败:", error);
  // 即使初始化失败也渲染应用
  createRoot(document.getElementById("root")!).render(
    <App />
  );
});
