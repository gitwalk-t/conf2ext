# Минимальный стартовый контекст

## Назначение проекта

`conf2ext` — Go CLI для преобразования конфигурации 1С в расширение `.cfe` через переписывание XML-выгрузки.

Главный рискованный слой системы:

`internal/utils/xmlutil/change.go`

## Основной pipeline

1. Загрузка конфига
2. Получение XML-выгрузки (`cfConvert` или `srcConvert`)
3. `ChangeFiles` переписывает XML
4. XML загружается как extension
5. Формируется `.cfe`

## Главные каталоги

- `internal/utils/xmlutil/` — XML/classification logic
- `internal/converter/` — orchestration
- `internal/config/` — config merge и normalization
- `configs/config.json` — активный локальный конфиг
- `output/_log` — логи
- `output/_tmp` — временные XML-дампы
- `.codex/skills/` — run/debug helpers

## Базовые команды

```powershell
go build ./...
go test ./...
go run . --config .\\configs\\config.json
```

## Основные инварианты

- `excluded_*` — soft exclude
- `forbidden_*` — hard exclude
- BSL не анализируется
- формы не режутся частично
- `Native` и `AdoptedStub` — основные технические режимы
- `included_Native_objects` сильнее soft-exclude, но слабее `forbidden_*`

## Что читать дальше

- Архитектура: `docs/architecture.md`
- XML/classification rules: `.codex/context/xml-rules.md`
- Термины: `.codex/context/terms.md`
- Debugging: `docs/debugging.md`
- Запуск прогона: `.codex/skills/run-conversion.md`
- Текущий статус расследования: `.codex/handoff.md`

## Что не делать без необходимости

- Не читать весь исторический handoff для обычной правки.
- Не делать широкий рефакторинг `change.go`.
- Не смешивать правила XML и временные debugging-заметки в одном документе.
