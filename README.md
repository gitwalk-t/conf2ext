# files-converter

Go CLI-инструмент для конвертации исходников конфигурации 1С или файла `*.cf` в пакет расширения `*.cfe`. В этой рабочей копии основной активный сценарий — `srcConvert`: взять готовое дерево XML-исходников, переписать XML по правилам расширения и собрать `*.cfe`.

## Запуск

Основная CLI-точка входа:

```powershell
go run . --config .\configs\config.json
```

Упрощенный пример без паузы на TTY, с локальными CLI-флагами `-c` / `--config` и `--check-config`:

```powershell
go run .\cmd\app
```

Сборка:

```powershell
go build ./...
```

Тесты:

```powershell
go test ./...
```

Примечания:
- в [go.mod](D:\Codex\files-converter_ver2\go.mod) зафиксирован Go `1.23` и `toolchain go1.23.12`
- для реальной конвертации нужна локально доступная платформа 1С нужной версии
- если `--config` не передан, CLI по умолчанию берет `./configs/config.json`
- `input_path` и `output_path` в конфиге можно задавать абсолютными путями или путями от корня проекта; для checked-in конфига используется краткая форма вроде `/input` и `/output/demo.cfe`
- имя, префикс и стабильный идентификатор расширения теперь живут в `extension_properties`; старые `extension` / `prefix` остаются как backward-compatible alias

## Основные модули

- [main.go](D:\Codex\files-converter_ver2\main.go): корневой исполняемый файл
- [cmd/root.go](D:\Codex\files-converter_ver2\cmd\root.go): Cobra CLI, загрузка конфига, пауза в интерактивном терминале
- [cmd/app/main.go](D:\Codex\files-converter_ver2\cmd\app\main.go): минимальный пример запуска без интерактивной паузы
- [pkg/config/config.go](D:\Codex\files-converter_ver2\pkg\config\config.go): публичная обертка загрузки конфига
- [pkg/converter/converter.go](D:\Codex\files-converter_ver2\pkg\converter\converter.go): публичная обертка запуска конвертации
- [internal/converter/converter.go](D:\Codex\files-converter_ver2\internal\converter\converter.go): оркестрация конвертации, временные каталоги, вызовы 1С
- [internal/config](D:\Codex\files-converter_ver2\internal\config): схема конфига, значения по умолчанию, нормализация путей, объединение настроек
- [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go): движок переписывания XML и текущая горячая точка
- [configs/config.json](D:\Codex\files-converter_ver2\configs\config.json): текущий рабочий локальный конфиг

## Обязательный стартовый контекст агента

1. [AGENTS.md](D:\Codex\files-converter_ver2\AGENTS.md)
2. [.codex/context.md](D:\Codex\files-converter_ver2\.codex\context.md)
3. [.codex/handoff.md](D:\Codex\files-converter_ver2\.codex\handoff.md)
4. [docs/debugging.md](D:\Codex\files-converter_ver2\docs\debugging.md)

## Дополнительные документы

- [docs/architecture.md](D:\Codex\files-converter_ver2\docs\architecture.md): полезен для архитектурных задач и понимания границ модулей
- [docs/conventions.md](D:\Codex\files-converter_ver2\docs\conventions.md): полезен для терминологии и точечного выравнивания формулировок
- [docs/technical-spec.md](D:\Codex\files-converter_ver2\docs\technical-spec.md): дополнительная пользовательская спецификация; полезна для синхронизации требований, архитектурных задач и онбординга человека, но не обязательна как стартовый агентский контекст

## Переменные окружения

- `CONFIG_PATH`: запасной путь к конфигу, если `--config` не передан
- `FILES_CONVERTER_KEEP_TMP=1`: не удалять временную ИБ и выгрузку в `output/_tmp`
- `FILES_CONVERTER_SNAPSHOT_BEFORE_CHANGE=1`: сохранить снимок XML-выгрузки до `ChangeFiles`
- `FILES_CONVERTER_DEBUG_DECISIONS=1`: включить выборочный `decision debug` в XML-логике

## Полезный флаг конфига

- `keep_xml_dump`: если `true`, временная XML-выгрузка и временная ИБ не очищаются после прогона
- при `keep_xml_dump=true` копия XML-дампа сохраняется в `output/_log/xml_dumps/<v8_src*>`
- `stop_after_xml_dump`: если `true`, программа останавливается после переписывания XML и не переходит к сборке `.cfe`
- `enable_form_validation`: если `true`, после переписывания XML выполняется проверка form-driven dynamic list contracts; если `false`, этот этап пропускается
- `AdditionalProcessing.Use_MetaDataFile`: если `true`, `ChangeFiles` читает общий макет `упо_MetaDataFile` и добавляет специальные adopted-режимы для перечисленных там объектов: `AdoptedStubMetaData` для retained child metadata и `AdoptedStubExtMetaData` для merge-объектов `DefinedType` / `ExchangePlan` / `EventSubscription`, которые сохраняют `Type` / `Content` / `Source` без форм и BSL
- `AdditionalProcessing.Use_упо_SearchResult`: включает дополнительный overlay `AdoptedStubCode` для adopted-части состава; он читает `searchingTemplateText.json` рядом с активным конфигом и `CommonTemplates/упо_SearchResult/Ext/Template.txt`, может поднять default-excluded объект в `AdoptedStub`, сохранить нужные формы/команды/modules и наложить текст модулей, не переводя объект в `Native`
- при `AdditionalProcessing.Use_упо_SearchResult=true` конвертер дополнительно проверяет, что объекты, реально выбранные из `упо_SearchResult`, дошли до итогового XML-дампа как top-level adopted-объекты
- `AdditionalProcessing.UseExactTemplates`: управляет строгостью сопоставления шаблонов для `Use_упо_SearchResult`; по умолчанию `true`, а mismatch в strict-режиме пишется в `output/_log/searchresult-template-errors.log` и переводит перенос кода на общий fallback без падения прогона
- `extension_properties.identifier`: стабильный `uuid` корня расширения; он не генерируется заново и записывается в `Configuration.xml`
- `target.xml_dump`: optional XML-выгрузка конфигурации-приемника; для `DefinedType`, `EventSubscription`, `ExchangePlan` сначала формируется `targetCompatibilitySet`: `Native`-объекты этих типов допустимы всегда, а adopted-объекты допустимы только если их top-level XML реально существует в target; lightweight-коллектор читает только `DefinedTypes/*.xml`, `EventSubscriptions/*.xml`, `ExchangePlans/*.xml`
- после обычной classification/promotion-логики `target.xml_dump` все еще используется только для post-promotion merge объектов из `CommonTemplate.упо_MetaDataFile` (`DefinedType`, `ExchangePlan`, `EventSubscription`), которые в этом сценарии идут как `AdoptedStubExtMetaData`; target-ссылки из их сохраненного `Type` / `Content` / `Source` могут дотягивать отсутствующий top-level metadata-объект из target как обычный `AdoptedStub`, но merge не должен переигрывать `targetCompatibilitySet`
- если adopted top-level объект в source совпадает с target по top-level id, но имя в `target.xml_dump` уже переименовано, приоритетным считается имя из target: в extension должен остаться один adopted-объект под target-именем, а stale source-name не должен дублировать тот же identity bundle
- `included_Native_objects` — это явный `Native` override над soft-exclude из `excluded_subsystems` / `excluded_objects`, но не над `forbidden_*`; при этом обычный native-prefix по-прежнему слабее soft-exclude, а ненативная подсистема попадает в состав только если служит предком реально вложенной native-подсистемы

## Identity mapping

- `output/_state/identity-map.json`: persisted runtime state только для `Adopted-*` metadata-path; хранит стабильные `extension_id` между генерациями
- `configs/base-bindings.json`: пользовательские override bindings для `base_object_id`; файл может быть частично заполнен и не влияет на `Native`

## Что важно про текущее состояние репозитория

- HTTP/API слоя в проекте нет
- CI и линтер-конфига в репозитории нет
- `go test ./...` включает точечные unit-тесты для `cmd/app` и `internal/utils/xmlutil`, но полноценного CI и широкого integration/golden-покрытия в репозитории все еще нет
- основное место для локальных отладочных логов — `output/_log`; в корне еще лежат старые `run-*.log`
