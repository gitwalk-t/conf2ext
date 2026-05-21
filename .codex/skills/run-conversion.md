# Skill: запуск прогона

## Когда использовать

Используй этот orchestrator, когда пользователь просит:
- запустить прогон
- выполнить `go run`
- проверить сборку расширения
- получить свежий dump
- проверить загрузку после изменений

## Назначение

Этот skill только оркестрирует запуск и мониторинг.

Детали cleanup и status-check вынесены в helper skills:
- `.codex/skills/cleanup-run-tails.md`
- `.codex/skills/check-run-status.md`

## Обязательный контекст

По умолчанию использовать:

```text
configs/config.json
```

Все пути трактуются относительно текущего worktree / текущего `cwd`.

## Порядок работы

1. Выполнить:

```text
.codex/skills/cleanup-run-tails.md
```

2. Выполнить:

```powershell
go build ./...
go test ./...
```

3. Запустить hidden wrapper:

```powershell
$runTimestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logDir = Join-Path (Get-Location) 'output\\_log'
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

$stdoutLog = Join-Path $logDir "run-$runTimestamp.stdout.log"
$stderrLog = Join-Path $logDir "run-$runTimestamp.stderr.log"
$pidFile = Join-Path $logDir "run-$runTimestamp.pid"

$process = Start-Process powershell `
    -WindowStyle Hidden `
    -PassThru `
    -RedirectStandardOutput $stdoutLog `
    -RedirectStandardError $stderrLog `
    -ArgumentList @(
        '-NoProfile',
        '-ExecutionPolicy', 'Bypass',
        '-Command',
        'go run . --config .\\configs\\config.json'
    )

$process.Id | Set-Content -Encoding UTF8 $pidFile
```

Не запускать `go run` напрямую в активной консоли.

## Heartbeat monitoring

Сразу после запуска:
- создать heartbeat-monitoring раз в 10 минут
- heartbeat должен использовать только:

```text
.codex/skills/check-run-status.md
```

- heartbeat не должен:
  - перезапускать прогон
  - завершать процессы
  - запускать cleanup

## Resume-from-validation

Если прогон упал после:

```text
xml step: validate dynamic list contracts
```

использовать:

```powershell
go run .\\cmd\\changefiles\\main.go .\\configs\\config.json <path-to-v8_src*> --resume-from-validation
```

Перед resume обязательно выполнить:

```text
.codex/skills/check-run-status.md
```

## PowerShell gotchas

Не использовать `$PID` как имя переменной.

Использовать нейминг:
- `$pidVal`
- `$wrapperPid`
- `$runPid`

## Запреты

- Не завершать процессы напрямую из orchestrator.
- Любой cleanup — только через `cleanup-run-tails.md`.
- Не делать широкий рефакторинг во время расследования.
- Не считать старый dump свежим без проверки `Configuration.xml`.
- Не анализировать BSL как источник XML/classification правил.
