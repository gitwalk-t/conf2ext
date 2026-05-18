# Архитектура

## Верхнеуровневый поток

1. CLI или пример приложения загружает конфиг через `pkg/config`
2. `internal/config` объединяет встроенные значения по умолчанию с пользовательским JSON и нормализует пути
3. `internal/converter.RunConversion` выбирает `cfConvert` или `srcConvert`
4. Конвертер подготавливает временные каталоги и временную 1С ИБ
5. Метаданные выгружаются в файлы
6. `internal/utils/xmlutil.ChangeFiles` загружает persisted identity map для `Adopted-*`, применяет optional base bindings и переписывает выгрузку в XML, пригодный для загрузки как расширение
7. При `stop_after_xml_dump=true` конвейер останавливается после переписывания XML
8. Переписанные файлы загружаются как расширение и выгружаются в `.cfe`

## Границы модулей

- `cmd/`: CLI-оболочка вокруг загрузки конфига и запуска конвертации
- `pkg/`: стабильные публичные обертки
- `internal/config/`: схема конфига и логика объединения настроек
- `internal/converter/`: оркестрация, временный ввод-вывод, интеграция с 1С runner
- `internal/utils/xmlutil/`: классификация метаданных, нормализация XML, очистка, замена GUID
- `internal/export_format/`: встроенная карта версий для выгрузки исходников

## XML rewrite pipeline

`ChangeFiles` сейчас отвечает за:
- загрузку XML-контекстов
- определение владельца по пути/документу
- специальные случаи для корня и языка
- классификацию объектов с разделением на мягко исключенные и жестко исключенные
- сбор единого soft-excluded набора: ссылки из `excluded_subsystems` плюс единичные объекты из `excluded_objects`
- приоритетное soft-исключение top-level объектов, связанных с `excluded_subsystems` или явно перечисленных в `excluded_objects`, даже если имя объекта попадает в primary `Native` по native-prefix
- `RefDrivenInclusion` по XML-ссылкам из оставленных `Native`-объектов
- для восстановления soft-excluded объектов источниками `RefDrivenInclusion` считаем только `Native`-объекты и их формы; `Native`-подсистемы сами по себе для такого восстановления источником не являются
- особый случай для ненативных `DefinedType`, перенесенных как `AdoptedStubExt`: их состав тоже остается источником `RefDrivenInclusion`
- возврат мягко исключенных объектов в `Native`, если на них есть ссылка из допустимого `Native`
- первичный `Native` не демотируется в `Excluded` из-за мягких списков; для него действует только `forbidden_*`
- добавление новых зависимостей как `AdoptedStub` или `AdoptedStubExt`
- ограничение `DefinedType` и `EventSubscription` через `targetCompatibilitySet`: `Native`-объекты допустимы всегда, а adopted-объекты этих kind могут пережить `RefDrivenInclusion` только если существуют в `target.xml_dump`
- очистку ссылок на жестко исключенные объекты из `forbidden_*`
- нормализацию adopted stub
- замену GUID
- отдельный binding-pass для `base_object_id`: он переписывает ссылки на связанные Adopted-объекты по итоговым XML-документам, но не подменяет их собственные `uuid`
- очистку forbidden movements
- запись файлов и проверку отсутствия old GUID

## Этап валидации

После записи переписанных XML `ChangeFiles` может выполнять отдельный этап `validate dynamic list contracts`, если он включен конфигом.

Этот этап проверяет только form-driven contract dynamic list:
- target-объект не исключен итоговым `decision`
- target-объект не остался урезанным, если форма требует `AdoptedStubExt(Form)`
- для non-Native target-объекта в top-level metadata XML реально сохранены все поля, которые требует dynamic list

Для `Native` объектов field-level проверка не выполняется: валидатор здесь нужен для контроля `AdoptedStubExt`, а не для полного native-состава.

Источники полей для этой проверки:
- `StandardAttributes`
- `Attribute`
- `Dimension`
- `Resource`
- `Measure`

Для standard attributes используются alias-имена, если form-contract приходит по русскому имени поля, например:
- `Recorder -> Регистратор`
- `Period -> Период`
- `LineNumber -> НомерСтроки`
- `Active -> Активность`

Эта валидация не проверяет весь XML целиком и не заменяет загрузку в 1С. Если `enable_form_validation=false`, этап пропускается целиком.

Это самый плотный и рискованный слой системы.

## Терминология слоя XML

- `Native` — объект, оставляемый как нативный по правилам классификации
- в XML обычный `Native` не должен получать явный `<ObjectBelonging>Native</ObjectBelonging>`; отсутствие свойства означает дефолтный native-режим, а adopted-режимы записываются явно
- `AdoptedStub` — заимствованный объект в урезанном виде
- `AdoptedStubMetaData` — частный случай `AdoptedStub`, который включается через `AdditionalProcessing.Use_MetaDataFile`: top-level объект переносится как adopted, а child metadata с префиксом `упо_`, перечисленные в `CommonTemplate.упо_MetaDataFile` в множествах `Реквизиты` / `ТабличныеЧасти`, сохраняются как retained-состав
- У `AdoptedStubMetaData` retained child metadata остается в режиме `Native`: top-level объект остается `Adopted`, но сохраненные по макету `Attribute` / `TabularSection` и их child metadata не должны сохранять ни `ObjectBelonging=Adopted`, ни `ExtendedConfigurationObject`
- `AdoptedStubMetaData` не должен переигрывать явный soft-exclude: если top-level объект попал в единый `excluded`-набор из `excluded_subsystems` / `excluded_objects`, правило `упо_MetaDataFile` не возвращает его обратно в состав
- `AdoptedStubExt` — заимствованный объект без форм и кода, но с сохраненным реквизитным составом; бывает как `AdoptedStubExt(Form)` и как `AdoptedStubExt(DefinedType)`
- Для `AdoptedStub` не переносим тексты `ManagerModule` и `ObjectModule`; для adopted-констант по тому же правилу не переносим `ValueManagerModule`; для adopted общих модулей не переносим `Module`; для adopted команд не переносим `CommandModule`. Если top-level объект остается adopted, child-записи этих модулей нужно убирать и из `ConfigDumpInfo.xml`, а у `CommonModule` и команд еще и удалять соответствующие `Ext/*.bsl`
- у `AdoptedStubExt` не сериализуем `Properties/StandardAttributes`; стандартные поля остаются доступными платформенно и не должны считаться заимствованным составом
- `AdoptedStubExt(EventSubscription)` — ненативная подписка на событие в полном составе; она тоже остается источником `RefDrivenInclusion`
- формы не режутся частично: если форма осталась в составе, она должна оставаться целой; если форма не должна переноситься, ее не оставляем фрагментами
- `excluded` — он же мягко исключенный объект; объект, снятый с первичного включения и допускающий возврат по ссылкам, если не входит в `forbidden_*`
- единый `excluded`-набор сначала собирается из объектов, найденных по `excluded_subsystems`, и затем дополняется объектами из `excluded_objects`; дальше для всего этого набора используется одна и та же логика soft-exclude и `RefDrivenInclusion`
- мягко исключенный объект не должен возвращаться в `AdoptedStub`, если все его допустимые входящие ref-driven ссылки идут только из `Native`-подсистем; одна лишь подсистемная группировка не считается достаточным основанием возвращать такой объект в состав расширения
- `Native`-подсистема вообще не должна сама по себе восстанавливать excluded-объект по `RefDrivenInclusion`; для такого возврата нужен хотя бы один другой `Native`-источник, то есть реальный объект или его форма
- ссылки из `Role/Ext/Rights.xml` на excluded-объекты тоже не считаются источником `RefDrivenInclusion`; такие rights-записи должны очищаться из роли, а не возвращать объект в итоговый состав
- top-level объект, который связан с веткой из `excluded_subsystems` или явно указан в `excluded_objects`, должен исключаться до проверки primary `Native` по native-prefix; одна принадлежность имени к `упо_` не должна удерживать такой объект в составе
- `forbidden_*` — он же жестко исключенный объект; запрещает включение объекта в любом виде
- `RefDrivenInclusion` / `рефдривен` — включение по XML-ссылкам из уже оставленных `Native`-объектов; ненативные `DefinedType` и `EventSubscription` в статусе `AdoptedStubExt` тоже могут быть источником; BSL не участвует
- если `AccumulationRegister`/`InformationRegister`/другой регистр остается `Native`, его документы-registrators тоже дотягиваются в `Native`, чтобы в `RegisterRecords` не исчезала регистрация документа
- для `DefinedType` и `EventSubscription` общий cleanup мягко исключенных ссылок не применяется, их состав сохраняется целиком
- для `AdoptedStubExt(Form)` источником решения служит не только `MainTable`, но и field contract dynamic list: `DataPath`, `Field`, `RowPictureDataPath` и родственные поля. Этот контракт должен проверяться конвертером до загрузки в 1С.
- `extension_properties.name` / `extension_properties.prefix` — основной источник имени и префикса корня; старые `extension` / `prefix` поддерживаются как алиасы совместимости и тоже должны сходиться с `Configuration.xml` и `ConfigDumpInfo.xml`
- `extension_properties.identifier` — стабильный `uuid` корня `Configuration.xml`; он не должен генерироваться заново между прогонами
- если задан `target.xml_dump`, для `DefinedType` и `EventSubscription` применяется `targetCompatibilitySet`: `Native`-объекты расширения допустимы всегда, а adopted-объекты этих kind допускаются только если существуют в top-level наборе XML-выгрузки конфигурации-приемника
- у корня `Configuration.xml` обязаны быть `InternalInfo/xr:PropertyState`, `Caption`, `ShortCaption`, `Language.Русский`
- `ChildObjects` корня должны совпадать с итоговым top-level составом: top-level `Native` и `AdoptedStub`, не помеченные как excluded/forbidden
- при `normalizeAdoptedObjectComposition` сохраняем каркас формата: если у исходного top-level metadata был `ChildObjects`, после очистки он должен остаться хотя бы пустым
- у корневого `Ext` не должно быть `ParentConfigurations*`
- `Ext/CommandInterface.xml` должен быть командным зеркалом `Ext/MainSectionCommandInterface.xml`; подсистемные секции пишутся в `MainSectionCommandInterface`, а не в `CommandInterface`
- для `Subsystem` нельзя вырезать `Content`, если подсистема переносится в расширение
- для `Subsystem` сам контейнер `Content` сохраняется, но `xr:MDObjectRef` внутри него должны совпадать с реально оставленным top-level составом; ссылки на отсутствующие metadata нужно удалять
- для `Subsystem` в adopted-режиме нельзя терять верхний `ChildObjects`: он нужен для состава подсистем и должен доживать до записи
- для `Subsystem` содержимое `ChildObjects` тоже должно нормализоваться по итоговому составу: контейнер остается, но ссылки на дочерние подсистемы без итогового XML должны удаляться, включая ветки из `excluded_subsystems`
- `Role/Ext/Rights.xml` должен очищаться от `object/name = Configuration.<oldConfigName>`; права на исходную конфигурацию не являются частью состава расширения
- `Role/Ext/Rights.xml` также должен очищаться от ссылок на дочерние metadata, которых уже нет в итоговом XML top-level объекта; иначе Конфигуратор падает на `Неизвестный объект метаданных`
- `CommandInterface.xml` и `MainSectionCommandInterface.xml` должны очищаться от `Command name=...`, если соответствующий metadata-path команды уже не существует в итоговом XML; иначе загрузка падает на `Неизвестная команда`
- `Form.xml` должен очищаться от элементов интерфейса формы, если их `<Command>` ссылается на metadata-command, которого уже нет в итоговом XML; иначе загрузка падает на `Неверное имя команды элемента формы`
- `ConfigDumpInfo.xml` должен очищаться не только от вложенных `Metadata`, но и от плоских top-level child-записей вида `Catalog.X.Command.Y.*`, если top-level объект уже non-`Native` и этот child-path не сохранен в итоговом составе

В документации стоит сохранять именно эти термины, потому что они совпадают с текущей моделью в коде и обсуждениях.

## Инварианты

- Финальный результат всегда `.cfe`
- `cfConvert` на практике требует явную версию платформы
- Временная работа идет под `output/_tmp`, если output указывает на `.cfe`
- Основное место для отладочных и исследовательских логов — `output/_log`, но в корне репозитория еще есть старые `run-*.log`
- `--config` имеет приоритет над `CONFIG_PATH`
- Внешний код не должен импортировать `internal/*`
- HTTP/service слоя в проекте нет
