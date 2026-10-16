// 验证码相关类型定义

// 验证码类型（与后端保持一致）
export type CaptchaType = "normal" | "slider";

// 验证码开关类型
export type CaptchaEnabled = "disabled" | "normal" | "slider";

// 验证码响应
export interface CaptchaResponse {
  captchaId: string;
  captchaType: CaptchaType;
  captchaImg?: string; // 文字验证码图片 (base64)
  sliderImg?: string; // 滑动底图 (base64)
  pieceImg?: string; // 拼图块 (base64)
  yPos?: number; // 拼图块Y坐标
  token?: string; // 验证token (滑动类型)
}

// 滑动验证码验证请求
export interface SliderVerifyRequest {
  captchaId: string;
  xPos: number;
  token: string;
}

// 滑动验证码验证响应
export interface SliderVerifyResponse {
  success: boolean;
  token: string;
  message?: string;
}

// 验证码配置
export interface CaptchaConfig {
  enabled: CaptchaEnabled;
  type: number; // 文字验证码长度
  expireTime: number; // 有效期(分钟)
  maxAttempts: number; // 最大验证次数
}

// 验证码组件Props
export interface CaptchaProps {
  value?: string;
  captchaId?: string;
  onChange?: (value: string, captchaId: string) => void;
  onError?: (error: string) => void;
}

// ==================== 验证码背景图管理类型 ====================

// 拼图形状
export type PieceShape = "circle" | "square" | "star" | "heart";

// 难度级别
export type DifficultyLevel = 1 | 2 | 3; // 1:简单 2:中等 3:困难

// 背景图状态
export type CaptchaBackgroundStatus = 0 | 1; // 0:禁用 1:启用

// 验证码背景图
export interface CaptchaBackground {
  id: string;
  fileName: string;
  filePath: string;
  fileSize: number;
  fileWidth: number;
  fileHeight: number;
  fileMD5?: string;
  pieceShape: PieceShape;
  difficultyLevel: DifficultyLevel;
  allowedShapes?: string[];
  useCount: number;
  lastUsedAt?: string;
  sortOrder: number;
  status: CaptchaBackgroundStatus;
  remark?: string;
  createdAt: string;
  updatedAt: string;
  previewUrl: string; // 预览图片 URL（web 可访问的相对路径）
}

// 背景图列表请求参数
export interface CaptchaBackgroundListRequest {
  current?: number;
  pageSize?: number;
  fileName?: string;
  pieceShape?: PieceShape;
  difficultyLevel?: DifficultyLevel;
  status?: CaptchaBackgroundStatus;
}

// 背景图列表响应
export interface CaptchaBackgroundListResponse {
  items: CaptchaBackground[];
  total: number;
}

// 背景图更新请求
export interface CaptchaBackgroundUpdateRequest {
  pieceShape?: PieceShape;
  difficultyLevel?: DifficultyLevel;
  allowedShapes?: string[];
  status?: CaptchaBackgroundStatus;
  sortOrder?: number;
  remark?: string;
}

// 统计响应
export interface StatisticsResponse {
  totalCount: number;
  enabledCount: number;
  disabledCount: number;
  shapeDistribution: Record<string, number>;
  difficultyDist: Record<number, number>;
  totalUsage: number;
}

// 背景图模式
export type BackgroundMode = "auto" | "custom" | "mixed";
