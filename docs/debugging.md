# Отладка

Документ содержит troubleshooting cookbook и debugging-паттерны.

Run orchestration и monitoring вынесены в:

- `.codex/skills/run-conversion.md`
- `.codex/skills/check-run-status.md`
- `.codex/skills/cleanup-run-tails.md`

Детальные XML/classification rules вынесены в:

```text
.codex/context/xml-rules.md
```

## Базовый debugging cycle

```powershell
go build ./...
go test ./...
go run . --config .\\configs\\config.json
```

Если прогон упал:

1. Смотреть stderr-log текущего запуска.
2. Смотреть соответствующий `output/_tmp/v8_src*`.
3. После правки повторять цикл.

## Resume после validate dynamic list contracts

Если прогон упал после:

```text
xml step: validate dynamic list contracts
```

можно продолжить:

```powershell
go run .\\cmd\\changefiles\\main.go .\\configs\\config.json <path-to-v8_src*> --resume-from-validation
```

Resume-mode:
- не переписывает XML заново;
- повторяет validation/final cleanup;
- не повторяет old GUID verification.

## Как понимать dump

Каталог `v8_src*` сам по себе не гарантирует dump extension.

Проверять `Configuration.xml`:
- `ObjectBelonging=Adopted`;
- extension name/prefix соответствуют конфигу;
- dump не является исходной конфигурацией.

## Типовые ошибки

### Неверный путь к данным

Признаки:
- ошибка `DataPath`;
- ошибка dynamic list;
- ошибка field contract.

В первую очередь смотреть:
- diff `input/.../Form.xml` vs `output/_log/xml_dumps/.../Form.xml`;
- form-cleanup/form-normalization в `change.go`;
- присутствует ли объект в итоговом top-level составе.

Частые причины:
- агрессивное переписывание `DataPath`;
- удаленный `CommonAttribute`;
- excluded-объект удержался в составе;
- форма ссылается на metadata-path, которого больше нет.

Детальные правила form-driven cleanup:

```text
.codex/context/xml-rules.md
```

### Неизвестный объект метаданных

Обычно:
- top-level объект уже вырезан правильно;
- в служебном XML осталась висячая ссылка.

В первую очередь смотреть:
- `Role/Ext/Rights.xml`;
- `Subsystem/.../Content`;
- `ConfigDumpInfo.xml`;
- root `Ext/*.xml`;
- metadata-path ссылки в `name`, `Metadata`, `DataPath`, `Field`, `Command`, `object`.

Для owner-command отдельно проверять файловый каркас `Commands/<Имя>`.

### Ошибка XDTO / QName / TypeSet

Обычно проблема в type-bearing XML:
- `Properties/Type`;
- `Ext/Predefined.xml`;
- namespace alias;
- qualifier synchronization.

В первую очередь смотреть:
- diff `input` vs `dump`;
- aliases вроде `d4p1:`;
- cleanup/normalization type-bearing XML.

Не переписывать корректные current-config aliases.

### Excluded-объект остался в extension

Обычно проблема в порядке promotion/ref-driven восстановления.

В первую очередь смотреть:
- `excluded_subsystems`;
- `excluded_objects`;
- `forbidden_*`;
- `Role/Ext/Rights.xml`;
- `Subsystem/Content`;
- порядок classification/promote logic.

Основные XML/classification rules вынесены в:

```text
.codex/context/xml-rules.md
```

## Полезные env vars

- `FILES_CONVERTER_KEEP_TMP=1`
- `FILES_CONVERTER_SNAPSHOT_BEFORE_CHANGE=1`
- `FILES_CONVERTER_DEBUG_DECISIONS=1`
- `CONFIG_PATH`

## Полезные config flags

- `keep_xml_dump=true`
- `stop_after_xml_dump=true`
- `enable_form_validation=false`

## Источники истины

Код:
- `internal/utils/xmlutil/change.go`
- `configs/config.json`

Рабочие артефакты:
- `output/_tmp`
- `output/_log`
