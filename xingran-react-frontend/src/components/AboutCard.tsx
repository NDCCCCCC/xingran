/**
 * 「关于」展示组件。
 *
 * 在个人中心 / 系统设置等页面露出应用版本号。版本号来自 Vite 编译期注入的
 * 全局常量 __APP_VERSION__（见 vite.config.ts 的 define 与 src/vite-env.d.ts
 * 的类型声明），与后端 -ldflags -X main.Version 注入的发布版本号同源。
 *
 * 说明：本组件只读、不发起任何网络请求、不读取任何配置或密钥，仅展示版本号。
 * typeof 守卫让组件在未经 Vite 转换的环境（如独立单测）中也不会抛
 * ReferenceError，而是回落显示 "dev"。
 */
import type { FC } from "react";
import { Card, Typography } from "antd";

const { Text } = Typography;

const appVersion: string = typeof __APP_VERSION__ === "undefined" ? "dev" : __APP_VERSION__;

export interface AboutCardProps {
  /** 卡片标题，默认「关于」 */
  title?: string;
  /** 是否显示带边框的 Card，false 时仅输出一行文本，便于嵌入任意容器 */
  bordered?: boolean;
}

const AboutCard: FC<AboutCardProps> = ({ title = "关于", bordered = true }) => {
  const content = (
    <Text>
      应用版本：<Text code>{appVersion}</Text>
    </Text>
  );

  if (!bordered) {
    return content;
  }

  return (
    <Card title={title} size="small">
      {content}
    </Card>
  );
};

export default AboutCard;
