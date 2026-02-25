param(
    [string]$ProtocVersion = $(if ($env:PROTOC_VERSION) { $env:PROTOC_VERSION } else { "27.3" }),
    [string]$ToolsDir = "",
    [string]$GoProxy = $(if ($env:GOPROXY) { $env:GOPROXY } else { "" }),
    [switch]$ForceDownload,
    [switch]$PersistPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
if ([string]::IsNullOrWhiteSpace($ToolsDir)) {
    $ToolsDir = Join-Path $RepoRoot ".tools"
}

$DownloadDir = Join-Path $RepoRoot ".cache\downloads"
$ProtocDir = Join-Path $ToolsDir ("protoc-" + $ProtocVersion)
$ProtocBinDir = Join-Path $ProtocDir "bin"
$ProtocExe = Join-Path $ProtocBinDir "protoc.exe"
$ZipName = "protoc-$ProtocVersion-win64.zip"
$ZipPath = Join-Path $DownloadDir $ZipName
$ZipUrl = "https://github.com/protocolbuffers/protobuf/releases/download/v$ProtocVersion/$ZipName"

function Add-PathItem {
    param([string]$PathItem)
    if ([string]::IsNullOrWhiteSpace($PathItem)) {
        return
    }
    if (-not (Test-Path $PathItem)) {
        return
    }
    if ($env:PATH -notlike "*$PathItem*") {
        $env:PATH = "$PathItem;$env:PATH"
    }
}

New-Item -ItemType Directory -Force -Path $ToolsDir | Out-Null
New-Item -ItemType Directory -Force -Path $DownloadDir | Out-Null

if (-not (Test-Path $ProtocExe) -or $ForceDownload) {
    Write-Host "Downloading protoc $ProtocVersion..."
    Invoke-WebRequest -Uri $ZipUrl -OutFile $ZipPath
    if (Test-Path $ProtocDir) {
        Remove-Item -Recurse -Force $ProtocDir
    }
    Expand-Archive -Path $ZipPath -DestinationPath $ProtocDir
}

if (-not (Test-Path $ProtocExe)) {
    throw "protoc.exe not found after download: $ProtocExe"
}

Add-PathItem $ProtocBinDir

if (Get-Command go -ErrorAction SilentlyContinue) {
    $GoBin = Join-Path $ToolsDir "go\bin"
    New-Item -ItemType Directory -Force -Path $GoBin | Out-Null
    $env:GOBIN = $GoBin
    if (-not [string]::IsNullOrWhiteSpace($GoProxy)) {
        $env:GOPROXY = $GoProxy
    }
    Add-PathItem $GoBin

    Write-Host "Installing protoc-gen-go and protoc-gen-go-grpc..."
    go install "google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11"
    go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1"
} else {
    Write-Warning "Go not found in PATH. Skipping protoc-gen-go install."
}

Write-Host "protoc installed at: $ProtocExe"
Write-Host "PATH updated for current session."

if ($PersistPath) {
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($UserPath -notlike "*$ProtocBinDir*") {
        $UserPath = "$ProtocBinDir;$UserPath"
    }
    if ($env:GOBIN -and $UserPath -notlike "*$env:GOBIN*") {
        $UserPath = "$env:GOBIN;$UserPath"
    }
    [Environment]::SetEnvironmentVariable("PATH", $UserPath, "User")
    Write-Host "User PATH updated. Restart your shell to pick it up."
} else {
    Write-Host "To persist PATH, re-run with -PersistPath."
}

Write-Host "Next: run make proto."
