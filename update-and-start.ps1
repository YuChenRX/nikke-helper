param(
    [switch]$NoStart,
    [switch]$CheckOnly
)

$ErrorActionPreference = "Stop"

$repo = "YuChenRX/nikke-helper"
$repoApi = "https://api.github.com/repos/$repo/releases/latest"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$releaseDir = if (Test-Path -LiteralPath (Join-Path $root "interface.json")) {
    $root
} else {
    Join-Path $root "release"
}
$interfaceFile = Join-Path $releaseDir "interface.json"
$appExePath = Join-Path $releaseDir "alas-app.exe"
$legacyExePath = Join-Path $releaseDir "alas.exe"
$exePath = if (Test-Path -LiteralPath $appExePath) { $appExePath } else { $legacyExePath }
$workDir = Join-Path $root ".update"
$archivePath = Join-Path $workDir "latest.zip"
$extractDir = Join-Path $workDir "extract"

function Write-Step($message) {
    Write-Host "[nikke-helper] $message"
}

function Test-TcpPort($hostName, $port) {
    $client = [System.Net.Sockets.TcpClient]::new()
    try {
        $task = $client.ConnectAsync($hostName, [int]$port)
        if (-not $task.Wait(300)) {
            return $false
        }
        return $client.Connected
    } catch {
        return $false
    } finally {
        $client.Dispose()
    }
}

function Test-ProxyAvailable($proxyUrl) {
    if (-not $proxyUrl) {
        return $false
    }

    try {
        $uri = [Uri]$proxyUrl
        if ($uri.Host -in @("127.0.0.1", "localhost")) {
            return Test-TcpPort $uri.Host $uri.Port
        }
        return $true
    } catch {
        return $false
    }
}

function Get-WebProxy {
    $candidates = @(
        $env:HTTPS_PROXY,
        $env:HTTP_PROXY,
        $env:ALL_PROXY,
        "http://127.0.0.1:7890"
    )

    foreach ($candidate in $candidates) {
        if (Test-ProxyAvailable $candidate) {
            return $candidate
        }
    }

    return $null
}

function Invoke-WithProxyEnv($proxy, [scriptblock]$body) {
    $oldHttp = $env:HTTP_PROXY
    $oldHttps = $env:HTTPS_PROXY
    $oldAll = $env:ALL_PROXY

    try {
        if ($proxy) {
            $env:HTTP_PROXY = $proxy
            $env:HTTPS_PROXY = $proxy
            $env:ALL_PROXY = $proxy
        }
        return & $body
    } finally {
        $env:HTTP_PROXY = $oldHttp
        $env:HTTPS_PROXY = $oldHttps
        $env:ALL_PROXY = $oldAll
    }
}

function Get-CurrentVersion {
    if (-not (Test-Path -LiteralPath $interfaceFile)) {
        return ""
    }

    try {
        $json = Get-Content -Raw -LiteralPath $interfaceFile | ConvertFrom-Json
        return [string]$json.version
    } catch {
        Write-Step "读取本地版本失败：$($_.Exception.Message)"
        return ""
    }
}

function Get-LatestRelease {
    $headers = @{
        "User-Agent" = "nikke-helper-updater"
        "Accept" = "application/vnd.github+json"
    }

    $request = @{
        Uri = $repoApi
        Headers = $headers
    }
    $proxy = Get-WebProxy
    if ($proxy) {
        Write-Step "使用代理：$proxy"
        $request.Proxy = $proxy
    }

    try {
        return Invoke-RestMethod @request
    } catch {
        Write-Step "PowerShell 请求 GitHub 失败，尝试 gh：$($_.Exception.Message)"
    }

    if (Get-Command gh -ErrorAction SilentlyContinue) {
        return Invoke-WithProxyEnv $proxy {
            $json = & gh api "repos/$repo/releases/latest"
            if ($LASTEXITCODE -ne 0) {
                throw "gh api 请求失败"
            }
            return $json | ConvertFrom-Json
        }
    }

    if (Get-Command curl.exe -ErrorAction SilentlyContinue) {
        return Invoke-WithProxyEnv $proxy {
            $curlArgs = @("-L", "-H", "User-Agent: nikke-helper-updater", $repoApi)
            if ($proxy) {
                $curlArgs = @("--proxy", $proxy) + $curlArgs
            }
            $json = & curl.exe @curlArgs
            if ($LASTEXITCODE -ne 0) {
                throw "curl 请求 GitHub 失败"
            }
            return $json | ConvertFrom-Json
        }
    }

    throw "无法请求 GitHub：PowerShell 请求失败，且未找到 gh 或 curl.exe"
}

function Select-WindowsAsset($release) {
    $assets = @($release.assets)
    $asset = $assets |
        Where-Object { $_.name -match "win" -and $_.name -match "x86_64" -and $_.name -match "\.zip$" } |
        Select-Object -First 1

    if (-not $asset) {
        $asset = $assets |
            Where-Object { $_.name -match "\.zip$" } |
            Sort-Object size -Descending |
            Select-Object -First 1
    }

    return $asset
}

function Test-AppRunning {
    $target = [System.IO.Path]::GetFullPath($exePath)
    $processes = Get-Process -Name "alas" -ErrorAction SilentlyContinue
    foreach ($process in $processes) {
        try {
            if ($process.Path -and ([System.IO.Path]::GetFullPath($process.Path) -eq $target)) {
                return $true
            }
        } catch {
            return $true
        }
    }
    return $false
}

function Clear-UpdateWorkDir {
    if (Test-Path -LiteralPath $workDir) {
        Remove-Item -LiteralPath $workDir -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $workDir | Out-Null
}

function Get-PackageRoot {
    $interface = Get-ChildItem -LiteralPath $extractDir -Recurse -Filter "interface.json" |
        Select-Object -First 1

    if (-not $interface) {
        throw "更新包中没有 interface.json"
    }

    return $interface.Directory.FullName
}

function Install-Package($packageRoot) {
    $preserved = @("config", "cache", "debug", ".update")

    New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null

    Get-ChildItem -LiteralPath $packageRoot -Force | ForEach-Object {
        if ($preserved -contains $_.Name) {
            return
        }

        if ($_.Name -eq "alas.exe") {
            Write-Step "保留当前启动器 alas.exe"
            return
        }

        $target = Join-Path $releaseDir $_.Name
        if (Test-Path -LiteralPath $target) {
            Remove-Item -LiteralPath $target -Recurse -Force
        }
        Copy-Item -LiteralPath $_.FullName -Destination $target -Recurse -Force
    }
}

function Download-Asset($asset, $tagName) {
    Clear-UpdateWorkDir

    $downloadRequest = @{
        Uri = $asset.browser_download_url
        OutFile = $archivePath
        Headers = @{ "User-Agent" = "nikke-helper-updater" }
    }
    $proxy = Get-WebProxy
    if ($proxy) {
        $downloadRequest.Proxy = $proxy
    }

    try {
        Invoke-WebRequest @downloadRequest
        return
    } catch {
        Write-Step "PowerShell 下载失败，尝试 gh：$($_.Exception.Message)"
    }

    if (Get-Command gh -ErrorAction SilentlyContinue) {
        Invoke-WithProxyEnv $proxy {
            & gh release download $tagName --repo $repo --pattern $asset.name --dir $workDir --clobber
            if ($LASTEXITCODE -ne 0) {
                throw "gh release download 失败"
            }
        }
        $downloaded = Get-ChildItem -LiteralPath $workDir -Filter "*.zip" | Select-Object -First 1
        if (-not $downloaded) {
            throw "gh 下载后没有找到 zip 文件"
        }
        Move-Item -LiteralPath $downloaded.FullName -Destination $archivePath -Force
        return
    }

    if (Get-Command curl.exe -ErrorAction SilentlyContinue) {
        Invoke-WithProxyEnv $proxy {
            $curlArgs = @("-L", "-H", "User-Agent: nikke-helper-updater", "-o", $archivePath, $asset.browser_download_url)
            if ($proxy) {
                $curlArgs = @("--proxy", $proxy) + $curlArgs
            }
            & curl.exe @curlArgs
            if ($LASTEXITCODE -ne 0) {
                throw "curl 下载失败"
            }
        }
        return
    }

    throw "无法下载更新包：PowerShell 下载失败，且未找到 gh 或 curl.exe"
}

function Start-App {
    if ($NoStart) {
        Write-Step "已跳过启动"
        return
    }

    if (Test-Path -LiteralPath $exePath) {
        Write-Step "启动 $([System.IO.Path]::GetFileName($exePath))"
        Start-Process -FilePath $exePath -WorkingDirectory $releaseDir
    } else {
        Write-Step "未找到 $exePath"
    }
}

try {
    $currentVersion = Get-CurrentVersion
    if ($currentVersion) {
        Write-Step "当前版本：$currentVersion"
    } else {
        Write-Step "未找到本地版本，将尝试检查最新版本"
    }

    $latest = Get-LatestRelease
    $latestVersion = [string]$latest.tag_name
    Write-Step "GitHub 最新版本：$latestVersion"

    if ($CheckOnly) {
        if ($currentVersion -eq $latestVersion) {
            Write-Step "检查结果：已是最新版本"
        } else {
            Write-Step "检查结果：发现新版本 $latestVersion"
        }
        Start-App
        exit 0
    }

    if ($currentVersion -eq $latestVersion) {
        Write-Step "已是最新版本"
        Start-App
        exit 0
    }

    if (Test-AppRunning) {
        Write-Step "alas.exe 正在运行，跳过更新。请退出程序后再运行此启动器。"
        Start-App
        exit 0
    }

    $asset = Select-WindowsAsset $latest
    if (-not $asset) {
        Write-Step "没有找到 Windows zip 更新包，跳过更新"
        Start-App
        exit 0
    }

    Write-Step "下载更新包：$($asset.name)"
    Download-Asset $asset $latestVersion

    Write-Step "解压更新包"
    New-Item -ItemType Directory -Force -Path $extractDir | Out-Null
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force

    $packageRoot = Get-PackageRoot
    Write-Step "安装更新：$latestVersion"
    Install-Package $packageRoot

    Write-Step "更新完成"
} catch {
    Write-Step "更新失败：$($_.Exception.Message)"
    Write-Step "继续启动当前版本"
} finally {
    try {
        if (Test-Path -LiteralPath $workDir) {
            Remove-Item -LiteralPath $workDir -Recurse -Force
        }
    } catch {
        Write-Step "清理临时目录失败：$($_.Exception.Message)"
    }
}

Start-App
