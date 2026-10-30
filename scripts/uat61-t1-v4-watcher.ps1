
# UAT61 Test1 v4 看门人：等到 03:13 后自动分析
Start-Sleep -Seconds 1
$target = Get-Date -Date '2026-10-22 03:13:00'
while ((Get-Date) -lt $target) { Start-Sleep -Seconds 30 }
$csv = 'D:/CODE/ClaudeCode/xingran-go-backend/.planning/tmp-uat61-t1-samples-v4.csv'
$rows = Import-Csv $csv | Where-Object { $_.ts -ne 'ts' -and $_.ts -ne '' }
$report = @()
$report += "=== UAT61 Test1 v4 analysis $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') ==="
$report += "total samples: $($rows.Count)"
$window = $rows | Where-Object { $_.member_count -match '^\d+$' }
$counts = $window | Group-Object member_count | Sort-Object Count -Descending | ForEach-Object { "$($_.Name) x$($_.Count)" }
$report += "member_count distribution: $($counts -join ' | ')"
# V 形检测：窗口内出现 <基线-50 的样本即为空窗
$nums = $window | ForEach-Object { [int]$_.member_count }
if ($nums.Count -gt 0) {
  $mode = ($window | Group-Object member_count | Sort-Object Count -Descending | Select-Object -First 1).Name
  $modeN = [int]$mode
  $anomalies = $window | Where-Object { [int]$_.member_count -lt ($modeN - 50) }
  $report += "baseline: $mode"
  if ($anomalies.Count -gt 0) {
    $report += "VERDICT: V-SHAPE DETECTED ($($anomalies.Count) anomaly samples)"
    $report += ($anomalies | Select-Object -First 15 | ForEach-Object { "$($_.ts) $($_.member_count),$($_.distinct_groups)" })
  } else {
    $report += "VERDICT: NO GAP OBSERVED - all samples stable at baseline"
  }
} else {
  $report += "VERDICT: NO DATA - poller failed"
}
$outFile = 'D:/CODE/ClaudeCode/xingran-go-backend/.planning/tmp-uat61-t1-v4-verdict.txt'
$report | Set-Content -Path $outFile -Encoding utf8
Write-Output "verdict written"
$report | ForEach-Object { Write-Output $_ }
