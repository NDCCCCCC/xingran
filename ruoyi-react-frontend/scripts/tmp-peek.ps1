# 读取源码 any 上下文（用后删）
$spots = @(
  'src/components/dashboard/settings/EndpointSelector.tsx:49',
  'src/components/dashboard/widgets/WidgetRenderer.tsx:41',
  'src/components/operations/WorkstationDeviceTable/index.tsx:446',
  'src/components/shared/ExcelExport.tsx:20',
  'src/components/shared/ExcelExport.tsx:148',
  'src/components/shared/ExcelImport.tsx:261',
  'src/components/shared/NetworkExport.tsx:15',
  'src/components/shared/NetworkExport.tsx:38',
  'src/lib/api.ts:99',
  'src/lib/api.ts:106',
  'src/pages/knowledge/articles/index.tsx:164',
  'src/pages/monitor/logs/index.tsx:192',
  'src/pages/monitor/server/index.tsx:152',
  'src/pages/network/backups/index.tsx:262',
  'src/pages/network/command/index.tsx:141',
  'src/pages/network/credentials/index.tsx:198',
  'src/pages/network/devices/index.tsx:384',
  'src/pages/network/discoveries/index.tsx:197',
  'src/pages/network/executions/index.tsx:137',
  'src/pages/network/mac/index.tsx:249',
  'src/pages/network/templates/index.tsx:104',
  'src/pages/network/templates/index.tsx:190',
  'src/pages/network/templates/index.tsx:331',
  'src/pages/operations/rpa/tasks/modals/EditModal.tsx:182',
  'src/pages/system/config/index.tsx:152',
  'src/pages/system/settings/default-theme.tsx:73',
  'src/pages/system/settings/default-theme.tsx:79',
  'src/utils/iconUtils.tsx:115',
  'src/utils/sm-crypto.d.ts:21',
  'src/utils/tableHelpers.tsx:122'
)
foreach ($s in $spots) {
  $i = $s.LastIndexOf(':')
  $p = $s.Substring(0, $i); $l = [int]$s.Substring($i + 1)
  Write-Output "=== $s ==="
  Get-Content $p | Select-Object -Skip ([Math]::Max(0, $l - 2)) -First 3
}
