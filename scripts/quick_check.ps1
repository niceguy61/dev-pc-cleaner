$ErrorActionPreference = 'Stop'
$root = Resolve-Path D:\windows_cleaner
Write-Host "Repo: $root"
Write-Host "Scanning help output..."
$help = go run $root --help 2>&1
$lines = $help -split "`n" | Select-Object -First 30
$lines | ForEach-Object { $_.TrimEnd() } | ForEach-Object { Write-Host $_ }

Write-Host "Running scan (table)..."
go run $root scan -min-mb 100 -project-root $root -project-depth 2 | Select-Object -First 40 | ForEach-Object { Write-Host $_ }

Write-Host "Running scan (json)..."
go run $root scan -output json -project-root $root -project-depth 2 | Select-Object -First 20 | ForEach-Object { Write-Host $_ }
