# Skill: статус прогона

## Когда использовать

Используй этот helper:
- для heartbeat monitoring
- если пользователь просит проверить статус прогона
- если нужно понять, жив ли текущий запуск
- если нужно подтвердить свежий dump
- перед resume-from-validation

## Назначение

Skill отвечает только за проверку состояния текущего прогона.

Skill не завершает процессы.
Skill не запускает новый прогон.
Skill не делает cleanup.

## Что проверять

Проверять:
- `output/_log/last_run.txt`
- stdout log текущего запуска
- stderr log текущего запуска
- `run-<timestamp>.pid`
- наличие wrapper `powershell`
- наличие дочернего `go`
- наличие связанных `1cv8.exe`
- свежие `output/_tmp/v8_src*`
- свежие `output/_log/xml_dumps/v8_src*`
- `Configuration.xml`

Все эти пути проверяются внутри текущего worktree / текущего `cwd`, для которого был запущен прогон. Если в системе есть несколько worktree репозитория, нельзя смешивать их `output/_log`, `output/_tmp` и `configs`.

## Как определить текущий запуск

Текущий запуск определяется только по совокупности:
- timestamp запуска
- PID wrapper-процесса
- stdout/stderr log path
- `CommandLine` с текущим `--config`
- parent-child process chain
- свежие temp/dump каталоги

Дополнительно:
- `output/_log/last_run.txt` не является достаточным доказательством текущего запуска
- `last_run.txt` может остаться от старого wrapper-процесса и отставать от реального нового запуска
- если `run-<timestamp>.pid`, stdout/stderr и process-chain указывают на новый запуск, а `last_run.txt` еще старый, источником истины считается новый `runTimestamp`, а не stale `last_run.txt`

## Как определить свежий dump

Каталог `v8_src*` сам по себе не доказывает, что это dump расширения.

Проверить `Configuration.xml`:
- `ObjectBelonging=Adopted`
- имя/префикс соответствуют `extension_properties`
- dump не является исходной конфигурацией

## Возможные статусы

### RUNNING

Прогон ещё выполняется.

### FAILED

Прогон завершился ошибкой.

Вывести:
- ключевую ошибку
- файл
- последнюю стадию

### SUCCESS

Прогон завершился успешно.

### PAUSED_AFTER_SUCCESS

`ChangeFiles` завершился успешно, dump snapshot сохранён, но wrapper `powershell` ещё висит на:

```text
Press any key to exit...
```

В этом случае orchestrator должен вызвать:

`.codex/skills/cleanup-run-tails.md`

## Формат status summary

Всегда выводить:
- status
- run timestamp
- wrapper PID
- stdout log
- stderr log
- текущую стадию
- свежий dump snapshot
- подтвержден ли dump как dump расширения
- есть ли активный `go`
- есть ли связанные `1cv8.exe`
- next action

## Правило остановки heartbeat

Если мониторинг запущен как heartbeat и статус **не** `RUNNING` (например, `SUCCESS`, `FAILED`, `PAUSED_AFTER_SUCCESS`, или прогона нет), heartbeat нужно **остановить**:
- удалить heartbeat (предпочтительно), либо поставить на `PAUSED`
- в сообщении пользователю явно указать, что heartbeat остановлен, чтобы не было ощущения “фон продолжает что-то делать”
