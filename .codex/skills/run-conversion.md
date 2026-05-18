# Skill: запуск прогона

## Когда использовать

Используй этот skill, когда пользователь просит:
- запустить прогон
- проверить сборку расширения
- выполнить `go run`
- получить свежий дамп
- проверить ошибку загрузки после изменений

## Обязательный контекст

Перед запуском прочитай:
- `README.md`
- `AGENTS.md`
- `.codex/context.md`
- `.codex/handoff.md`
- `docs/debugging.md`

Активный конфиг по умолчанию:

`configs/config.json`

## Перед запуском

1. Зафиксируй в ответе пользователю timestamp начала запуска.
2. Останови только процессы, относящиеся к текущему проекту:
   - старые `go`
   - связанные с ними `1cv8.exe`
3. Не трогай чужие фоновые `1cv8.exe`.
4. Текущий прогон отличай по совокупности признаков:
   - timestamp запуска
   - PID wrapper-процесса
   - пути stdout/stderr логов
   - более поздний `StartTime`
   - `CommandLine` с текущим `--config`
   - связь с новым `go run`

## Базовый цикл

Всегда используй только фоновый запуск без всплывающего окна:

```powershell
go build ./...
go test ./...

$runTimestamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$logDir = Join-Path (Get-Location) 'output\_log'
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
        'go run . --config .\configs\config.json'
    )

$process.Id | Set-Content -Encoding UTF8 $pidFile
"run timestamp: $runTimestamp"
"wrapper pid: $($process.Id)"
"stdout: $stdoutLog"
"stderr: $stderrLog"
```

Не запускать `go run` напрямую в активной консоли. В ответе пользователю укажи `runTimestamp`, PID wrapper-процесса и пути stdout/stderr логов.

## Мониторинг долгого прогона

После старта долгого прогона сразу включи проверку состояния раз в 10 минут.

Проверять:
- `output/_log/last_run.txt`
- stdout/stderr текущего запуска по путям `run-<timestamp>.stdout.log` и `run-<timestamp>.stderr.log`
- PID wrapper-процесса из `run-<timestamp>.pid`
- свежие каталоги `output/_tmp/v8_src*`
- свежие дампы `output/_log/xml_dumps/v8_src*`
- наличие текущих `go` / `1cv8.exe`

Текущий запуск определяется только по совокупности:
- timestamp запуска
- PID wrapper-процесса
- stdout/stderr log path
- `CommandLine` с текущим `--config`
- свежие `output/_tmp/v8_src*` и `output/_log/xml_dumps/v8_src*`

Запрещено:
- автоматически перезапускать прогон
- трогать чужие `1cv8.exe`
- считать старый `v8_src*` результатом нового запуска без проверки времени и `Configuration.xml`

## Автозавершение зависшего процесса после успешного ChangeFiles

Во время мониторинга, если wrapper `powershell` / дочерний `go run` ещё жив, но фактически висит только на паузе `Press any key to exit...`, его нужно завершить без отдельного подтверждения пользователя.

Завершать можно только если одновременно выполнены все условия:

- stdout текущего запуска содержит `Press any key to exit...`
- лог показывает, что `ChangeFiles` завершился успешно
- прогресс объектов дошёл до конца, например:
  `apply object changes N/N file=done`
- свежий XML dump snapshot уже скопирован в:
  `output/_log/xml_dumps/v8_src*`
- процесс относится именно к текущему запуску:
  совпадают timestamp, PID wrapper-процесса, log path и `CommandLine`

После завершения:
- снять heartbeat/monitoring automation для этого прогона
- сообщить пользователю, что процесс был остановлен как штатно зависший на паузе
- указать подтверждённый путь к свежему дампу

## Как определить свежий дамп

Каталог `v8_src*` сам по себе не доказывает, что это дамп расширения.

Перед тем как назвать путь пользователю, проверь `Configuration.xml`:
- корень должен иметь `ObjectBelonging=Adopted`
- имя/префикс должны соответствовать `extension_properties`
- это не должна быть исходная конфигурация с пустым `NamePrefix`

## Если прогон упал

Сначала смотри:
1. stderr текущего запуска
2. `output/_log`
3. соответствующий `output/_tmp/v8_src*`
4. свежий дамп в `output/_log/xml_dumps`

Типовые направления диагностики:

### Неверный путь к данным

Проверять:
- формы
- dynamic list
- `DataPath`
- `Field`
- `CommonAttribute`

### Неизвестный объект метаданных

Проверять:
- `Role/Ext/Rights.xml`
- `Subsystem/Content`
- `ConfigDumpInfo.xml`
- root `Ext/*.xml`

### XDTO/type errors

Проверять:
- `Properties/Type`
- `Ext/Predefined.xml`
- namespace alias
- qualifiers

## Resume после validate dynamic list contracts

Если прогон уже дошёл до:

```text
xml step: validate dynamic list contracts
```

и упал там, можно продолжить на текущем temp-дампе.

Перед resume обязательно проверь, что temp-дамп относится к тому же прогону:

- путь свежий относительно timestamp текущего запуска
- `Configuration.xml` подтверждает, что это дамп расширения:
  - `ObjectBelonging=Adopted`
  - имя/префикс соответствуют `extension_properties` / активному конфигу
- это не старый `v8_src*`, выбранный только потому, что он существует

```powershell
go run .\cmd\changefiles\main.go .\configs\config.json <path-to-output\_tmp\v8_src*> --resume-from-validation
```

Resume-режим:
- не переписывает XML заново
- повторяет расчет decisions и form-contract
- запускает validation/final cleanup
- не повторяет проверку old GUID

## После успешного завершения

Если wrapper `powershell` / дочерний `go run` ещё жив, stdout текущего запуска содержит:

```text
Press any key to exit...
```

и одновременно подтверждено, что `ChangeFiles` завершился успешно, свежий dump snapshot уже сохранён, а процесс однозначно относится к текущему запуску, заверши именно этот wrapper/дочерний процесс автоматически.

В финальном ответе укажи:
- результат `go build ./...`
- результат `go test ./...`
- результат `go run`
- путь к свежему дампу
- путь к `.cfe`, если он создан
- ключевую ошибку и файл, если прогон упал

## Запреты

- Не делай широкий рефакторинг во время запуска прогона.
- Не удаляй временные каталоги без необходимости.
- Не называй старый дамп свежим.
- Не анализируй BSL как источник правил классификации.
- Не меняй бизнес-логику ради прохождения прогона без отдельной задачи.
- Не завершай зависший `powershell` / `go run`, если нет подтверждения, что `ChangeFiles` дошёл до конца и dump snapshot уже сохранён.
