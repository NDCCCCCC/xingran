import React, { useState } from 'react';
import { Card, Tabs } from 'antd';
import { MailOutlined, ApiOutlined, PictureOutlined } from '@ant-design/icons';
import { EmailConfigPage, APIConfigPage, CaptchaBackgroundSettingsPage } from '../settings';

/**
 * 系统设置页面
 * 使用Tabs组织邮箱配置、API配置和验证码背景图配置
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
    {
      key: 'captcha-background',
      label: (
        <span>
          <PictureOutlined />
          验证码背景图
        </span>
      ),
      children: <CaptchaBackgroundSettingsPage />,
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
