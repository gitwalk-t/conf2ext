# conf2ext

Go CLI-инструмент для конвертации конфигурации 1С в расширение `.cfe` через переписывание XML-выгрузки.

## Основной сценарий

`srcConvert`:
- взять готовое дерево XML-исходников;
- переписать XML по правилам extension;
- собрать `.cfe`.

## Быстрый старт

Запуск:

```powershell
go run . --config .\\configs\\config.json
```

Минимальный запуск:

```powershell
go run .\\cmd\\app
```

Сборка и тесты:

```powershell
go build ./...
go test ./...
```

## Архитектура

- `cmd/` — CLI
- `pkg/` — публичные wrapper API
- `internal/config/` — config merge и normalization
- `internal/converter/` — orchestration
- `internal/utils/xmlutil/` — XML rewrite и classification logic
- `internal/export_format/` — mapping export format -> platform version

Ключевой файл XML rewrite:

```text
internal/utils/xmlutil/change.go
```

## Основные каталоги

- `configs/config.json` — активный локальный конфиг
- `output/_log` — логи
- `output/_tmp` — временные XML-выгрузки
- `docs/` — human-facing документация
- `.codex/` — агентский контекст и skills

## Документация

Архитектура:
- `docs/architecture.md`
- `docs/technical-spec.md`

Debugging:
- `docs/debugging.md`

Агентский bootstrap:
- `AGENTS.md`

## Инварианты

- `excluded_*` — soft exclude
- `forbidden_*` — hard exclude
- BSL не анализируется как источник classification rules
- формы не режутся частично
- `Native` и `AdoptedStub` — основные режимы обработки

## Переменные окружения

- `CONFIG_PATH`
- `FILES_CONVERTER_KEEP_TMP=1`
- `FILES_CONVERTER_SNAPSHOT_BEFORE_CHANGE=1`
- `FILES_CONVERTER_DEBUG_DECISIONS=1`

## Ограничения проекта

- HTTP/API слоя нет
- полноценного CI пока нет
- базовая проверка:

```powershell
go build ./...
go test ./...
```
