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
import { ArrowLeftOutlined } from '@ant-design/icons';
import { getNoticeDetail } from '@/lib/noticeApi';
import type { Notice } from '@/types/notice';
import { PRIORITY_COLORS, PRIORITY_LABELS, NOTICE_TYPE_COLORS, NOTICE_TYPE_LABELS, PUBLISH_STATUS_COLORS, PUBLISH_STATUS_LABELS } from '@/types/notice';
import { formatDateTime } from '@/utils/datetime';

/**
 * 管理端通知详情页面
 */
const AdminNoticeDetailPage: FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const [loading, setLoading] = useState(true);
  const [notice, setNotice] = useState<Notice | null>(null);

  // 加载通知详情
  const loadNoticeDetail = useCallback(async () => {
    if (!id) return;

    setLoading(true);
    try {
      const response = await getNoticeDetail(id);
      const noticeData = response.data;
      if (noticeData) {
        setNotice(noticeData);
      }
    } catch (error) {
      console.error('加载通知详情失败:', error);
      message.error('加载失败，请稍后重试');
      navigate('/system/notice');
    } finally {
      setLoading(false);
    }
  }, [id, navigate]);

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
        <Button type="link" onClick={() => navigate('/system/notice')}>
          返回通知列表
        </Button>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* 头部操作栏 */}
      <div className="mb-4">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/system/notice')}>
          返回通知列表
        </Button>
      </div>

      {/* 通知内容卡片 */}
      <Card variant="borderless">
        {/* 标题和标签 */}
        <div className="mb-4">
          <div className="flex items-start justify-between gap-4">
            <h1 className="text-2xl font-bold m-0">{notice.noticeTitle}</h1>
            <Space size="small">
              <Tag color={NOTICE_TYPE_COLORS[notice.noticeType as '1' | '2']}>
                {NOTICE_TYPE_LABELS[notice.noticeType as '1' | '2']}
              </Tag>
              {notice.priority > 0 && (
                <Tag color={PRIORITY_COLORS[notice.priority]}>
                  {PRIORITY_LABELS[notice.priority]}
                </Tag>
              )}
              <Tag color={PUBLISH_STATUS_COLORS[notice.publishStatus]}>
                {PUBLISH_STATUS_LABELS[notice.publishStatus]}
              </Tag>
              {notice.isMarkdown && <Tag color="purple">Markdown</Tag>}
            </Space>
          </div>
        </div>

        <Divider />

        {/* 通知元信息 */}
        <Descriptions column={3} size="small" className="mb-6">
          <Descriptions.Item label="创建人">{notice.createdByName || '系统管理员'}</Descriptions.Item>
          <Descriptions.Item label="创建时间">
            {formatDateTime(notice.createdAt)}
          </Descriptions.Item>
          <Descriptions.Item label="发布时间">
            {notice.publishTime
              ? formatDateTime(notice.publishTime)
              : '尚未发布'}
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
              <Space direction="vertical" className="w-full">
                {notice.attachments.map((file: any) => (
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
          <Button
            onClick={() => navigate('/system/notice')}
          >
            关闭
          </Button>
        </div>
      </Card>
    </div>
  );
};

export default AdminNoticeDetailPage;
