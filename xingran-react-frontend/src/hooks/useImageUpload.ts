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
      setImageUrl(`/uploads/${response.storagePath || ""}`);
      message.success("图片上传成功");
      onSuccess?.(response.id, `/uploads/${response.storagePath || ""}`);
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
          response: { id: fileId, storagePath: fileUrl.replace("/uploads/", "") },
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
