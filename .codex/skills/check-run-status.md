# Skill: статус прогона

## Когда использовать

Используй этот helper:
- для heartbeat monitoring
- если пользователь просит проверить статус прогона
- если нужно понять, жив ли текущий запуск
- если нужно подтвердить свежий dump
- перед resume-from-validation

## Назначение

Skill отвечает за проверку и финализацию состояния текущего прогона.

Skill не запускает новый прогон.

Cleanup не является основной задачей skill, кроме special-case `PAUSED_AFTER_SUCCESS`.

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

## Как определить текущий запуск

Текущий запуск определяется по совокупности:
- timestamp
- PID
- stdout/stderr
- `CommandLine`
- process chain
- свежие temp/dump каталоги

`last_run.txt` сам по себе не является источником истины.

## Как определить свежий dump

Проверить `Configuration.xml`:
- `ObjectBelonging=Adopted`
- имя/префикс соответствуют `extension_properties`
- dump не является исходной конфигурацией

## Возможные статусы

### RUNNING

Прогон еще выполняется.

### FAILED

Прогон завершился ошибкой.

### SUCCESS

Прогон завершился успешно.

### PAUSED_AFTER_SUCCESS

Wrapper завис только на:

```text
Press any key to exit...
```

В этом состоянии cleanup обязателен.

Skill должен выполнить:

```text
.codex/skills/cleanup-run-tails.md
```

Это не считается отдельным orchestration flow.

Cleanup является частью финализации успешного прогона:
- dump уже подтвержден как корректный;
- `ChangeFiles` завершился успешно;
- остались только зависшие wrapper/process tails.

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

## Финальный формат ответа пользователю

Если проверка выполняется как часть полного прогона или финального отчета, дополнительно выводить:
- результат `go build ./...`
- результат `go test ./...`
- результат `go run`
- путь к свежему dump snapshot
- путь к `.cfe`, если он создан
- ключевую ошибку и файл, если прогон упал

## Правило остановки heartbeat

Если статус не `RUNNING`:
- heartbeat нужно остановить
- удалить heartbeat или перевести его в paused
- явно сообщить пользователю, что monitoring остановлен
