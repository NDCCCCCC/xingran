import json
d = json.load(open('coverage/coverage-final.json'))
out = []
for path, info in d.items():
    npath = path.replace('\\', '/')
    if '__tests__' in npath: continue
    stmts = len(info['statementMap'])
    if stmts < 30 or stmts > 100: continue
    cov = sum(1 for v in info['s'].values() if v > 0)
    pct = cov/stmts*100
    if pct < 30:
        short = npath.split('src/', 1)[1]
        out.append((pct, stmts, short))
out.sort()
print('Found:', len(out))
for p, s, p2 in out[:25]:
    print(f'{p:5.1f}% {s:5d} {p2}')
