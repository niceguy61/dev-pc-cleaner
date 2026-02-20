# dev-pc-cleaner

![Go](https://img.shields.io/badge/Go-1.20%2B-00ADD8?logo=go&logoColor=white)

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
- `-docker-all`: Prune all unused images (requires `-docker-prune)`
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

## Example Output

```text
Language Detection
+-------------------+------------+-----------+------------------------------------------------------------+----------------------+
| Language          | Category   | Status    | Version                                                    | Command              |
+-------------------+------------+-----------+------------------------------------------------------------+----------------------+
| Go                | Systems    | Installed | go version go1.25.6 windows/amd64                          | go version           |
| Java              | General    | Installed | openjdk version "21.0.7" 2025-04-15 LTS                    | java -version        |
| JavaScript        | Web        | Installed | v22.21.1                                                   | node -v              |
| Python            | General    | Installed | Python 3.12.10                                             | python --version     |
| TypeScript        | Web        | Installed |                                                            | tsc -v               |
+-------------------+------------+-----------+------------------------------------------------------------+----------------------+

Cache Scan
+--------------------------------+----------+----------+---------+----------+-------+------------------------------------------+
| Item                           | Category | Priority | Status  | Size     | Files | Path                                     |
+--------------------------------+----------+----------+---------+----------+-------+------------------------------------------+
| go build cache                 | Go       | Low      | OK      | 40.4 MB  | 561   | C:\Users\USER\AppData\Local\go-build     |
| user temp                      | Logs     | Low      | Partial | 144.9 MB | 245   | C:\Users\USER\AppData\Local\Temp         |
| software distribution download | System   | Medium   | OK      | 972.5 MB | 33573 | C:\WINDOWS\SoftwareDistribution\Download |
| recycle bin                    | System   | Low      | Partial | 3.1 KB   | 20    | D:\$Recycle.Bin                          |
+--------------------------------+----------+----------+---------+----------+-------+------------------------------------------+

Summary
+-------+-------+--------+
| Items | Files | Size   |
+-------+-------+--------+
| 10    | 34834 | 1.2 GB |
+-------+-------+--------+
```

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
