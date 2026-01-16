/**
 * 文件上传组件
 * 支持单文件/多文件上传、图片预览、上传进度显示
 */

import { useState } from 'react';
import type { FC } from 'react';
import { Upload, message, Image, Progress, Button, Space } from 'antd';
import { PlusOutlined, LoadingOutlined } from '@ant-design/icons';
import type { UploadFile, UploadProps } from 'antd';

export interface FileUploadProps {
  value?: UploadFile[];
  onChange?: (files: UploadFile[]) => void;
  accept?: string; // 接受的文件类型，如 'image/*,.pdf'
  maxCount?: number; // 最大文件数量
  maxSize?: number; // 最大文件大小（字节）
  listType?: 'text' | 'picture' | 'picture-card' | 'picture-circle';
  disabled?: boolean;
  category?: string; // 文件分类，如 'avatar', 'room-photo', 'document'
  onUploadSuccess?: (file: UploadFile, response: any) => void;
  onUploadError?: (file: UploadFile, error: any) => void;
}

const FileUpload: FC<FileUploadProps> = ({
  value = [],
  onChange,
  accept = 'image/*',
  maxCount = 1,
  maxSize = 5 * 1024 * 1024, // 默认5MB
  listType = 'picture-card',
  disabled = false,
  category = 'document',
  onUploadSuccess,
  onUploadError,
}) => {
  const [fileList, setFileList] = useState<UploadFile[]>(value);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);

  // 处理文件变化
  const handleChange: UploadProps['onChange'] = ({ fileList: newFileList }) => {
    setFileList(newFileList);
    onChange?.(newFileList);
  };

  // 上传前验证
  const beforeUpload = (file: File) => {
    // 验证文件大小
    if (file.size > maxSize) {
      message.error(`文件大小不能超过 ${(maxSize / 1024 / 1024).toFixed(0)}MB`);
      return Upload.LIST_IGNORE;
    }

    // 验证文件类型
    if (accept) {
      const acceptTypes = accept.split(',').map(t => t.trim());
      const isAccepted = acceptTypes.some(type => {
        if (type.startsWith('.')) {
          return file.name.toLowerCase().endsWith(type.toLowerCase());
        }
        if (type.endsWith('/*')) {
          const mimeType = type.split('/')[0];
          return file.type.startsWith(mimeType + '/');
        }
        return file.type === type;
      });

      if (!isAccepted) {
        message.error('不支持的文件类型');
        return Upload.LIST_IGNORE;
      }
    }

    return true;
  };

  // 自定义上传请求
  const customRequest = async (options: any) => {
    const { file, onProgress, onSuccess, onError } = options;
    const formData = new FormData();
    formData.append('file', file);
    if (category) {
      formData.append('category', category);
    }

    setUploading(true);
    setUploadProgress(0);

    try {
      // 使用 XMLHttpRequest 来跟踪上传进度
      const xhr = new XMLHttpRequest();

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          const percent = Math.round((e.loaded / e.total) * 100);
          setUploadProgress(percent);
          onProgress({ percent }, file);
        }
      };

      xhr.onload = () => {
        if (xhr.status === 200) {
          const response = JSON.parse(xhr.responseText);
          setUploading(false);
          setUploadProgress(100);
          onSuccess(response, file);
          onUploadSuccess?.(file, response);
          message.success('上传成功');
        } else {
          throw new Error(xhr.responseText || '上传失败');
        }
      };

      xhr.onerror = () => {
        setUploading(false);
        const error = new Error('网络错误');
        onError(error);
        onUploadError?.(file, error);
        message.error('上传失败');
      };

      xhr.open('POST', '/system/files/upload', true);
      // 从 localStorage 获取 token
      const token = localStorage.getItem('token');
      if (token) {
        xhr.setRequestHeader('Authorization', `Bearer ${token}`);
      }
      xhr.send(formData);
    } catch (error) {
      setUploading(false);
      onError(error);
      onUploadError?.(file, error);
      message.error('上传失败');
    }
  };

  // 上传按钮
  const uploadButton = (
    <div>
      {uploading ? <LoadingOutlined /> : <PlusOutlined />}
      <div style={{ marginTop: 8 }}>{uploading ? '上传中' : '上传'}</div>
      {uploading && (
        <Progress
          percent={uploadProgress}
          size="small"
          showInfo={false}
          style={{ marginTop: 4 }}
        />
      )}
    </div>
  );

  // 删除文件
  const handleRemove = async (file: UploadFile) => {
    if (file.response?.data?.id) {
      try {
        await fetch(`/system/files/${file.response.data.id}`, {
          method: 'DELETE',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
          },
        });
      } catch (error) {
        console.error('删除文件失败:', error);
      }
    }
  };

  if (listType === 'picture-card' && maxCount === 1) {
    return (
      <Upload
        listType="picture-card"
        fileList={fileList}
        onChange={handleChange}
        beforeUpload={beforeUpload}
        customRequest={customRequest}
        accept={accept}
        maxCount={maxCount}
        disabled={disabled || uploading}
        onRemove={handleRemove}
      >
        {fileList.length >= maxCount ? null : uploadButton}
      </Upload>
    );
  }

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Upload
        listType={listType}
        fileList={fileList}
        onChange={handleChange}
        beforeUpload={beforeUpload}
        customRequest={customRequest}
        accept={accept}
        maxCount={maxCount}
        disabled={disabled || uploading}
        onRemove={handleRemove}
      >
        <Button icon={<PlusOutlined />} disabled={disabled || uploading}>
          选择文件
        </Button>
      </Upload>
      {fileList.length > 0 && listType === 'picture' && (
        <Image.PreviewGroup>
          {fileList.map(file => (
            <Image
              key={file.uid}
              src={file.url || file.response?.data?.fileUrl}
              width={100}
              style={{ marginRight: 8 }}
            />
          ))}
        </Image.PreviewGroup>
      )}
    </Space>
  );
};

export default FileUpload;
