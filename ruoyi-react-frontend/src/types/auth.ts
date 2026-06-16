/**
 * 认证和个人信息相关类型
 */

import type { Gender } from "./base";
import type { User } from "./system";

// ==================== 登录相关 ====================

/**
 * 登录请求
 */
export interface LoginRequest {
  username: string;
  password: string;
  encryptedPassword?: boolean;
  captcha?: string;
  captchaId?: string;
}

/**
 * 登录响应
 */
export interface LoginResponse {
  user: User;
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  tokenType: string;
}

// ==================== 个人信息相关 ====================

/**
 * 用户详细信息
 */
export interface UserProfile extends User {
  remark?: string;
  pwdUpdateTime?: string;
}

/**
 * 个人信息更新请求
 */
export interface UpdateProfileRequest {
  nickname?: string;
  email?: string;
  phone?: string;
  gender: Gender;
  remark?: string;
}

/**
 * 修改密码请求
 */
export interface ChangePasswordRequest {
  oldPassword: string;
  newPassword: string;
}

// ==================== 用户设置相关 ====================

/**
 * 主题模式
 */
export type ThemeMode = "light" | "dark";

/**
 * 语言设置
 */
export type Language = "zh-CN" | "en-US";

/**
 * 用户个人设置
 */
export interface UserPreferences {
  theme: ThemeMode;
  language: Language;
  pageSize: number;
  sidebarCollapsed: boolean;
}
