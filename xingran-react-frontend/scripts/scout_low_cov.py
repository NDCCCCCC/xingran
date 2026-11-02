import json
d = json.load(open('coverage/coverage-final.json'))
out = []
for path, info in d.items():
    npath = path.replace('\\', '/')
    if 'src/pages/' in npath and '__tests__' not in npath:
        stmts = len(info['statementMap'])
        if stmts == 0: continue
        cov = sum(1 for v in info['s'].values() if v > 0)
        pct = cov/stmts*100
        if pct < 50 and stmts > 30:
            # Trim prefix
            short = npath.split('src/pages/', 1)[1]
            out.append((pct, stmts, 'pages/' + short))
out.sort()
for p,s,p2 in out[:40]:
    print(f'{p:5.1f}% {s:5d} {p2}')
