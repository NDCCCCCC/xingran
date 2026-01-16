/**
 * Captcha Background Modals Hook
 * 验证码背景模态框管理 Hook
 */

import { useState, useCallback } from 'react';
import { message } from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import type { CaptchaBackground, CaptchaBackgroundUpdateRequest } from '@/types/captcha';
import * as captchaService from '@/services/captcha';
import { SHAPE_OPTIONS } from '../constants';

export interface UseCaptchaModalsReturn {
  uploadModalVisible: boolean;
  editModalVisible: boolean;
  editingBg: CaptchaBackground | null;
  fileList: UploadFile[];
  uploading: boolean;
  uploadProps: UploadProps;

  setUploadModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setEditModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setEditingBg: React.Dispatch<React.SetStateAction<CaptchaBackground | null>>;
  setFileList: React.Dispatch<React.SetStateAction<UploadFile[]>>;
  setUploading: React.Dispatch<React.SetStateAction<boolean>>;

  openEditModal: (record: CaptchaBackground, editForm: any) => void;
  closeUploadModal: (uploadForm: any) => void;
  closeEditModal: (editForm: any) => void;
  handleUpload: (fileList: UploadFile[], uploadForm: any, onSuccess: () => void) => Promise<void>;
  handleUpdate: (editingBg: CaptchaBackground, editForm: any, onSuccess: () => void) => Promise<void>;
  handleDelete: (id: string, onSuccess: () => void) => Promise<void>;
  handleToggle: (id: string, onSuccess: () => void) => Promise<void>;
  handlePreload: () => Promise<void>;
}

export function useCaptchaModals(): UseCaptchaModalsReturn {
  const [uploadModalVisible, setUploadModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingBg, setEditingBg] = useState<CaptchaBackground | null>(null);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [uploading, setUploading] = useState(false);

  const uploadProps: UploadProps = {
    listType: 'picture-card',
    fileList,
    onChange: ({ fileList: newFileList }) => {
      setFileList(newFileList);
    },
    beforeUpload: (file) => {
      const isImage = file.type.startsWith('image/');
      if (!isImage) {
        message.error('只能上传图片文件');
        return Upload.LIST_IGNORE;
      }
      const isLt2M = file.size / 1024 / 1024 < 2;
      if (!isLt2M) {
        message.error('图片大小不能超过 2MB');
        return Upload.LIST_IGNORE;
      }
      return false; // 阻止自动上传
    },
    maxCount: 1,
  };

  const openEditModal = useCallback((record: CaptchaBackground, editForm: any) => {
    setEditingBg(record);
    editForm.setFieldsValue({
      pieceShape: record.pieceShape,
      difficultyLevel: record.difficultyLevel,
      allowedShapes: record.allowedShapes,
      status: record.status,
      sortOrder: record.sortOrder,
      remark: record.remark,
    });
    setEditModalVisible(true);
  }, []);

  const closeUploadModal = useCallback((uploadForm: any) => {
    setUploadModalVisible(false);
    uploadForm.resetFields();
    setFileList([]);
  }, []);

  const closeEditModal = useCallback((editForm: any) => {
    setEditModalVisible(false);
    editForm.resetFields();
    setEditingBg(null);
  }, []);

  const handleUpload = useCallback(
    async (fileList: UploadFile[], uploadForm: any, onSuccess: () => void) => {
      try {
        const values = await uploadForm.validateFields();
        if (fileList.length === 0) {
          message.warning('请选择要上传的图片');
          return;
        }

        setUploading(true);
        const file = fileList[0].originFileObj as File;

        await captchaService.uploadCaptchaBackground(file, {
          pieceShape: values.pieceShape,
          difficultyLevel: values.difficultyLevel,
          allowedShapes: values.allowedShapes,
          remark: values.remark,
        });

        message.success('上传成功');
        closeUploadModal(uploadForm);
        onSuccess();
      } catch (error: any) {
        if (error.errorFields) {
          return;
        }
        message.error('上传失败');
      } finally {
        setUploading(false);
      }
    },
    [closeUploadModal]
  );

  const handleUpdate = useCallback(
    async (editingBg: CaptchaBackground, editForm: any, onSuccess: () => void) => {
      try {
        const values = await editForm.validateFields();
        const updateData: CaptchaBackgroundUpdateRequest = {
          pieceShape: values.pieceShape,
          difficultyLevel: values.difficultyLevel,
          allowedShapes: values.allowedShapes,
          status: values.status,
          sortOrder: values.sortOrder,
          remark: values.remark,
        };

        await captchaService.updateCaptchaBackground(editingBg.id, updateData);
        message.success('更新成功');
        setEditModalVisible(false);
        editForm.resetFields();
        setEditingBg(null);
        onSuccess();
      } catch (error: any) {
        if (error.errorFields) {
          return;
        }
        message.error('更新失败');
      }
    },
    []
  );

  const handleDelete = useCallback(async (id: string, onSuccess: () => void) => {
    try {
      await captchaService.deleteCaptchaBackground(id);
      message.success('删除成功');
      onSuccess();
    } catch (error) {
      message.error('删除失败');
    }
  }, []);

  const handleToggle = useCallback(async (id: string, onSuccess: () => void) => {
    try {
      await captchaService.toggleCaptchaBackgroundStatus(id);
      message.success('状态更新成功');
      onSuccess();
    } catch (error) {
      message.error('状态更新失败');
    }
  }, []);

  const handlePreload = useCallback(async () => {
    try {
      await captchaService.preloadCaptchaCache();
      message.success('预加载成功');
    } catch (error) {
      message.error('预加载失败');
    }
  }, []);

  return {
    uploadModalVisible,
    editModalVisible,
    editingBg,
    fileList,
    uploading,
    uploadProps,
    setUploadModalVisible,
    setEditModalVisible,
    setEditingBg,
    setFileList,
    setUploading,
    openEditModal,
    closeUploadModal,
    closeEditModal,
    handleUpload,
    handleUpdate,
    handleDelete,
    handleToggle,
    handlePreload,
  };
}
