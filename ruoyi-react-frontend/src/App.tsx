import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { useAuthStore } from '@/store/authStore';
import Layout from '@/components/layout';
import Login from '@/pages/login';
import Dashboard from '@/pages/dashboard';
// 系统管理页面
import UserManagement from '@/pages/system/user';
import RoleManagement from '@/pages/system/role';
import MenuManagement from '@/pages/system/menu';
import DepartmentManagement from '@/pages/system/dept';
// 系统监控页面
import MonitorDashboard from '@/pages/monitor/dashboard';
import ServerMonitor from '@/pages/monitor/server';
import LogMonitor from '@/pages/monitor/logs';
import JobManager from '@/pages/monitor/job';
import CacheManager from '@/pages/monitor/cache';
import './App.css';

// 创建React Query客户端
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

// 受保护的路由组件
const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();

  // 检查 Zustand store 和 localStorage 中的认证状态
  const token = localStorage.getItem('accessToken');

  if (!isAuthenticated || !token) {
    // 如果 store 或 localStorage 中没有认证信息，重定向到登录页
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
};

// 公共路由组件（已登录用户不能访问）
const PublicRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore();
  const token = localStorage.getItem('accessToken');

  // 如果 store 和 localStorage 中都有认证信息，重定向到 dashboard
  if (isAuthenticated && token) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
};

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ConfigProvider locale={zhCN}>
        <Router>
          <Routes>
            {/* 登录页面 - 未登录才能访问 */}
            <Route
              path="/login"
              element={
                <PublicRoute>
                  <Login />
                </PublicRoute>
              }
            />

            {/* 需要认证的路由 */}
            <Route
              path="/*"
              element={
                <ProtectedRoute>
                  <Layout>
                    <Routes>
                      <Route path="/" element={<Navigate to="/dashboard" replace />} />
                      <Route path="/dashboard" element={<Dashboard />} />

                      {/* 系统管理路由 */}
                      <Route path="/system" element={<Navigate to="/system/user" replace />} />
                      <Route path="/system/user" element={<UserManagement />} />
                      <Route path="/system/role" element={<RoleManagement />} />
                      <Route path="/system/menu" element={<MenuManagement />} />
                      <Route path="/system/dept" element={<DepartmentManagement />} />

                      {/* 系统监控路由 */}
                      <Route path="/monitor" element={<Navigate to="/monitor/dashboard" replace />} />
                      <Route path="/monitor/dashboard" element={<MonitorDashboard />} />
                      <Route path="/monitor/server" element={<ServerMonitor />} />
                      <Route path="/monitor/logs" element={<LogMonitor />} />
                      <Route path="/monitor/job" element={<JobManager />} />
                      <Route path="/monitor/cache" element={<CacheManager />} />

                      <Route path="*" element={<Navigate to="/dashboard" replace />} />
                    </Routes>
                  </Layout>
                </ProtectedRoute>
              }
            />
          </Routes>
        </Router>
      </ConfigProvider>
    </QueryClientProvider>
  );
}

export default App;