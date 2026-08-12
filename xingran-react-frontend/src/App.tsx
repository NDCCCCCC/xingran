import { useEffect } from "react";
import { BrowserRouter as Router } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import zhCN from "antd/locale/zh_CN";
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
        AntdThemeBridge 读取 settingsStore 中的 customColors / mode / density
        并把 AntD ThemeConfig 喂给 ConfigProvider。这是解决"AntD 组件硬编码蓝色"
        （如工位管理页面表格/卡片/平面图 Radio.Group）跟随用户主色变化的关键。
        locale 通过 AntdThemeBridge 内部直接传给 ConfigProvider。
      */}
      <AntdThemeBridge>
        <ConfigProvider>
          <ThemeProvider>
            <LayoutProvider>
              <Router>
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
