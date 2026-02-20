# ROADMAP

## Next: Cache Rules Expansion
- Rust
  - rustup toolchains cache
  - cargo cache for git/checkouts
- Python
  - pip cache dir discovery via `pip cache dir`
  - uv cache
  - poetry virtualenvs
- Java
  - Maven/Gradle caches for custom user home
- Go
  - GOPATH discovery via `go env` and dynamic cache paths
- C/C++
  - clangd/llvm cache directories
- Data/ML
  - Jupyter, Hugging Face, pip wheel cache

## Next: Language Detection Enhancements
- Registry-based detection on Windows for common installs
- `go env`, `python -c` detection for cache paths
- Prefer `where.exe` + command probe for accuracy

## Next: Project-Level Cache Options
- Optional project scan for:
  - `node_modules`, `.gradle`, `.mvn`, `.venv`, `target`, `dist`, `build`, `.next`, `.nuxt`, `.cache`
- Scope-limited by workspace root and exclusions list

## Next: Safety & UX
- Review mode with interactive selection per item
- Configurable allow/deny lists per path
- Per-category thresholds (size/files)

## Next: Reporting
- JSON schema versioning
- CSV/JSON output with timestamps and host info
- Summary breakdown by language/tool
