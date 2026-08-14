import { useEffect, useState, useCallback } from "react";
import type { FC } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { App, Button, Spin } from "antd";
import { ArrowLeftOutlined } from "@ant-design/icons";
import { getNoticeDetail } from "@/lib/noticeApi";
import type { Notice } from "@/types/notice";
import { formatDateTime } from "@/utils/datetime";
import NoticeDetailContent from "@/components/NoticeDetail/NoticeDetailContent";

/**
 * 管理端通知详情页面
 */
const AdminNoticeDetailPage: FC = () => {
  const { message } = App.useApp();
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
      console.error("加载通知详情失败:", error);
      message.error("加载失败，请稍后重试");
      navigate("/system/notice");
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
        <Button type="link" onClick={() => navigate("/system/notice")}>
          返回通知列表
        </Button>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* 头部操作栏 */}
      <div className="mb-4">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate("/system/notice")}>
          返回通知列表
        </Button>
      </div>

      {/* 通知内容卡片 */}
      <NoticeDetailContent notice={notice} showPublishStatus showCreator />

      {/* 底部关闭按钮 */}
      <div className="flex justify-end mt-4">
        <Button onClick={() => navigate("/system/notice")}>关闭</Button>
      </div>
    </div>
  );
};

export default AdminNoticeDetailPage;
