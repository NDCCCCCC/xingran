import React from 'react';
import { Form, Input, Select, Button, Modal, Row, Col, Radio, Switch, DatePicker } from 'antd';
import MDEditor from '@uiw/react-md-editor';
import '@uiw/react-md-editor/markdown-editor.css';
import { ChannelSelector } from './ChannelSelector';
import { TargetSelector } from './TargetSelector';
import { RecurrenceConfig } from './RecurrenceConfig';
import type { Notice } from '@/types/notice';
import type { APINotificationConfig } from '@/lib/notificationConfigApi';
import type { Target } from '../hooks/useTargetSelector';
import type { ExecutionType, RecurrenceType } from '@/types/notice';

const { Option } = Select;
const { TextArea } = Input;

export interface NoticeFormProps {
  visible: boolean;
  editingNotice: Notice | null;
  executionType: ExecutionType;
  recurrenceType: RecurrenceType;
  selectedChannels: string[];
  selectedAPIConfigId: string | undefined;
  apiConfigs: APINotificationConfig[];
  loadingAPIConfigs: boolean;
  deptTree: Target[];
  roles: Target[];
  users: Target[];
  loadingDepts: boolean;
  loadingRoles: boolean;
  loadingUsers: boolean;
  markdownContent: string;
  customEmails: string;
  customWeComUsers: string;
  form: any;
  onCancel: () => void;
  onSubmit: () => void;
  onExecutionTypeChange: (type: ExecutionType) => void;
  onRecurrenceTypeChange: (type: RecurrenceType) => void;
  onChannelsChange: (channels: string[]) => void;
  onAPIConfigChange: (configId: string | undefined) => void;
  onMarkdownContentChange: (content: string) => void;
  onCustomEmailsChange: (emails: string) => void;
  onCustomWeComUsersChange: (users: string) => void;
  onTargetTypeChange: (value: number) => void;
  onDeptChange: (keys: React.Key[]) => void;
  onRoleChange: (values: any[]) => void;
  onUserChange: (values: any[]) => void;
}

/**
 * 通知编辑/创建表单组件
 * 包含执行类型、接收范围、发送渠道、内容等配置
 */
export const NoticeForm: React.FC<NoticeFormProps> = ({
  visible,
  editingNotice,
  executionType,
  recurrenceType,
  selectedChannels,
  selectedAPIConfigId,
  apiConfigs,
  loadingAPIConfigs,
  deptTree,
  roles,
  users,
  loadingDepts,
  loadingRoles,
  loadingUsers,
  markdownContent,
  customEmails,
  customWeComUsers,
  form,
  onCancel,
  onSubmit,
  onExecutionTypeChange,
  onRecurrenceTypeChange,
  onChannelsChange,
  onAPIConfigChange,
  onMarkdownContentChange,
  onCustomEmailsChange,
  onCustomWeComUsersChange,
  onTargetTypeChange,
  onDeptChange,
  onRoleChange,
  onUserChange,
}) => {
  const targetType = Form.useWatch('targetType', form);
  const targetDepts = Form.useWatch('targetDepts', form) || [];
  const targetRoles = Form.useWatch('targetRoles', form) || [];
  const targetUsers = Form.useWatch('targetUsers', form) || [];
  const isMarkdown = Form.useWatch('isMarkdown', form) || false;

  return (
    <Modal
      title={editingNotice ? '编辑通知公告' : '新增通知公告'}
      open={visible}
      onOk={onSubmit}
      onCancel={onCancel}
      width={900}
      footer={[
        <Button key="cancel" onClick={onCancel}>
          取消
        </Button>,
        <Button key="submit" type="primary" onClick={onSubmit}>
          {editingNotice ? '更新' : '创建'}
        </Button>,
      ]}
    >
      <Form form={form} labelCol={{ span: 4 }} wrapperCol={{ span: 18 }}>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="noticeTitle" label="标题" rules={[{ required: true, message: '请输入公告标题' }]}>
              <Input placeholder="请输入公告标题" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="noticeType" label="类型" rules={[{ required: true }]}>
              <Select>
                <Option value="1">公告</Option>
                <Option value="2">警告</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="priority" label="优先级" initialValue={0}>
              <Select>
                <Option value={0}>普通</Option>
                <Option value={1}>重要</Option>
                <Option value={2}>紧急</Option>
              </Select>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="status" label="状态" initialValue={0}>
              <Select>
                <Option value={0}>正常</Option>
                <Option value={1}>关闭</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>

        {/* 执行类型和发布配置 */}
        <Form.Item label="执行方式">
          <Radio.Group
            value={executionType}
            onChange={(e) => {
              onExecutionTypeChange(e.target.value);
              form.setFieldValue('publishTime', null);
            }}
          >
            <Radio value="once">立即发布/定时发布</Radio>
            <Radio value="recurring">周期性发送</Radio>
          </Radio.Group>
        </Form.Item>

        {executionType === 'once' ? (
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="publishTime" label="发布时间">
                <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" className="w-full" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="isMarkdown" label="Markdown" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
        ) : (
          <>
            <RecurrenceConfig
              recurrenceType={recurrenceType}
              onRecurrenceTypeChange={onRecurrenceTypeChange}
            />
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="isMarkdown" label="Markdown" valuePropName="checked">
                  <Switch />
                </Form.Item>
              </Col>
            </Row>
          </>
        )}

        {/* 隐藏字段：执行类型 */}
        <Form.Item name="executionType" hidden>
          <Input />
        </Form.Item>
        <Form.Item name="recurrenceConfig" hidden>
          <Input />
        </Form.Item>

        {/* 目标选择器 */}
        <Form.Item label="接收范围" required>
          <Row gutter={8}>
            <Col span={12}>
              <Form.Item name="targetType" noStyle initialValue={0}>
                <Select onChange={onTargetTypeChange}>
                  <Option value={0}>全部用户</Option>
                  <Option value={1}>指定部门</Option>
                  <Option value={2}>指定角色</Option>
                  <Option value={3}>指定用户</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
        </Form.Item>

        <Form.Item label=" " colon={false}>
          <div className="border rounded p-4 bg-gray-50">
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
        </Form.Item>

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
        <ChannelSelector
          selectedChannels={selectedChannels}
          selectedAPIConfigId={selectedAPIConfigId}
          apiConfigs={apiConfigs}
          loadingAPIConfigs={loadingAPIConfigs}
          customEmails={customEmails}
          customWeComUsers={customWeComUsers}
          onChannelsChange={onChannelsChange}
          onAPIConfigChange={onAPIConfigChange}
          onCustomEmailsChange={onCustomEmailsChange}
          onCustomWeComUsersChange={onCustomWeComUsersChange}
        />

        <Form.Item name="noticeContent" label="内容" rules={[{ required: true, message: '请输入公告内容' }]}>
          {isMarkdown ? (
            <React.Fragment>
              {React.createElement(MDEditor as any, {
                value: markdownContent,
                onChange: (val: string | undefined) => {
                  const content = val || '';
                  onMarkdownContentChange(content);
                  form.setFieldValue('noticeContent', content);
                },
                preview: 'live',
                height: 400,
                'data-color-mode': 'light',
              })}
            </React.Fragment>
          ) : (
            <TextArea
              rows={10}
              placeholder="请输入公告内容"
              value={form.getFieldValue('noticeContent')}
              onChange={(e) => {
                form.setFieldValue('noticeContent', e.target.value);
              }}
            />
          )}
        </Form.Item>
      </Form>
    </Modal>
  );
};
