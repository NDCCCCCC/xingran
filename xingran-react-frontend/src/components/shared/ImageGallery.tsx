/**
 * 图片画廊组件
 * 用于展示和管理图片集合，支持设置主图、预览、删除、描述编辑
 */

import { useState } from "react";
import type { FC } from "react";
import {
  Image,
  Button,
  Space,
  Modal,
  Input,
  Tag,
  Popconfirm,
  App,
  Spin,
  Empty,
  Row,
  Col,
} from "antd";
import { DeleteOutlined, EditOutlined, StarFilled } from "@ant-design/icons";

export interface Photo {
  id: string;
  roomId: string;
  fileId: string;
  fileName?: string;
  fileUrl: string;
  sortOrder: number;
  isPrimary: boolean;
  description?: string;
  createdAt: string;
}

export interface ImageGalleryProps {
  photos: Photo[];
  loading?: boolean;
  editable?: boolean;
  onSetPrimary?: (photoId: string) => void;
  onUpdateDescription?: (photoId: string, description: string) => void;
  onDelete?: (photoId: string) => void;
  onSortChange?: (photoIds: string[]) => void;
  onUpload?: () => void;
}

const ImageGallery: FC<ImageGalleryProps> = ({
  photos,
  loading = false,
  editable = true,
  onSetPrimary,
  onUpdateDescription,
  onDelete,
  onSortChange: _onSortChange,
  onUpload,
}) => {
  const { message } = App.useApp();
  const [previewVisible, setPreviewVisible] = useState(false);
  const [previewImage, setPreviewImage] = useState("");
  const [editingPhoto, setEditingPhoto] = useState<Photo | null>(null);
  const [description, setDescription] = useState("");
  const [modalVisible, setModalVisible] = useState(false);

  // 预览图片
  const handlePreview = (url: string) => {
    setPreviewImage(url);
    setPreviewVisible(true);
  };

  // 设置主图
  const handleSetPrimary = async (photo: Photo) => {
    if (!onSetPrimary) return;
    try {
      await onSetPrimary(photo.id);
      message.success("已设置为主图");
    } catch (error) {
      message.error("设置主图失败");
    }
  };

  // 编辑描述
  const handleEditDescription = (photo: Photo) => {
    setEditingPhoto(photo);
    setDescription(photo.description || "");
    setModalVisible(true);
  };

  // 保存描述
  const handleSaveDescription = async () => {
    if (!editingPhoto || !onUpdateDescription) return;
    try {
      await onUpdateDescription(editingPhoto.id, description);
      message.success("描述已更新");
      setModalVisible(false);
    } catch (error) {
      message.error("更新描述失败");
    }
  };

  // 删除照片
  const handleDelete = async (photo: Photo) => {
    if (!onDelete) return;
    try {
      await onDelete(photo.id);
      message.success("删除成功");
    } catch (error) {
      message.error("删除失败");
    }
  };

  // 渲染单张图片卡片
  const renderPhotoCard = (photo: Photo) => (
    <div key={photo.id} style={{ marginBottom: 16 }}>
      <div
        style={{
          position: "relative",
          paddingBottom: "75%",
          borderRadius: 8,
          overflow: "hidden",
          boxShadow: "0 2px 8px rgba(0,0,0,0.1)",
          backgroundColor: photo.isPrimary ? "#fff7e6" : "#fff",
        }}
      >
        {/* 图片 */}
        <img
          src={photo.fileUrl}
          alt={photo.description || photo.fileName}
          onClick={() => handlePreview(photo.fileUrl)}
          style={{
            position: "absolute",
            top: 0,
            left: 0,
            width: "100%",
            height: "100%",
            objectFit: "cover",
            cursor: "pointer",
          }}
        />

        {/* 主图标识 */}
        {photo.isPrimary && (
          <Tag
            color="gold"
            icon={<StarFilled />}
            style={{
              position: "absolute",
              top: 8,
              left: 8,
              margin: 0,
            }}
          >
            主图
          </Tag>
        )}
      </div>

      {/* 操作按钮 */}
      {editable && (
        <Space size="small" style={{ marginTop: 8 }}>
          {!photo.isPrimary && onSetPrimary && (
            <Button size="small" icon={<StarFilled />} onClick={() => handleSetPrimary(photo)}>
              设为主图
            </Button>
          )}
          {onUpdateDescription && (
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEditDescription(photo)}
            >
              编辑
            </Button>
          )}
          {onDelete && (
            <Popconfirm
              title="确定要删除这张照片吗？"
              onConfirm={() => handleDelete(photo)}
              okText="确定"
              cancelText="取消"
            >
              <Button size="small" danger icon={<DeleteOutlined />}>
                删除
              </Button>
            </Popconfirm>
          )}
        </Space>
      )}

      {/* 描述 */}
      {photo.description && (
        <div style={{ marginTop: 8, color: "#666", fontSize: 12 }}>{photo.description}</div>
      )}
    </div>
  );

  return (
    <div className="image-gallery">
      <Spin spinning={loading}>
        {/* 上传按钮 */}
        {editable && onUpload && (
          <div style={{ marginBottom: 16 }}>
            <Space>
              <Button type="primary" icon={<StarFilled />} onClick={onUpload}>
                上传照片
              </Button>
              {photos.length > 0 && <Tag>共 {photos.length} 张照片</Tag>}
            </Space>
          </div>
        )}

        {/* 图片网格 */}
        {photos.length === 0 ? (
          <Empty description="暂无照片" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Row gutter={[16, 16]}>
            {photos.map((photo) => (
              <Col key={photo.id} xs={24} sm={12} md={8} lg={6} xl={4}>
                {renderPhotoCard(photo)}
              </Col>
            ))}
          </Row>
        )}
      </Spin>

      {/* 预览模态框 */}
      <Image
        style={{ display: "none" }}
        preview={{
          visible: previewVisible,
          src: previewImage,
          onVisibleChange: (vis) => setPreviewVisible(vis),
        }}
      />

      {/* 编辑描述模态框 */}
      <Modal
        title="编辑照片描述"
        open={modalVisible}
        onOk={handleSaveDescription}
        onCancel={() => setModalVisible(false)}
        okText="保存"
        cancelText="取消"
      >
        <Input.TextArea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="请输入照片描述"
          rows={4}
          maxLength={200}
          showCount
        />
      </Modal>
    </div>
  );
};

export default ImageGallery;
