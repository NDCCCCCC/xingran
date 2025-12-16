import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import Login from './index';

// Mock useAuthStore
jest.mock('@/store/authStore', () => ({
  useAuthStore: jest.fn(() => ({
    login: jest.fn(),
  })),
}));

describe('Login Component', () => {
  const renderLogin = () => {
    return render(
      <ConfigProvider>
        <BrowserRouter>
          <Login />
        </BrowserRouter>
      </ConfigProvider>
    );
  };

  test('renders login form', () => {
    renderLogin();

    expect(screen.getByText('RuoYi Next')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('用户名')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('密码')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '登录' })).toBeInTheDocument();
  });

  test('shows validation errors for empty fields', async () => {
    renderLogin();

    const loginButton = screen.getByRole('button', { name: '登录' });
    fireEvent.click(loginButton);

    await waitFor(() => {
      expect(screen.getByText('请输入用户名')).toBeInTheDocument();
      expect(screen.getByText('请输入密码')).toBeInTheDocument();
    });
  });

  test('submits form with valid credentials', async () => {
    const mockLogin = jest.fn();
    jest.doMock('@/store/authStore', () => ({
      useAuthStore: () => ({
        login: mockLogin,
      }),
    }));

    renderLogin();

    const usernameInput = screen.getByPlaceholderText('用户名');
    const passwordInput = screen.getByPlaceholderText('密码');
    const loginButton = screen.getByRole('button', { name: '登录' });

    fireEvent.change(usernameInput, { target: { value: 'admin' } });
    fireEvent.change(passwordInput, { target: { value: 'admin123' } });
    fireEvent.click(loginButton);

    await waitFor(() => {
      expect(mockLogin).toHaveBeenCalledWith({
        username: 'admin',
        password: 'admin123',
      });
    });
  });
});