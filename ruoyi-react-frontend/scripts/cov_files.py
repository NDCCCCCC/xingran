"""Phase 88 — 列指定子串的文件覆盖明细(正确读 s 数组)."""
import json
import sys

data = json.load(open(r'D:\CODE\ClaudeCode\guoguo\xingran-react-frontend\coverage\coverage-final.json'))
pat = sys.argv[1].replace('/', '\\').lower()
show_all = '-a' in sys.argv

rows = []
for k in sorted(data.keys()):
    if pat not in k.lower():
        continue
    v = data[k]
    s_arr = v['s']
    hit = sum(1 for x in s_arr.values() if x > 0)
    if hit or show_all:
        src_idx = k.find('src\\')
        name = k[src_idx + 4:] if src_idx >= 0 else k
        rows.append((hit, len(s_arr), name))

for hit, total, name in sorted(rows):
    print(f'  {hit:>4}/{total:<4} {name}')