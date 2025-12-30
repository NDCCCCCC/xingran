import React, { useState } from 'react';
import { Card, Tabs } from 'antd';
import { MailOutlined, ApiOutlined } from '@ant-design/icons';
import { EmailConfigPage, APIConfigPage } from '../settings';

/**
 * 系统设置页面
 * 使用Tabs组织邮箱配置和API配置
 */
const SystemSettingsPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState('email');

  const tabItems = [
    {
      key: 'email',
      label: (
        <span>
          <MailOutlined />
          邮箱配置
        </span>
      ),
      children: <EmailConfigPage />,
    },
    {
      key: 'api',
      label: (
        <span>
          <ApiOutlined />
          API配置
        </span>
      ),
      children: <APIConfigPage />,
    },
  ];

  return (
    <div className="p-6">
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
        />
      </Card>
    </div>
  );
};

export default SystemSettingsPage;
