import { useEffect } from 'react';
import { BrowserRouter as Router } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ConfigProvider as AntConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { useMenuStore } from '@/store/menuStore';
import { routeConfigManager } from '@/router/routeConfigManager';
import { ThemeProvider } from '@/design-system/components/ThemeProvider';
import LayoutProvider from '@/design-system/components/LayoutProvider';
import ConfigProvider from '@/components/ConfigProvider';
import { DynamicRoutes } from '@/router/DynamicRoutes';
import './App.css';

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
      <AntConfigProvider locale={zhCN}>
        <ConfigProvider>
          <ThemeProvider>
            <LayoutProvider>
              <Router>
                <DynamicRoutes />
              </Router>
            </LayoutProvider>
          </ThemeProvider>
        </ConfigProvider>
      </AntConfigProvider>
    </QueryClientProvider>
  );
}

export default App;
