import React from 'react';
import { Layout as AntLayout } from 'antd';
import Header from './header';
import Sidebar from './Sidebar';
import Breadcrumb from './breadcrumb';

const { Content } = AntLayout;

interface LayoutProps {
  children: React.ReactNode;
}

const Layout: React.FC<LayoutProps> = ({ children }) => {
  return (
    <AntLayout className="h-screen">
      <Sidebar />
      <AntLayout className="h-full">
        <Header />
        <Breadcrumb />
        <Content className="flex-1 m-6 overflow-auto">
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 min-h-full">
            {children}
          </div>
        </Content>
      </AntLayout>
    </AntLayout>
  );
};

export default Layout;