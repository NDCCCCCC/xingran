import json, collections, sys
with open('coverage/coverage-final.json') as f:
    data = json.load(f)
agg = collections.defaultdict(lambda: [0,0])
for path, cov in data.items():
    p = path.replace('\\', '/').split('xingran-react-frontend/')[-1]
    if not p.startswith('src/pages/network'):
        continue
    s = cov.get('s', {})
    for stmt_id, hit in s.items():
        agg[p][1] += 1
        if hit > 0:
            agg[p][0] += 1
prefix = sys.argv[1] if len(sys.argv) > 1 else ''
for p, (h, t) in sorted(agg.items(), key=lambda kv: kv[1][0]-kv[1][1]):
    if prefix and prefix not in p:
        continue
    print(f'{h:5}/{t:5}  {p}')
