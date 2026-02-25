Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $RepoRoot

$ToolsDir = Join-Path $RepoRoot ".tools"
$GoBin = Join-Path $ToolsDir "go\bin"

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

$ProtocDir = Get-ChildItem -Path $ToolsDir -Directory -Filter "protoc-*" |
    Sort-Object Name -Descending |
    Select-Object -First 1
$ProtocExe = $null
if ($ProtocDir) {
    $ProtocExe = Join-Path $ProtocDir.FullName "bin\protoc.exe"
    Add-PathItem (Join-Path $ProtocDir.FullName "bin")
}

Add-PathItem $GoBin

if (-not (Test-Path $ProtocExe) -and -not (Get-Command protoc -ErrorAction SilentlyContinue)) {
    throw "protoc not found. Run script/install-protoc.ps1 first."
}
if (-not (Test-Path (Join-Path $GoBin "protoc-gen-go.exe")) -and -not (Get-Command protoc-gen-go -ErrorAction SilentlyContinue)) {
    throw "protoc-gen-go not found. Run script/install-protoc.ps1 first."
}
if (-not (Test-Path (Join-Path $GoBin "protoc-gen-go-grpc.exe")) -and -not (Get-Command protoc-gen-go-grpc -ErrorAction SilentlyContinue)) {
    throw "protoc-gen-go-grpc not found. Run script/install-protoc.ps1 first."
}

$OutDir = Join-Path $RepoRoot "pkg/grpc/pb/v1"
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

protoc --go_out=. --go_opt=module=github.com/goclaw/goclaw `
    --go-grpc_out=. --go-grpc_opt=module=github.com/goclaw/goclaw `
    --proto_path=api/proto `
    api/proto/goclaw/v1/*.proto

Write-Host "Protobuf files generated."
