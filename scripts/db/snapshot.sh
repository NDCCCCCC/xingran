#!/usr/bin/env bash
#
# snapshot.sh — 生成 XingRan-Next 系统的 schema + seed snapshot SQL 文件
#
# 用途:
#   从生产 PostgreSQL DB 导出 schema (DDL) + seed (基础种子数据),
#   给新部署机器做一次性 psql -f 导入,取代 200+ 次启动期 migration 调用。
#
# 前置:
#   - pg_dump 命令在 PATH 中(或通过 PGDUMP_BIN 环境变量指定)
#   - 当前用户能连接到目标 DB
#   - 凭据可通过 PG* 环境变量或 .env 文件(DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME)提供
#
# 用法:
#   scripts/db/snapshot.sh                    # 默认导出到 docs/deployment/snapshots/
#   scripts/db/snapshot.sh --output /tmp/dbdump  # 自定义输出目录
#   scripts/db/snapshot.sh --schema-only      # 只导 schema (skip seed)
#   scripts/db/snapshot.sh --seed-only        # 只导 seed (skip schema)
#   scripts/db/snapshot.sh --help             # 打印本帮助
#
# 新部署流程:
#   1. 在生产 DB 上跑本脚本,得到 schema-{YYYY-MM-DD}.sql + seed-{YYYY-MM-DD}.sql
#   2. 拷贝两文件到新机器
#   3. psql -d newdb -f schema-{YYYY-MM-DD}.sql
#      psql -d newdb -f seed-{YYYY-MM-DD}.sql
#   4. ./xingran-backend   # AutoMigrate 仅做 model 字段增量,不再跑 200+ migration
#
# 历史: 260704-ne5 — 启动期 AutoMigrate 精简后,补一个独立 snapshot 工具
#

set -euo pipefail

# ---------- 默认值 ----------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUTPUT_DIR="${PROJECT_ROOT}/docs/deployment/snapshots"
EXPORT_SCHEMA=1
EXPORT_SEED=1
DATE="$(date -u +%Y-%m-%d)"

# ---------- 帮助 ----------
print_help() {
    cat <<'EOF'
snapshot.sh — 生成 schema + seed snapshot SQL 文件

用法:
  scripts/db/snapshot.sh [选项]

选项:
  --output <dir>     输出目录 (默认: docs/deployment/snapshots)
  --schema-only      只导 schema (DDL)
  --seed-only        只导 seed (基础种子)
  --help, -h         打印本帮助

环境变量:
  PGHOST / PGUSER / PGPASSWORD / PGDATABASE  标准 psql/libpq 变量
  PGPORT                                    标准 psql/libpq 变量 (默认 5432)
  PGDUMP_BIN                                pg_dump 路径 (默认: PATH 里的 pg_dump)
  DB_HOST / DB_PORT / DB_USER / DB_PASSWORD / DB_NAME   .env 风格回退
EOF
}

# ---------- 参数解析 ----------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        --schema-only)
            EXPORT_SEED=0
            shift
            ;;
        --seed-only)
            EXPORT_SCHEMA=0
            shift
            ;;
        --help|-h)
            print_help
            exit 0
            ;;
        *)
            echo "ERROR: 未知参数 $1" >&2
            print_help >&2
            exit 1
            ;;
    esac
done

# ---------- .env 加载 (DB_* 回退到 PG*) ----------
# .env 可能含特殊字符(如密码括号),直接 source 会失败;按白名单 key 一行行读,容忍格式噪音。
if [[ -f "${PROJECT_ROOT}/.env" ]]; then
    while IFS='=' read -r key value; do
        # 跳过空行 / 注释
        [[ -z "${key}" || "${key}" =~ ^[[:space:]]*# ]] && continue
        # 仅引入 DB_* 变量,且仅当 PG* 还未设置时
        case "${key}" in
            DB_HOST)    [[ -z "${PGHOST:-}"     ]] && export PGHOST="${value}" ;;
            DB_PORT)    [[ -z "${PGPORT:-}"     ]] && export PGPORT="${value}" ;;
            DB_USER)    [[ -z "${PGUSER:-}"     ]] && export PGUSER="${value}" ;;
            DB_PASSWORD) [[ -z "${PGPASSWORD:-}" ]] && export PGPASSWORD="${value}" ;;
            DB_NAME)    [[ -z "${PGDATABASE:-}" ]] && export PGDATABASE="${value}" ;;
        esac
    done < "${PROJECT_ROOT}/.env"
fi

# 回退: DB_* → PG*
: "${PGHOST:=${DB_HOST:-localhost}}"
: "${PGPORT:=${DB_PORT:-5432}}"
: "${PGUSER:=${DB_USER:-}}"
: "${PGPASSWORD:=${DB_PASSWORD:-}}"
: "${PGDATABASE:=${DB_NAME:-xingran_next}}"
: "${PGDUMP_BIN:=pg_dump}"

# ---------- 前置检查 ----------
if ! command -v "${PGDUMP_BIN}" >/dev/null 2>&1; then
    echo "ERROR: pg_dump 未找到 (PGDUMP_BIN='${PGDUMP_BIN}')." >&2
    echo "       请安装 PostgreSQL client 或设置 PGDUMP_BIN 指向 pg_dump 路径。" >&2
    exit 1
fi

if [[ -z "${PGUSER}" || -z "${PGDATABASE}" ]]; then
    echo "ERROR: 缺少连接凭据。设置 PGUSER/PGPASSWORD/PGDATABASE 或 DB_USER/DB_PASSWORD/DB_NAME。" >&2
    exit 1
fi

mkdir -p "${OUTPUT_DIR}"

SCHEMA_FILE="${OUTPUT_DIR}/schema-${DATE}.sql"
SEED_FILE="${OUTPUT_DIR}/seed-${DATE}.sql"

echo "============================================================"
echo "XingRan-Next DB Snapshot Generator"
echo "============================================================"
echo "Host:     ${PGHOST}:${PGPORT}"
echo "Database: ${PGDATABASE}"
echo "User:     ${PGUSER}"
echo "Output:   ${OUTPUT_DIR}"
echo "Date:     ${DATE}"
echo "Schema:   $([[ ${EXPORT_SCHEMA} -eq 1 ]] && echo "yes" || echo "skip")"
echo "Seed:     $([[ ${EXPORT_SEED} -eq 1 ]] && echo "yes" || echo "skip")"
echo "------------------------------------------------------------"

# pg_dump 公共参数:
#   --no-owner   跳过 OWNER TO 语句(部署用 superuser 建表,owner 不一致无关)
#   --no-acl     跳过 GRANT/REVOKE(权限另走 init_data 或运维手工)
PG_DUMP_COMMON=(
    --no-owner
    --no-acl
    --host="${PGHOST}"
    --port="${PGPORT}"
    --username="${PGUSER}"
    --dbname="${PGDATABASE}"
)

# ---------- Schema dump ----------
if [[ ${EXPORT_SCHEMA} -eq 1 ]]; then
    echo "[1/2] 导出 schema (DDL) ..."
    "${PGDUMP_BIN}" \
        --schema-only \
        -t 'sys_*' -t 'ops_*' \
        "${PG_DUMP_COMMON[@]}" \
        > "${SCHEMA_FILE}"
    echo "      -> ${SCHEMA_FILE}"
fi

# ---------- Seed dump ----------
# 仅导"基础种子"表(sys_*/ops_* 已建,导出必要字典/角色/菜单/配置/账号):
#   sys_menu, sys_role, sys_role_menu, sys_user, sys_config,
#   sys_dict_type, sys_dict_data, sys_ad_service_accounts, sys_post
# 用户密码哈希等敏感列天然导出,但落到磁盘后须按运维规范保护(snapshot 是机密文件)。
if [[ ${EXPORT_SEED} -eq 1 ]]; then
    echo "[2/2] 导出 seed (基础数据) ..."
    "${PGDUMP_BIN}" \
        --data-only \
        --inserts \
        -t sys_menu \
        -t sys_role \
        -t sys_role_menu \
        -t sys_user \
        -t sys_config \
        -t sys_dict_type \
        -t sys_dict_data \
        -t sys_ad_service_accounts \
        -t sys_post \
        "${PG_DUMP_COMMON[@]}" \
        > "${SEED_FILE}"
    echo "      -> ${SEED_FILE}"
fi

# ---------- 总结 ----------
echo "------------------------------------------------------------"
echo "完成。生成的 snapshot:"
[[ ${EXPORT_SCHEMA} -eq 1 ]] && {
    lines=$(wc -l < "${SCHEMA_FILE}")
    bytes=$(wc -c < "${SCHEMA_FILE}")
    echo "  schema : ${SCHEMA_FILE}  (${lines} 行, ${bytes} 字节)"
}
[[ ${EXPORT_SEED} -eq 1 ]] && {
    lines=$(wc -l < "${SEED_FILE}")
    bytes=$(wc -c < "${SEED_FILE}")
    echo "  seed   : ${SEED_FILE}  (${lines} 行, ${bytes} 字节)"
}
echo ""
echo "新部署用法:"
echo "  psql -d NEWDB -f ${SCHEMA_FILE}"
echo "  psql -d NEWDB -f ${SEED_FILE}"
echo "============================================================"