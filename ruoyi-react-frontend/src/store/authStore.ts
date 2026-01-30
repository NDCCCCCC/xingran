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
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      loading: false,

      login: async (credentials: LoginRequest) => {
        set({ loading: true });
        try {
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

          const response = await post('/system/auth/login', loginRequest) as { data: { user: any; accessToken: string; refreshToken: string } };
          const { user, accessToken, refreshToken } = response.data;

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

      logout: () => {
        localStorage.removeItem('accessToken');
        localStorage.removeItem('refreshToken');

        useMenuStore.getState().clearMenus();

        set({
          user: null,
          accessToken: null,
          refreshToken: null,
          isAuthenticated: false,
        });
      },

      refreshTokenAction: async () => {
        const { refreshToken } = get();
        if (!refreshToken) {
          get().logout();
          return;
        }

        try {
          const response = await post('/system/auth/refresh', { refreshToken }) as { data: { accessToken: string; refreshToken: string } };
          const { accessToken: newAccessToken, refreshToken: newRefreshToken } = response.data;

          localStorage.setItem('accessToken', newAccessToken);
          localStorage.setItem('refreshToken', newRefreshToken);

          set({
            accessToken: newAccessToken,
            refreshToken: newRefreshToken,
          });
        } catch (error) {
          get().logout();
          throw error;
        }
      },

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
);
