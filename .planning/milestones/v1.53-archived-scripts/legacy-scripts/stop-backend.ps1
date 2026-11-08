$ErrorActionPreference = 'SilentlyContinue'
foreach ($n in @('ruoyi-backend','xingran-backend','main','go')) {
  Get-Process -Name $n | ForEach-Object {
    Write-Output ("stopping {0} PID={1}" -f $_.ProcessName, $_.Id)
    Stop-Process -Id $_.Id -Force
  }
}
Start-Sleep -Seconds 2
Write-Output "--- port 9000 listeners ---"
$lines = netstat -ano | Select-String ':9000'
if ($lines) { $lines | ForEach-Object { Write-Output $_.ToString() } } else { Write-Output "none" }
