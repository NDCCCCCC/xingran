import { useEffect } from "react";
import { BrowserRouter as Router } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useMenuStore } from "@/store/menuStore";
import { routeConfigManager } from "@/router/routeConfigManager";
import { ThemeProvider } from "@/design-system/components/ThemeProvider";
import LayoutProvider from "@/design-system/components/LayoutProvider";
import AntdThemeBridge from "@/design-system/components/AntdThemeBridge";
import ConfigProvider from "@/components/ConfigProvider";
import { DynamicRoutes } from "@/router/DynamicRoutes";
import "./App.css";

// 创建React Query客户端
// 默认配置遵循 D-11：5 分钟 stale、30 分钟 gc、切窗口不重拉。
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000,
      gcTime: 30 * 60 * 1000,
    },
  },
});

function App() {
  const { allMenus } = useMenuStore();

  // 初始化路由配置管理器
  // 只在菜单数量变化时重新初始化，避免循环渲染
  useEffect(() => {
    if (allMenus.length > 0) {
      routeConfigManager.initialize(allMenus);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allMenus.length]);

  return (
    <QueryClientProvider client={queryClient}>
      {/*
        AntdThemeBridge 读取 settingsStore 中的 mode / density，
        并把 xingranBrand 品牌令牌喂给 AntD ThemeConfig。
        这是解决"AntD 组件硬编码蓝色"（如工位管理页面表格/卡片/平面图
        Radio.Group）的关键；v1.22 Phase 65 起主色固定为品牌绿 #156031。
        locale 通过 AntdThemeBridge 内部直接传给 ConfigProvider。
      */}
      <AntdThemeBridge>
        <ConfigProvider>
          <ThemeProvider>
            <LayoutProvider>
              {/* basename 取 Vite 的 base（VITE_BASE），dev 时 '/' 等价于未设；生产为 '/xingran/'
                  时让 React Router 内部所有 <Link>/navigate/history 都自动加此前缀，
                  与 nginx 子路径部署模型一致。 */}
              <Router
                basename={import.meta.env.BASE_URL !== "/" ? import.meta.env.BASE_URL : undefined}
              >
                <DynamicRoutes />
              </Router>
            </LayoutProvider>
          </ThemeProvider>
        </ConfigProvider>
      </AntdThemeBridge>
    </QueryClientProvider>
  );
}

export default App;
