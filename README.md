# conf2ext

Go CLI-инструмент для конвертации конфигурации 1С в расширение `.cfe` через переписывание XML-выгрузки.

Основной активный сценарий:
- `srcConvert` — взять готовое дерево XML-исходников, переписать XML по правилам extension и собрать `.cfe`.

## Запуск

Основная точка входа:

```powershell
go run . --config .\\configs\\config.json
```

Минимальный запуск:

```powershell
go run .\\cmd\\app
```

Сборка:

```powershell
go build ./...
```

Тесты:

```powershell
go test ./...
```

## Основные модули

- `cmd/` — CLI
- `pkg/` — публичные wrapper API
- `internal/config/` — config merge и normalization
- `internal/converter/` — orchestration
- `internal/utils/xmlutil/` — XML rewrite и classification logic
- `internal/export_format/` — mapping export format -> platform version

Главный рискованный файл:

```text
internal/utils/xmlutil/change.go
```

## Основные каталоги

- `configs/config.json` — активный локальный конфиг
- `output/_log` — логи
- `output/_tmp` — временные XML-выгрузки
- `.codex/skills/` — run/debug helpers
- `.codex/context/` — agent context

## Документация

### Для архитектуры

- `docs/architecture.md`
- `docs/technical-spec.md`

### Для debugging

- `docs/debugging.md`

### Для агентского контекста

Стартовая точка:

- `.codex/index.md`

Минимальный startup-context:

- `.codex/agent-start.md`

XML/classification rules:

- `.codex/context/xml-rules.md`

Терминология:

- `.codex/context/terms.md`

## Основные инварианты

- `excluded_*` — soft exclude
- `forbidden_*` — hard exclude
- BSL не анализируется
- формы не режутся частично
- `Native` и `AdoptedStub` — основные technical modes

## Переменные окружения

- `CONFIG_PATH` — fallback path к конфигу
- `FILES_CONVERTER_KEEP_TMP=1` — не удалять временные каталоги
- `FILES_CONVERTER_SNAPSHOT_BEFORE_CHANGE=1` — сохранить snapshot до XML rewrite
- `FILES_CONVERTER_DEBUG_DECISIONS=1` — decision debug для XML logic

## Что важно про репозиторий

- HTTP/API слоя нет
- полноценного CI пока нет
- основные проверки:

```powershell
go build ./...
go test ./...
```
