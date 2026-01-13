import { useState } from 'react';
import type { FC } from 'react';
import { Tabs } from 'antd';
import { MailOutlined, ApiOutlined, PictureOutlined } from '@ant-design/icons';
import { EmailConfigPage, APIConfigPage, CaptchaBackgroundSettingsPage } from '../settings';

const { TabPane } = Tabs;

/**
 * 系统设置页面
 * 使用Tabs组织邮箱配置、API配置和验证码背景图配置
 */
const SystemSettingsPage: FC = () => {
  const [activeTab, setActiveTab] = useState('email');

  return (
    <div>
      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab={<span><MailOutlined />邮箱配置</span>} key="email">
          <EmailConfigPage />
        </TabPane>
        <TabPane tab={<span><ApiOutlined />API配置</span>} key="api">
          <APIConfigPage />
        </TabPane>
        <TabPane tab={<span><PictureOutlined />验证码背景图</span>} key="captcha-background">
          <CaptchaBackgroundSettingsPage />
        </TabPane>
      </Tabs>
    </div>
  );
};

export default SystemSettingsPage;
