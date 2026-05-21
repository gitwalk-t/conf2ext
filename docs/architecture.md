# Архитектура

Документ описывает границы модулей и верхнеуровневый поток. Детальные XML/classification rules живут в `.codex/context/xml-rules.md`, терминология — в `.codex/context/terms.md`.

## Верхнеуровневый поток

1. CLI или пример приложения загружает конфиг через `pkg/config`.
2. `internal/config` объединяет defaults с пользовательским JSON и нормализует пути.
3. `internal/converter.RunConversion` выбирает `cfConvert` или `srcConvert`.
4. Конвертер подготавливает временные каталоги и временную 1С ИБ.
5. Метаданные выгружаются в XML-файлы.
6. `internal/utils/xmlutil.ChangeFiles` переписывает XML-выгрузку в extension-compatible dump.
7. При `stop_after_xml_dump=true` pipeline останавливается после XML rewrite.
8. Переписанные файлы загружаются как extension и выгружаются в `.cfe`.

## Границы модулей

- `cmd/` — CLI-оболочка вокруг загрузки конфига и запуска конвертации.
- `pkg/` — стабильные публичные wrappers.
- `internal/config/` — схема конфига, defaults, merge, path normalization, backward compatibility.
- `internal/converter/` — orchestration, temp I/O, интеграция с 1С runner.
- `internal/utils/xmlutil/` — XML loading, metadata owner detection, classification, cleanup, GUID rewrite.
- `internal/export_format/` — встроенная карта export format -> platform version.
- `internal/utils/fileutil/` — копирование дерева файлов для `srcConvert`.

## XML rewrite boundary

`ChangeFiles` — самый плотный и рискованный слой системы.

Он отвечает за:
- загрузку XML-контекстов;
- определение owner metadata;
- special cases для root/language;
- top-level classification;
- `RefDrivenInclusion`;
- adopted normalization;
- target-sensitive merge cases;
- cleanup служебных XML-ссылок;
- GUID/base binding rewrite;
- запись файлов и проверки.

Детальные правила этого слоя не дублируются здесь. Читать:

```text
.codex/context/xml-rules.md
```

## Validation boundary

`validate dynamic list contracts` — отдельный этап после записи XML и до финального cleanup/verification.

Назначение:
- поймать ошибки form-driven dynamic list до загрузки в 1С;
- проверить, что non-Native target object не остался слишком урезанным;
- не выполнять полный platform-level validation.

Детали field contract и aliases описаны в `.codex/context/xml-rules.md`.

## Публичная поверхность

Внешний код должен использовать только:

- `pkg/config`
- `pkg/converter`

`internal/*` не является публичным API.

## Инварианты архитектуры

- Финальный результат — `.cfe`.
- `srcConvert` работает с готовым деревом XML-исходников.
- `cfConvert` требует платформу 1С и практическую версию платформы.
- Временная работа идет под `output/_tmp`, если output указывает на `.cfe`.
- Основное место логов — `output/_log`.
- `--config` имеет приоритет над `CONFIG_PATH`.
- HTTP/service слоя в проекте нет.
- Полноценного CI пока нет; базовые проверки — `go build ./...` и `go test ./...`.

## Что читать дальше

- XML/classification details: `.codex/context/xml-rules.md`
- Термины: `.codex/context/terms.md`
- Debugging cookbook: `docs/debugging.md`
- Run orchestration: `.codex/skills/run-conversion.md`
