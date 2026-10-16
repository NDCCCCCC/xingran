import json
data = json.load(open(r'D:\CODE\ClaudeCode\guoguo\xingran-react-frontend\coverage\coverage-final.json'))

ops_keys = [k for k in data if 'pages\\operations' in k or 'components\\operations' in k]
total_stmts = sum(len(data[k]['statementMap']) for k in ops_keys)
hit_stmts = sum(sum(1 for s in data[k]['statementMap'].values() if s.get('count', 0) > 0) for k in ops_keys)
print(f'operations: files={len(ops_keys)} total={total_stmts} hit={hit_stmts} ({hit_stmts * 100 / max(total_stmts, 1):.1f}%)')

print()
print('workstations 子组件:')
for k in sorted([x for x in ops_keys if 'workstations' in x]):
    sm = data[k]['statementMap']
    s_count = sum(1 for s in sm.values() if s.get('count', 0) > 0)
    idx = k.find('src\\')
    name = k[idx + 4:] if idx >= 0 else k
    print(f'  {name:60s} {s_count}/{len(sm)}')

print()
print('operations/index.tsx:')
for k in sorted([x for x in ops_keys if x.endswith('operations\\index.tsx') or x.endswith('operations/index.tsx')]):
    sm = data[k]['statementMap']
    s_count = sum(1 for s in sm.values() if s.get('count', 0) > 0)
    print(f'  {k} = {s_count}/{len(sm)}')