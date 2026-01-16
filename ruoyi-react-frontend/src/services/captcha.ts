// 验证码相关API服务
import { post, postFormData } from '@/lib/api';
import type {
  CaptchaResponse,
  CaptchaConfig,
  SliderVerifyRequest,
  SliderVerifyResponse,
  CaptchaBackgroundListRequest,
  CaptchaBackgroundListResponse,
  CaptchaBackgroundUpdateRequest,
  CaptchaBackground,
  StatisticsResponse,
} from '@/types/captcha';

// API响应类型 - 响应拦截器已经返回了 data 对象
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  timestamp: number;
  request_id: string;
}

// ==================== 验证码相关API ====================

// 获取验证码
export async function getCaptcha(): Promise<CaptchaResponse> {
  const response = await post<ApiResponse<CaptchaResponse>>('/system/auth/captcha', {});
  return response.data;  // 响应拦截器返回的是完整的 data 对象，需要访问嵌套的 data 属性
}

// 获取验证码配置
export async function getCaptchaConfig(): Promise<CaptchaConfig> {
  const response = await post<ApiResponse<CaptchaConfig>>('/system/auth/captcha/config', {});
  return response.data;  // 响应拦截器返回的是完整的 data 对象，需要访问嵌套的 data 属性
}

// 验证滑动验证码
export async function verifySliderCaptcha(data: SliderVerifyRequest): Promise<SliderVerifyResponse> {
  const response = await post<ApiResponse<SliderVerifyResponse>>('/system/auth/captcha/verify/slider', data);
  return response.data;  // 响应拦截器返回的是完整的 data 对象，需要访问嵌套的 data 属性
}

// ==================== 验证码背景图管理API ====================

// 获取背景图列表
export async function getCaptchaBackgroundList(params: CaptchaBackgroundListRequest): Promise<CaptchaBackgroundListResponse> {
  const response = await post<ApiResponse<CaptchaBackgroundListResponse>>('/system/captcha-backgrounds/list', params);
  return response.data;
}

// 上传背景图
export async function uploadCaptchaBackground(file: File, params: {
  pieceShape: string;
  difficultyLevel: number;
  allowedShapes?: string[];
  remark?: string;
}): Promise<CaptchaBackground> {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('pieceShape', params.pieceShape);
  formData.append('difficultyLevel', params.difficultyLevel.toString());
  if (params.allowedShapes) {
    formData.append('allowedShapes', JSON.stringify(params.allowedShapes));
  }
  if (params.remark) {
    formData.append('remark', params.remark);
  }
  return postFormData('/system/captcha-backgrounds/upload', formData);
}

// 获取背景图详情
export async function getCaptchaBackground(id: string): Promise<CaptchaBackground> {
  const response = await post<ApiResponse<CaptchaBackground>>(`/system/captcha-backgrounds/${id}`, {});
  return response.data;
}

// 更新背景图
export async function updateCaptchaBackground(id: string, data: CaptchaBackgroundUpdateRequest): Promise<CaptchaBackground> {
  const response = await post<ApiResponse<CaptchaBackground>>(`/system/captcha-backgrounds/${id}/update`, data);
  return response.data;
}

// 删除背景图
export async function deleteCaptchaBackground(id: string): Promise<{ success: boolean; message: string }> {
  const response = await post<ApiResponse<{ success: boolean; message: string }>>(`/system/captcha-backgrounds/${id}/delete`, {});
  return response.data;
}

// 切换背景图状态
export async function toggleCaptchaBackgroundStatus(id: string): Promise<CaptchaBackground> {
  const response = await post<ApiResponse<CaptchaBackground>>(`/system/captcha-backgrounds/${id}/toggle`, {});
  return response.data;
}

// 获取统计信息
export async function getCaptchaBackgroundStatistics(): Promise<StatisticsResponse> {
  const response = await post<ApiResponse<StatisticsResponse>>('/system/captcha-backgrounds/statistics', {});
  return response.data;
}

// 预加载缓存
export async function preloadCaptchaCache(): Promise<{ preloaded: number; message: string }> {
  const response = await post<ApiResponse<{ preloaded: number; message: string }>>('/system/captcha-backgrounds/preload', {});
  return response.data;
}

export default {
  getCaptcha,
  getCaptchaConfig,
  verifySliderCaptcha,
  // 背景图管理
  getCaptchaBackgroundList,
  uploadCaptchaBackground,
  getCaptchaBackground,
  updateCaptchaBackground,
  deleteCaptchaBackground,
  toggleCaptchaBackgroundStatus,
  getCaptchaBackgroundStatistics,
  preloadCaptchaCache,
};
