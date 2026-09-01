import json
d = json.load(open('coverage/coverage-final.json'))
out = []
for path, info in d.items():
    npath = path.replace('\\', '/')
    if 'pages/operations/workstations' in npath and '__tests__' not in npath:
        stmts = len(info['statementMap'])
        if stmts == 0: continue
        cov = sum(1 for v in info['s'].values() if v > 0)
        pct = cov/stmts*100
        out.append((pct, stmts, npath.split('src/')[1]))
out.sort()
for p, s, p2 in out[:15]:
    print(f'{p:5.1f}% {s:5d} {p2}')
