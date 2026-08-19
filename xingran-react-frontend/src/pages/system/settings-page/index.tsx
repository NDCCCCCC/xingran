import type { FC } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { Tabs } from "antd";
import { MailOutlined, ApiOutlined, PictureOutlined } from "@ant-design/icons";
import { EmailConfigPage, APIConfigPage, CaptchaBackgroundSettingsPage } from "../settings";

/**
 * 系统设置页面
 * 使用Tabs组织邮箱配置、API配置、验证码背景图配置
 *
 * v1.22 收尾：移除"默认主题" Tab（默认主题页面与后端 sys.theme.default / theme API
 * 全部删除，管理员不再有"全局默认主题"配置入口）。
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
      label: (
        <span>
          <MailOutlined />
          邮箱配置
        </span>
      ),
      children: <EmailConfigPage />,
    },
    {
      key: "api",
      label: (
        <span>
          <ApiOutlined />
          API配置
        </span>
      ),
      children: <APIConfigPage />,
    },
    {
      key: "captcha-background",
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
    <div>
      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} />
    </div>
  );
};

export default SystemSettingsPage;
