
$target = Get-Date -Date '2026-10-22 08:19:00'
while ((Get-Date) -lt $target) { Start-Sleep -Seconds 30 }
$csv = 'D:/CODE/ClaudeCode/xingran-go-backend/.planning/tmp-uat61-t1-samples-v5.csv'
$rows = @()
foreach ($line in Get-Content $csv) {
  if ($line -match '^(\d{2}:\d{2}:\d{2}\.\d{3}),(\d+),(\d+)$') { $rows += [pscustomobject]@{ ts=$Matches[1]; c=[int]$Matches[2]; g=[int]$Matches[3] } }
}
$rep = @()
$rep += "=== UAT61 Test1 v5 analysis $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') ==="
$rep += "parsed samples: $($rows.Count)"
if ($rows.Count -gt 0) {
  $rep += "first: $($rows[0].ts)  last: $($rows[-1].ts)"
  $dist = $rows | Group-Object c | Sort-Object Count -Descending | ForEach-Object { "$($_.Name)x$($_.Count)" }
  $rep += "member_count distribution: $($dist -join ' | ')"
  $mode = [int](($rows | Group-Object c | Sort-Object Count -Descending | Select-Object -First 1).Name)
  $gdist = $rows | Group-Object g | Sort-Object Count -Descending | ForEach-Object { "$($_.Name)x$($_.Count)" }
  $rep += "distinct_groups distribution: $($gdist -join ' | ')"
  $anom = $rows | Where-Object { $_.c -lt ($mode - 50) }
  if ($anom.Count -gt 0) {
    $rep += "VERDICT: V-SHAPE DETECTED ($($anom.Count) anomaly samples)"
    $rep += ($anom | Select-Object -First 15 | ForEach-Object { "$($_.ts) $($_.c),$($_.g)" })
  } else {
    $rep += "VERDICT: NO GAP OBSERVED - all $mode samples stable"
  }
} else { $rep += "VERDICT: NO DATA" }
$rep | Set-Content -Path 'D:/CODE/ClaudeCode/xingran-go-backend/.planning/tmp-uat61-t1-v5-verdict.txt' -Encoding utf8
$rep | ForEach-Object { Write-Output $_ }
