# Техническое задание

Дополнительная пользовательская спецификация: полезна для синхронизации требований, архитектурных задач и онбординга человека, но не является обязательным стартовым контекстом агента.

## Назначение

`files-converter` — это CLI-инструмент на Go для конвертации конфигурации 1С в пакет расширения `*.cfe`.

В текущей рабочей копии основной сценарий такой:

1. взять исходную конфигурацию из XML-выгрузки или `*.cf`
2. переписать XML по правилам расширения
3. получить XML-дамп расширения
4. загрузить его в 1С как расширение
5. выгрузить итоговый `*.cfe`

Главная практическая цель проекта сейчас: добиться стабильной сборки расширения из активного локального конфига [`../configs/config.json`](../configs/config.json) без ошибок загрузки.

## Область применения

Инструмент должен поддерживать два режима входа:

- `cfConvert`: загрузить `*.cf` во временную ИБ, выгрузить XML, переписать XML, собрать расширение
- `srcConvert`: взять уже готовое дерево XML-исходников, переписать XML, собрать расширение

В текущем checked-in конфиге активен локальный сценарий `srcConvert`.

## Точка входа

Основной запуск:

```powershell
go run . --config .\\configs\\config.json
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
- [`../configs/config.json`](../configs/config.json)

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

### 7. Поддержка type-bearing XML

Нужно корректно обрабатывать:

- `Properties/Type`
- `DefinedType`
- `ChartsOfCharacteristicTypes/.../Ext/Predefined.xml`

### 8. Специальная обработка через макеты

Сейчас в рабочем коде активна только дополнительная обработка:

- `AdditionalProcessing.Use_MetaDataFile`

`AdditionalProcessing.Use_упо_SearchResult` в текущем репозитории остается зарезервированным флагом без рабочего поведения.

## 9. Persisted identity mapping для Adopted-объектов

Для обновляемых расширений система должна сохранять идентификаторы только Adopted-объектов между повторными генерациями.

Persisted state (сохранённое состояние) — это состояние, которое сохраняется между повторными запусками генератора.

### Область применения

Persisted identity mapping применяется только к:

- `AdoptedStub`
- `AdoptedStubExt(Form)`
- `AdoptedStubExt(DefinedType)`
- `AdoptedStubExt(EventSubscription)`
- `AdoptedStubMetaData`

`Native`-объекты не участвуют в identity mapping и всегда сохраняют исходные идентификаторы конфигурации.

### Persisted state

Система должна поддерживать файл сохранённого состояния:

```text
output/_state/identity-map.json
```

Само наличие файла `identity-map.json` является флагом использования сохраненных идентификаторов.

Поведение:

- если `identity-map.json` существует, генератор обязан переиспользовать сохраненные `extension_id`;
- если файла нет, генератор работает в режиме первичного формирования identity mapping и создает новый state.

Формат:

```json
{
  "version": 1,
  "objects": {
    "Catalog.Пользователи": {
      "extension_id": "..."
    }
  }
}
```

Где:

- key — metadata-path;
- `extension_id` — стабильный идентификатор объекта расширения.
