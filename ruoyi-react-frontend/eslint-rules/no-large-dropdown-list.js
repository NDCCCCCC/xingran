/**
 * Custom ESLint rule: 禁止 Select/AutoComplete/Cascader 反模式
 *
 * 触发场景:
 *  1. <Select> / <AutoComplete> / <Cascader> JSX 元素缺少 onSearch 属性
 *     → 必须配合服务端搜索 + 防抖,不能依赖客户端 filterOption
 *  2. *.list({ pageSize: 1000|2000|5000|10000 }) 在 src/pages/ 下且
 *     函数名匹配 dropdown loader 命名约定
 *     → 应该用 xxxApi.searchOptions() 或 pageSize: 50 + keyword 搜索
 *
 * 排除:
 *  - useRoleList.ts / useDict.ts (小数据集,文档明示)
 *  - 所有非 pages/ 目录下的 .list() 调用(可能在 hooks/utils 内合理使用)
 */

const { ESLintUtils } = require("typescript-eslint");

const createRule = ESLintUtils.RuleCreator(
  (name) => `https://example.com/rules/${name}`
);

// 危险 pageSize 值集合
const BAD_PAGE_SIZES = new Set([1000, 2000, 5000, 10000]);

// dropdown loader 函数名约定(任意 loadXxxOptions 或 fetchXxxOptions)
const DROPDOWN_FUNC_NAME = /^(load|fetch).+(Options|List|Search)$/i;

// 仅在 src/pages/ 下触发
const PAGE_DIR = /[\\/]+src[\\/]+pages[\\/]+/;

module.exports = createRule({
  name: "no-large-dropdown-list",
  meta: {
    type: "problem",
    docs: {
      description:
        "禁止 Select/AutoComplete/Cascader 反模式: 必须有 onSearch, 且禁止在 dropdown loader 中使用 pageSize >= 1000 的 list 接口",
    },
    schema: [],
    messages: {
      selectNoOnSearch:
        "Select/AutoComplete/Cascader 缺少 onSearch prop;必须使用服务端搜索 + 防抖 (debounce)。",
      largePageSizeInDropdownLoader:
        "下拉加载函数中禁止 pageSize >= 1000;使用 xxxApi.searchOptions() (后端 /dropdown-options) 或 pageSize:50 + onSearch 远程搜索。",
    },
  },
  defaultOptions: [],
  create(context) {
    const filename = context.getFilename();
    const inPage = PAGE_DIR.test(filename);

    return {
      // 规则 1: Select/AutoComplete/Cascader 必须有 onSearch prop
      "JSXOpeningElement[name.name=/^(Select|AutoComplete|Cascader)$/]"(node) {
        const hasOnSearch = node.attributes.some(
          (attr) =>
            attr.type === "JSXAttribute" &&
            attr.name &&
            attr.name.name === "onSearch"
        );
        if (!hasOnSearch) {
          context.report({ node, messageId: "selectNoOnSearch" });
        }
      },

      // 规则 2: *.list() / getUserList() 在 dropdown loader 中使用大 pageSize
      CallExpression(node) {
        if (!inPage) return;

        const callee = node.callee;
        if (
          callee.type !== "MemberExpression" ||
          !/^(list|getUserList)$/.test(callee.property.name || "")
        ) {
          return;
        }

        const first = node.arguments[0];
        if (!first || first.type !== "ObjectExpression") return;

        const pageSizeProp = first.properties.find(
          (p) =>
            p.type === "Property" &&
            p.key &&
            ((p.key.type === "Identifier" && p.key.name === "pageSize") ||
              (p.key.type === "Literal" && p.key.value === "pageSize")) &&
            p.value.type === "Literal" &&
            typeof p.value.value === "number"
        );
        if (!pageSizeProp) return;
        if (!BAD_PAGE_SIZES.has(pageSizeProp.value.value)) return;

        // 向上回溯函数声明或 const 赋值,确认是 dropdown loader
        const ancestors = context.getAncestors();
        const inDropdownLoader = ancestors.some((a) => {
          if (a.type === "FunctionDeclaration" && a.id && DROPDOWN_FUNC_NAME.test(a.id.name)) {
            return true;
          }
          if (
            a.type === "VariableDeclarator" &&
            a.id &&
            a.id.type === "Identifier" &&
            DROPDOWN_FUNC_NAME.test(a.id.name)
          ) {
            return true;
          }
          if (a.type === "MethodDefinition" && a.key && DROPDOWN_FUNC_NAME.test(a.key.name || "")) {
            return true;
          }
          return false;
        });

        if (inDropdownLoader) {
          context.report({ node, messageId: "largePageSizeInDropdownLoader" });
        }
      },
    };
  },
});