/**
 * 可复用的图片上传 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { UploadFile } from "antd";
import type { FileUploadResponse } from "@/components/shared/FileUpload";
import { MAX_IMAGE_SIZE } from "@/constants/upload";

export interface UseImageUploadOptions {
  businessType?: string;
  maxSize?: number;
  onSuccess?: (fileId: string, fileUrl: string) => void;
  onError?: (error: Error) => void;
}

export interface UseImageUploadReturn {
  uploading: boolean;
  fileList: UploadFile[];
  imageId: string | undefined;
  imageUrl: string | undefined;
  handleUploadChange: (fileList: UploadFile[]) => void;
  handleUploadSuccess: (file: UploadFile, response: FileUploadResponse) => void;
  handleUploadError: (file: UploadFile, error: unknown) => void;
  resetUpload: () => void;
  setInitialValue: (fileId: string, fileUrl?: string) => void;
}

const DEFAULT_MAX_SIZE = MAX_IMAGE_SIZE;

export function useImageUpload(options: UseImageUploadOptions = {}): UseImageUploadReturn {
  const {
    businessType: _businessType = "image",
    maxSize: _maxSize = DEFAULT_MAX_SIZE,
    onSuccess,
    onError,
  } = options;

  const { message } = App.useApp();
  const [uploading, setUploading] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [imageId, setImageId] = useState<string>();
  const [imageUrl, setImageUrl] = useState<string>();

  const handleUploadChange = useCallback((newFileList: UploadFile[]) => {
    setFileList(newFileList);
    if (newFileList.length === 0) {
      setImageId(undefined);
      setImageUrl(undefined);
    }
  }, []);

  const handleUploadSuccess = useCallback(
    (file: UploadFile, response: FileUploadResponse) => {
      setUploading(false);
      setImageId(response.id);
      // 前缀用 Vite base：dev 下 BASE_URL='/' 行为不变，prod 下产出 /xingran/uploads/...
      // 与 nginx location /xingran/ 对齐；后端 /uploads/* 在 nginx 剥前缀后正常服务。
      const uploadsBase = `${import.meta.env.BASE_URL}uploads/`;
      setImageUrl(`${uploadsBase}${response.storagePath || ""}`);
      message.success("图片上传成功");
      onSuccess?.(response.id, `${uploadsBase}${response.storagePath || ""}`);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [onSuccess]
  );

  const handleUploadError = useCallback(
    (_file: UploadFile, error: unknown) => {
      setUploading(false);
      message.error("图片上传失败");
      onError?.(error instanceof Error ? error : new Error(String(error)));
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [onError]
  );

  const resetUpload = useCallback(() => {
    setFileList([]);
    setImageId(undefined);
    setImageUrl(undefined);
    setUploading(false);
  }, []);

  const setInitialValue = useCallback((fileId: string, fileUrl?: string) => {
    if (!fileId) return;

    setImageId(fileId);
    setImageUrl(fileUrl);

    if (fileUrl) {
      setFileList([
        {
          uid: fileId,
          name: "平面图",
          status: "done",
          url: fileUrl,
          // 取 fileUrl 中 "/uploads/" 最后一次出现之后的部分作为 storagePath，
          // 兼容不带前缀('/uploads/abc')和带 BASE_URL 前缀('/xingran/uploads/abc')两种旧/新格式。
          response: { id: fileId, storagePath: fileUrl.split("/uploads/").pop() || "" },
        },
      ]);
    } else {
      setFileList([]);
    }
  }, []);

  return {
    uploading,
    fileList,
    imageId,
    imageUrl,
    handleUploadChange,
    handleUploadSuccess,
    handleUploadError,
    resetUpload,
    setInitialValue,
  };
}

export default useImageUpload;
