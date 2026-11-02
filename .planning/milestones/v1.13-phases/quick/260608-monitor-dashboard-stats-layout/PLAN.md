---
quick_id: "260608-mdsl"
status: "in-progress"
---

# 优化监控仪表盘统计组件布局

## 问题描述

监控仪表盘（`xingran-react-frontend/src/pages/monitor/dashboard/index.tsx`）的统计卡片占据过多垂直空间：
- 每个卡片包含 Statistic + Progress 双重内容
- 响应式断点 `xs={24} sm={12} md={6}` 在小屏幕导致垂直堆叠
- 整体占用空间大于其他模块（公告统计、楼层统计）

## 优化方案

1. **移除 Progress 组件**：每个卡片仅保留 Statistic，参考公告统计的简洁设计
2. **调整布局**：改为固定 `span={6}`，4个卡片并排显示
3. **保持功能**：颜色编码（>80% 红色警告）保留在 Statistic 的样式中

## 参考模式

**公告统计**（`NoticeStatistics.tsx`）：
```tsx
<Row gutter={16} style={{ marginBottom: 16 }}>
  <Col span={6}>
    <Card>
      <Statistic title="总公告数" value={statistics.total} />
    </Card>
  </Col>
  {/* 其他3个卡片类似 */}
</Row>
```

**楼层统计**（`FloorStatisticsCards.tsx`）：
```tsx
<Row gutter={16} style={{ marginBottom: 16 }}>
  <Col span={8}>
    <Card>
      <Statistic title="总楼层数" value={statistics.total} />
    </Card>
  </Col>
  {/* 其他2个卡片类似 */}
</Row>
```

## 执行步骤

1. 读取当前文件 `xingran-react-frontend/src/pages/monitor/dashboard/index.tsx`
2. 移除第 212-219 行（Progress 组件及条件渲染）
3. 移除第 232-239 行（内存 Progress）
4. 移除第 252-259 行（磁盘 Progress）
5. 将第 202 行 `<Col xs={24} sm={12} md={6}>` 改为 `<Col span={6}>`（4处）
6. 保持 Statistic 的样式和颜色编码不变
7. 验证编译：`cd xingran-react-frontend && npm run type-check`
8. 提交变更

## 文件

- `xingran-react-frontend/src/pages/monitor/dashboard/index.tsx`
