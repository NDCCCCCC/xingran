# Phase 6: 实时数据刷新 - Research

**Researched:** 2026-04-21
**Domain:** React Hooks、WebSocket、轮询机制、缓存策略
**Confidence:** HIGH

## Summary

本阶段需要实现仪表盘数据的实时更新机制，支持 WebSocket 推送和轮询两种模式。经过代码审计，发现项目已有部分基础设施（dashboardStore 的 widgetDataCache、dashboardService 的 getBatchWidgetData），需要补充 WebSocket 连接管理、轮询机制和网络状态检测。

**Primary recommendation:** 使用原生 WebSocket API 封装 React Hook，结合 Zustand store 实现状态管理，使用 Page Visibility API 优化轮询性能。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### WebSocket 连接管理
- **D-01:** 连接方式 — 原生 WebSocket API
- **D-02:** 连接状态管理 — Zustand Store
- **D-03:** 重连策略 — 指数退避

#### 轮询机制
- **D-04:** 轮询实现 — setInterval + 可配置间隔
- **D-05:** 轮询触发 — 仪表盘级别

#### 缓存策略
- **D-06:** 缓存位置 — dashboardStore
- **D-07:** 缓存过期 — 基于刷新间隔
- **D-08:** 缓存失效 — 手动 + 自动

#### 前端数据订阅
- **D-09:** 订阅模式 — 发布-订阅
- **D-10:** 数据流 — 单向数据流

### Claude's Discretion
- 错误处理策略
- 加载状态显示
- 网络状态检测
- 性能优化

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RT-03a | WebSocket 连接管理 | 需实现 useWebSocket Hook，支持连接/断开/重连 |
| RT-03b | 基于刷新间隔的轮询机制 | 需实现 useWidgetPolling Hook，支持可配置间隔 |
| RT-03c | 缓存策略 | 扩展现有 dashboardStore 的 widgetDataCache |
| RT-03d | 前端实时数据订阅 | 使用 Zustand store 的订阅功能 |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| WebSocket 连接 | Browser / Client | — | 纯前端连接管理 |
| 轮询调度 | Browser / Client | — | 前端定时器管理 |
| 缓存管理 | Browser / Client | — | 前端内存缓存 |
| 数据订阅 | Browser / Client | — | React 状态订阅 |
| 网络状态检测 | Browser / Client | — | 浏览器 API |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.2 | UI 框架 | 项目标准，已验证 |
| TypeScript | 5.9 | 类型系统 | 项目标准，已验证 |
| Zustand | 5.0.9 | 状态管理 | 项目标准，dashboardStore 已实现 |

### Browser APIs
| API | Purpose | Browser Support |
|-----|---------|-----------------|
| WebSocket | 实时连接 | All modern browsers |
| Page Visibility API | 优化轮询 | All modern browsers |
| navigator.onLine | 网络状态 | All modern browsers |
| online/offline events | 网络变化监听 | All modern browsers |

**Installation:** 无需安装新依赖，所有依赖已存在于项目中。

## Architecture Patterns

### System Architecture Diagram

```
用户操作
    │
    ├─► 查看仪表盘 ──► useWidgetPolling Hook
    │                       │
    │                       ├─► 检查缓存 (widgetDataCache)
    │                       ├─► 发起请求 (getBatchWidgetData)
    │                       └─► 更新 Store → UI 重渲染
    │
    ├─► WebSocket 连接 ──► useWebSocket Hook
    │                           │
    │                           ├─► 建立连接
    │                           ├─► 监听消息 → 更新 Store
    │                           ├─► 连接断开 → 自动重连
    │                           └─► 网络恢复 → 重连
    │
    └─► 网络状态变化 ──► useNetworkStatus Hook
                                │
                                ├─► online → 恢复连接/轮询
                                └─► offline → 显示提示
```

### Recommended Project Structure
```
src/hooks/
├── useWebSocket.ts         # WebSocket 连接管理
├── useWidgetPolling.ts     # Widget 数据轮询
└── useNetworkStatus.ts     # 网络状态检测

src/store/
└── dashboardStore.ts       # 扩展 WebSocket 状态
```

### Pattern 1: WebSocket Hook
**What:** 封装 WebSocket 连接为 React Hook
**When to use:** 需要实时数据推送的场景
**Example:**
```typescript
// Source: 需新建 useWebSocket.ts
import { useEffect, useRef, useCallback, useState } from 'react';

interface UseWebSocketOptions {
  url: string;
  onMessage: (data: unknown) => void;
  onOpen?: () => void;
  onClose?: () => void;
  onError?: (error: Event) => void;
  reconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

interface UseWebSocketReturn {
  status: 'connecting' | 'connected' | 'disconnected' | 'error';
  connect: () => void;
  disconnect: () => void;
  send: (data: unknown) => void;
}

export function useWebSocket(options: UseWebSocketOptions): UseWebSocketReturn {
  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  // 连接逻辑...
  // 重连逻辑（指数退避）...
  // 消息处理...

  return { status, connect, disconnect, send };
}
```

### Pattern 2: 轮询 Hook
**What:** 封装数据轮询为 React Hook
**When to use:** 需要定时刷新数据的场景
**Example:**
```typescript
// Source: 需新建 useWidgetPolling.ts
import { useEffect, useRef, useCallback } from 'react';
import { useDashboardStore } from '@/store/dashboardStore';
import { dashboardService } from '@/services/dashboardService';

interface UseWidgetPollingOptions {
  widgetIds: string[];
  interval: number; // 秒
  enabled?: boolean;
}

export function useWidgetPolling(options: UseWidgetPollingOptions) {
  const { widgetIds, interval, enabled = true } = options;
  const { cacheWidgetData, getCachedWidgetData, currentDashboard } = useDashboardStore();
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const fetchData = useCallback(async () => {
    // 检查缓存
    // 发起请求
    // 更新缓存
  }, [widgetIds]);

  useEffect(() => {
    if (!enabled || widgetIds.length === 0) return;

    // 首次加载
    fetchData();

    // 设置轮询
    intervalRef.current = setInterval(fetchData, interval * 1000);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [enabled, widgetIds, interval, fetchData]);

  // Page Visibility API 优化
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.hidden) {
        // 页面不可见时暂停轮询
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      } else {
        // 页面可见时恢复轮询
        fetchData();
        intervalRef.current = setInterval(fetchData, interval * 1000);
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [fetchData, interval]);
}
```

### Pattern 3: 网络状态 Hook
**What:** 检测网络连接状态
**When to use:** 需要处理离线/在线状态变化
**Example:**
```typescript
// Source: 需新建 useNetworkStatus.ts
import { useState, useEffect } from 'react';

interface UseNetworkStatusReturn {
  isOnline: boolean;
  wasOffline: boolean;
}

export function useNetworkStatus(): UseNetworkStatusReturn {
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const [wasOffline, setWasOffline] = useState(false);

  useEffect(() => {
    const handleOnline = () => {
      setIsOnline(true);
    };

    const handleOffline = () => {
      setIsOnline(false);
      setWasOffline(true);
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  return { isOnline, wasOffline };
}
```

### Anti-Patterns to Avoid
- **直接在组件中使用 setInterval:** 应封装为 Hook，便于管理和清理
- **忽略 Page Visibility API:** 页面不可见时应暂停轮询，节省资源
- **硬编码刷新间隔:** 应使用仪表盘配置的 refreshInterval

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| WebSocket 重连 | 自定义重连逻辑 | 指数退避算法 | 避免服务器过载 |
| 网络状态检测 | 轮询 navigator.onLine | online/offline 事件 | 实时响应，无延迟 |
| 缓存过期 | 手动时间戳比较 | 现有 widgetDataCache | 已验证的实现 |

**Key insight:** 项目已有完善的缓存基础设施，只需扩展使用。

## Common Pitfalls

### Pitfall 1: WebSocket 连接泄漏
**What goes wrong:** 组件卸载时未关闭 WebSocket 连接
**Why it happens:** useEffect 清理函数未正确处理
**How to avoid:** 在 useEffect 返回的清理函数中调用 ws.close()
**Warning signs:** 控制台显示多个 WebSocket 连接

### Pitfall 2: 轮询间隔过短
**What goes wrong:** 频繁请求导致服务器压力过大
**Why it happens:** 用户配置了过短的刷新间隔
**How to avoid:** 设置最小刷新间隔限制（30秒）
**Warning signs:** 服务器响应变慢，请求堆积

### Pitfall 3: 缓存未失效
**What goes wrong:** Widget 配置变更后仍显示旧数据
**Why it happens:** 缓存未正确失效
**How to avoid:** Widget 配置变更时调用 clearWidgetCache(widgetId)
**Warning signs:** 修改数据源后数据未更新

### Pitfall 4: 页面不可见时继续轮询
**What goes wrong:** 用户切换标签页后仍继续请求
**Why it happens:** 未使用 Page Visibility API
**How to avoid:** 监听 visibilitychange 事件，暂停/恢复轮询
**Warning signs:** 浏览器标签页显示高 CPU 使用率

## Code Examples

### WebSocket 连接管理 (RT-03a)
```typescript
// Source: 需新建 useWebSocket.ts
import { useEffect, useRef, useCallback, useState } from 'react';
import { message } from 'antd';

interface UseWebSocketOptions {
  url: string;
  onMessage: (data: unknown) => void;
  reconnect?: boolean;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

export function useWebSocket(options: UseWebSocketOptions) {
  const {
    url,
    onMessage,
    reconnect = true,
    reconnectInterval = 1000,
    maxReconnectAttempts = 10,
  } = options;

  const [status, setStatus] = useState<'connecting' | 'connected' | 'disconnected' | 'error'>('disconnected');
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    setStatus('connecting');
    const ws = new WebSocket(url);

    ws.onopen = () => {
      setStatus('connected');
      reconnectAttemptsRef.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        onMessage(data);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    ws.onclose = () => {
      setStatus('disconnected');
      wsRef.current = null;

      // 自动重连
      if (reconnect && reconnectAttemptsRef.current < maxReconnectAttempts) {
        const delay = reconnectInterval * Math.pow(2, reconnectAttemptsRef.current);
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectAttemptsRef.current++;
          connect();
        }, Math.min(delay, 30000)); // 最大 30 秒
      }
    };

    ws.onerror = (error) => {
      setStatus('error');
      console.error('WebSocket error:', error);
    };

    wsRef.current = ws;
  }, [url, onMessage, reconnect, reconnectInterval, maxReconnectAttempts]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setStatus('disconnected');
  }, []);

  const send = useCallback((data: unknown) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data));
    }
  }, []);

  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  return { status, connect, disconnect, send };
}
```

### 轮询机制 (RT-03b)
```typescript
// Source: 需新建 useWidgetPolling.ts
import { useEffect, useRef, useCallback, useState } from 'react';
import { useDashboardStore } from '@/store/dashboardStore';
import { dashboardService } from '@/services/dashboardService';

interface UseWidgetPollingOptions {
  widgetIds: string[];
  interval: number;
  enabled?: boolean;
}

export function useWidgetPolling(options: UseWidgetPollingOptions) {
  const { widgetIds, interval, enabled = true } = options;
  const { cacheWidgetData, getCachedWidgetData, clearWidgetCache } = useDashboardStore();
  const [loading, setLoading] = useState(false);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const fetchData = useCallback(async () => {
    if (widgetIds.length === 0) return;

    // 检查缓存
    const cacheExpiry = Math.max(interval / 2, 30) * 1000;
    const now = Date.now();
    const uncachedIds: string[] = [];

    for (const id of widgetIds) {
      const cached = getCachedWidgetData(id);
      if (!cached || (now - (cached as { timestamp: number }).timestamp) > cacheExpiry) {
        uncachedIds.push(id);
      }
    }

    if (uncachedIds.length === 0) return;

    setLoading(true);
    try {
      const data = await dashboardService.getBatchWidgetData(uncachedIds);
      for (const [id, widgetData] of data) {
        cacheWidgetData(id, widgetData);
      }
    } catch (error) {
      console.error('Failed to fetch widget data:', error);
    } finally {
      setLoading(false);
    }
  }, [widgetIds, interval, cacheWidgetData, getCachedWidgetData]);

  useEffect(() => {
    if (!enabled || widgetIds.length === 0) return;

    fetchData();
    intervalRef.current = setInterval(fetchData, interval * 1000);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [enabled, widgetIds, interval, fetchData]);

  // Page Visibility API
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.hidden) {
        if (intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
        }
      } else {
        fetchData();
        intervalRef.current = setInterval(fetchData, interval * 1000);
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [fetchData, interval]);

  return { loading, refresh: fetchData };
}
```

### 网络状态检测
```typescript
// Source: 需新建 useNetworkStatus.ts
import { useState, useEffect } from 'react';

export function useNetworkStatus() {
  const [isOnline, setIsOnline] = useState(navigator.onLine);
  const [wasOffline, setWasOffline] = useState(false);

  useEffect(() => {
    const handleOnline = () => {
      setIsOnline(true);
    };

    const handleOffline = () => {
      setIsOnline(false);
      setWasOffline(true);
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, []);

  return { isOnline, wasOffline, resetWasOffline: () => setWasOffline(false) };
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 轮询无暂停 | Page Visibility API | 本阶段 | 节省资源，提升性能 |
| 固定重连间隔 | 指数退避 | 本阶段 | 避免服务器过载 |
| 无网络状态检测 | online/offline 事件 | 本阶段 | 更好的用户体验 |

**Deprecated/outdated:**
- 无已弃用模式，项目使用现代 React 19 + TypeScript 技术栈

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | 后端 WebSocket 端点已实现 | Architecture Patterns | 需确认 Phase 4 实现 |
| A2 | dashboardStore 支持扩展 | Code Examples | 需验证 store 结构 |

**验证结果:**
- A1: Phase 4 已实现 GetWidgetData 方法，支持 WebSocket 数据源类型 [VERIFIED: 04-01-SUMMARY.md]
- A2: dashboardStore 已有 widgetDataCache 和相关方法，支持扩展 [VERIFIED: dashboardStore.ts]

## Open Questions

1. **WebSocket 端点 URL**
   - What we know: 后端支持 WebSocket 数据源类型
   - What's unclear: WebSocket 连接的具体 URL 格式
   - Recommendation: 使用 `/ws/dashboard/{dashboardId}` 格式，与后端协商

2. **推送数据格式**
   - What we know: 后端支持 WebSocket 数据源
   - What's unclear: 推送数据的 JSON 格式
   - Recommendation: 使用 `{ widgetId: string, data: unknown, timestamp: number }` 格式

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| React | 所有组件 | ✓ | 19.2 | — |
| TypeScript | 类型检查 | ✓ | 5.9 | — |
| Zustand | 状态管理 | ✓ | 5.0.9 | — |
| Ant Design | 提示组件 | ✓ | 6.1.1 | — |

**Missing dependencies with no fallback:** 无

**Missing dependencies with fallback:** 无

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.0.18 |
| Config file | vitest.config.ts (项目根目录) |
| Quick run command | `npm run test` |
| Full suite command | `npm run test:coverage` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RT-03a | WebSocket 连接建立 | unit | `vitest run useWebSocket.test.ts` | ❌ Wave 0 |
| RT-03a | WebSocket 自动重连 | unit | `vitest run useWebSocket.test.ts` | ❌ Wave 0 |
| RT-03b | 轮询定时触发 | unit | `vitest run useWidgetPolling.test.ts` | ❌ Wave 0 |
| RT-03b | 页面不可见暂停轮询 | unit | `vitest run useWidgetPolling.test.ts` | ❌ Wave 0 |
| RT-03c | 缓存命中跳过请求 | unit | `vitest run useWidgetPolling.test.ts` | ❌ Wave 0 |
| RT-03d | 数据订阅触发 UI 更新 | integration | `vitest run dashboardStore.test.ts` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `npm run test` (快速运行)
- **Per wave merge:** `npm run test:coverage` (覆盖率检查)
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `ruoyi-react-frontend/src/hooks/__tests__/useWebSocket.test.ts` — covers RT-03a
- [ ] `ruoyi-react-frontend/src/hooks/__tests__/useWidgetPolling.test.ts` — covers RT-03b, RT-03c
- [ ] `ruoyi-react-frontend/src/hooks/__tests__/useNetworkStatus.test.ts` — covers network detection
- [ ] Framework install: 已安装 — 无需额外安装

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes | WebSocket 消息验证 |
| V6 Cryptography | no | — |

### Known Threat Patterns for WebSocket

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 消息注入 | Tampering | JSON 解析错误处理，不执行动态代码 |
| 连接劫持 | Tampering | 使用认证 Token，验证连接身份 |
| 资源耗尽 | Denial | 重连次数限制，指数退避 |

## Sources

### Primary (HIGH confidence)
- 现有代码文件: `dashboardStore.ts`, `dashboardService.ts` - [完整实现参考]
- 类型定义: `ruoyi-react-frontend/src/types/dashboard.ts` - [完整类型定义]
- Phase 4 实现: `.planning/phases/04-widget-data-fetching/*-SUMMARY.md` - [后端支持验证]

### Secondary (MEDIUM confidence)
- package.json - [依赖版本确认]

### Tertiary (LOW confidence)
- 无

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - 所有依赖已存在于项目中，版本已验证
- Architecture: HIGH - 现有组件提供完整参考，模式清晰
- Pitfalls: HIGH - 基于 React 和 WebSocket 最佳实践

**Research date:** 2026-04-21
**Valid until:** 30 days (技术栈稳定)
