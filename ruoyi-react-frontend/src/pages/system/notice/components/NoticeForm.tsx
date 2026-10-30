import React, { useState } from "react";
import { Form, Button, Modal, Row, Col, Radio, Switch, DatePicker, Checkbox } from "antd";
// Phase 71 (W2)：裸 Input/Select/TextArea → 契约组件。本文件 Select 全为 children 形态，
// 一律改写为 options 形态（AppSelect 无条件 optionFilterProp="label"，children 下破坏性）
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import type { FormInstance } from "antd/es/form";
import { MarkdownEditor as MDEditor } from "@/components/markdown/MarkdownEditor";
import CronSelector from "@/components/CronSelector";
import { TargetSelector } from "./TargetSelector";
import type { Notice } from "@/types/notice";
import type { APINotificationConfig } from "@/lib/notificationConfigApi";
import type { Target } from "../hooks/useTargetSelector";
import type { ExecutionType } from "@/types/notice";

// 内部组件：使用 Form.useWatch 订阅 noticeContent 字段，
// 避免在 Form 未连接时调用 getFieldValue 触发告警
const NoticeContentField: React.FC<{
  form: FormInstance;
  isMarkdown: boolean;
  markdownContent: string;
  onMarkdownContentChange: (content: string) => void;
}> = ({ form, isMarkdown, markdownContent, onMarkdownContentChange }) => {
  const noticeContent = Form.useWatch("noticeContent", form) as string | undefined;

  return isMarkdown ? (
    <MDEditor
      value={markdownContent}
      onChange={(val) => {
        const content = val || "";
        onMarkdownContentChange(content);
        form.setFieldValue("noticeContent", content);
      }}
      preview="live"
      height={400}
    />
  ) : (
    <AppTextArea
      rows={10}
      placeholder="请输入公告内容"
      value={noticeContent ?? ""}
      onChange={(e) => {
        form.setFieldValue("noticeContent", e.target.value);
      }}
    />
  );
};

// ==================== 形态分类器产物：children 形态 <Option> → options 常量 ====================

/** 公告类型（固定 2 项） */
const NOTICE_TYPE_OPTIONS = [
  { value: "1", label: "公告" },
  { value: "2", label: "警告" },
];

/** 重要程度（固定 3 项） */
const NOTICE_LEVEL_OPTIONS = [
  { value: 0, label: "普通" },
  { value: 1, label: "重要" },
  { value: 2, label: "紧急" },
];

/** 公告状态（固定 2 项） */
const NOTICE_STATUS_OPTIONS = [
  { value: 0, label: "正常" },
  { value: 1, label: "关闭" },
];

/** 接收范围（固定 4 项） */
const TARGET_TYPE_OPTIONS = [
  { value: 0, label: "全部用户" },
  { value: 1, label: "指定部门" },
  { value: 2, label: "指定角色" },
  { value: 3, label: "指定用户" },
];

export interface NoticeFormProps {
  visible: boolean;
  editingNotice: Notice | null;
  executionType: ExecutionType;
  selectedChannels: string[];
  apiConfigs: APINotificationConfig[];
  loadingAPIConfigs: boolean;
  deptTree: Target[];
  roles: Target[];
  users: Target[];
  loadingDepts: boolean;
  loadingRoles: boolean;
  loadingUsers: boolean;
  markdownContent: string;
  form: FormInstance;
  onCancel: () => void;
  onSubmit: () => void | Promise<void>;
  onExecutionTypeChange: (type: ExecutionType) => void;
  onChannelsChange: (channels: string[]) => void;
  onMarkdownContentChange: (content: string) => void;
  onTargetTypeChange: (value: number) => void;
  onDeptChange: (keys: React.Key[]) => void;
  onRoleChange: (values: string[]) => void;
  onUserChange: (values: string[]) => void;
}

/**
 * 通知编辑/创建表单组件
 * 采用统一布局，所有标签完美对齐
 */
export const NoticeForm: React.FC<NoticeFormProps> = ({
  visible,
  editingNotice,
  executionType,
  selectedChannels,
  apiConfigs,
  loadingAPIConfigs,
  deptTree,
  roles,
  users,
  loadingDepts,
  loadingRoles,
  loadingUsers,
  markdownContent,
  form,
  onCancel,
  onSubmit,
  onExecutionTypeChange,
  onChannelsChange,
  onMarkdownContentChange,
  onTargetTypeChange,
  onDeptChange,
  onRoleChange,
  onUserChange,
}) => {
  const targetType = Form.useWatch("targetType", form);
  const targetDepts = Form.useWatch("targetDepts", form) || [];
  const targetRoles = Form.useWatch("targetRoles", form) || [];
  const targetUsers = Form.useWatch("targetUsers", form) || [];
  const isMarkdown = Form.useWatch("isMarkdown", form) || false;
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async () => {
    setSubmitting(true);
    try {
      await onSubmit();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={editingNotice ? "编辑通知公告" : "新增通知公告"}
      open={visible}
      onCancel={onCancel}
      width={900}
      footer={[
        <Button key="cancel" onClick={onCancel} disabled={submitting}>
          取消
        </Button>,
        <Button key="submit" type="primary" onClick={handleSubmit} loading={submitting}>
          {editingNotice ? "更新" : "创建"}
        </Button>,
      ]}
    >
      <Form form={form}>
        {/* 第一行：标题、类型 */}
        <Row>
          <Col span={12}>
            <Form.Item
              name="noticeTitle"
              label="标题"
              labelCol={{ span: 10 }}
              wrapperCol={{ span: 14 }}
              rules={[{ required: true, message: "请输入公告标题" }]}
            >
              <AppInput placeholder="请输入公告标题" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="noticeType"
              label="类型"
              labelCol={{ span: 10 }}
              wrapperCol={{ span: 14 }}
              rules={[{ required: true, message: "请选择公告类型" }]}
            >
              {/* 形态分类器②④：公告类型固定 2 项，业务禁搜 */}
              <AppSelect
                allowClear
                onSearch={() => {}}
                showSearch={false}
                options={NOTICE_TYPE_OPTIONS}
              />
            </Form.Item>
          </Col>
        </Row>

        {/* 第二行：优先级、状态 */}
        <Row>
          <Col span={12}>
            <Form.Item
              name="priority"
              label="优先级"
              labelCol={{ span: 10 }}
              wrapperCol={{ span: 14 }}
              initialValue={0}
            >
              {/* 形态分类器②④：重要程度固定 3 项，业务禁搜 */}
              <AppSelect
                allowClear
                onSearch={() => {}}
                showSearch={false}
                options={NOTICE_LEVEL_OPTIONS}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="status"
              label="状态"
              labelCol={{ span: 10 }}
              wrapperCol={{ span: 14 }}
              initialValue={0}
            >
              {/* 形态分类器②④：公告状态固定 2 项，业务禁搜 */}
              <AppSelect
                allowClear
                onSearch={() => {}}
                showSearch={false}
                options={NOTICE_STATUS_OPTIONS}
              />
            </Form.Item>
          </Col>
        </Row>

        {/* 执行方式 */}
        <Row>
          <Col span={24}>
            <Form.Item label="执行方式" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
              <Radio.Group
                value={executionType}
                onChange={(e) => {
                  onExecutionTypeChange(e.target.value);
                  form.setFieldValue("publishTime", null);
                }}
              >
                <Radio value="once">立即发布/定时发布</Radio>
                <Radio value="recurring">周期性发送</Radio>
              </Radio.Group>
            </Form.Item>
          </Col>
        </Row>

        {/* 时间配置和 Markdown */}
        <Row>
          <Col span={12}>
            {executionType === "once" ? (
              <Form.Item
                name="publishTime"
                label="发布时间"
                labelCol={{ span: 10 }}
                wrapperCol={{ span: 14 }}
              >
                <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" className="w-full" />
              </Form.Item>
            ) : (
              <Form.Item
                name={["recurrenceConfig", "endDate"]}
                label="结束时间"
                labelCol={{ span: 10 }}
                wrapperCol={{ span: 14 }}
              >
                <DatePicker format="YYYY-MM-DD HH:mm:ss" className="w-full" showTime />
              </Form.Item>
            )}
          </Col>
          <Col span={12}>
            <Form.Item
              name="isMarkdown"
              label="Markdown"
              labelCol={{ span: 10 }}
              wrapperCol={{ span: 14 }}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Col>
        </Row>

        {/* Cron 表达式（仅周期性发送时显示） */}
        {executionType === "recurring" && (
          <Row>
            <Col span={24}>
              <Form.Item
                name={["recurrenceConfig", "cronExpression"]}
                label="Cron表达式"
                labelCol={{ span: 5 }}
                wrapperCol={{ span: 19 }}
                required
                rules={[{ required: true, message: "请输入 Cron 表达式" }]}
              >
                <CronSelector />
              </Form.Item>
            </Col>
          </Row>
        )}

        {/* 隐藏字段 */}
        <Form.Item name="executionType" hidden>
          <AppInput />
        </Form.Item>
        <Form.Item name="recurrenceConfig" hidden>
          <AppInput />
        </Form.Item>

        {/* 接收范围配置 */}
        <Row>
          <Col span={24}>
            <Form.Item label="接收" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }} required>
              <Row gutter={8}>
                <Col span={6}>
                  <Form.Item name="targetType" noStyle initialValue={0}>
                    {/* 形态分类器②：接收范围固定 4 项 */}
                    <AppSelect
                      allowClear
                      // AppSelect 泛型被 ComponentProps 擦除 → onChange 值为 unknown，收敛回业务类型
                      onChange={(value) => onTargetTypeChange(value as number)}
                      onSearch={() => {}}
                      options={TARGET_TYPE_OPTIONS}
                    />
                  </Form.Item>
                </Col>
                <Col span={18}>
                  {targetType !== 0 && (
                    <div className="border rounded p-3 bg-gray-50">
                      <TargetSelector
                        targetType={targetType}
                        targetDepts={targetDepts}
                        targetRoles={targetRoles}
                        targetUsers={targetUsers}
                        deptTree={deptTree}
                        roles={roles}
                        users={users}
                        loadingDepts={loadingDepts}
                        loadingRoles={loadingRoles}
                        loadingUsers={loadingUsers}
                        onDeptChange={onDeptChange}
                        onRoleChange={onRoleChange}
                        onUserChange={onUserChange}
                      />
                    </div>
                  )}
                </Col>
              </Row>
            </Form.Item>
          </Col>
        </Row>

        <Form.Item name="targetDepts" hidden>
          <AppInput />
        </Form.Item>
        <Form.Item name="targetRoles" hidden>
          <AppInput />
        </Form.Item>
        <Form.Item name="targetUsers" hidden>
          <AppInput />
        </Form.Item>

        {/* 发送渠道配置 */}
        <Row>
          <Col span={24}>
            <Form.Item label="发送渠道" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
              <Checkbox.Group
                value={selectedChannels}
                onChange={(checkedValues) => {
                  const newChannels = checkedValues as string[];
                  onChannelsChange(newChannels);
                  if (!newChannels.includes("api")) {
                    form.setFieldValue("apiConfigId", undefined);
                  }
                }}
              >
                <Checkbox value="web">站内信</Checkbox>
                <Checkbox value="email">邮件通知</Checkbox>
                <Checkbox value="api">企微机器人</Checkbox>
              </Checkbox.Group>
            </Form.Item>
          </Col>
        </Row>

        {/* 企微配置选择 */}
        {selectedChannels.includes("api") && (
          <Row>
            <Col span={24}>
              <Form.Item
                name="apiConfigId"
                label="企微配置"
                labelCol={{ span: 5 }}
                wrapperCol={{ span: 19 }}
              >
                {/* 形态分类器③：动态 children → options 改写；label 拼接原样保留 */}
                <AppSelect
                  placeholder="请选择企微机器人配置"
                  loading={loadingAPIConfigs}
                  allowClear
                  onChange={(value) => {
                    form.setFieldValue("apiConfigId", value);
                  }}
                  onSearch={() => {}}
                  options={apiConfigs
                    .filter((config) => config.id != null)
                    .map((config) => ({
                      value: config.id,
                      label: `${config.configName}${
                        config.configType === "webhook" ? ` (${config.apiMethod})` : ""
                      }`,
                    }))}
                />
              </Form.Item>
            </Col>
          </Row>
        )}

        {/* 自定义邮箱 */}
        {selectedChannels.includes("email") && (
          <Row>
            <Col span={24}>
              <Form.Item
                name="customEmails"
                label="自定义邮箱"
                labelCol={{ span: 5 }}
                wrapperCol={{ span: 19 }}
              >
                <AppInput placeholder="多个邮箱用逗号分隔，如: user1@example.com, user2@example.com" />
              </Form.Item>
            </Col>
          </Row>
        )}

        {/* 自定义企微用户 */}
        {selectedChannels.includes("api") && (
          <Row>
            <Col span={24}>
              <Form.Item
                name="customWeComUsers"
                label="企微用户"
                labelCol={{ span: 5 }}
                wrapperCol={{ span: 19 }}
              >
                <AppInput placeholder="多个用户代码用逗号分隔，如: USER001, USER002" />
              </Form.Item>
            </Col>
          </Row>
        )}

        {/* 公告内容 */}
        <Row>
          <Col span={24}>
            <Form.Item
              name="noticeContent"
              label="内容"
              labelCol={{ span: 5 }}
              wrapperCol={{ span: 19 }}
              rules={[{ required: true, message: "请输入公告内容" }]}
            >
              <NoticeContentField
                form={form}
                isMarkdown={isMarkdown}
                markdownContent={markdownContent}
                onMarkdownContentChange={onMarkdownContentChange}
              />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
};
