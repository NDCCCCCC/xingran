import type { FC } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { Tabs } from "antd";
import { MailOutlined, ApiOutlined, PictureOutlined, BgColorsOutlined } from "@ant-design/icons";
import { EmailConfigPage, APIConfigPage, CaptchaBackgroundSettingsPage } from "../settings";
import DefaultThemePage from "../settings/default-theme";

/**
 * 系统设置页面
 * 使用Tabs组织邮箱配置、API配置、验证码背景图配置和默认主题配置
 */
const SystemSettingsPage: FC = () => {
  const location = useLocation();
  const [activeTab, setActiveTab] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "activeTab",
    defaultValue: "email",
  });

  const tabItems = [
    {
      key: "email",
      label: <span><MailOutlined />邮箱配置</span>,
      children: <EmailConfigPage />
    },
    {
      key: "api",
      label: <span><ApiOutlined />API配置</span>,
      children: <APIConfigPage />
    },
    {
      key: "captcha-background",
      label: <span><PictureOutlined />验证码背景图</span>,
      children: <CaptchaBackgroundSettingsPage />
    },
    {
      key: "default-theme",
      label: <span><BgColorsOutlined />默认主题</span>,
      children: <DefaultThemePage />
    }
  ];

  return (
    <div>
      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} />
    </div>
  );
};

export default SystemSettingsPage;
