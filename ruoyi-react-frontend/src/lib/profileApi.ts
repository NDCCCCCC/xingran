import { get, post, put, upload } from "./api";
import type { UserProfile, UpdateProfileRequest, ChangePasswordRequest } from "@/types";
import type { BackendUserPreferences } from "@/types/config";

// ==================== 个人信息 API ====================

/**
 * 获取当前用户个人信息
 */
export async function getProfileInfo(): Promise<UserProfile> {
  const response = await get<UserProfile>("/system/profile/info");
  return response.data!;
}

/**
 * 更新个人信息
 */
export async function updateProfileInfo(data: UpdateProfileRequest): Promise<{ message: string }> {
  const response = await put<{ message: string }>("/system/profile/info", data);
  return response.data!;
}

/**
 * 修改密码
 */
export async function changePassword(data: ChangePasswordRequest): Promise<{ message: string }> {
  const response = await post<{ message: string }>("/system/profile/change-password", data);
  return response.data!;
}

/**
 * 上传头像
 */
export async function uploadAvatar(file: File): Promise<{ avatar: string; message: string }> {
  const response = await upload<{ avatar: string; message: string }>(
    "/system/profile/avatar",
    file
  );
  return response.data!;
}

// ==================== 系统设置 API ====================

/**
 * 获取用户个人设置
 * 注意：后端返回的是 BackendUserPreferences 格式（扁平结构）
 */
export async function getUserPreferences(): Promise<BackendUserPreferences> {
  const response = await get<BackendUserPreferences>("/system/settings/preferences");
  return response.data!;
}

/**
 * 更新用户个人设置
 * 注意：后端接收的是 BackendUserPreferences 格式（扁平结构）
 */
export async function updateUserPreferences(
  data: BackendUserPreferences
): Promise<{ message: string }> {
  const response = await put<{ message: string }>("/system/settings/preferences", data);
  return response.data!;
}
