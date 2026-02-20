param(
  [string[]]$Paths,
  [switch]$Force,
  [switch]$ShutdownOnly
)

$ErrorActionPreference = 'Stop'

function Get-WslBasePaths {
  Get-ChildItem "HKCU:\Software\Microsoft\Windows\CurrentVersion\Lxss" | ForEach-Object {
    $p = Get-ItemProperty $_.PsPath
    if ($null -ne $p.BasePath -and $null -ne $p.DistributionName) {
      [pscustomobject]@{
        Name = $p.DistributionName
        BasePath = $p.BasePath
      }
    }
  }
}

function Resolve-Ext4Vhdx {
  param([string]$BasePath)
  $path = Join-Path $BasePath "ext4.vhdx"
  if (Test-Path $path) { return $path }
  return $null
}

Write-Host "WSL shutdown..."
try { wsl --shutdown | Out-Null } catch {}
if ($ShutdownOnly) { Write-Host "Shutdown only complete."; exit 0 }

$targets = @()
if ($Paths -and $Paths.Count -gt 0) {
  foreach ($p in $Paths) {
    $full = Resolve-Path $p -ErrorAction SilentlyContinue
    if ($null -ne $full) { $targets += $full.Path }
  }
} else {
  $entries = Get-WslBasePaths
  foreach ($e in $entries) {
    $vhdx = Resolve-Ext4Vhdx $e.BasePath
    if ($vhdx) {
      $targets += $vhdx
    }
  }
}

$targets = $targets | Select-Object -Unique
if ($targets.Count -eq 0) {
  Write-Host "No ext4.vhdx targets found."
  exit 1
}

Write-Host "Targets:"
$targets | ForEach-Object { Write-Host "  $_" }

if (-not $Force) {
  Write-Host "Proceed to Optimize-VHD on these targets? (y/n)"
  $resp = Read-Host
  if ($resp -notin @('y','Y','yes','YES')) {
    Write-Host "Aborted."
    exit 1
  }
}

foreach ($t in $targets) {
  Write-Host "Optimizing: $t"
  Optimize-VHD -Path $t -Mode Full
}

Write-Host "Done."
