import { Checkbox, Form } from "antd";
// Phase 71 (W2)：裸 Select/Input → 契约组件。API 配置下拉为 children 形态，改写为 options
// 形态（AppSelect 无条件 optionFilterProp="label"，children 下破坏性）
import { AppInput, AppSelect } from "@/components/ui";
import type { APINotificationConfig } from "@/lib/notificationConfigApi";

interface ChannelSelectorProps {
  selectedChannels: string[];
  selectedAPIConfigId: string | undefined;
  apiConfigs: APINotificationConfig[];
  loadingAPIConfigs: boolean;
  customEmails: string;
  customWeComUsers: string;
  onChannelsChange: (channels: string[]) => void;
  onAPIConfigChange: (configId: string | undefined) => void;
  onCustomEmailsChange: (emails: string) => void;
  onCustomWeComUsersChange: (users: string) => void;
}

/**
 * 渠道选择器组件
 * 支持选择站内信、邮件通知、企微机器人等渠道
 */
export const ChannelSelector: React.FC<ChannelSelectorProps> = ({
  selectedChannels,
  selectedAPIConfigId,
  apiConfigs,
  loadingAPIConfigs,
  customEmails,
  customWeComUsers,
  onChannelsChange,
  onAPIConfigChange,
  onCustomEmailsChange,
  onCustomWeComUsersChange,
}) => {
  return (
    <>
      <Form.Item label="发送渠道">
        <Checkbox.Group
          value={selectedChannels}
          onChange={(checkedValues) => {
            const newChannels = checkedValues as string[];
            onChannelsChange(newChannels);
            // 如果取消选择API渠道，清除API配置选择
            if (!newChannels.includes("api")) {
              onAPIConfigChange(undefined);
            }
          }}
        >
          <Checkbox value="web">站内信</Checkbox>
          <Checkbox value="email">邮件通知</Checkbox>
          <Checkbox value="api">企微机器人</Checkbox>
        </Checkbox.Group>
        <div className="text-xs text-gray-500 mt-1">
          邮件通知：使用系统邮件配置发送；企微机器人：需要选择具体配置
        </div>
      </Form.Item>

      {/* API配置选择器（仅当选择API渠道时显示） */}
      {selectedChannels.includes("api") && (
        <Form.Item
          label="企微机器人配置"
          rules={[{ required: true, message: "请选择企微机器人配置" }]}
        >
          {/* 形态分类器③：动态 children → options 改写；label 拼接逻辑原样保留 */}
          <AppSelect
            placeholder="请选择企微机器人配置"
            value={selectedAPIConfigId}
            // AppSelect 泛型被 ComponentProps 擦除 → onChange 值为 unknown，收敛回业务类型
            onChange={(value) => onAPIConfigChange(value as string | undefined)}
            loading={loadingAPIConfigs}
            allowClear
            onSearch={() => {}}
            options={apiConfigs.map((config) => ({
              value: config.id,
              label: `${config.configName}${
                config.configType === "webhook" ? ` (${config.apiMethod})` : ""
              }`,
            }))}
          />
          <div className="text-xs text-gray-500 mt-1">
            {apiConfigs.length === 0 && "暂无可用的API配置，请先在系统设置中添加"}
          </div>
        </Form.Item>
      )}

      {/* 自定义邮件地址（仅当选择邮件渠道时显示） */}
      {selectedChannels.includes("email") && (
        <Form.Item label="自定义邮件地址" extra="多个邮件地址用逗号分隔，留空则发送给目标用户">
          <AppInput
            placeholder="例如: user1@example.com, user2@example.com"
            value={customEmails}
            onChange={(e) => onCustomEmailsChange(e.target.value)}
            allowClear
          />
        </Form.Item>
      )}

      {/* 自定义企微用户代码（仅当选择API渠道时显示） */}
      {selectedChannels.includes("api") && (
        <Form.Item label="自定义企微用户" extra="多个用户代码用逗号分隔，留空则发送给目标用户">
          <AppInput
            placeholder="例如: USER001, USER002"
            value={customWeComUsers}
            onChange={(e) => onCustomWeComUsersChange(e.target.value)}
            allowClear
          />
        </Form.Item>
      )}
    </>
  );
};
