
$log = 'D:/CODE/ClaudeCode/xingran-go-backend/.planning/tmp-uat61-t1-samples-v5.csv'
'ts,member_count,distinct_groups' | Set-Content -Path $log -Encoding utf8
function Sample {
  $out = & C:\Windows\System32\wsl.exe -d Ubuntu-22.04 -- bash -c "PGPASSWORD=Cpic1234 psql -X -h 10.62.10.34 -p 5432 -U postgres -d xingran -A -t -F, -c 'SELECT count(*), count(DISTINCT group_dn) FROM sys_ad_group_member'" 2>$null
  return ($out | Select-Object -First 1)
}
$s = Sample
"$(Get-Date -Format 'HH:mm:ss.fff'),$s" | Add-Content $log
Write-Output "selftest: $s"
$target = Get-Date -Date '2026-10-22 08:00:00'
while ((Get-Date) -lt $target) { Start-Sleep -Milliseconds 500 }
Write-Output "window start $(Get-Date -Format 'HH:mm:ss')"
$end = Get-Date -Date '2026-10-22 08:18:00'
while ((Get-Date) -lt $end) {
  $ts = Get-Date -Format 'HH:mm:ss.fff'
  $v = Sample
  "$ts,$v" | Add-Content $log
  Start-Sleep -Milliseconds 150
}
'DONE' | Add-Content $log
Write-Output "done $(Get-Date -Format 'HH:mm:ss')"
