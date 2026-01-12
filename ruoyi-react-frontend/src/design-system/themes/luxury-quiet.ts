/**
 * 静谧奢华主题
 * 特点：深邃色调 + 金属点缀 + 优雅动效 + 奢华质感
 */

import type { ThemeConfig } from '@/types/theme';
import { spacing } from '../tokens/spacing';
import { typography } from '../tokens/typography';
import { shadows, radius } from '../tokens/shadows';

export const luxuryQuietTheme: ThemeConfig = {
  id: 'luxury-quiet',
  name: '静谧奢华',
  description: '深邃炭灰背景配铜金色点缀，优雅奢华的视觉体验',

  colors: {
    // 主色：铜金色系列
    primary: [
      '#fef3c7', // 50 - 浅金
      '#fde68a', // 100 - 浅金黄
      '#d4af37', // 200 - 铜金色（主题色）
      '#b8960f', // 300 - 深金
      '#9a7b0a', // 400 - 古铜
    ],
    secondary: ['#f8f9fa', '#e9ecef', '#dee2e6', '#adb5bd', '#6c757d'],
    accent: ['#d4af37', '#f4d03f', '#b8960f'],
    neutral: [
      '#f8f9fa', '#e9ecef', '#dee2e6', '#ced4da', '#adb5bd',
      '#6c757d', '#495057', '#343a40', '#212529'
    ],
    // 功能色 - 柔和版本
    success: ['#73d13d', '#52c41a', '#389e0d'],
    warning: ['#ffd666', '#faad14', '#d48806'],
    error: ['#ff7875', '#ff4d4f', '#cf1322'],
    info: ['#69c0ff', '#1890ff', '#096dd9'],

    // 背景：深邃炭黑系列
    background: {
      primary: '#0a0e12',    // 深邃炭黑
      secondary: '#11151a',  // 次级背景
      tertiary: '#1a1f26',   // 三级背景
      surface: '#1a1f26',    // 表面背景
      elevated: '#222830',   // 悬浮背景
    },

    // 文字：银灰系列
    text: {
      primary: '#f0f2f5',    // 主要文字 - 接近白色
      secondary: '#b8c5d6',  // 次要文字 - 银灰
      tertiary: '#7d8ca1',   // 三级文字 - 灰蓝
      disabled: '#4a5568',   // 禁用文字
      inverse: '#0a0e12',    // 反色文字
    },

    // 边框：金色半透明
    border: {
      primary: 'rgba(212, 175, 55, 0.3)',     // 金色半透明
      secondary: 'rgba(184, 149, 15, 0.2)',   // 暗金色
      divider: 'rgba(255, 255, 255, 0.06)',   // 分割线
    },
  },

  spacing: {
    xs: spacing.xs,
    sm: spacing.sm,
    md: spacing.md,
    lg: spacing.lg,
    xl: spacing.xl,
    '2xl': spacing['2xl'],
  },

  typography: {
    fontFamily: typography.fontFamily,
    fontSize: typography.fontSize,
    fontWeight: typography.fontWeight,
    lineHeight: typography.lineHeight,
  },

  shadows: {
    xs: '0 1px 2px rgba(0, 0, 0, 0.3)',
    sm: '0 2px 4px rgba(0, 0, 0, 0.4)',
    md: '0 4px 8px rgba(0, 0, 0, 0.5)',
    lg: '0 8px 16px rgba(0, 0, 0, 0.6)',
    xl: '0 16px 32px rgba(0, 0, 0, 0.7)',
    '2xl': '0 24px 48px rgba(0, 0, 0, 0.8)',
    inner: 'inset 0 1px 3px rgba(0, 0, 0, 0.3)',
  },

  radius: {
    sm: radius.sm,
    md: radius.md,
    lg: radius.lg,
    xl: radius.xl,
    '2xl': radius['2xl'],
    full: radius.full,
  },

  effects: {
    transition: {
      fast: '150ms cubic-bezier(0.4, 0, 0.2, 1)',
      base: '250ms cubic-bezier(0.4, 0, 0.2, 1)',
      slow: '350ms cubic-bezier(0.4, 0, 0.2, 1)',
    },
  },
};

export default luxuryQuietTheme;
