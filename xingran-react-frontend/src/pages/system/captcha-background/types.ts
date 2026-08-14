/**
 * Captcha Background Types
 * 验证码背景类型定义
 */

import type { UploadFile } from "antd";
import type { UploadProps } from "antd";
import type { CaptchaBackground, PieceShape, DifficultyLevel } from "@/types/captcha";

// 统计数据
export interface CaptchaStatistics {
  totalCount: number;
  enabledCount: number;
  disabledCount: number;
  totalUsage: number;
}

// 模态框状态
export interface CaptchaModalState {
  uploadModalVisible: boolean;
  editModalVisible: boolean;
  editingBg: CaptchaBackground | null;
  fileList: UploadFile[];
  uploading: boolean;
}

// 选项类型
export interface SelectOption {
  label: string;
  value: string | number;
}
