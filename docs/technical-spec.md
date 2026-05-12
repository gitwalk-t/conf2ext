# Техническое задание

## Назначение

`files-converter` — это CLI-инструмент на Go для конвертации конфигурации 1С в пакет расширения `*.cfe`.

В текущей рабочей копии основной сценарий такой:

1. взять исходную конфигурацию из XML-выгрузки или `*.cf`
2. переписать XML по правилам расширения
3. получить XML-дамп расширения
4. загрузить его в 1С как расширение
5. выгрузить итоговый `*.cfe`

Главная практическая цель проекта сейчас: добиться стабильной сборки расширения из активного локального конфига [D:\Codex\files-converter_ver2\configs\config.json](</D:/Codex/files-converter_ver2/configs/config.json>) без ошибок загрузки.

## Область применения

Инструмент должен поддерживать два режима входа:

- `cfConvert`: загрузить `*.cf` во временную ИБ, выгрузить XML, переписать XML, собрать расширение
- `srcConvert`: взять уже готовое дерево XML-исходников, переписать XML, собрать расширение

В текущем checked-in конфиге активен локальный сценарий `srcConvert`.

## Точка входа

Основной запуск:

```powershell
go run . --config .\configs\config.json
```

Базовые проверки перед завершением любой правки:

```powershell
go test ./...
go build ./...
```

## Основные требования

### 1. Загрузка и нормализация конфига

Система должна:

- загружать встроенные defaults и пользовательский JSON
- поддерживать обратную совместимость по устаревшим алиасам полей конфига
- нормализовать пути до абсолютных
- использовать `./configs/config.json`, если `--config` не передан

Ключевой рабочий конфиг:
- [D:\Codex\files-converter_ver2\configs\config.json](</D:/Codex/files-converter_ver2/configs/config.json>)

### 2. Классификация метаданных

XML-конвейер должен классифицировать top-level объекты в терминах:

- `Native`
- `AdoptedStub`
- `AdoptedStubExt(Form)`
- `AdoptedStubExt(DefinedType)`
- `AdoptedStubExt(EventSubscription)`
- `AdoptedStubMetaData`
- `excluded` как мягкое исключение
- `forbidden` как жесткое исключение

Обязательные инварианты:

- обычный `Native` не сериализуется как явный `<ObjectBelonging>Native</ObjectBelonging>`
- `excluded_subsystems` и `excluded_objects` образуют единый soft-excluded набор
- `forbidden_*` всегда сильнее soft-include и `RefDrivenInclusion`
- `Native`-подсистемы и `Role/Ext/Rights.xml` не должны сами по себе восстанавливать excluded-объекты
- объекты из `included_Native_objects` должны обрабатываться так же, как prefix-native

### 3. RefDrivenInclusion

После первичной классификации конвейер должен уметь:

- строить XML reference graph
- возвращать допустимые soft-excluded объекты по ссылкам из сохраненного `Native`
- расширять состав до `AdoptedStub` или `AdoptedStubExt`, если этого требуют формы, типы или служебные XML

При этом:

- источником считаются только допустимые XML-ссылки
- BSL не участвует в принятии решений
- формы являются источником только для `Native`-владельцев

### 4. Переписывание XML

Этап `ChangeFiles` должен:

- нормализовать `Configuration.xml`, `ConfigDumpInfo.xml`, `Ext/*`
- согласованно выставлять имя расширения и префикс из `extension` / `prefix`
- перестраивать `Configuration.xml/ChildObjects` по итоговым `decisions`
- чистить служебные XML от ссылок на отсутствующие metadata-path
- сохранять обязательный структурный каркас XML, даже если состав объекта минимизирован
- выполнять cleanup ссылок на excluded/forbidden metadata
- заменять GUID по правилам adopted-состава

Особенно важно:

- не терять контейнер `ChildObjects`, если формат его требует
- не ломать `Subsystem/Content` и состав дочерних подсистем
- не оставлять в `ConfigDumpInfo.xml`, формах, правах и command interface ссылки на уже удаленные metadata-path

### 5. Работа с adopted-объектами

Для adopted-режимов действуют отдельные ограничения:

- формы non-native top-level объектов не должны переноситься частично
- у `AdoptedStub` не переносим тексты:
  - `ManagerModule`
  - `ObjectModule`
  - `ValueManagerModule` у констант
  - `Module` у общих модулей
  - `CommandModule` у команд
- retained child metadata у `AdoptedStubMetaData` должны оставаться в native-режиме
- child-команды adopted-владельца могут сохраняться как metadata-path без переноса `CommandModule`, если на них есть живая XML-ссылка

### 6. Поддержка form-driven dynamic list

Конвейер должен:

- собирать полный field contract dynamic list, а не только `MainTable`
- учитывать `DataPath`, `Field`, `RowPictureDataPath` и другие field-bearing ссылки формы
- валидировать retained-поля для non-native target-объектов
- поддерживать alias стандартных атрибутов (`Ref`, `Recorder`, `Period`, `LineNumber`, `Active` и т.д.)
- чистить invalid form bindings, если они ссылаются на отсутствующие metadata-path

Отдельные уже зафиксированные требования:

- для manual-query dynamic list без `MainTable` не оставлять невалидные standard row commands
- удалять form-controls и declared fields, если они ссылаются на отсутствующий `CommonAttribute.X`, а owner metadata такого поля не содержит
- удалять form-elements с висячими metadata-command ссылками

### 7. Поддержка type-bearing XML

Нужно корректно обрабатывать:

- `Properties/Type`
- `DefinedType`
- `ChartsOfCharacteristicTypes/.../Ext/Predefined.xml`

Обязательные требования:

- cleanup metadata-ссылок должен работать независимо от namespace alias
- current-config alias вроде `d4p1:` нельзя агрессивно переписывать в `cfg:`
- qualifier-блоки (`StringQualifiers`, `NumberQualifiers`, `DateQualifiers`, `BinaryDataQualifiers`) должны быть согласованы между owner type-set и predefined item

### 8. Специальная обработка через макеты

Сейчас в рабочем коде активна только дополнительная обработка:

- `AdditionalProcessing.Use_MetaDataFile`

Она должна:

- читать общий макет `упо_MetaDataFile`
- переводить перечисленные объекты в режим `AdoptedStubMetaData`
- сохранять только разрешенный retained child-состав с префиксом `упо_`
- не возвращать объект в состав, если он уже soft-excluded

`AdditionalProcessing.Use_упо_SearchResult` в текущем репозитории остается зарезервированным флагом без рабочего поведения.

## Нефункциональные требования

### 1. Минимальный дифф

При изменениях в коде нужно:

- предпочитать локальные правки
- не делать широкий рефакторинг, особенно в [D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go](</D:/Codex/files-converter_ver2/internal/utils/xmlutil/change.go>)
- не менять бизнес-правила ради косметики

### 2. Надежность расследований

Инструмент должен быть пригоден для долгой итеративной отладки:

- логи и диагностические артефакты живут в `output/_log`
- временные XML и ИБ — в `output/_tmp`
- при `keep_xml_dump=true` копия дампа сохраняется в `output/_log/xml_dumps/<v8_src*>`

### 3. Совместимость

Нельзя ломать:

- старые алиасы полей конфига
- уже зафиксированные XML-правила
- текущий словарь терминов (`Native`, `AdoptedStub`, `excluded`, `forbidden`, `RefDrivenInclusion`)

## Ограничения

- проект не анализирует BSL как источник правил
- CI в репозитории нет
- автоматическая проверка в 1С не формализована, финальная валидация по-прежнему опирается на локальный прогон
- основное узкое место и главный риск проекта — [D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go](</D:/Codex/files-converter_ver2/internal/utils/xmlutil/change.go>)

## Источники истины

Если требования расходятся, в первую очередь ориентироваться на:

1. [D:\Codex\files-converter_ver2\configs\config.json](</D:/Codex/files-converter_ver2/configs/config.json>)
2. [D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go](</D:/Codex/files-converter_ver2/internal/utils/xmlutil/change.go>)
3. [D:\Codex\files-converter_ver2\.codex\decisions.md](</D:/Codex/files-converter_ver2/.codex/decisions.md>)
4. [D:\Codex\files-converter_ver2\docs\debugging.md](</D:/Codex/files-converter_ver2/docs/debugging.md>)

## Связанные документы

- [D:\Codex\files-converter_ver2\README.md](</D:/Codex/files-converter_ver2/README.md>)
- [D:\Codex\files-converter_ver2\docs\architecture.md](</D:/Codex/files-converter_ver2/docs/architecture.md>)
- [D:\Codex\files-converter_ver2\docs\conventions.md](</D:/Codex/files-converter_ver2/docs/conventions.md>)
- [D:\Codex\files-converter_ver2\docs\debugging.md](</D:/Codex/files-converter_ver2/docs/debugging.md>)
- [D:\Codex\files-converter_ver2\.codex\context.md](</D:/Codex/files-converter_ver2/.codex/context.md>)
- [D:\Codex\files-converter_ver2\.codex\decisions.md](</D:/Codex/files-converter_ver2/.codex/decisions.md>)
- [D:\Codex\files-converter_ver2\.codex\tasks.md](</D:/Codex/files-converter_ver2/.codex/tasks.md>)
