import { useState, useCallback, useEffect } from "react";
import type { FC } from "react";
import { Upload, App, Image, Progress, Button, Space } from "antd";
import { PlusOutlined, LoadingOutlined } from "@ant-design/icons";
import { getAuthHeaders } from "@/utils/authHelpers";
import type { UploadFile, UploadProps } from "antd";
import { MAX_IMAGE_SIZE } from "@/constants/upload";

// 自定义 UploadRequestOption 类型定义
export interface UploadRequestOption {
  action?: string;
  data?: Record<string, unknown> | ((file: UploadFile) => Record<string, unknown>);
  filename?: string;
  file: File | Blob | UploadFile | string;
  headers?: Record<string, string>;
  method?: "POST" | "PUT" | "post" | "put";
  onError?: (error: Error) => void;
  onProgress?: (percent: { percent: number }) => void;
  onSuccess?: (response: unknown) => void;
  withCredentials?: boolean;
}

export interface FileUploadResponse {
  id: string;
  fileName: string;
  fileSize: number;
  fileType: string;
  extension: string;
  url: string;
  storagePath?: string;
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
  listType?: "text" | "picture" | "picture-card" | "picture-circle";
  disabled?: boolean;
  category?: string;
  onUploadSuccess?: (file: UploadFile, response: FileUploadResponse) => void;
  onUploadError?: (file: UploadFile, error: unknown) => void;
}

const DEFAULT_MAX_SIZE = MAX_IMAGE_SIZE;

const FileUpload: FC<FileUploadProps> = ({
  value = [],
  onChange,
  accept = "image/*",
  maxCount = 1,
  maxSize = DEFAULT_MAX_SIZE,
  listType = "picture-card",
  disabled = false,
  category = "document",
  onUploadSuccess,
  onUploadError,
}) => {
  const { message } = App.useApp();
  const [fileList, setFileList] = useState<UploadFile[]>(Array.isArray(value) ? value : []);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);

  useEffect(() => {
    setFileList(Array.isArray(value) ? value : []);
  }, [value]);

  const handleChange: UploadProps["onChange"] = useCallback(({ fileList: newFileList }: { fileList: UploadFile[] }) => {
    setFileList(newFileList);
    onChange?.(newFileList);
  }, [onChange]);

  const isFileTypeAccepted = useCallback((file: File): boolean => {
    if (!accept) return true;

    const acceptTypes = accept.split(",").map(t => t.trim());
    const fileName = file.name.toLowerCase();
    return acceptTypes.some(type => {
      if (type.startsWith(".")) {
        return fileName.endsWith(type.toLowerCase());
      }
      if (type.endsWith("/*")) {
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
      message.error("不支持的文件类型");
      return Upload.LIST_IGNORE;
    }

    return true;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [maxSize, isFileTypeAccepted]);

  const handleUploadSuccess = useCallback((response: unknown, uploadFile: UploadFile) => {
    setUploading(false);
    setUploadProgress(100);
    onUploadSuccess?.(uploadFile, response as FileUploadResponse);
    message.success("上传成功");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [onUploadSuccess]);

  const handleUploadError = useCallback((error: Error, uploadFile: UploadFile) => {
    setUploading(false);
    onUploadError?.(uploadFile, error);
    message.error("上传失败");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [onUploadError]);

  const customRequest = useCallback(async (options: UploadRequestOption) => {
    const { file, onProgress, onSuccess, onError } = options;

    // 获取原始文件对象 - 处理 Ant Design 6.x 的 UploadRequestFile 类型
    let rawFile: File | Blob | null = null;
    let uploadFile: UploadFile;

    if (file instanceof File || file instanceof Blob) {
      rawFile = file;
      uploadFile = file instanceof File
        ? { originFileObj: file, name: file.name, uid: Date.now().toString() } as UploadFile
        : { originFileObj: file as File, name: "blob", uid: Date.now().toString() } as UploadFile;
    } else if (typeof file === "string") {
      // file 是字符串（URL）的情况，不应该发生在这里
      onError?.(new Error("不支持URL类型的文件"));
      return;
    } else if (typeof file === "object" && file !== null && "originFileObj" in file) {
      // file 是 UploadFile 对象
      uploadFile = file as UploadFile;
      rawFile = (file as UploadFile).originFileObj || null;
    } else {
      onError?.(new Error("无法获取文件"));
      return;
    }

    if (!rawFile) {
      onError?.(new Error("无法获取文件"));
      return;
    }

    const formData = new FormData();
    formData.append("file", rawFile);
    if (category) {
      formData.append("category", category);
    }

    setUploading(true);
    setUploadProgress(0);

    const xhr = new XMLHttpRequest();

    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        const percent = Math.round((e.loaded / e.total) * 100);
        setUploadProgress(percent);
        onProgress({ percent });
      }
    };

    xhr.onload = () => {
      if (xhr.status === 200) {
        const apiResponse = JSON.parse(xhr.responseText);
        if (apiResponse.code === 0 && apiResponse.data) {
          handleUploadSuccess(apiResponse.data, uploadFile);
          onSuccess?.(apiResponse.data);
        } else {
          const error = new Error(apiResponse.message || "上传失败");
          handleUploadError(error, uploadFile);
          onError?.(error);
        }
      } else {
        const error = new Error(xhr.responseText || "上传失败");
        handleUploadError(error, uploadFile);
        onError?.(error);
      }
    };

    xhr.onerror = () => {
      const error = new Error("网络错误");
      handleUploadError(error, uploadFile);
      onError?.(error);
    };

    const baseUrl = import.meta.env.VITE_API_BASE_URL || "";
    xhr.open("POST", `${baseUrl}/system/files/upload`, true);
    const headers = await getAuthHeaders();
    if (headers["Authorization"]) {
      xhr.setRequestHeader("Authorization", headers["Authorization"]);
    }
    xhr.send(formData);
  }, [category, handleUploadSuccess, handleUploadError]);

  const uploadButton = (
    <div>
      {uploading ? <LoadingOutlined /> : <PlusOutlined />}
      <div style={{ marginTop: 8 }}>{uploading ? "上传中" : "上传"}</div>
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
        const baseUrl = import.meta.env.VITE_API_BASE_URL || "";
        const headers = await getAuthHeaders();
        await fetch(`${baseUrl}/system/files/${file.response.data.id}`, {
          method: "DELETE",
          headers,
        });
      } catch (error) {
        console.error("删除文件失败:", error);
      }
    }
  }, []);

  const uploadProps = {
    listType,
    fileList,
    onChange: handleChange,
    beforeUpload,
    customRequest: customRequest as any, // eslint-disable-line @typescript-eslint/no-explicit-any
    accept,
    maxCount,
    disabled: disabled || uploading,
    onRemove: handleRemove,
  };

  if (listType === "picture-card" && maxCount === 1) {
    return (
      <Upload {...uploadProps}>
        {fileList.length >= maxCount ? null : uploadButton}
      </Upload>
    );
  }

  return (
    <Space orientation="vertical" style={{ width: "100%" }}>
      <Upload {...uploadProps}>
        <Button icon={<PlusOutlined />} disabled={disabled || uploading}>
          选择文件
        </Button>
      </Upload>
      {fileList.length > 0 && listType === "picture" && (
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
