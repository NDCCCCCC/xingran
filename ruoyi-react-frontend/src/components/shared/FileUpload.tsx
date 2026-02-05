import { useState, useCallback, useEffect } from 'react';
import type { FC } from 'react';
import { Upload, message, Image, Progress, Button, Space } from 'antd';
import { PlusOutlined, LoadingOutlined } from '@ant-design/icons';
import type { UploadFile, UploadProps, UploadRequestOption } from 'antd';

export interface FileUploadResponse {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  extension: string;
  url: string;
  width?: number;
  height?: number;
  metadata?: string;
  createdAt: string;
}

export interface FileUploadProps {
  value?: UploadFile[];
  onChange?: (files: UploadFile[]) => void;
  accept?: string;
  maxCount?: number;
  maxSize?: number;
  listType?: 'text' | 'picture' | 'picture-card' | 'picture-circle';
  disabled?: boolean;
  category?: string;
  onUploadSuccess?: (file: UploadFile, response: FileUploadResponse) => void;
  onUploadError?: (file: UploadFile, error: unknown) => void;
}

const DEFAULT_MAX_SIZE = 5 * 1024 * 1024;

const FileUpload: FC<FileUploadProps> = ({
  value = [],
  onChange,
  accept = 'image/*',
  maxCount = 1,
  maxSize = DEFAULT_MAX_SIZE,
  listType = 'picture-card',
  disabled = false,
  category = 'document',
  onUploadSuccess,
  onUploadError,
}) => {
  const [fileList, setFileList] = useState<UploadFile[]>(Array.isArray(value) ? value : []);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);

  useEffect(() => {
    setFileList(Array.isArray(value) ? value : []);
  }, [value]);

  const handleChange: UploadProps['onChange'] = useCallback(({ fileList: newFileList }) => {
    setFileList(newFileList);
    onChange?.(newFileList);
  }, [onChange]);

  const isFileTypeAccepted = useCallback((file: File): boolean => {
    if (!accept) return true;

    const acceptTypes = accept.split(',').map(t => t.trim());
    const fileName = file.name.toLowerCase();
    return acceptTypes.some(type => {
      if (type.startsWith('.')) {
        return fileName.endsWith(type.toLowerCase());
      }
      if (type.endsWith('/*')) {
        return file.type.startsWith(type.slice(0, -2));
      }
      return file.type === type;
    });
  }, [accept]);

  const beforeUpload = useCallback((file: File) => {
    if (file.size > maxSize) {
      message.error(`文件大小不能超过 ${(maxSize / 1024 / 1024).toFixed(0)}MB`);
      return Upload.LIST_IGNORE;
    }

    if (!isFileTypeAccepted(file)) {
      message.error('不支持的文件类型');
      return Upload.LIST_IGNORE;
    }

    return true;
  }, [maxSize, isFileTypeAccepted]);

  const handleUploadProgress = useCallback((e: ProgressEvent, file: UploadFile, onProgress: (progress: { percent: number }, file: UploadFile) => void) => {
    if (e.lengthComputable) {
      const percent = Math.round((e.loaded / e.total) * 100);
      setUploadProgress(percent);
      onProgress({ percent }, file);
    }
  }, []);

  const handleUploadSuccess = useCallback((response: unknown, file: UploadFile) => {
    setUploading(false);
    setUploadProgress(100);
    onUploadSuccess?.(file, response as FileUploadResponse);
    message.success('上传成功');
  }, [onUploadSuccess]);

  const handleUploadError = useCallback((error: Error, file: UploadFile, onError: (error: Error) => void) => {
    setUploading(false);
    onError(error);
    onUploadError?.(file, error);
    message.error('上传失败');
  }, [onUploadError]);

  const customRequest = useCallback((options: UploadRequestOption) => {
    const { file, onProgress, onSuccess, onError } = options;
    const formData = new FormData();
    formData.append('file', file);
    if (category) {
      formData.append('category', category);
    }

    setUploading(true);
    setUploadProgress(0);

    const xhr = new XMLHttpRequest();

    xhr.upload.onprogress = (e) => handleUploadProgress(e, file, onProgress);

    xhr.onload = () => {
      if (xhr.status === 200) {
        const apiResponse = JSON.parse(xhr.responseText);
        if (apiResponse.code === 0 && apiResponse.data) {
          handleUploadSuccess(apiResponse.data, file);
          onSuccess(apiResponse.data, file);
        } else {
          const error = new Error(apiResponse.message || '上传失败');
          handleUploadError(error, file, onError);
        }
      } else {
        const error = new Error(xhr.responseText || '上传失败');
        handleUploadError(error, file, onError);
      }
    };

    xhr.onerror = () => {
      const error = new Error('网络错误');
      handleUploadError(error, file, onError);
    };

    xhr.open('POST', '/api/v1/system/files/upload', true);
    const token = localStorage.getItem('accessToken');
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`);
    }
    xhr.send(formData);
  }, [category, handleUploadProgress, handleUploadSuccess, handleUploadError]);

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

  const handleRemove = useCallback(async (file: UploadFile) => {
    if (file.response?.data?.id) {
      try {
        await fetch(`/api/v1/system/files/${file.response.data.id}`, {
          method: 'DELETE',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('accessToken')}`,
          },
        });
      } catch (error) {
        console.error('删除文件失败:', error);
      }
    }
  }, []);

  const uploadProps = {
    listType,
    fileList,
    onChange: handleChange,
    beforeUpload,
    customRequest,
    accept,
    maxCount,
    disabled: disabled || uploading,
    onRemove: handleRemove,
  };

  if (listType === 'picture-card' && maxCount === 1) {
    return (
      <Upload {...uploadProps}>
        {fileList.length >= maxCount ? null : uploadButton}
      </Upload>
    );
  }

  return (
    <Space orientation="vertical" style={{ width: '100%' }}>
      <Upload {...uploadProps}>
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
