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
import PostManagement from '@/pages/system/post';
import DictManagement from '@/pages/system/dict';
import ConfigManagement from '@/pages/system/config';
import NoticeManagement from '@/pages/system/notice';
import WorkstationManagement from '@/pages/system/workstation';
// 系统设置页面
import SystemSettingsPage from '@/pages/system/settings-page';
// 系统监控页面
import MonitorDashboard from '@/pages/monitor/dashboard';
import ServerMonitor from '@/pages/monitor/server';
import LogMonitor from '@/pages/monitor/logs';
import JobManager from '@/pages/monitor/job';
import CacheManager from '@/pages/monitor/cache';
// 网络设备管理页面
import DeviceManagement from '@/pages/network/devices';
import CredentialManagement from '@/pages/network/credentials';
import TemplateManagement from '@/pages/network/templates';
import CommandDispatch from '@/pages/network/command';
import ConfigExecutionPage from '@/pages/network/executions';
import ConfigBackupPage from '@/pages/network/backups';
import DeviceDiscoveryPage from '@/pages/network/discoveries';
import MACAddressPage from '@/pages/network/mac';
import PortStatusPage from '@/pages/network/ports';
// 个人中心和系统设置页面
import ProfilePage from '@/pages/profile';
import SettingsPage from '@/pages/settings';
// 用户通知中心页面
import MyNoticesPage from '@/pages/my-notices';
import NoticeDetailPage from '@/pages/my-notices/detail';
// 管理端通知页面
import SystemNoticesPage from '@/pages/system/notice';
import AdminNoticeDetailPage from '@/pages/system/notice/detail';
// 值班管理页面
import DutyPoolPage from '@/pages/duty/pools';
import DutySchedulePage from '@/pages/duty/schedules';
import DutyHolidayPage from '@/pages/duty/holidays';
import DutyConfigPage from '@/pages/duty/config';
import MyDutyPage from '@/pages/duty/my-duty';
// 运维工单管理页面
import WorkOrderPage from '@/pages/workorder/orders';
import WorkOrderCategoryPage from '@/pages/workorder/categories';
import WorkOrderStatisticsPage from '@/pages/workorder/statistics';
import PeriodicTemplatePage from '@/pages/workorder/periodic/templates';
// 知识库管理页面
import KnowledgeArticlePage from '@/pages/knowledge/articles';
import KnowledgeViewPage from '@/pages/knowledge/view';
// AD域管理页面
import ADConfigPage from '@/pages/ad-domain/configs';
import ADOUPage from '@/pages/ad-domain/ous';
import ADGroupPage from '@/pages/ad-domain/groups';
import ADUserPage from '@/pages/ad-domain/users';
import ADSyncLogsPage from '@/pages/ad-domain/logs';
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

                      {/* 个人中心和系统设置路由 */}
                      <Route path="/profile" element={<ProfilePage />} />
                      <Route path="/settings" element={<SettingsPage />} />

                      {/* 用户通知中心路由 */}
                      <Route path="/my-notices" element={<MyNoticesPage />} />
                      <Route path="/my-notices/:id" element={<NoticeDetailPage />} />

                      {/* 系统管理路由 */}
                      <Route path="/system" element={<Navigate to="/system/user" replace />} />
                      <Route path="/system/user" element={<UserManagement />} />
                      <Route path="/system/role" element={<RoleManagement />} />
                      <Route path="/system/menu" element={<MenuManagement />} />
                      <Route path="/system/dept" element={<DepartmentManagement />} />
                      <Route path="/system/post" element={<PostManagement />} />
                      <Route path="/system/dict" element={<DictManagement />} />
                      <Route path="/system/config" element={<ConfigManagement />} />
                      <Route path="/system/notice" element={<NoticeManagement />} />
                      <Route path="/system/notice/:id" element={<AdminNoticeDetailPage />} />
                      <Route path="/system/workstation" element={<WorkstationManagement />} />
                      <Route path="/system/settings-page" element={<SystemSettingsPage />} />

                      {/* 系统监控路由 */}
                      <Route path="/monitor" element={<Navigate to="/monitor/dashboard" replace />} />
                      <Route path="/monitor/dashboard" element={<MonitorDashboard />} />
                      <Route path="/monitor/server" element={<ServerMonitor />} />
                      <Route path="/monitor/logs" element={<LogMonitor />} />
                      <Route path="/monitor/job" element={<JobManager />} />
                      <Route path="/monitor/cache" element={<CacheManager />} />

                      {/* 网络设备管理路由 - 注意：不要为 /network 添加重定向，它应该作为父菜单展开 */}
                      <Route path="/network/devices" element={<DeviceManagement />} />
                      <Route path="/network/credentials" element={<CredentialManagement />} />
                      <Route path="/network/templates" element={<TemplateManagement />} />
                      <Route path="/network/command" element={<CommandDispatch />} />
                      <Route path="/network/executions" element={<ConfigExecutionPage />} />
                      <Route path="/network/backups" element={<ConfigBackupPage />} />
                      <Route path="/network/discoveries" element={<DeviceDiscoveryPage />} />
                      <Route path="/network/mac" element={<MACAddressPage />} />
                      <Route path="/network/ports" element={<PortStatusPage />} />

                      {/* 值班管理路由 */}
                      <Route path="/duty/pools" element={<DutyPoolPage />} />
                      <Route path="/duty/schedules" element={<DutySchedulePage />} />
                      <Route path="/duty/holidays" element={<DutyHolidayPage />} />
                      <Route path="/duty/config" element={<DutyConfigPage />} />
                      <Route path="/duty/my-duty" element={<MyDutyPage />} />

                      {/* 运维管理 - 值班路由（带 ops 前缀） */}
                      <Route path="/ops/duty/pools" element={<DutyPoolPage />} />
                      <Route path="/ops/duty/schedules" element={<DutySchedulePage />} />
                      <Route path="/ops/duty/holidays" element={<DutyHolidayPage />} />
                      <Route path="/ops/duty/config" element={<DutyConfigPage />} />
                      <Route path="/ops/duty/my-duty" element={<MyDutyPage />} />

                      {/* 运维工单管理路由 */}
                      <Route path="/workorder/orders" element={<WorkOrderPage />} />
                      <Route path="/workorder/categories" element={<WorkOrderCategoryPage />} />
                      <Route path="/workorder/statistics" element={<WorkOrderStatisticsPage />} />
                      <Route path="/workorder/periodic/templates" element={<PeriodicTemplatePage />} />

                      {/* 运维管理 - 工单路由（带 ops 前缀） */}
                      <Route path="/ops/workorder/orders" element={<WorkOrderPage />} />
                      <Route path="/ops/workorder/categories" element={<WorkOrderCategoryPage />} />
                      <Route path="/ops/workorder/statistics" element={<WorkOrderStatisticsPage />} />
                      <Route path="/ops/workorder/periodic/templates" element={<PeriodicTemplatePage />} />

                      {/* 知识库管理路由 */}
                      <Route path="/knowledge/articles" element={<KnowledgeArticlePage />} />
                      <Route path="/knowledge/view" element={<KnowledgeViewPage />} />

                      {/* AD域管理路由 */}
                      <Route path="/ad-domain/configs" element={<ADConfigPage />} />
                      <Route path="/ad-domain/ous" element={<ADOUPage />} />
                      <Route path="/ad-domain/groups" element={<ADGroupPage />} />
                      <Route path="/ad-domain/users" element={<ADUserPage />} />
                      <Route path="/ad-domain/logs" element={<ADSyncLogsPage />} />
                      {/* 运维管理路由 */}
                      <Route path="/ops/ad-domain/configs" element={<ADConfigPage />} />
                      <Route path="/ops/ad-domain/ous" element={<ADOUPage />} />
                      <Route path="/ops/ad-domain/groups" element={<ADGroupPage />} />
                      <Route path="/ops/ad-domain/users" element={<ADUserPage />} />
                      <Route path="/ops/ad-domain/logs" element={<ADSyncLogsPage />} />
                      {/* 运维管理三级菜单路由（注意：ad-domain 重复是数据问题） */}
                      <Route path="/ops/ad-domain/ad-domain/configs" element={<ADConfigPage />} />
                      <Route path="/ops/ad-domain/ad-domain/ous" element={<ADOUPage />} />
                      <Route path="/ops/ad-domain/ad-domain/groups" element={<ADGroupPage />} />
                      <Route path="/ops/ad-domain/ad-domain/users" element={<ADUserPage />} />
                      <Route path="/ops/ad-domain/ad-domain/logs" element={<ADSyncLogsPage />} />

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