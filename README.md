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
- `AdditionalProcessing.Use_MetaDataFile`: если `true`, `ChangeFiles` читает общий макет `упо_MetaDataFile` и добавляет режим `AdoptedStubMetaData` для перечисленных там объектов
- `AdditionalProcessing.Use_упо_SearchResult`: зарезервированный флаг; пока не влияет на поведение

## Identity mapping

- `output/_state/identity-map.json`: persisted runtime state только для `Adopted-*` metadata-path; хранит стабильные `extension_id` между генерациями
- `configs/base-bindings.json`: пользовательские override bindings для `base_object_id`; файл может быть частично заполнен и не влияет на `Native`

## Что важно про текущее состояние репозитория

- HTTP/API слоя в проекте нет
- CI и линтер-конфига в репозитории нет
- `go test ./...` включает точечные unit-тесты для `cmd/app` и `internal/utils/xmlutil`, но полноценного CI и широкого integration/golden-покрытия в репозитории все еще нет
- основное место для локальных отладочных логов — `output/_log`; в корне еще лежат старые `run-*.log`
