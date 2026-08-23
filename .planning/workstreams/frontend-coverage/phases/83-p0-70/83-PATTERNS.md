# Phase 83: P0 基建层全清 ≥70% - Pattern Map

**Mapped:** 2026-08-24
**Files analyzed:** 38 new/modified files/groups
**Analogs found:** 35 / 38

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `.github/scripts/check-frontend-diff-coverage.sh` | config/gate | request-response | `.github/scripts/check-diff-coverage.sh` | exact |
| `.github/scripts/check-frontend-coverage.sh` | config/gate | request-response | `.github/scripts/check-coverage.sh` | exact |
| `.github/workflows/ci.yml` | config | event-driven | existing `ci.yml` frontend job section | exact |
| `.coverage-fe-floors` | config/data | batch | `.coverage-threshold` (backend twin) | role-match |
| `.planning/frontend-coverage-baseline.md` | config/doc | batch | existing baseline.md ratchet rows | exact |
| `xingran-react-frontend/package.json` | config | static | existing `package.json` devDependencies | exact |
| `xingran-react-frontend/src/test/utils/renderWithProviders.tsx` | utility/harness | request-response | existing hook tests' `wrapper` pattern (`usePagination.test.tsx`) | role-match |
| `xingran-react-frontend/src/test/utils/createApiMock.ts` | utility/harness | request-response | `src/lib/api/__tests__/networkApi.test.ts` vi.mock pattern | role-match |
| `xingran-react-frontend/src/test/utils/mockAntdMessage.ts` | utility/harness | event-driven | `src/utils/antdMessage.ts` + `loginPreflight.test.ts` vi.mock pattern | role-match |
| `xingran-react-frontend/src/test/utils/*.test.ts` | test | request-response | `src/design-system/tokens/colors.test.ts` | exact |
| `xingran-react-frontend/src/lib/api.test.ts` | test | request-response | `src/lib/__tests__/loginPreflight.test.ts` (vi.mock dependencies) | role-match |
| `xingran-react-frontend/src/lib/opsApi.test.ts` | test | request-response | `src/lib/api/__tests__/networkApi.test.ts` | exact |
| `xingran-react-frontend/src/lib/menuApi.test.ts` | test | request-response | `src/lib/api/__tests__/networkApi.test.ts` | exact |
| `xingran-react-frontend/src/utils/sm4.test.ts` | test | transform | `src/utils/sm2.test.ts` (crypto real-call pattern) | role-match |
| `xingran-react-frontend/src/utils/encoding.test.ts` | test | transform | `src/utils/sm2.test.ts` (real algorithm + deterministic vectors) | role-match |
| `xingran-react-frontend/src/utils/token/TokenManager.test.ts` | test | event-driven | `src/lib/__tests__/loginPreflight.test.ts` (fake timers) | role-match |
| `xingran-react-frontend/src/utils/token/SecureTokenStorageImpl.test.ts` | test | transform | `src/utils/sm2.test.ts` (real crypto + beforeEach reset) | role-match |
| `xingran-react-frontend/src/utils/dualLevelCache.test.ts` | test | CRUD | `src/hooks/usePersistedState.test.ts` (sessionStorage/localStorage setup) | role-match |
| `xingran-react-frontend/src/utils/errorHandler.test.ts` | test | request-response | `src/lib/__tests__/loginPreflight.test.ts` (mock antd message / console.error) | role-match |
| `xingran-react-frontend/src/utils/authHelpers.test.ts` | test | request-response | `src/utils/sm2.test.ts` (mock @/lib/api) | role-match |
| `xingran-react-frontend/src/hooks/useTableManager.test.tsx` | test | request-response | `src/hooks/usePagination.test.tsx` (renderHook + MemoryRouter) | role-match |
| `xingran-react-frontend/src/hooks/useColumnConfig.test.tsx` | test | request-response | `src/hooks/useServerSort.test.tsx` (renderHook + persistence assertions) | role-match |
| `xingran-react-frontend/src/hooks/useWidgetData.test.tsx` | test | request-response | `src/hooks/usePagination.test.tsx` | role-match |
| `xingran-react-frontend/src/store/authStore.test.ts` | test | event-driven | `src/utils/sm2.test.ts` (vi.mock @/lib/api) + Zustand reset pattern | role-match |
| `xingran-react-frontend/src/store/tabsStore.test.ts` | test | CRUD | `src/hooks/usePersistedState.test.ts` (storage isolation) | role-match |
| `xingran-react-frontend/src/store/menuStore.test.ts` | test | request-response | `src/lib/api/__tests__/networkApi.test.ts` (mock API wrappers) | role-match |
| `xingran-react-frontend/src/services/encryptionConfig.test.ts` | test | request-response | `src/lib/api/__tests__/networkApi.test.ts` | exact |
| `xingran-react-frontend/src/router/routeConfigManager.test.ts` | test | transform | `src/constants/status.test.ts` (static assertion + fixture data) | role-match |
| `xingran-react-frontend/src/router/routeGenerator.test.ts` | test | transform | `src/constants/status.test.ts` | role-match |
| `xingran-react-frontend/src/constants/storage.test.ts` | test | transform | `src/hooks/usePersistedState.test.ts` (sessionStorage key cleanup) | role-match |
| `xingran-react-frontend/src/constants/pageTitles.test.ts` | test | static | `src/constants/status.test.ts` | exact |
| `xingran-react-frontend/src/constants/routes.test.ts` | test | static | `src/constants/status.test.ts` | exact |
| `xingran-react-frontend/src/types/config.test.ts` | test | static | `src/constants/status.test.ts` | exact |
| `xingran-react-frontend/src/types/dashboard.test.ts` | test | static | `src/constants/status.test.ts` | exact |
| `xingran-react-frontend/src/types/notice.test.ts` | test | static | `src/constants/status.test.ts` | exact |
| `xingran-react-frontend/src/types/common.test.ts` | test | static | `src/constants/status.test.ts` | exact |

## Pattern Assignments

### Gate / Config Files

#### `.github/scripts/check-frontend-diff-coverage.sh` (config, request-response)

**Analog:** `.github/scripts/check-diff-coverage.sh`

**State:** CR-01 / WR-01 already committed to main (`60f712c`, `27f275e`). Phase 83 plan0 should verify + clean comments, not re-implement.

**Pattern to preserve** (lines 101-112):
```bash
# Fail-closed (WR-01): this job carries `needs: frontend`, so reaching the
# gate means the frontend job already passed and the json MUST exist. A
# missing profile here is configuration drift (artifact name/path, reporter
# change), not an upstream failure — soft-skipping would silently disable
# GOV-04. Mirror the backend twin check-diff-coverage.sh: exit 2.
if [ ! -f "$PROFILE" ]; then
  echo "check-frontend-diff-coverage.sh: coverage profile $PROFILE missing — configuration drift? ..."
  exit 2
fi
```

**Pathspec mirror pattern** (lines 144-151):
```bash
if ! git diff --unified=0 "${DIFF_ARGS[@]} -- \
  'xingran-react-frontend/src/*.ts' 'xingran-react-frontend/src/*.tsx' \
  ':(exclude)xingran-react-frontend/src/*.test.*' \
  ':(exclude)xingran-react-frontend/src/**/__tests__/**' \
  ':(exclude)xingran-react-frontend/src/**/*.d.ts' \
  ':(exclude)xingran-react-frontend/src/test/**' \
  ':(exclude)xingran-react-frontend/src/components/cad-editor/**' \
  ':(exclude)xingran-react-frontend/src/components/cad-elements/**' | awk '...'
```

---

#### `.github/scripts/check-frontend-coverage.sh` (config, request-response)

**Analog:** `.github/scripts/check-coverage.sh`

**State:** WR-02 / WR-03 already committed to main (`94d3a16`, `aa3bf0c`). Phase 83 plan0 should verify + clean comments, not re-implement.

**Whitelist drift detection** (lines 183-189):
```bash
if printf '%s\n' "$FLAT" | grep -Eq '^xingran-react-frontend/src/components/cad-(editor|elements)/'; then
  echo "check-frontend-coverage.sh: WHITELIST DRIFT (D-10) — cad-editor/cad-elements files present in the profile:" >&2
  printf '%s\n' "$FLAT" | grep -E '^xingran-react-frontend/src/components/cad-(editor|elements)/' | cut -f1 >&2
  echo "coverage.exclude in xingran-react-frontend/vitest.config.ts is the single truth source —" >&2
  exit 6
fi
```

**Floors numeric validation** (lines 260-267):
```bash
if ! awk -v v="$GLOBAL_FLOOR" 'BEGIN { exit (v !~ /^[0-9]+([.][0-9]+)?$/) }'; then
  echo "check-frontend-coverage.sh: floors file $FLOORS_FILE has no numeric GLOBAL line" >&2
  exit 2
fi
```

---

#### `.github/workflows/ci.yml` (config, event-driven)

**Analog:** existing `ci.yml`

**Pattern to preserve** (lines 172-179, 226-231):
```yaml
- name: Coverage gate
  working-directory: .
  run: bash .github/scripts/check-frontend-coverage.sh xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors

- name: Diff coverage gate (≥80%)
  run: bash .github/scripts/check-frontend-diff-coverage.sh xingran-react-frontend/coverage/coverage-final.json origin/${{ github.base_ref }} 80
```

**Plan0 cleanup scope:** If 82-REVIEW-FIX.md notes any stale/misleading comment about "json 缺失软跳过" or pathspec, update inline comments to reflect committed truth.

---

#### `.coverage-fe-floors` (config/data, batch)

**Analog:** `.coverage-threshold` (backend) + existing `.coverage-fe-floors`

**Pattern to copy** (lines 12-40):
```text
GLOBAL 3.8
(src root)	0.0
api	0.0
components	4.9
constants	38.8
...
utils	7.7
```

**Ratchet rule** (lines 7-11):
```text
# Ratchet bump = edit the numbers in THIS file only, never the script (D-07); every
# bump lands in the same commit as a row appended to
# .planning/frontend-coverage-baseline.md (floors only move UP).
```

**D-11 bump targets for Phase 83:**
- lib → 70.0+
- utils → 70.0+
- hooks → 70.0+
- store → 70.0+
- services → 70.0+
- router → 70.0+
- constants → 70.0+
- types → 70.0+

---

#### `.planning/frontend-coverage-baseline.md` (config/doc, batch)

**Analog:** existing baseline rows

**Pattern to append** (lines 13-14):
```markdown
| 2026-08-23 | 起点 | 3.85 | 21574 | 830 | 15 | bddb2fc | n/a | n/a | n/a |
```

**Per-directory table pattern** (lines 18-48):
```markdown
| 目录                          | stmts   | covered | pct    | 文件数 |
| components                    | 3958    | 215     | 5.43%  | 118    |
```

---

#### `xingran-react-frontend/package.json` (config, static)

**Analog:** existing `package.json`

**Pattern to adjust** (lines 76-78, 98):
```json
"@vitest/coverage-v8": "^4.1.10",
"@vitest/ui": "^4.1.10",
"vitest": "^4.1.10",
```

---

### Harness Files (`src/test/utils/`)

#### `src/test/utils/renderWithProviders.tsx` (utility/harness, request-response)

**Analog:** `src/hooks/usePagination.test.tsx` wrapper pattern + `src/test/setup.ts` polyfill

**Imports pattern** (from `usePagination.test.tsx` lines 10-13):
```typescript
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
```

**Wrapper pattern** (from `usePagination.test.tsx` lines 16-20):
```typescript
const wrapper =
  (initialPath: string) =>
  ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
  );
```

**renderWithProviders shape** (D-05):
```typescript
import { render, type RenderOptions } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ConfigProvider } from "antd";

export function renderWithProviders(
  ui: React.ReactNode,
  { route = "/", ...renderOptions }: { route?: string } & RenderOptions = {}
) {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <ConfigProvider>{ui}</ConfigProvider>
    </MemoryRouter>,
    renderOptions
  );
}
```

---

#### `src/test/utils/createApiMock.ts` (utility/harness, request-response)

**Analog:** `src/lib/api/__tests__/networkApi.test.ts`

**Mock factory pattern** (lines 16-24):
```typescript
const mockPost = vi.fn();
vi.mock("../api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));
```

**Endpoint factory shape** (D-06):
```typescript
import { vi } from "vitest";

export function createApiMock() {
  const post = vi.fn();
  const get = vi.fn();
  const put = vi.fn();
  const del = vi.fn();

  vi.mock("@/lib/api", () => ({
    post: (...args: unknown[]) => post(...args),
    get: (...args: unknown[]) => get(...args),
    put: (...args: unknown[]) => put(...args),
    del: (...args: unknown[]) => del(...args),
    upload: (...args: unknown[]) => post(...args),
    postFormData: (...args: unknown[]) => post(...args),
  }));

  return { post, get, put, del };
}
```

---

#### `src/test/utils/mockAntdMessage.ts` (utility/harness, event-driven)

**Analog:** `src/utils/antdMessage.ts` + `src/lib/__tests__/loginPreflight.test.ts`

**Message shape** (from `antdMessage.ts` lines 27-37):
```typescript
const noop = () => {};
const noopMessage = {
  success: noop,
  error: noop,
  info: noop,
  warning: noop,
  loading: noop,
  warn: noop,
  open: noop,
  destroy: noop,
} as unknown as MessageInstance;
```

**Mock helper shape**:
```typescript
import { vi } from "vitest";

export function mockAntdMessage() {
  const message = {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
    loading: vi.fn(),
    warn: vi.fn(),
    open: vi.fn(),
    destroy: vi.fn(),
  };
  vi.mock("@/utils/antdMessage", () => ({
    getAppMessage: () => message,
  }));
  return message;
}
```

---

### lib Tests (INFRA-01)

#### `src/lib/api.test.ts` (test, request-response)

**Analog:** `src/lib/__tests__/loginPreflight.test.ts`

**Dependency mock pattern** (lines 1-27):
```typescript
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { mockRefreshEncryptionConfig, mockGetCaptchaConfig, mockFetchPublicKey, mockClearPublicKeyCache } = vi.hoisted(() => ({
  mockRefreshEncryptionConfig: vi.fn<() => Promise<boolean>>(),
  mockGetCaptchaConfig: vi.fn(),
  mockFetchPublicKey: vi.fn(),
  mockClearPublicKeyCache: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  refreshEncryptionConfig: mockRefreshEncryptionConfig,
}));
vi.mock("@/utils/sm2", () => ({
  fetchPublicKey: mockFetchPublicKey,
  clearPublicKeyCache: mockClearPublicKeyCache,
}));
```

**Test structure** (lines 30-45):
```typescript
describe("submitLoginPreflight", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    mockRefreshEncryptionConfig.mockReset();
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });
});
```

---

#### `src/lib/opsApi.test.ts` (test, request-response)

**Analog:** `src/lib/api/__tests__/networkApi.test.ts`

**Full pattern** (lines 12-53):
```typescript
import { describe, it, expect, vi, beforeEach } from "vitest";

const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));

import { buildingApi } from "./opsApi";

describe("opsApi CRUD factory", () => {
  beforeEach(() => {
    mockPost.mockReset();
  });

  it("buildingApi.list calls correct endpoint", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { list: [], total: 0 } });
    await buildingApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenCalledWith("/ops/building/list", {
      current: 1,
      pageSize: 10,
    });
  });
});
```

---

### utils Tests (INFRA-02)

#### `src/utils/sm4.test.ts` (test, transform)

**Analog:** `src/utils/sm2.test.ts`

**Real crypto pattern** (from `sm2.test.ts` lines 1-9):
```typescript
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.hoisted(() => vi.fn());
vi.mock("@/lib/api", () => ({
  get: mockGet,
}));
```

**Test vector pattern** (from RESEARCH code example):
```typescript
import { encryptSM4CBC, decryptSM4CBC } from "@/utils/sm4";

describe("SM4-CBC", () => {
  const key = "0123456789abcdeffedcba9876543210";
  const iv = "abcdef98765432100123456789abcdef";

  it("加解密往返", async () => {
    const plain = "hello 国密";
    const cipher = await encryptSM4CBC(plain, key, iv);
    expect(await decryptSM4CBC(cipher, key, iv)).toBe(plain);
  });
});
```

---

#### `src/utils/encoding.test.ts` (test, transform)

**Analog:** `src/utils/sm2.test.ts`

**Pattern to copy** (round-trip + edge cases):
```typescript
import { describe, it, expect } from "vitest";
import { hexToBase64, base64ToHex, bytesToHex, hexToBytes, generateRandomHex } from "./encoding";

describe("encoding", () => {
  it("hexToBase64 / base64ToHex round-trip", () => {
    const hex = "0123456789abcdef";
    expect(base64ToHex(hexToBase64(hex))).toBe(hex);
  });
});
```

---

#### `src/utils/token/TokenManager.test.ts` (test, event-driven)

**Analog:** `src/lib/__tests__/loginPreflight.test.ts` (fake timers)

**Fake timer pattern** (lines 121-135):
```typescript
it("刷新超过 5 秒时停止等待并返回友好提示", async () => {
  mockRefreshEncryptionConfig.mockImplementation(() => new Promise(() => {}));

  vi.useFakeTimers();
  try {
    const pending = submitLoginPreflight();
    await vi.advanceTimersByTimeAsync(5000);

    expect(await pending).toEqual({ ok: false, friendlyMessage: "..." });
  } finally {
    vi.useRealTimers();
  }
});
```

**TokenManager fake timer shape** (D-09):
```typescript
import { TokenManager } from "./TokenManager";

it("过期前 30 秒自动触发刷新", async () => {
  vi.useFakeTimers();
  const storage = createFakeStorage();
  const manager = new TokenManager(storage, {
    refreshEndpoint: "/refresh",
    refreshBeforeSeconds: 30,
    refreshTimeout: 10000,
  });
  await manager.initializeTokens("acc", "ref", 60);

  const refreshSpy = vi.spyOn(manager, "refreshToken").mockResolvedValue({
    accessToken: "new-acc",
    refreshToken: "new-ref",
    expiresIn: 60,
  });

  vi.advanceTimersByTime(31_000);
  expect(refreshSpy).toHaveBeenCalled();
  vi.useRealTimers();
});
```

---

#### `src/utils/token/SecureTokenStorageImpl.test.ts` (test, transform)

**Analog:** `src/utils/sm2.test.ts`

**Storage reset pattern** (from `usePersistedState.test.ts` lines 13-17):
```typescript
beforeEach(() => {
  sessionStorage.clear();
  localStorage.clear();
});
```

**Encryption round-trip shape**:
```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { SecureTokenStorageImpl } from "./SecureTokenStorageImpl";

describe("SecureTokenStorageImpl", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("encrypts and decrypts refresh token", async () => {
    const storage = new SecureTokenStorageImpl();
    storage.setAccessToken("access");
    await storage.setRefreshToken("refresh-secret");
    expect(await storage.getRefreshToken()).toBe("refresh-secret");
  });
});
```

---

#### `src/utils/dualLevelCache.test.ts` (test, CRUD)

**Analog:** `src/hooks/usePersistedState.test.ts`

**Storage cleanup pattern**:
```typescript
beforeEach(() => {
  localStorage.clear();
});
```

**Cache CRUD shape**:
```typescript
import { DualLevelCache } from "./dualLevelCache";

describe("DualLevelCache", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("memory hit returns value without touching storage", () => {
    const cache = new DualLevelCache<string>();
    cache.set("k", "v");
    expect(cache.get("k")).toBe("v");
  });
});
```

---

#### `src/utils/errorHandler.test.ts` (test, request-response)

**Analog:** `src/lib/__tests__/loginPreflight.test.ts`

**Console/message mock pattern**:
```typescript
beforeEach(() => {
  vi.spyOn(console, "error").mockImplementation(() => {});
});
afterEach(() => {
  vi.restoreAllMocks();
});
```

---

### hooks Tests (INFRA-03)

#### `src/hooks/useTableManager.test.tsx` (test, request-response)

**Analog:** `src/hooks/usePagination.test.tsx`

**Full pattern** (lines 10-33):
```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { usePagination } from "./usePagination";

const wrapper =
  (initialPath: string) =>
  ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
  );

describe("usePagination", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it("setCurrent persists to sessionStorage", () => {
    const { result } = renderHook(() => usePagination(), {
      wrapper: wrapper("/system/user"),
    });
    act(() => result.current.setCurrent(5));
    expect(result.current.current).toBe(5);
  });
});
```

---

#### `src/hooks/useColumnConfig.test.tsx` (test, request-response)

**Analog:** `src/hooks/useServerSort.test.tsx`

**Sorter meta pattern** (lines 21-22):
```typescript
const sorterMetas = [{ field: "username" }] as unknown as Array<{ field: string } | undefined>;
```

---

### store Tests (INFRA-04)

#### `src/store/authStore.test.ts` (test, event-driven)

**Analog:** `src/utils/sm2.test.ts` + Zustand reset pattern

**Mock pattern**:
```typescript
const mockPost = vi.hoisted(() => vi.fn());
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));
```

**Zustand reset pattern** (D-05, from RESEARCH):
```typescript
import { useAuthStore } from "@/store/authStore";

beforeEach(() => {
  useAuthStore.setState({
    user: null,
    isAuthenticated: false,
    loading: false,
    menusLoaded: false,
    initialized: false,
  });
});
```

---

#### `src/store/tabsStore.test.ts` (test, CRUD)

**Analog:** `src/hooks/usePersistedState.test.ts`

**Storage cleanup pattern**:
```typescript
beforeEach(() => {
  localStorage.clear();
  useTabsStore.setState({
    tabs: [],
    activeTab: "",
    history: [],
  });
});
```

---

### services / router / constants / types Tests (INFRA-05)

#### `src/services/encryptionConfig.test.ts` (test, request-response)

**Analog:** `src/lib/api/__tests__/networkApi.test.ts`

**Mock pattern**:
```typescript
const mockGet = vi.fn();
vi.mock("@/lib/api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
}));

import { getEncryptionConfig, getCachedEncryptionConfig, clearEncryptionConfigCache } from "./encryptionConfig";
```

---

#### `src/router/routeConfigManager.test.ts` (test, transform)

**Analog:** `src/constants/status.test.ts`

**Fixture + static assertion pattern** (lines 46-122):
```typescript
import { describe, it, expect } from "vitest";
import { RouteConfigManager } from "./routeConfigManager";

const menus = [
  {
    id: "1",
    menuName: "用户管理",
    menuType: "C",
    path: "system/user",
    visible: 1,
    meta: { title: "用户管理" },
  },
];

describe("RouteConfigManager", () => {
  it("initializes from menu list", () => {
    const manager = new RouteConfigManager();
    manager.initialize(menus);
    expect(manager.isInitialized()).toBe(true);
    expect(manager.getRouteTitle("system/user")).toBe("用户管理");
  });
});
```

---

#### `src/constants/storage.test.ts` (test, transform)

**Analog:** `src/hooks/usePersistedState.test.ts`

**Storage key cleanup pattern**:
```typescript
beforeEach(() => {
  sessionStorage.clear();
});

it("sanitizes path for key", () => {
  expect(sanitizePathForKey("/system/user")).toBe("system_user");
});

it("clears table state by path", () => {
  sessionStorage.setItem("xingran_table_state_system_user_current", "5");
  clearTableStateByPath("/system/user");
  expect(sessionStorage.getItem("xingran_table_state_system_user_current")).toBeNull();
});
```

---

#### `src/types/config.test.ts` (test, static)

**Analog:** `src/constants/status.test.ts`

**Static assertion pattern** (lines 46-64):
```typescript
describe("shared status constants", () => {
  it("options lock 0=启用 / 1=禁用", () => {
    expectOptions(ENABLE_DISABLE_OPTIONS, [
      { label: "启用", value: 0 },
      { label: "禁用", value: 1 },
    ]);
  });
});
```

**Type guard shape**:
```typescript
import { isValidLayoutType, isValidDensityMode, isValidColorMode, defaultUserPreferences } from "./config";

describe("config type guards", () => {
  it("validates layout type", () => {
    expect(isValidLayoutType("classic")).toBe(true);
    expect(isValidLayoutType("invalid")).toBe(false);
  });

  it("default preferences are well-formed", () => {
    expect(defaultUserPreferences.version).toBe(2);
    expect(defaultUserPreferences.theme.mode).toBe("light");
  });
});
```

---

## Shared Patterns

### 1. `vi.mock` Placement & Hoisting

**Source:** `src/lib/__tests__/loginPreflight.test.ts` (lines 1-27)
**Apply to:** All tests that mock `@/lib/api`, `@/utils/sm2`, `@/utils/sm4`, `@/utils/antdMessage`
```typescript
const mockPost = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
}));
```

**Pitfall:** module-level variables declared before `vi.mock` are hoisted and unreachable; always use `vi.hoisted()`.

---

### 2. Storage Cleanup in `beforeEach`

**Source:** `src/hooks/usePersistedState.test.ts` (lines 13-17)
**Apply to:** All hooks/store/utils tests touching `localStorage` / `sessionStorage`
```typescript
beforeEach(() => {
  sessionStorage.clear();
  localStorage.clear();
});
```

---

### 3. Zustand Store Reset

**Source:** Research pattern + `src/store/authStore.ts`
**Apply to:** All store tests
```typescript
beforeEach(() => {
  useAuthStore.setState({
    user: null,
    isAuthenticated: false,
    loading: false,
    menusLoaded: false,
    initialized: false,
  });
});
```

---

### 4. Console Error Suppression

**Source:** `src/lib/__tests__/loginPreflight.test.ts` (lines 31-45)
**Apply to:** Tests exercising error branches of API client / errorHandler / TokenManager
```typescript
beforeEach(() => {
  vi.spyOn(console, "error").mockImplementation(() => {});
});
afterEach(() => {
  vi.restoreAllMocks();
});
```

---

### 5. API Wrapper Contract Tests

**Source:** `src/lib/api/__tests__/networkApi.test.ts` (lines 50-67)
**Apply to:** `opsApi.test.ts`, `menuApi.test.ts`, `profileApi.test.ts`, `columnConfigApi.test.ts`, etc.
```typescript
describe("buildingApi", () => {
  beforeEach(() => {
    mockPost.mockReset();
  });

  it("calls correct endpoint", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { list: [], total: 0 } });
    await buildingApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenCalledWith("/ops/building/list", { current: 1, pageSize: 10 });
  });
});
```

---

### 6. Real Crypto + Deterministic Vectors

**Source:** Research pattern + `src/utils/sm2.test.ts`
**Apply to:** `sm4.test.ts`, `encoding.test.ts`, `SecureTokenStorageImpl.test.ts`
```typescript
const key = "0123456789abcdeffedcba9876543210";
const iv = "abcdef98765432100123456789abcdef";

it("round-trips plaintext", async () => {
  const cipher = await encryptSM4CBC("hello 国密", key, iv);
  expect(await decryptSM4CBC(cipher, key, iv)).toBe("hello 国密");
});
```

---

### 7. Fake Timers for Token Refresh

**Source:** `src/lib/__tests__/loginPreflight.test.ts` (lines 121-135)
**Apply to:** `TokenManager.test.ts`, `authStore.test.ts`
```typescript
vi.useFakeTimers();
try {
  const pending = manager.getAccessToken();
  await vi.advanceTimersByTimeAsync(31_000);
  await pending;
} finally {
  vi.useRealTimers();
}
```

---

### 8. vitest.config.ts Coverage Exclusions

**Source:** `xingran-react-frontend/vitest.config.ts` (lines 16-35)
**Apply to:** Gate script pathspec mirror; harness placement
```typescript
coverage: {
  provider: "v8",
  include: ["src/**/*.{ts,tsx}"],
  reporter: ["text", "json", "html"],
  exclude: [
    "src/test/",
    "**/*.d.ts",
    "src/components/cad-editor/**",
    "src/components/cad-elements/**",
  ],
}
```

**Critical:** `src/test/utils/` is excluded by `"src/test/"`; harness files must live there or they will pollute coverage denominator.

---

### 9. Gate Script Exit Codes

**Source:** `check-frontend-coverage.sh` header (lines 28-33)
**Apply to:** Plan0 verification
```text
Exit codes:
  0 — global + per-dir + no drift
  1 — global threshold not met OR parse failure
  2 — usage error / unreadable floors file
  4 — per-dir floor violation
  6 — whitelist drift
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `src/test/utils/renderWithProviders.tsx` | utility/harness | request-response | No existing shared harness; build from existing `MemoryRouter` + `ConfigProvider` fragments |
| `src/test/utils/createApiMock.ts` | utility/harness | request-response | No existing centralized API mock factory; build from per-test `vi.mock` patterns |
| `src/test/utils/mockAntdMessage.ts` | utility/harness | event-driven | No existing centralized message mock; build from `antdMessage.ts` + test usages |
| `src/lib/api.test.ts` | test | request-response | No existing test directly tests `api.ts` interceptors; D-07 double-track pattern is new |
| `src/router/routeConfigManager.test.ts` / `routeGenerator.test.ts` | test | transform | No existing router-specific tests; closest analog is constants/type guard tests |

---

## Metadata

**Analog search scope:**
- `xingran-react-frontend/src/**/*.test.ts`
- `xingran-react-frontend/src/**/*.test.tsx`
- `.github/scripts/check-*.sh`
- `.github/workflows/ci.yml`
- `.coverage-fe-floors`
- `.planning/frontend-coverage-baseline.md`
- `xingran-react-frontend/vitest.config.ts`
- `xingran-react-frontend/package.json`

**Files scanned:** 26 analog source files
**Pattern extraction date:** 2026-08-24

**Key conventions to enforce:**
1. All new test files use `*.test.ts` / `*.test.tsx` and sit next to source files (not `__tests__`, except existing component tests).
2. Harness files strictly under `src/test/utils/` (covered by `coverage.exclude`).
3. Gate fixes already committed; plan0 is verify + comment cleanup + trial PR.
4. Per-dir floor bumps happen in same commit as baseline.md append (D-11 ratchet).
5. No new dependencies; align vitest caret ranges to `^4.1.10` only if doing IN-06.
