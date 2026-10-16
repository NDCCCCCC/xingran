"""Phase 88 诊断 — 正确读 v8 istanbul 格式: 命中数在 's' 对象, statementMap 只有坐标."""
import json
import sys
from collections import defaultdict

data = json.load(open(r'D:\CODE\ClaudeCode\guoguo\xingran-react-frontend\coverage\coverage-final.json'))

by_dir = defaultdict(lambda: [0, 0])
for k, v in data.items():
    src_idx = k.find('src\\')
    if src_idx < 0:
        continue
    rel = k[src_idx + 4:]
    parts = rel.split('\\')
    # gate 粒度: components 二级 / pages 二级 / 其它一级
    if parts[0] == 'components' and len(parts) > 1:
        top = 'components/' + parts[1]
    elif parts[0] == 'pages' and len(parts) > 1:
        top = 'pages/' + parts[1]
    else:
        top = parts[0] if parts else 'root'
    s_arr = v['s']
    total = len(s_arr)
    hit = sum(1 for x in s_arr.values() if x > 0)
    by_dir[top][0] += total
    by_dir[top][1] += hit

rows = []
for d, (t, h) in by_dir.items():
    pct = h * 100 / max(t, 1)
    rows.append((pct, d, h, t))

print(f'{"目录":<32}{"覆盖":>8}  {"hit/total":>14}')
for pct, d, h, t in sorted(rows):
    print(f'  {d:<30}{pct:>6.1f}%  {h:>6}/{t}')

if len(sys.argv) > 1 and sys.argv[1] == '-f':
    # 单文件明细过滤
    pat = sys.argv[2]
    print()
    print(f'== 文件明细: *{pat}* ==')
    for k in sorted(data.keys()):
        if pat.lower() not in k.lower():
            continue
        v = data[k]
        s_arr = v['s']
        hit = sum(1 for x in s_arr.values() if x > 0)
        src_idx = k.find('src\\')
        name = k[src_idx + 4:] if src_idx >= 0 else k
        if hit or '-a' in sys.argv:
            print(f'  {name:<60} {hit:>3}/{len(s_arr)}')