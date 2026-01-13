import { get, post, put, upload } from './api';
import type {
  UserProfile,
  UpdateProfileRequest,
  ChangePasswordRequest,
  UserPreferences,
} from '@/types';

// ==================== 个人信息 API ====================

/**
 * 获取当前用户个人信息
 */
export function getProfileInfo(): Promise<UserProfile> {
  return get<UserProfile>('/system/profile/info');
}

/**
 * 更新个人信息
 */
export function updateProfileInfo(data: UpdateProfileRequest): Promise<{ message: string }> {
  return put('/system/profile/info', data);
}

/**
 * 修改密码
 */
export function changePassword(data: ChangePasswordRequest): Promise<{ message: string }> {
  return post('/system/profile/change-password', data);
}

/**
 * 上传头像
 */
export function uploadAvatar(file: File): Promise<{ avatar: string; message: string }> {
  return upload('/system/profile/avatar', file);
}

// ==================== 系统设置 API ====================

/**
 * 获取用户个人设置
 */
export function getUserPreferences(): Promise<UserPreferences> {
  return get<UserPreferences>('/system/settings/preferences');
}

/**
 * 更新用户个人设置
 */
export function updateUserPreferences(data: UserPreferences): Promise<{ message: string }> {
  return put('/system/settings/preferences', data);
}
