import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { post } from '@/lib/api';
import type { User, LoginRequest } from '@/types';
import { useMenuStore } from './menuStore';
import { getEncryptedLoginRequest } from '@/utils/sm2';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  loading: boolean;
}

interface AuthActions {
  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => void;
  refreshTokenAction: () => Promise<void>;
  updateUser: (user: Partial<User>) => void;
}

export const useAuthStore = create<AuthState & AuthActions>()(
  persist(
    (set, get) => ({
      // 初始状态
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      loading: false,

      // 登录
      login: async (credentials: LoginRequest) => {
        set({ loading: true });
        try {
          // 使用 SM2 加密密码（带回退机制）
          const encryptedRequest = await getEncryptedLoginRequest(
            credentials.username,
            credentials.password
          );

          const loginRequest = {
            username: encryptedRequest.username,
            password: encryptedRequest.password,
            encryptedPassword: encryptedRequest.encryptedPassword,
            captcha: credentials.captcha,
            captchaId: credentials.captchaId,
          };

          const response = await post('/system/auth/login', loginRequest);
          const { user, accessToken, refreshToken } = response.data;

          // 保存到localStorage
          localStorage.setItem('accessToken', accessToken);
          localStorage.setItem('refreshToken', refreshToken);

          set({
            user,
            accessToken,
            refreshToken,
            isAuthenticated: true,
            loading: false,
          });
        } catch (error) {
          set({ loading: false });
          throw error;
        }
      },

      // 登出
      logout: () => {
        // 清除localStorage
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');

        // 清除菜单数据
        useMenuStore.getState().clearMenus();

        // 清除状态
        set({
          user: null,
          accessToken: null,
          refreshToken: null,
          isAuthenticated: false,
        });
      },

      // 刷新令牌
      refreshTokenAction: async () => {
        const { refreshToken } = get();
        if (!refreshToken) {
          get().logout();
          return;
        }

        try {
          const response = await post('/system/auth/refresh', { refreshToken });
          const { accessToken: newAccessToken, refreshToken: newRefreshToken } = response.data;

          // 更新localStorage
          localStorage.setItem('accessToken', newAccessToken);
          localStorage.setItem('refreshToken', newRefreshToken);

          set({
            accessToken: newAccessToken,
            refreshToken: newRefreshToken,
          });
        } catch (error) {
          // 刷新失败，清除状态
          get().logout();
          throw error;
        }
      },

      // 更新用户信息
      updateUser: (userData: Partial<User>) => {
        const { user } = get();
        if (user) {
          set({
            user: { ...user, ...userData },
          });
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);// Last updated: 2025年12月16日 13:58:00
