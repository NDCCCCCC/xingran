/**
 * 通知详情内容组件
 * 用于展示通知的完整内容，支持用户端和管理端
 */

import type { FC } from "react";
import { Card, Tag, Space, Descriptions, Divider, Button } from "antd";
import { CheckCircleOutlined } from "@ant-design/icons";
import type { Notice, NoticeAttachment } from "@/types/notice";
import {
  PRIORITY_COLORS,
  PRIORITY_LABELS,
  NOTICE_TYPE_COLORS,
  NOTICE_TYPE_LABELS,
  PUBLISH_STATUS_COLORS,
  PUBLISH_STATUS_LABELS,
} from "@/types/notice";
import { formatDateTime } from "@/utils/datetime";

export interface NoticeDetailContentProps {
  notice: Notice;
  showReadStatus?: boolean; // 是否显示阅读状态（用户端）
  showPublishStatus?: boolean; // 是否显示发布状态（管理端）
  showCreator?: boolean; // 是否显示创建人（管理端）
  showMarkAsReadButton?: boolean; // 是否显示"标记为已读"按钮
  onMarkAsRead?: () => Promise<void>;
}

const NoticeDetailContent: FC<NoticeDetailContentProps> = ({
  notice,
  showReadStatus = false,
  showPublishStatus = false,
  showCreator = false,
  showMarkAsReadButton = false,
  onMarkAsRead,
}) => {
  const handleMarkAsReadClick = async () => {
    if (onMarkAsRead) {
      await onMarkAsRead();
    }
  };

  return (
    <Card variant="borderless">
      {/* 标题和标签 */}
      <div className="mb-4">
        <div className="flex items-start justify-between gap-4">
          <h1 className="text-2xl font-bold m-0">{notice.noticeTitle}</h1>
          <Space size="small">
            <Tag color={NOTICE_TYPE_COLORS[notice.noticeType as "1" | "2"]}>
              {NOTICE_TYPE_LABELS[notice.noticeType as "1" | "2"]}
            </Tag>
            {notice.priority > 0 && (
              <Tag color={PRIORITY_COLORS[notice.priority]}>{PRIORITY_LABELS[notice.priority]}</Tag>
            )}
            {showPublishStatus && (
              <Tag color={PUBLISH_STATUS_COLORS[notice.publishStatus]}>
                {PUBLISH_STATUS_LABELS[notice.publishStatus]}
              </Tag>
            )}
            {notice.isMarkdown && <Tag color="purple">Markdown</Tag>}
          </Space>
        </div>
      </div>

      <Divider />

      {/* 通知元信息 */}
      <Descriptions column={3} size="small" className="mb-6">
        {showCreator && (
          <Descriptions.Item label="创建人">
            {notice.createdByName || "系统管理员"}
          </Descriptions.Item>
        )}
        <Descriptions.Item label={showCreator ? "创建时间" : "发布人"}>
          {showCreator ? formatDateTime(notice.createdAt) : notice.createdByName || "系统管理员"}
        </Descriptions.Item>
        {showPublishStatus ? (
          <Descriptions.Item label="发布时间">
            {notice.publishTime ? formatDateTime(notice.publishTime) : "尚未发布"}
          </Descriptions.Item>
        ) : showReadStatus ? (
          <Descriptions.Item label="发布时间">
            {formatDateTime(notice.publishTime || notice.createdAt)}
          </Descriptions.Item>
        ) : null}
        {showReadStatus && (
          <Descriptions.Item label="阅读状态">
            {notice.isRead ? <Tag color="default">已读</Tag> : <Tag color="blue">未读</Tag>}
          </Descriptions.Item>
        )}
      </Descriptions>

      {/* 通知内容 */}
      <div className="min-h-64">
        {notice.isMarkdown ? (
          <div className="prose max-w-none">
            <div className="whitespace-pre-wrap">{notice.noticeContent}</div>
            <div className="text-xs text-gray-400 mt-2">
              （Markdown 渲染需要安装 react-markdown 库）
            </div>
          </div>
        ) : (
          <div className="whitespace-pre-wrap">{notice.noticeContent}</div>
        )}
      </div>

      {/* 附件列表 */}
      {notice.attachments && notice.attachments.length > 0 && (
        <>
          <Divider />
          <div>
            <h3 className="text-base font-medium mb-3">附件</h3>
            <Space orientation="vertical" className="w-full">
              {notice.attachments.map((file: NoticeAttachment) => (
                <Card key={file.id} size="small" className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="text-blue-600">{file.fileName}</span>
                    <span className="text-gray-400 text-sm">
                      ({(file.fileSize / 1024).toFixed(2)} KB)
                    </span>
                  </div>
                  <Button type="link" size="small">
                    下载
                  </Button>
                </Card>
              ))}
            </Space>
          </div>
        </>
      )}

      <Divider />

      {/* 底部操作栏 */}
      <div className="flex justify-end">
        {showMarkAsReadButton && !notice.isRead && onMarkAsRead && (
          <Button type="primary" icon={<CheckCircleOutlined />} onClick={handleMarkAsReadClick}>
            标记为已读
          </Button>
        )}
      </div>
    </Card>
  );
};

export default NoticeDetailContent;
