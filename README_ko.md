# dev-pc-cleaner

![Go](https://img.shields.io/badge/Go-1.20%2B-00ADD8?logo=go&logoColor=white)

[English](README.md) | [한국어](README_ko.md)

개발자용 Windows 클리너. 언어/도구 캐시, 로그, Docker 정리를 CLI 중심으로 제공합니다.

## 요구 사항

- Go 1.20+ 설치
  - Windows (PowerShell):
    - `winget install GoLang.Go`
  - Linux (Debian/Ubuntu):
    - `sudo apt-get update && sudo apt-get install -y golang`

## 빠른 시작

```powershell
go run . scan
go run . clean -apply
```

## 명령어
- `scan`: 언어 감지 + 캐시 스캔 (기본)
- `detect`: 설치된 언어만 감지
- `clean`: 캐시 삭제 (기본은 dry-run, `-apply` 필요)

## 공통 옵션
- `-no-color`: ANSI 색상 끄기
- `-timeout`: 커맨드 타임아웃 (예: `2s`, `1500ms`)
- `-output`: `table|json|csv`

## 캐시 스캔 옵션
- `-show-missing`: 존재하지 않는 경로도 표시
- `-include-system`: 시스템 캐시 포함 (기본: true)
- `-min-mb`: 이 크기(MB)보다 작은 항목 숨김
- `-min-files`: 이 파일 수보다 작은 항목 숨김

## 정리 옵션
- `-apply`: 실제 삭제 수행
- `-allow-system-delete`: 시스템 캐시 삭제 허용 (`-apply` 필요)
- `-docker-prune`: `docker system prune` 실행 (기본: true)
- `-docker-all`: 미사용 이미지 모두 정리 (`-docker-prune` 필요)
- `-docker-volumes`: 미사용 볼륨 정리 (`-docker-prune` 필요)

## 프로젝트 스캔 옵션
- `-project-root`: 프로젝트 캐시 스캔 루트
- `-project-depth`: 최대 깊이 (`-1` = 제한 없음)
- `-project-clean`: 프로젝트 캐시 삭제 포함
- `-project-review`: 삭제 전 선택(리뷰) 모드
- `-project-exclude`: 제외 경로(쉼표 구분)
- `-project-no-default-exclude`: 기본 제외 목록 비활성화
- `-recycle-bin-only`: 휴지통만 스캔/정리 (시스템 범위)

테이블 출력:
- `-cmd-max`: Command 컬럼 최대 폭 (0 = 제한 없음)
- `-path-max`: Path 컬럼 최대 폭 (0 = 제한 없음)

리뷰 입력 지원:
- `all`, `none`
- `gt:500mb`
- `cat:web`
- `project:<path>`

## 출력 예시

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

## 설정
- `-config`: 설정 파일 로드
- `-save-config`: 설정 파일 저장

예시:
```powershell
go run . scan -project-root D:\repo -project-depth 6 -min-mb 200 -save-config D:\cleaner.json
go run . clean -config D:\cleaner.json -apply -project-clean -project-review
```

## JSON 출력
`scan` JSON 포함 항목:
- `languages`
- `items`
- `summary`
- `project`
- `projectSummary`

`clean` JSON 포함 항목:
- `languages`
- `items`
- `summary`
- `clean`
- `dockerPrune`
- `project`
- `projectSummary`
- `projectClean`

## WSL 디스크 축소

`docker system prune` 이후에도 디스크 용량이 자동으로 줄지 않을 수 있습니다. 아래 스크립트를 사용하세요.

```powershell
# interactive
powershell -ExecutionPolicy Bypass -File .\scripts\shrink_wsl.ps1

# non-interactive (CI/pipeline)
powershell -ExecutionPolicy Bypass -File .\scripts\shrink_wsl.ps1 -Force
```

옵션:
- `-Paths`: 최적화할 VHDX 경로 직접 지정
- `-ShutdownOnly`: `wsl --shutdown`만 수행
- 스크립트는 사용자 레지스트리에서 VHDX 경로를 자동 탐색합니다.
