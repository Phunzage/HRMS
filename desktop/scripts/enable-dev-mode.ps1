# 尝试开启 Windows 开发者模式（授予 SeCreateSymbolicLinkPrivilege），
# 使 7zip 能创建符号链接，规避 electron-builder winCodeSign 解压失败。
$ErrorActionPreference = 'Stop'
try {
    $key = 'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\AppModelUnlock'
    New-Item -Path $key -Force | Out-Null
    New-ItemProperty -Path $key -Name 'AllowDevelopmentWithoutDevLicense' -Value 1 -PropertyType DWord -Force | Out-Null
    Write-Output 'DEV_MODE_ENABLED'
} catch {
    Write-Output ('FAILED: ' + $_.Exception.Message)
}
