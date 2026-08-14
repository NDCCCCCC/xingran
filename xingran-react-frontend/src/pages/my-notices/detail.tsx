import { useEffect, useState, useCallback } from "react";
import type { FC } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { App, Button, Spin } from "antd";
import { ArrowLeftOutlined } from "@ant-design/icons";
import { getMyNoticeDetail, markNoticeAsRead } from "@/lib/noticeApi";
import { useNoticeStore } from "@/store/noticeStore";
import type { Notice } from "@/types/notice";
import NoticeDetailContent from "@/components/NoticeDetail/NoticeDetailContent";
import { USER_NOTICES } from "@/constants/routes";

/**
 * 用户通知详情页面
 */
const NoticeDetailPage: FC = () => {
  const { message } = App.useApp();
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
      console.error("加载通知详情失败:", error);
      message.error("加载失败，请稍后重试");
      navigate(USER_NOTICES);
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
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
        <Button type="link" onClick={() => navigate(USER_NOTICES)}>
          返回通知中心
        </Button>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* 头部操作栏 */}
      <div className="mb-4">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(USER_NOTICES)}>
          返回通知中心
        </Button>
      </div>

      {/* 通知内容卡片 */}
      <NoticeDetailContent
        notice={notice}
        showReadStatus
        showMarkAsReadButton={!notice.isRead}
        onMarkAsRead={async () => {
          try {
            await markNoticeAsRead(notice.id);
            markAsRead(notice.id);
            setNotice({ ...notice, isRead: true });
            message.success("已标记为已读");
          } catch (error) {
            console.error("标记已读失败:", error);
            message.error("操作失败");
          }
        }}
      />
    </div>
  );
};

export default NoticeDetailPage;
