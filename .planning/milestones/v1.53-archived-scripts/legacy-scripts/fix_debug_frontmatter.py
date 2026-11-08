#!/usr/bin/env python3
"""Phase 40 批量 frontmatter 修复脚本（用户批准的 Standard 1 批量修复）。

对 .planning/debug/*.md 与 .planning/debug/resolved/*.md：
  - 无 frontmatter → 补最小标准块（slug=status=trigger=created=updated）
  - 缺 slug → 从文件名推导（slug = 文件名 stem，小写连字符，符合 validator 格式）
  - 缺 status → resolved/ 目录补 resolved，否则 investigating
  - 缺 trigger/created/updated → 补默认值
  - messy 中文长 status（如 "RESOLVED - 等待用户确认修复方案"）→ 归一为 fix_applied
  - skip_audit: true 的文件跳过（knowledge-base.md）

合法但非 D-11 枚举的状态（checkpoint_reached 等）不改动 —— 由 validator 枚举扩充接受。
幂等：重复运行不产生额外变更。仅改动的文件写回。
"""
import os
import sys

VALID_STATUSES = {
    'resolved', 'root_cause_found', 'root_cause_identified',
    'awaiting_human_verify', 'investigating', 'investigation_in_progress',
    'verifying', 'fixed', 'fixing', 'fix_applied', 'fixed_pending_restart',
    'diagnosed', 'debug_complete', 'complete', 'checkpoint_reached', 'applied',
}
REQUIRED = ['slug', 'status', 'trigger', 'created', 'updated']
MESSY_STATUS_MAP = {
    'resolved - 等待用户确认修复方案': 'fix_applied',
    '"resolved - 等待用户确认修复方案"': 'fix_applied',
    '"fixing"': 'fixing',  # 去引号归一到枚举值 fixing
}


def parse_frontmatter(content):
    """返回 (fm_lines, body_lines, has_fm)。fm_lines 不含首尾 ---。"""
    lines = content.split('\n')
    if not lines or lines[0].strip() != '---':
        return [], lines, False
    for i in range(1, len(lines)):
        if lines[i].strip() == '---':
            return lines[1:i], lines[i + 1:], True
    # 有开头 --- 但无闭合 → 视为无 frontmatter
    return [], lines, False


def get_field(fm_lines, key):
    for ln in fm_lines:
        s = ln.strip()
        if s.startswith(key + ':'):
            return s[len(key) + 1:].strip()
    return None


def fix_file(path, in_resolved):
    with open(path, encoding='utf-8') as f:
        content = f.read()
    stem = os.path.splitext(os.path.basename(path))[0]
    fm_lines, body_lines, has_fm = parse_frontmatter(content)

    # skip_audit 顶层识别 → 不动
    if has_fm and any(l.strip() == 'skip_audit: true' for l in fm_lines):
        return False

    changed = False
    if not has_fm:
        # 无 frontmatter → 构造最小块
        status = 'resolved' if in_resolved else 'investigating'
        fm_lines = [
            f'slug: {stem}',
            f'status: {status}',
            'trigger: (legacy session, see body)',
            'created: 2026-06-25',
            'updated: 2026-06-25',
            'session_type: bug',
        ]
        changed = True
    else:
        # 归一 messy status
        cur_status = get_field(fm_lines, 'status')
        if cur_status is not None:
            key = cur_status.lower()
            if key in MESSY_STATUS_MAP:
                new_val = MESSY_STATUS_MAP[key]
                for idx, ln in enumerate(fm_lines):
                    if ln.strip().startswith('status:'):
                        fm_lines[idx] = f'status: {new_val}'
                        changed = True
                        break
        # 补缺字段
        existing = {ln.strip().split(':', 1)[0] for ln in fm_lines if ':' in ln}
        if 'slug' not in existing:
            fm_lines.insert(0, f'slug: {stem}')
            changed = True
        if 'status' not in existing:
            status = 'resolved' if in_resolved else 'investigating'
            fm_lines.append(f'status: {status}')
            changed = True
        if 'trigger' not in existing:
            fm_lines.append('trigger: (legacy session, see body)')
            changed = True
        if 'created' not in existing:
            fm_lines.append('created: 2026-06-25')
            changed = True
        if 'updated' not in existing:
            fm_lines.append('updated: 2026-06-25')
            changed = True

    if not changed:
        return False
    new_content = '---\n' + '\n'.join(fm_lines) + '\n---\n' + '\n'.join(body_lines)
    # 保持原文件末尾换行
    if content.endswith('\n') and not new_content.endswith('\n'):
        new_content += '\n'
    with open(path, 'w', encoding='utf-8', newline='\n') as f:
        f.write(new_content)
    return True


def main():
    root = sys.argv[1] if len(sys.argv) > 1 else '.'
    debug_dir = os.path.join(root, '.planning', 'debug')
    resolved_dir = os.path.join(debug_dir, 'resolved')
    fixed = 0
    for d in sorted(glob_md(debug_dir)) + sorted(glob_md(resolved_dir)):
        in_resolved = os.path.dirname(d).endswith('resolved')
        if fix_file(d, in_resolved):
            fixed += 1
            print(f'  fixed: {os.path.relpath(d, root)}')
    print(f'\nTotal fixed: {fixed}')


def glob_md(directory):
    if not os.path.isdir(directory):
        return []
    return [os.path.join(directory, f) for f in sorted(os.listdir(directory))
            if f.endswith('.md') and os.path.isfile(os.path.join(directory, f))]


if __name__ == '__main__':
    main()
