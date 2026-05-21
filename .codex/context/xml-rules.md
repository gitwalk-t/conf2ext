# XML/classification rules

Этот файл — источник долгоживущих правил XML-переписывания и классификации. Для словаря терминов см. `.codex/context/terms.md`.

## Базовая модель

- `ChangeFiles` переписывает XML-выгрузку в extension-compatible dump.
- Решения по `ObjectBelonging` формируются в XML pipeline, а не делегируются платформе.
- BSL не анализируется и не является источником classification/ref rules.
- Для обычного `Native` не сериализуется явный `<ObjectBelonging>Native</ObjectBelonging>`: это default mode.
- Adopted-режимы сериализуются явно.
- При замене GUID нельзя менять `ClassId`.
- Корень конфигурации и `Language.Русский` обрабатываются отдельно.

## Приоритеты top-level classification

Порядок смысловых приоритетов:

1. `forbidden_*` / hard exclude — сильнее всего.
2. `included_Native_objects` — explicit native override над soft exclude.
3. `excluded_subsystems` + `excluded_objects` — единый soft-excluded набор.
4. обычный `Native` по native-prefix.
5. ref-driven adopted/native promotion.

Правила:
- `excluded_objects` и `excluded_subsystems` — мягкие исключения.
- `forbidden_AdoptedStub_objects` — жесткое исключение.
- Top-level объект из `included_Native_objects` остается `Native`, даже если попал в soft-excluded набор, но `forbidden_*` сильнее.
- Объект из `excluded_subsystems` или `excluded_objects` должен стать excluded раньше обычного native-prefix.
- Единый `excluded` набор сначала собирается по `excluded_subsystems`, затем дополняется `excluded_objects`.

## RefDrivenInclusion

- `RefDrivenInclusion` работает по XML-ссылкам из уже оставленных допустимых `Native`-объектов.
- Форменные XML участвуют только если владелец формы уже `Native`.
- Non-native формы не являются источником ссылок и не порождают `AdoptedStubExt(Form)`.
- `Native`-подсистема сама по себе не восстанавливает excluded-объект.
- Ссылки из `Role/Ext/Rights.xml` на excluded-объекты не восстанавливают эти объекты; такие rights-записи чистятся.
- Мягко исключенный объект не возвращается в состав, если все допустимые входящие ссылки идут только из `Native`-подсистем.
- `DefinedType` и `EventSubscription` в режиме `AdoptedStubExt` могут сохранять состав как источник ссылок.
- Если регистр остается `Native`, его документы-registrators дотягиваются в `Native`, чтобы не потерять `RegisterRecords`.

## AdoptedStub и adopted-cleanup

- На текущем этапе внешняя модель — `AdoptedStub`; полноценного отдельного business-mode `Adopted` нет.
- Для `AdoptedStub` не переносить тексты:
  - `ManagerModule`
  - `ObjectModule`
  - `ValueManagerModule` у констант
  - `Module` у общих модулей
  - `CommandModule` у команд
- Child-записи этих модулей удаляются из `ConfigDumpInfo.xml`; физические `Ext/*.bsl` для common modules и команд тоже удаляются.
- Для top-level adopted metadata нельзя удалять обязательный форматный каркас: если исходный объект имел `ChildObjects`, контейнер должен остаться.
- Для non-native top-level metadata формы должны исчезать полностью: XML-файлы форм, `Default*Form` / `Auxiliary*Form` и дочерние `Metadata` форм в `ConfigDumpInfo.xml`.
- Child metadata (`Attribute`, `Dimension`, `Resource`, `Measure`) удаляется целиком, если его `Properties/Type` ссылается на excluded-объект.

## AdoptedStubExt и dynamic list contract

- `AdoptedStubExt` — частный случай `AdoptedStub` с сохраненным metadata composition, но без форм и BSL.
- Для `AdoptedStubExt(Form)` реквизитная часть определяется по полному dynamic list contract, а не только по `MainTable`.
- Учитывать `DataPath`, `Field`, `RowPictureDataPath` и родственные ссылки формы.
- `validate dynamic list contracts` запускается после записи XML и перед `verify old GUID` / cleanup.
- Валидация проверяет только form-driven dynamic list для non-Native target objects.
- Для `Native` field-level проверка не применяется.
- Поле считается доступным через `StandardAttributes`, `Attribute`, `Dimension`, `Resource`, `Measure`.
- Для standard attributes используются alias-имена, минимум:
  - `Recorder -> Регистратор`
  - `Period -> Период`
  - `LineNumber -> НомерСтроки`
  - `Active -> Активность`

## MetaDataFile и target.xml_dump

- `AdditionalProcessing.Use_MetaDataFile=true` добавляет `AdoptedStubMetaData` из `CommonTemplate.упо_MetaDataFile`.
- Для `AdoptedStubMetaData` top-level объект adopted, retained child metadata ограничен реквизитами/табличными частями с префиксом `упо_`, перечисленными в макете.
- Retained child metadata остаются `Native`: без `ObjectBelonging=Adopted` и без `ExtendedConfigurationObject`.
- `AdoptedStubMetaData` не переигрывает explicit soft exclude.
- Для merge-объектов `DefinedType`, `ExchangePlan`, `EventSubscription` из `CommonTemplate.упо_MetaDataFile` используется `AdoptedStubExtMetaData`.
- `AdoptedStubExtMetaData` сохраняет соответственно `Properties/Type`, `Ext/Content.xml`, `Properties/Source`, но не переносит формы и BSL.
- Если задан `target.xml_dump`, сначала собирается `targetCompatibilitySet` для `DefinedType`, `EventSubscription`, `ExchangePlan`.
- `targetCompatibilitySet`: `Native`-объекты этих типов допустимы всегда; adopted-объекты допустимы только при наличии top-level XML в target.
- Target merge не превращает `target.xml_dump` в глобальный source graph.
- Если source и target совпадают по top-level `id/uuid`, canonical name берется из target; stale source-name не должен создавать дубль.

## Root, subsystem, role, command interface

- `Configuration.xml` должен иметь `InternalInfo/xr:PropertyState`, `Caption`, `ShortCaption`, `Language.Русский`.
- `Configuration.xml/ChildObjects` должен перечислять весь фактический top-level состав extension, включая `AdoptedStub`.
- У корневого `Ext` не должно быть `ParentConfigurations*`.
- `Ext/CommandInterface.xml` должен повторять командную структуру `Ext/MainSectionCommandInterface.xml`; подсистемные секции в него не пишутся.
- Для `Subsystem` нельзя вырезать `Content`, если подсистема переносится.
- У `Subsystem` контейнер `Content` сохраняется, но ссылки внутри чистятся по фактическому top-level составу.
- Для `Subsystem` adopted-нормализация не должна удалять верхний `ChildObjects`.
- Ненативная top-level `Subsystem` может попасть в extension только как adopted-предок реально вложенной native-подсистемы.
- `Role/Ext/Rights.xml` чистится от прав на `Configuration.<oldConfigName>` и отсутствующие metadata/child metadata paths.
- `CommandInterface.xml`, `MainSectionCommandInterface.xml` и `Form.xml` чистятся от ссылок на отсутствующие metadata commands.

## Type-bearing XML

- Metadata-ссылки внутри type-bearing XML считаются обычными XML-ссылками.
- Учитывать `Properties/Type/v8:Type`, `Ext/Predefined.xml`, namespace aliases вроде `d4p1:CatalogRef.X`.
- Нельзя переписывать корректные current-config aliases вроде `d4p1:` в `cfg:`.
- Для `ChartOfCharacteristicTypes/Ext/Predefined.xml` нужно сохранять согласованность qualifier-блоков с owner `Properties/Type`.

## Resume и проверки

- Resume после падения на `validate dynamic list contracts` не выполняет повторную проверку old GUID: old->new карта уже недоступна.
- Меняя classification или cleanup, проверять не только XML, но и последующую загрузку расширения в 1С.
- Не превращать adopted-объекты в новый источник `RefDrivenInclusion` без явного правила и записи в документации.
