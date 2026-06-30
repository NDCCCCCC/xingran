import React from "react";
import { Form, Input, Select, Button, Modal, Row, Col, Radio, Switch, DatePicker, Checkbox } from "antd";
import type { FormInstance } from "antd/es/form";
import { MarkdownEditor as MDEditor } from "@/components/markdown/MarkdownEditor";
import "@uiw/react-md-editor/markdown-editor.css";
import CronSelector from "@/components/CronSelector";
import { TargetSelector } from "./TargetSelector";
import type { Notice } from "@/types/notice";
import type { APINotificationConfig } from "@/lib/notificationConfigApi";
import type { Target } from "../hooks/useTargetSelector";
import type { ExecutionType } from "@/types/notice";

const { Option } = Select;
const { TextArea } = Input;

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
    <TextArea
      rows={10}
      placeholder="请输入公告内容"
      value={noticeContent ?? ""}
      onChange={(e) => {
        form.setFieldValue("noticeContent", e.target.value);
      }}
    />
  );
};

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
  onSubmit: () => void;
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

  return (
    <Modal
      title={editingNotice ? "编辑通知公告" : "新增通知公告"}
      open={visible}
      onOk={onSubmit}
      onCancel={onCancel}
      width={900}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          取消
        </Button>,
        <Button key="submit" type="primary" onClick={onSubmit}>
          {editingNotice ? "更新" : "创建"}
        </Button>,
      ]}
    >
      <Form form={form}>
        {/* 第一行：标题、类型 */}
        <Row>
          <Col span={12}>
            <Form.Item name="noticeTitle" label="标题" labelCol={{ span: 10 }} wrapperCol={{ span: 14 }} rules={[{ required: true, message: "请输入公告标题" }]}>
              <Input placeholder="请输入公告标题" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="noticeType" label="类型" labelCol={{ span: 10 }} wrapperCol={{ span: 14 }} rules={[{ required: true, message: "请选择公告类型" }]}>
              <Select onSearch={() => {}}>
                <Option value="1">公告</Option>
                <Option value="2">警告</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>

        {/* 第二行：优先级、状态 */}
        <Row>
          <Col span={12}>
            <Form.Item name="priority" label="优先级" labelCol={{ span: 10 }} wrapperCol={{ span: 14 }} initialValue={0}>
              <Select onSearch={() => {}}>
                <Option value={0}>普通</Option>
                <Option value={1}>重要</Option>
                <Option value={2}>紧急</Option>
              </Select>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="status" label="状态" labelCol={{ span: 10 }} wrapperCol={{ span: 14 }} initialValue={0}>
              <Select onSearch={() => {}}>
                <Option value={0}>正常</Option>
                <Option value={1}>关闭</Option>
              </Select>
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
              <Form.Item name="publishTime" label="发布时间" labelCol={{ span: 10 }} wrapperCol={{ span: 14 }}>
                <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" className="w-full" />
              </Form.Item>
            ) : (
              <Form.Item name={["recurrenceConfig", "endDate"]} label="结束时间" labelCol={{ span: 10 }} wrapperCol={{ span: 14 }}>
                <DatePicker format="YYYY-MM-DD HH:mm:ss" className="w-full" showTime />
              </Form.Item>
            )}
          </Col>
          <Col span={12}>
            <Form.Item name="isMarkdown" label="Markdown" labelCol={{ span: 10 }} wrapperCol={{ span: 14 }} valuePropName="checked">
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
          <Input />
        </Form.Item>
        <Form.Item name="recurrenceConfig" hidden>
          <Input />
        </Form.Item>

        {/* 接收范围配置 */}
        <Row>
          <Col span={24}>
            <Form.Item label="接收" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }} required>
              <Row gutter={8}>
                <Col span={6}>
                  <Form.Item name="targetType" noStyle initialValue={0}>
                    <Select onChange={onTargetTypeChange} onSearch={() => {}}>
                      <Option value={0}>全部用户</Option>
                      <Option value={1}>指定部门</Option>
                      <Option value={2}>指定角色</Option>
                      <Option value={3}>指定用户</Option>
                    </Select>
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
          <Input />
        </Form.Item>
        <Form.Item name="targetRoles" hidden>
          <Input />
        </Form.Item>
        <Form.Item name="targetUsers" hidden>
          <Input />
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
              <Form.Item name="apiConfigId" label="企微配置" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
                <Select
                  placeholder="请选择企微机器人配置"
                  loading={loadingAPIConfigs}
                  allowClear
                  onChange={(value) =>    {
                    form.setFieldValue("apiConfigId", value);
                  }}
                 onSearch={() => {}}>
                  {apiConfigs.filter((config) => config.id != null).map((config) => (
                    <Option key={config.id} value={config.id}>
                      {config.configName}
                      {config.configType === "webhook" && ` (${config.apiMethod})`}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>
        )}

        {/* 自定义邮箱 */}
        {selectedChannels.includes("email") && (
          <Row>
            <Col span={24}>
              <Form.Item name="customEmails" label="自定义邮箱" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
                <Input
                  placeholder="多个邮箱用逗号分隔，如: user1@example.com, user2@example.com"
                  allowClear
                />
              </Form.Item>
            </Col>
          </Row>
        )}

        {/* 自定义企微用户 */}
        {selectedChannels.includes("api") && (
          <Row>
            <Col span={24}>
              <Form.Item name="customWeComUsers" label="企微用户" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
                <Input
                  placeholder="多个用户代码用逗号分隔，如: USER001, USER002"
                  allowClear
                />
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
