# Skill: запуск прогона

## Когда использовать

Используй этот skill, когда пользователь просит:
- запустить прогон
- проверить сборку расширения
- выполнить `go run`
- получить свежий дамп
- проверить ошибку загрузки после изменений

## Обязательный контекст

Активный конфиг по умолчанию:

`configs/config.json`

## Используемые helper skills

Перед запуском обязательно используй:
- `.codex/skills/cleanup-run-tails.md`
- `.codex/skills/check-run-status.md`

`cleanup-run-tails.md` отвечает за:
- безопасное завершение старых project-процессов
- cleanup зависших wrapper-процессов
- запрет завершения чужих `1cv8.exe`

`check-run-status.md` отвечает за:
- определение состояния текущего прогона
- проверку stdout/stderr
- проверку dump snapshot
- проверку `Configuration.xml`
- вывод status summary

## Порядок работы orchestrator

1. Выполнить `.codex/skills/cleanup-run-tails.md`
2. Выполнить:

```powershell
go build ./...
go test ./...
```

3. Запустить новый прогон только hidden wrapper-процессом:

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
"run timestamp: $runTimestamp"
"wrapper pid: $($process.Id)"
"stdout: $stdoutLog"
"stderr: $stderrLog"
```

Не запускать `go run` напрямую в активной консоли.

4. В ответе пользователю указать:
- `runTimestamp`
- PID wrapper-процесса
- stdout log
- stderr log

5. Для heartbeat/monitoring всегда использовать:

`.codex/skills/check-run-status.md`

6. Если status helper показывает:

```text
PAUSED_AFTER_SUCCESS
```

то вызвать:

`.codex/skills/cleanup-run-tails.md`

## Resume после validate dynamic list contracts

Если status helper показывает, что прогон упал после:

```text
xml step: validate dynamic list contracts
```

можно продолжить:

```powershell
go run .\\cmd\\changefiles\\main.go .\\configs\\config.json <path-to-output\\_tmp\\v8_src*> --resume-from-validation
```

Перед resume обязательно использовать:

`.codex/skills/check-run-status.md`

чтобы подтвердить:
- что temp dump относится к текущему запуску
- что dump является dump расширения
- что dump не является старым `v8_src*`

## Финальный ответ пользователю

После завершения прогона укажи:
- результат `go build ./...`
- результат `go test ./...`
- результат `go run`
- путь к свежему dump snapshot
- путь к `.cfe`, если он создан
- ключевую ошибку и файл, если прогон упал

## Запреты

- Не завершай процессы напрямую из orchestrator.
- Любое завершение процессов выполняется только через:
  `.codex/skills/cleanup-run-tails.md`
- Не делай широкий рефакторинг во время прогона.
- Не удаляй временные каталоги без необходимости.
- Не называй старый dump свежим.
- Не анализируй BSL как источник правил классификации.
- Не меняй бизнес-логику ради прохождения прогона без отдельной задачи.
