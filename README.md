# windows_cleaner

Developer Windows cleaner focused on language/tool caches, logs, and Docker, with a CLI-first workflow.

## Requirements

- Go 1.20+ installed
  - Windows (PowerShell):
    - `winget install GoLang.Go`
  - Linux (Debian/Ubuntu):
    - `sudo apt-get update && sudo apt-get install -y golang`

## Quick Start

```powershell
go run . scan
go run . clean -apply
```

## Commands
- `scan`: Detect languages + scan caches (default)
- `detect`: Detect installed languages only
- `clean`: Remove cache locations (dry-run unless `-apply`)

## Common Options
- `-no-color`: Disable ANSI colors
- `-timeout`: Per-command timeout (e.g. `2s`, `1500ms`)
- `-output`: `table|json|csv`

## Cache Scan Options
- `-show-missing`: Include missing cache paths in output
- `-include-system`: Include system-level cache paths (default: true)
- `-min-mb`: Filter items smaller than this size (MB)
- `-min-files`: Filter items with fewer files than this count

## Clean Options
- `-apply`: Apply destructive changes
- `-allow-system-delete`: Allow deletion of system cache paths (requires `-apply`)
- `-docker-prune`: Run `docker system prune` during clean (default: true)
- `-docker-all`: Prune all unused images (requires `-docker-prune`)
- `-docker-volumes`: Prune unused volumes (requires `-docker-prune)`

## Project Scan Options
- `-project-root`: Scan project caches under this root
- `-project-depth`: Max directory depth for project scan (`-1` = unlimited)
- `-project-clean`: Include project cache clean
- `-project-review`: Review and select project items before cleaning
- `-project-exclude`: Comma-separated exclude paths for project scan
- `-project-no-default-exclude`: Disable default excludes for project scan
- `-recycle-bin-only`: Only scan/clean recycle bin (system scope)

Table formatting:
- `-cmd-max`: Max width for command column (0 = no limit)
- `-path-max`: Max width for path column (0 = no limit)

Review input supports:
- `all`, `none`
- `gt:500mb`
- `cat:web`
- `project:<path>`

## Config
- `-config`: Load config from file
- `-save-config`: Save config to file

Example:
```powershell
go run . scan -project-root D:\repo -project-depth 6 -min-mb 200 -save-config D:\cleaner.json
go run . clean -config D:\cleaner.json -apply -project-clean -project-review
```

## Output (JSON)
`scan` JSON contains:
- `languages`
- `items`
- `summary`
- `project`
- `projectSummary`

`clean` JSON contains:
- `languages`
- `items`
- `summary`
- `clean`
- `dockerPrune`
- `project`
- `projectSummary`
- `projectClean`

## WSL Disk Shrink

After `docker system prune`, WSL/Docker disk usage may not shrink automatically. Use:

```powershell
# interactive
powershell -ExecutionPolicy Bypass -File .\scripts\shrink_wsl.ps1

# non-interactive (CI/pipeline)
powershell -ExecutionPolicy Bypass -File .\scripts\shrink_wsl.ps1 -Force
```

Options:
- `-Paths`: Explicit VHDX paths to optimize
- `-ShutdownOnly`: Only run `wsl --shutdown`
- The script auto-detects WSL VHDX paths from the current user's registry (no hardcoded paths).
