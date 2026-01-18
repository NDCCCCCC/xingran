import { useEffect, useState, useCallback } from 'react';
import type { FC } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Card,
  Button,
  Tag,
  Space,
  Descriptions,
  Divider,
  message,
  Spin,
} from 'antd';
import { ArrowLeftOutlined, CheckOutlined } from '@ant-design/icons';
import { getMyNoticeDetail, markNoticeAsRead } from '@/lib/noticeApi';
import { useNoticeStore } from '@/store/noticeStore';
import type { Notice } from '@/types/notice';
import { PRIORITY_COLORS, PRIORITY_LABELS, NOTICE_TYPE_COLORS, NOTICE_TYPE_LABELS } from '@/types/notice';
import { formatDateTime } from '@/utils/datetime';

/**
 * 用户通知详情页面
 */
const NoticeDetailPage: FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { markAsRead } = useNoticeStore();

  const [loading, setLoading] = useState(true);
  const [notice, setNotice] = useState<Notice | null>(null);

  // 加载通知详情
  const loadNoticeDetail = useCallback(async () => {
    if (!id) return;

    setLoading(true);
    try {
      const response = await getMyNoticeDetail(id);
      const noticeData = response.data;
      if (noticeData) {
        setNotice(noticeData);

        // 标记为已读
        if (!noticeData.isRead) {
          await markNoticeAsRead(id);
          markAsRead(id);
        }
      }
    } catch (error) {
      console.error('加载通知详情失败:', error);
      message.error('加载失败，请稍后重试');
      navigate('/my-notices');
    } finally {
      setLoading(false);
    }
  }, [id, navigate, markAsRead]);

  useEffect(() => {
    loadNoticeDetail();
  }, [loadNoticeDetail]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" />
      </div>
    );
  }

  if (!notice) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-gray-500">
        <p>通知不存在</p>
        <Button type="link" onClick={() => navigate('/my-notices')}>
          返回通知中心
        </Button>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* 头部操作栏 */}
      <div className="mb-4">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/my-notices')}>
          返回通知中心
        </Button>
      </div>

      {/* 通知内容卡片 */}
      <Card variant="borderless">
        {/* 标题和标签 */}
        <div className="mb-4">
          <div className="flex items-start justify-between gap-4">
            <h1 className="text-2xl font-bold m-0">{notice.noticeTitle}</h1>
            <Space size="small">
              <Tag color={NOTICE_TYPE_COLORS[notice.noticeType]}>{NOTICE_TYPE_LABELS[notice.noticeType]}</Tag>
              {notice.priority > 0 && (
                <Tag color={PRIORITY_COLORS[notice.priority]}>{PRIORITY_LABELS[notice.priority]}</Tag>
              )}
              {notice.isMarkdown && <Tag color="purple">Markdown</Tag>}
            </Space>
          </div>
        </div>

        <Divider />

        {/* 通知元信息 */}
        <Descriptions column={3} size="small" className="mb-6">
          <Descriptions.Item label="发布人">{notice.createdByName || '系统管理员'}</Descriptions.Item>
          <Descriptions.Item label="发布时间">
            {formatDateTime(notice.publishTime || notice.createdAt)}
          </Descriptions.Item>
          <Descriptions.Item label="阅读状态">
            {notice.isRead ? (
              <Tag color="default">已读</Tag>
            ) : (
              <Tag color="blue">未读</Tag>
            )}
          </Descriptions.Item>
        </Descriptions>

        {/* 通知内容 */}
        <div className="min-h-64">
          {notice.isMarkdown ? (
            <div className="prose max-w-none">
              <div className="whitespace-pre-wrap">{notice.noticeContent}</div>
              <div className="text-xs text-gray-400 mt-2">（Markdown 渲染需要安装 react-markdown 库）</div>
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
                {notice.attachments.map((file) => (
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
          {!notice.isRead && (
            <Button
              type="primary"
              icon={<CheckOutlined />}
              onClick={async () => {
                try {
                  await markNoticeAsRead(notice.id);
                  markAsRead(notice.id);
                  setNotice({ ...notice, isRead: true });
                  message.success('已标记为已读');
                } catch (error) {
                  console.error('标记已读失败:', error);
                  message.error('操作失败');
                }
              }}
            >
              标记为已读
            </Button>
          )}
        </div>
      </Card>
    </div>
  );
};

export default NoticeDetailPage;
