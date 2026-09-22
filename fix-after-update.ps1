# MDA 环境自检 / 更新后自检（AAA + Interception）
#
# 结论性说明（已实测验证）：
#   MaaFramework 的 Win32 控制器「Interception」模式是 **直接和驱动通信** 的
#   （引擎内部 CreateFile("\\.\interception00"..) + DeviceIoControl(IOCTL_WRITE)），
#   **不需要 interception.dll**。所以 MDA 更新后不需要补任何文件。
#
# 本脚本只做三件事：
#   1. 检查 Interception 驱动是否已安装并可用（设备 \\.\interception00 能否打开）
#   2. 检查 interface.json 是否带 AAA 控制器（mouse=Interception）
#   3. 检查各实例连接方式
#
# 用法：pwsh -File .\fix-after-update.ps1

$ErrorActionPreference = "Continue"
$root = $PSScriptRoot
$release = Join-Path $root "release"
$ok = $true

function Say($m, $level = "info") {
    $tag = switch ($level) { "ok" { "[ok]  " } "warn" { "[warn]" } "err" { "[err] " } default { "[info]" } }
    Write-Host "$tag $m"
}

Write-Host "=== MDA 环境自检（AAA / Interception） ==="
Write-Host ""

# ---------- 1. 驱动检查 ----------
$dev = "\\.\interception00"
$sig = [System.IO.File]::Open($dev, 'Open', 'ReadWrite', 'ReadWrite')
if ($sig) {
    $sig.Close()
    Say "Interception 驱动已安装且可用（$dev 可打开）" "ok"
} else {
    Say "Interception 驱动不可用！" "err"
    $ok = $false
}

$kb = "C:\Windows\System32\drivers\keyboard.sys"
if (Test-Path $kb) {
    $s = Get-AuthenticodeSignature $kb
    if ($s.SignerCertificate.Subject -match "Francisco Lopes") {
        Say "keyboard.sys 为 Interception 作者签名（驱动已替换）" "ok"
    } else {
        Say "keyboard.sys 签名不是 Interception 作者：$($s.SignerCertificate.Subject)" "warn"
    }
}

if (-not $ok) {
    Write-Host ""
    Say "如需安装驱动：下载 https://github.com/oblitum/Interception/releases/download/v1.0.1/Interception.zip" "info"
    Say "解压后以【管理员】运行：Interception\\command line installer\\install-interception.exe /install" "info"
    Say "然后重启电脑生效。" "info"
}

# ---------- 2. interface.json 的 AAA ----------
Write-Host ""
$iface = Join-Path $release "interface.json"
if (Test-Path $iface) {
    try {
        $json = Get-Content -Raw -LiteralPath $iface | ConvertFrom-Json
        $aaa = $json.controller | Where-Object { $_.name -eq "AAA" }
        if ($aaa) {
            Say "AAA 控制器存在（mouse=$($aaa.win32.mouse), keyboard=$($aaa.win32.keyboard)）" "ok"
        } else {
            Say "interface.json 里没有 AAA 控制器（版本过旧，建议更新 MDA）" "warn"
        }
    } catch {
        Say "interface.json 解析失败：$($_.Exception.Message)" "err"
    }
} else {
    Say "找不到 $iface" "err"
}

# ---------- 3. 实例连接方式 ----------
Write-Host ""
$cfg = Join-Path $release "config\mxu-MDA.json"
if (Test-Path $cfg) {
    try {
        $c = Get-Content -Raw -LiteralPath $cfg | ConvertFrom-Json
        foreach ($i in $c.instances) {
            $name = $i.name
            $ctrl = $i.controllerName
            if ($ctrl -eq "AAA") {
                Say "实例「$name」连接方式 = AAA" "ok"
            } else {
                Say "实例「$name」连接方式 = $ctrl（建议改为 AAA 以修复点击失效）" "warn"
            }
        }
    } catch {
        Say "实例配置解析失败：$($_.Exception.Message)" "warn"
    }
} else {
    Say "找不到实例配置 $cfg" "info"
}

Write-Host ""
Write-Host "完成。打开 MDA（release\alas-app.exe）后，连接方式选 AAA 即可。"
