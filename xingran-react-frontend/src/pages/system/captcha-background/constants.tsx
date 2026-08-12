/**
 * Captcha Background Constants
 * 验证码背景常量定义
 */

import { Tag } from "antd";
import type { PieceShape, DifficultyLevel } from "@/types/captcha";

// 拼图形状映射
export const PIECE_SHAPE_MAP: Record<PieceShape, string> = {
  circle: "圆形",
  square: "方形",
  star: "星形",
  heart: "心形",
};

// 难度级别映射
export const DIFFICULTY_MAP: Record<DifficultyLevel, string> = {
  1: "简单",
  2: "中等",
  3: "困难",
};

// 形状选项
export const SHAPE_OPTIONS = [
  { label: "圆形", value: "circle" },
  { label: "方形", value: "square" },
  { label: "星形", value: "star" },
  { label: "心形", value: "heart" },
];

// 难度选项
export const DIFFICULTY_OPTIONS = [
  { label: "简单", value: 1 },
  { label: "中等", value: 2 },
  { label: "困难", value: 3 },
];

// 状态选项
export const STATUS_OPTIONS = [
  { label: "启用", value: 1 },
  { label: "禁用", value: 0 },
];

// 渲染拼图形状标签
export function renderPieceShapeTag(shape: PieceShape) {
  return <Tag color="blue">{PIECE_SHAPE_MAP[shape]}</Tag>;
}

// 渲染难度级别标签
export function renderDifficultyTag(level: DifficultyLevel) {
  return <Tag color="orange">{DIFFICULTY_MAP[level]}</Tag>;
}

// 渲染状态标签
export function renderStatusTag(status: number) {
  return <Tag color={status === 1 ? "success" : "default"}>{status === 1 ? "启用" : "禁用"}</Tag>;
}
