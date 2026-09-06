/**
 * Phase 94 D-12: src/lib/*Api.ts 手写 CRUD 五件套模板残留扫描（双档分级）
 *
 * 镜像后端 internal/services/system/cache_invariants_92_test.go（go/ast 扫描 +
 * 白名单豁免 + 双档断言，Phase 92 D-10② 先例）。D-10「模板清零」定性锚点的
 * 可回归化：新增手写模板或删掉既有 KEEP 模板都会让本测试转红（防白名单腐烂）。
 *
 * 检测口径（AST 精确匹配，typescript 已在 devDependencies 零新装）：
 *   函数体（FunctionDeclaration / 对象字面量 PropertyAssignment / MethodDeclaration，
 *   含 concise 箭头体）为单条 return 语句、调用表达式标识符 ∈ {post,get,put,del}
 *   （api.ts 传输层四函数直调），且 URL 实参为【模板字符串】、其后缀匹配 CRUD
 *   形状（/list、/update、/delete、/batch-delete 结尾）——即与 createResourceApi
 *   8 方法同构的手写函数体。纯字符串字面量 URL（locationAliasApi pageNum 默认、
 *   getUserList 默认分页注入、batchDelete 系列 plain literal 等历史 KEEP 形态）
 *   不在本扫描口径内（Shared Pattern 5：前置参数注入不进工厂，勿误伤）。
 *
 * 断言双档（镜像 cache_invariants_92_test.go 的 allowedResidues 等值锁）：
 *   - 硬档：opsApi/rpaApi/vdiApi（对象形态，94-02 已接工厂）——实际计数必须逐文件
 *     == HARD_ALLOWED 登记基线（超出 = 新模板回归；减少 = 白名单腐烂，双向 fail）。
 *     基线非 0 的原因：迁移矩阵在这两文件内显式登记的 KEEP 结构（locationAliasApi /
 *     workstationDeviceApi / vmApi accounts 子资源族）含 CRUD 后缀形状模板。
 *   - warning 档：其余 *Api.ts——实际计数必须逐文件 == WARNING_WHITELIST 期望
 *     计数（实际 != 期望即 fail；每项附理由注释，新豁免必须显式登记）。
 *
 * 范围：仅 src/lib/*Api.ts（readdirSync 枚举 + import.meta.url 相对定位，禁本地
 * 绝对路径断言），豁免 api.ts / apiFactory.ts / download.ts；禁止扩大到全仓 src
 * （会误伤页面内联 post——RESEARCH Anti-Patterns）。
 *
 * 红绿演练（Phase 92 惯例，执行于 94-03 Task 3）：向任一硬档文件临时加入同构
 * 模板函数（单条 return post(`/x/${id}/delete`)），本测试转红；删除后恢复绿。
 */
import { readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import ts from "typescript";

// 注：ESLint typed-lint 程序未含 @types/node，node:fs/url 的调用可能被标记为
// unsafe —— 属工具链误报，测试运行时（vitest/node）类型完全正常（colors.test.ts
// 同款说明；本文件对 AST 节点全部走 ts.* 窄化，无 any）。

const LIB_DIR = dirname(fileURLToPath(import.meta.url));

/** 工厂/传输层自身豁免（D-12 范围 = 业务 *Api.ts） */
const EXEMPT_FILES = new Set(["api.ts", "apiFactory.ts", "download.ts"]);

/** 硬档：对象形态三文件，94-02 已接入 createResourceApi，新增模板残留即回归 */
const HARD_TIER_FILES = ["opsApi.ts", "rpaApi.ts", "vdiApi.ts"] as const;

const HARD_TIER = new Set<string>(HARD_TIER_FILES);

/**
 * 硬档允许残留基线（文件名 → 期望计数，实际 != 期望即 fail，双向锁）。
 * 非 0 项均为迁移矩阵（RESEARCH § 逐文件迁移矩阵 / PATTERNS KEEP 表）显式登记的
 * KEEP 结构，非未迁移模板：
 *   - opsApi.ts 4：locationAliasApi.update/delete（整对象 KEEP——pageNum/pageSize
 *     参数 + create 注入 scope:"workstation"，硬套工厂即行为变更）+
 *     workstationDeviceApi.update/delete（全异构对象 setPrimaryAndSave/syncAD 族中
 *     的两处标准形状方法）
 *   - rpaApi.ts 0：对象族已全量接入工厂（94-02）；Phase 100 spread→pick 后仍 0
 *   - vdiApi.ts 0：Phase 100 D-100-11 accounts 族删除后清零
 *     （原基线 1 = vmApi.deleteAccount，URL 恰为 /delete 后缀形状）；手写 CRUD 模板残留 0
 */
const HARD_ALLOWED: Record<string, number> = {
  "opsApi.ts": 4,
  "rpaApi.ts": 0,
  "vdiApi.ts": 0,
};

/** 13 个 *Api.ts 写死清单（sorted）：新增/删除 *Api.ts 必须显式更新本测试 */
const EXPECTED_FILES = [
  "adDomainApi.ts",
  "assetApi.ts",
  "columnConfigApi.ts",
  "dutyApi.ts",
  "knowledgeApi.ts",
  "menuApi.ts",
  "noticeApi.ts",
  "notificationConfigApi.ts",
  "opsApi.ts",
  "profileApi.ts",
  "rpaApi.ts",
  "vdiApi.ts",
  "workorderApi.ts",
];

/**
 * warning 档白名单：文件名 → 允许的手写 CRUD 模板残留数（期望计数写死）。
 * 实际 != 期望即 fail——新增模板与删除既有 KEEP 模板都会触发，防白名单腐烂。
 * 每项附理由（计数口径 = 本文件头部检测规则下的模板字符串命中数）。
 */
const WARNING_WHITELIST: Record<string, number> = {
  // —— KEEP 整文件 ——
  menuApi: 0, // D-06 非 CRUD：3 个只读函数（my-menus 树/全量/权限）
  profileApi: 0, // D-06 非 CRUD：单例 profile + 密码 + 头像，无资源集合语义
  columnConfigApi: 0, // D-06 非 CRUD：按 pageKey 存取，无标准 CRUD 路径形状
  notificationConfigApi: 0, // 整文件 KEEP：GET list / PUT /{id} / DELETE 动词 + page 参数，传输语义四处与工厂不同构（模板字符串 URL 亦无 CRUD 后缀）
  assetApi: 0, // 整文件 KEEP：reconciliation/fixSuggestion 全部解包 res.data 返回裸数据，与工厂「返回 BaseResponse」契约不同；无 CRUD 后缀形状模板命中
  // —— 部分委托文件的 KEEP 残留（delete 统一保持单参 post 直调：既有测试锁定
  //    单参契约，工厂 delete 传 {} 属 wire 级 body 变更——D-14 零改动红线）——
  noticeApi: 0, // admin notices 标准形状已委托；剩余 publish/withdraw/read/ignore 等全异构（非 CRUD 后缀）
  adDomainApi: 3, // deleteADConfig 单参直调 KEEP + updateADGroup/updateADUser（groups/users 异构族 KEEP）；configs/mappings/ou-group 其余标准形状已委托
  knowledgeApi: 5, // articles update/delete、categories delete、tags update/delete（update 直调 + delete 单参直调）；createKnowledgeArticle plain literal 不在模板口径
  dutyApi: 5, // pools update/delete、schedules delete、holidays update/delete；create/update 异构（memberIds/Omit idiom）与 plain literal list 不在模板口径
  workorderApi: 6, // orders update/delete、categories delete、periodic update/delete、comments/list 子资源列表；create/update 异构请求类型与 batchDelete plain literal 不在模板口径
};

interface TemplateHit {
  file: string;
  /** 函数名或对象方法 key */
  owner: string;
  /** 1-based 行号 */
  line: number;
}

/** api.ts 传输层四函数（工厂方法的底层依赖） */
const TRANSPORT_FNS = new Set(["post", "get", "put", "del"]);

/** CRUD 形状后缀（与 createResourceApi 8 方法的 URL 拼接形状对应） */
const CRUD_SUFFIXES = ["/list", "/update", "/delete", "/batch-delete"];

/** 枚举 src/lib 下业务 *Api.ts（豁免工厂/传输/下载自身）——范围仅此目录 */
function listApiFiles(): string[] {
  return readdirSync(LIB_DIR)
    .filter((f) => f.endsWith("Api.ts") && !EXEMPT_FILES.has(f))
    .sort();
}

/** URL 实参是否为「模板字符串 + CRUD 后缀结尾」：`${base}/${id}/delete` 等 */
function isCrudShapedTemplateUrl(arg: ts.Expression): boolean {
  // 无占位模板串 `/x/list`
  if (ts.isNoSubstitutionTemplateLiteral(arg)) {
    return CRUD_SUFFIXES.some((suffix) => arg.text.endsWith(suffix));
  }
  // 含占位模板串 `/x/${id}/delete`：取最后一个 span 的静态尾部判断
  if (ts.isTemplateExpression(arg)) {
    const tail = arg.templateSpans[arg.templateSpans.length - 1].literal.text;
    return CRUD_SUFFIXES.some((suffix) => tail.endsWith(suffix));
  }
  // 纯字符串字面量（plain literal）不在本扫描口径（历史 KEEP 形态，见文件头注释）
  return false;
}

/** 单条 return 语句的函数体：取出 return 表达式（剥 await / 括号 / 类型断言） */
function unwrapSingleReturn(body: ts.ConciseBody): ts.Expression | undefined {
  if (ts.isBlock(body)) {
    if (body.statements.length !== 1) return undefined;
    const only = body.statements[0];
    if (!ts.isReturnStatement(only) || !only.expression) return undefined;
    return only.expression;
  }
  // concise 箭头体（`list: async (p) => post(...)`）等价单 return
  return body;
}

/** 递归剥 await / Parenthesized / as 断言，露出底层调用表达式 */
function unwrapOuter(expr: ts.Expression): ts.Expression {
  for (;;) {
    if (ts.isAwaitExpression(expr) || ts.isParenthesizedExpression(expr)) {
      expr = expr.expression;
    } else if (
      ts.isAsExpression(expr) ||
      ts.isTypeAssertionExpression(expr) ||
      ts.isSatisfiesExpression(expr)
    ) {
      expr = expr.expression;
    } else {
      return expr;
    }
  }
}

/** 是否为手写 CRUD 模板：传输函数直调 + CRUD 形状模板字符串 URL */
function isCrudTemplateCall(expr: ts.Expression): boolean {
  const call = unwrapOuter(expr);
  if (!ts.isCallExpression(call)) return false;
  const callee = call.expression;
  if (!ts.isIdentifier(callee) || !TRANSPORT_FNS.has(callee.text)) return false;
  const urlArg = call.arguments[0];
  return urlArg !== undefined && isCrudShapedTemplateUrl(urlArg);
}

/** 扫描单文件：收集所有「单 return 手写 CRUD 模板」命中 */
function scanFile(fileName: string, sourceText: string): TemplateHit[] {
  const sf = ts.createSourceFile(fileName, sourceText, ts.ScriptTarget.Latest, true);
  const hits: TemplateHit[] = [];
  const visit = (node: ts.Node): void => {
    let owner: string | undefined;
    let body: ts.ConciseBody | undefined;
    if (ts.isFunctionDeclaration(node) && node.body) {
      owner = node.name?.text;
      body = node.body;
    } else if (ts.isPropertyAssignment(node)) {
      const init = node.initializer;
      if (ts.isArrowFunction(init) || ts.isFunctionExpression(init)) {
        owner = node.name.getText(sf);
        body = init.body;
      }
    } else if (ts.isMethodDeclaration(node) && node.body) {
      owner = node.name.getText(sf);
      body = node.body;
    } else if (ts.isVariableDeclaration(node)) {
      // WR-02 补强：`export const x = (id) => post(...)` 形态（箭头函数/函数表达式
      // 挂在变量声明器上）同样进扫描口径，堵住 tripwire 结构性盲区
      const init = node.initializer;
      if (
        ts.isIdentifier(node.name) &&
        init &&
        (ts.isArrowFunction(init) || ts.isFunctionExpression(init))
      ) {
        owner = node.name.text;
        body = init.body;
      }
    }
    if (owner !== undefined && body !== undefined) {
      const returnExpr = unwrapSingleReturn(body);
      if (returnExpr !== undefined && isCrudTemplateCall(returnExpr)) {
        const { line } = sf.getLineAndCharacterOfPosition(node.getStart(sf));
        hits.push({ file: fileName, owner, line: line + 1 });
      }
    }
    ts.forEachChild(node, visit);
  };
  ts.forEachChild(sf, visit);
  return hits;
}

function formatHits(hits: TemplateHit[]): string {
  return hits.map((h) => `  - ${h.file}:${h.line} ${h.owner}()`).join("\n");
}

describe("apiFactory invariants — src/lib/*Api.ts 手写 CRUD 模板扫描（D-12 双档）", () => {
  const files = listApiFiles();
  const hitsByFile = new Map<string, TemplateHit[]>(
    files.map((f) => [f, scanFile(f, readFileSync(join(LIB_DIR, f), "utf-8"))])
  );

  it("扫描范围恰为 13 个业务 *Api.ts（3 对象形态 + 10 扁平/KEEP），不越 src/lib 半步", () => {
    expect(files).toEqual(EXPECTED_FILES);
  });

  it("硬档 opsApi/rpaApi/vdiApi 模板残留 == 登记基线（新增模板 = 回归，双向 fail）", () => {
    const drifted = HARD_TIER_FILES.filter(
      (f) => (hitsByFile.get(f) ?? []).length !== HARD_ALLOWED[f]
    );
    const report = HARD_TIER_FILES.map((f) => {
      const actual = hitsByFile.get(f) ?? [];
      return `${f}: 实际 ${actual.length} != 期望 ${HARD_ALLOWED[f]}\n${formatHits(actual)}`;
    }).join("\n");
    expect(drifted, `硬档模板残留漂移（新模板即回归）:\n${report}`).toEqual([]);
  });

  it("warning 档白名单无多无漏（每个非硬档文件都必须显式登记期望计数）", () => {
    const nonHard = files.filter((f) => !HARD_TIER.has(f));
    expect(
      Object.keys(WARNING_WHITELIST)
        .map((f) => `${f}.ts`)
        .sort()
    ).toEqual(nonHard);
  });

  it("warning 档各文件实际计数 == whitelist 期望（不等即 fail，防白名单腐烂）", () => {
    const nonHard = files.filter((f) => !HARD_TIER.has(f));
    const drifted = nonHard.filter((f) => {
      const actual = hitsByFile.get(f) ?? [];
      return actual.length !== WARNING_WHITELIST[f.replace(/\.ts$/, "")];
    });
    const report = drifted
      .map((f) => {
        const key = f.replace(/\.ts$/, "");
        const actual = hitsByFile.get(f) ?? [];
        return `${f}: 实际 ${actual.length} != 期望 ${WARNING_WHITELIST[key]}\n${formatHits(actual)}`;
      })
      .join("\n");
    expect(drifted, `白名单期望计数漂移（新增残留或白名单腐烂）:\n${report}`).toEqual([]);
  });
});

/**
 * Phase 100 D-100-2/D-100-5: 对象 keys 基线锁（方法集防回增的第二道守卫）。
 *
 * AST 模板扫描管「手写 CRUD 模板」，keys 等值断言管「方法集漂移」——
 * 每个对象实际 keys == 后端已注册路由的实测存活集（D-100-9 端态;
 * D-100-12 register/heartbeat 前端方法已删,后端公开路由保留）。
 * 新增方法（含幽灵 batch/statistics/searchOptions 回流）或私删方法双向即红。
 *
 * 基线数值与 rpaApi.test.ts / vdiApi.test.ts 末尾 keys 断言互指（必须一致）;
 * 对账依据见 .planning/phases/100-frontend-contract-fixes/RECONCILIATION.md。
 */
import { aiApi, executionApi, taskApi, workerApi } from "./rpaApi";
import { vdiServerApi, vmApi } from "./vdiApi";

const POST_CLEANUP_BASELINE: Record<string, string[]> = {
  // taskApi: rpa_router.go:48-53（list/get/create/update/delete 工厂 + execute 手写）
  taskApi: ["create", "delete", "execute", "get", "list", "update"],
  // workerApi: rpa_router.go:64/:66（statistics 为 D-100-9 无参收窄版）
  workerApi: ["list", "statistics"],
  // executionApi: rpa_router.go:84-89（list/get/statistics 工厂 + cancel/logs 手写）
  executionApi: ["cancel", "get", "list", "logs", "statistics"],
  // aiApi: rpa_router.go:101-108
  aiApi: ["analyzeFailure", "decide", "generateScript", "optimizeScript"],
  // vmApi: vm_router.go:18-43 + D-100-10 补注册的 /operate（accounts 族 D-100-11 已删）
  vmApi: [
    "batchOperate",
    "bindUser",
    "create",
    "delete",
    "get",
    "list",
    "listNetworks",
    "listResourceGroups",
    "listResources",
    "listRunPositions",
    "listStorages",
    "listVTPPlatforms",
    "operate",
    "sync",
    "unbindUser",
    "update",
  ],
  // vdiServerApi: vdi_server_router.go:14-19 + testConnection（/:id/test）
  vdiServerApi: ["create", "delete", "get", "list", "testConnection", "update"],
};

describe("apiFactory invariants — Phase 100 对象 keys 基线（D-100-2/D-100-5）", () => {
  const exported: Record<string, object> = {
    taskApi,
    workerApi,
    executionApi,
    aiApi,
    vmApi,
    vdiServerApi,
  };

  for (const [name, baseline] of Object.entries(POST_CLEANUP_BASELINE)) {
    it(`${name} 方法集 == 后端路由实测存活集（${baseline.length} 方法,回增/私删双向即红）`, () => {
      expect(Object.keys(exported[name]).sort()).toEqual(baseline);
    });
  }
});
