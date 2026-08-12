# UI设计契约：滑动验证码模态框改进

## 概述

**目标**：将滑动验证码从登录页面内嵌显示改为模态框弹出，提升用户体验

**当前问题**：
- 滑动验证码在登录页面内嵌显示，占用页面空间
- 用户需要点击"下一步"才能看到验证码，流程不够直观
- 登录按钮文字在"下一步"和"登录"之间切换，造成用户困惑

**改进后效果**：
- 登录按钮始终显示"登录"，用户体验更一致
- 验证码在模态框中弹出，不占用登录页面空间
- 验证成功后自动关闭模态框并登录，流程更流畅

---

## 组件设计

### 1. 登录页面改进

**文件**：`xingran-react-frontend/src/pages/login/index.tsx`

**变更**：
- ✅ 移除验证码内嵌显示逻辑
- ✅ 简化登录按钮文字，始终显示"登录"
- ✅ 使用Ant Design Modal组件显示验证码

### 2. 模态框组件

**组件名**：`CaptchaModal`

**文件**：`xingran-react-frontend/src/components/captcha/CaptchaModal.tsx`（新建）

**Props**：
```typescript
interface CaptchaModalProps {
  visible: boolean;
  captchaType: 'normal' | 'slider';
  onSuccess: (captchaData: CaptchaSuccessData) => void;
  onCancel: () => void;
}

interface CaptchaSuccessData {
  captchaId: string;
  captcha?: string; // normal验证码的值
  verified?: boolean; // slider验证码的验证状态
}
```

**UI规范**：
- **宽度**：`400px`（normal）或`500px`（slider）
- **标题**："安全验证"
- **取消按钮**：不显示（验证码必须完成）
- **关闭图标**：不允许点击关闭

### 3. 交互流程

#### 滑动验证码流程

```
用户操作                     系统响应
─────────────────────────────────────────────────────────
输入账号密码                  -
点击"登录"按钮               1. 表单验证通过
                            2. 检查验证码配置
                            3. 如果需要slider验证：
                               - 显示CaptchaModal
                               - 传入captchaType="slider"
                            
模态框弹出                   -
┌─────────────────────────┐
│  安全验证              │
│  ┌───────────────────┐  │
│  │  滑动拼图验证码   │  │
│  │                   │  │
│  │  [滑块] ==========│  │
│  │                   │  │
│  └───────────────────┘  │
└─────────────────────────┘

用户拖动滑块                1. 实时更新滑块位置
                            2. 释放鼠标时验证位置
                            
验证成功                     1. 显示"验证成功"提示
                            2. 自动关闭模态框（延迟500ms）
                            3. 执行登录请求
                            4. 登录成功跳转Dashboard
                            
验证失败                     1. 显示"验证失败"提示
                            2. 重新加载验证码
                            3. 模态框保持打开
```

#### 数字验证码流程

```
用户操作                     系统响应
─────────────────────────────────────────────────────────
输入账号密码                  -
点击"登录"按钮               1. 表单验证通过
                            2. 检查验证码配置
                            3. 如果需要normal验证：
                               - 显示CaptchaModal
                               - 传入captchaType="normal"
                            
模态框弹出                   -
┌─────────────────────────┐
│  安全验证              │
│  ┌───────────────────┐  │
│  │  [验证码图片]     │  │
│  │                   │  │
│  │  [输入框]         │  │
│  │                   │  │
│  │  [刷新] [确定]    │  │
│  └───────────────────┘  │
└─────────────────────────┘

用户输入验证码并点击"确定"   1. 验证输入是否为空
                            2. 如果不为空：关闭模态框
                            3. 执行登录请求
                            4. 登录成功跳转Dashboard
```

---

## 技术规范

### 1. Modal配置

```typescript
<Modal
  title="安全验证"
  open={visible}
  onCancel={onCancel}
  footer={null} // 不显示底部按钮栏
  closable={false} // 不允许点击关闭
  maskClosable={false} // 不允许点击遮罩关闭
  width={captchaType === 'slider' ? 500 : 400}
  centered
>
  {/* 验证码组件 */}
</Modal>
```

### 2. 状态管理

**登录页面状态**：
```typescript
const [showCaptchaModal, setShowCaptchaModal] = useState(false);
const [captchaType, setCaptchaType] = useState<'normal' | 'slider'>('normal');
const [pendingCredentials, setPendingCredentials] = useState<LoginRequest | null>(null);
```

### 3. 验证码成功回调

```typescript
const handleCaptchaSuccess = (data: CaptchaSuccessData) => {
  // 关闭模态框
  setShowCaptchaModal(false);
  
  // 立即执行登录
  performLogin(pendingCredentials!, data);
};
```

---

## 设计token

### 颜色
- 模态框标题：`var(--theme-text-primary)`
- 模态框背景：`var(--theme-bg-surface)`
- 验证成功提示：Ant Design `success` color

### 间距
- 模态框内边距：`24px`
- 验证码组件间距：`16px`

### 字体
- 标题：16px, font-weight: 500
- 提示文本：14px

---

## 6维质量检查清单

### 1. 功能性 ✅
- [ ] 滑动验证码在模态框中正确显示
- [ ] 数字验证码在模态框中正确显示
- [ ] 验证成功后自动关闭模态框
- [ ] 验证成功后自动登录
- [ ] 验证失败后模态框保持打开

### 2. 可用性 ✅
- [ ] 用户无法通过点击遮罩关闭模态框
- [ ] 用户无法通过ESC键关闭模态框
- [ ] 验证码加载时显示loading状态
- [ ] 验证失败时提供清晰的错误提示

### 3. 可访问性 ✅
- [ ] 模态框有明确的标题
- [ ] 键盘导航支持（Tab键）
- [ ] 焦点管理（打开时聚焦，关闭时返回登录按钮）

### 4. 响应式 ✅
- [ ] 模态框在移动设备上正确显示
- [ ] 验证码组件在小屏幕上可操作

### 5. 视觉一致性 ✅
- [ ] 模态框样式与现有设计系统一致
- [ ] 使用项目的design tokens
- [ ] 动画效果与现有组件一致

### 6. 边界情况 ✅
- [ ] 网络错误时正确处理
- [ ] 验证码过期时重新加载
- [ ] 用户取消登录时正确清理状态

---

## 实现优先级

### P0（必须有）
- [ ] 创建CaptchaModal组件
- [ ] 修改登录页面移除内嵌验证码显示
- [ ] 实现滑动验证码的模态框流程
- [ ] 验证成功自动登录

### P1（重要）
- [ ] 实现数字验证码的模态框流程
- [ ] 错误处理和重试逻辑
- [ ] 键盘导航支持

### P2（可选）
- [ ] 响应式优化
- [ ] 动画效果优化
- [ ] 可访问性增强

---

## 验收标准

1. **功能验收**：
   - 滑动验证码在模态框中正确显示和验证
   - 验证成功后自动关闭模态框并登录
   - 登录失败时模态框保持打开

2. **体验验收**：
   - 登录按钮始终显示"登录"
   - 模态框弹出和关闭流畅
   - 验证码加载速度快

3. **兼容性验收**：
   - 支持Chrome、Firefox、Edge最新版本
   - 移动设备基本可用

---

## 相关文件

- 登录页面：`xingran-react-frontend/src/pages/login/index.tsx`
- 滑动验证码组件：`xingran-react-frontend/src/components/captcha/SliderCaptcha.tsx`
- 数字验证码组件：`xingran-react-frontend/src/components/captcha/TextCaptcha.tsx`
- 设计系统：`xingran-react-frontend/src/design-system/`
