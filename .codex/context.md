# Контекст

## Устройство системы

- CLI загружает итоговый конфиг: встроенные значения по умолчанию плюс пользовательский JSON.
- Путь конвертации выбирается по `conversion_type`:
  - `cfConvert`: загрузить `.cf` во временную ИБ, выгрузить метаданные в файлы, переписать XML, загрузить расширение из файлов, выгрузить `.cfe`
  - `srcConvert`: скопировать дерево исходников, определить версию платформы по формату выгрузки, переписать XML, загрузить расширение, выгрузить `.cfe`
- Публичная поверхность для встраивания специально маленькая:
  - [pkg/config/config.go](D:\Codex\files-converter_ver2\pkg\config\config.go)
  - [pkg/converter/converter.go](D:\Codex\files-converter_ver2\pkg\converter\converter.go)

## Главные модули

- `internal/config`:
  схема, значения по умолчанию, логика объединения, нормализация путей, совместимость со старыми полями
- `internal/converter`:
  оркестрация временных директорий, вызовов runner и формирования выходного пути
- `internal/utils/xmlutil`:
  загрузка XML, определение владельца метаданных, классификация (`Native` / `AdoptedStub` / excluded), очистка, замена GUID
- `internal/export_format`:
  встроенная карта `версия формата выгрузки -> версия платформы`
- `internal/utils/fileutil`:
  копирование дерева файлов для `srcConvert`

## Инварианты

- `configs/config.json` — практический локальный конфиг для запуска в этой рабочей копии
- `keep_xml_dump=true` в конфиге удерживает временную XML-выгрузку и временную ИБ после прогона
- `keep_xml_dump=true` еще и сохраняет копию XML-дампа в `output/_log/xml_dumps/<v8_src*>`
- `stop_after_xml_dump=true` останавливает прогон после `ChangeFiles`, до загрузки расширения и выгрузки `.cfe`
- `Ext/CommandInterface.xml` должен повторять командную структуру `Ext/MainSectionCommandInterface.xml`; подсистемные секции в него не пишем
- если `ChangeFiles` уже дошел до `validate dynamic list contracts` и упал, отладку можно продолжать на текущем `v8_src*` через `go run .\cmd\changefiles\main.go <config> <xml-dir> --resume-from-validation`; этот режим не переписывает XML заново, а повторяет только validation/final cleanup
- новый прогон сборки сначала начинается с остановки текущих `go` и связанных `1cv8.exe`, чтобы не путать старые процессы с новым запуском
- при запуске долгого прогона нужно сразу ставить 10-минутную автоматизацию проверки состояния по образцу `Files-converter run check (silent go, 10m)` / `check-run`: читать `output/_log/last_run.txt`, связанные stdout/stderr, свежие `output/_tmp` и `output/_log/xml_dumps`, не перезапускать прогон и не трогать чужие `1cv8.exe`
- если пользователь просит "дамп", это нужно трактовать как XML-дамп расширения после `ChangeFiles`; перед ответом нужно проверить `Configuration.xml` и убедиться, что корень уже переписан как расширение, а не остался исходной конфигурацией
- `Artefacts/` — папка примеров для агента; если пользователь пишет "пример в артефактах", сначала ищи файл там
- если пример в `Artefacts/` имеет расширение `.cfe`, его сначала нужно превратить в набор XML-файлов, а уже потом использовать как образец
- `--config` имеет приоритет, а `CONFIG_PATH` работает как запасной источник пути к конфигу
- `input_path` и `output_path` поддерживают абсолютные пути и project-root relative shorthand вида `/input`; для конфигов из `configs/` такой путь привязывается к корню репозитория, а затем нормализуется в абсолютный
- текущий `1cv8.exe` отличай от старых по `StartTime`, `CommandLine` и связи с новым `go run`; старые окна 1С не считай активным прогоном
- решения по `ObjectBelonging` формируются в конвейере переписывания XML, а не “доделываются” платформой
- для обычного `Native` не сериализуем явный `<ObjectBelonging>Native</ObjectBelonging>`: в XML это дефолтный режим; явное свойство нужно для adopted-режимов
- объекты из `included_Native_objects` после сборки `primaryNativeObjects` должны участвовать в тех же helper-ветках, что и prefix-native; нельзя опираться только на `hasNativePrefix(...)`, если рядом уже есть полный `primaryNativeObjects`
- resume-режим после падения на `validate dynamic list contracts` не выполняет повторную проверку old GUID: на уже переписанном temp исходная old->new карта замен больше недоступна
- на текущем этапе внешняя модель — это `AdoptedStub`; отдельного полноценного бизнес-режима `Adopted` нет
- для `AdoptedStub` не переносим тексты `ManagerModule` и `ObjectModule`; для adopted-констант по тому же правилу не переносим `ValueManagerModule`; для adopted общих модулей не переносим `Module`; для adopted команд не переносим `CommandModule`. Даже если соответствующие child-узлы еще видны в исходном metadata XML до adopted-cleanup, `ConfigDumpInfo.xml` не должен оставлять эти child-записи в итоговом составе, а у `CommonModule` и команд нужно еще удалять физические `Ext/*.bsl`
- в отдельных правилах может использоваться `AdoptedStubExt`: это все еще `AdoptedStub`, но с сохраненным реквизитным составом и без форм/кода; для `DefinedType` и `EventSubscription` это полный состав, а для `Form` — только реквизитная часть
- `AdditionalProcessing.Use_MetaDataFile=true` добавляет еще один частный случай `AdoptedStub`: `AdoptedStubMetaData`. Он читается из `CommonTemplate.упо_MetaDataFile`: top-level объект попадает в adopted-режим, а child metadata сохраняется только для реквизитов с префиксом `упо_`, перечисленных в `Реквизиты` / `ТабличныеЧасти`
- У `AdoptedStubMetaData` retained child metadata должно оставаться `Native`: top-level объект adopted, но сохраненные по макету реквизиты и табличные части не должны сохранять ни `ObjectBelonging=Adopted`, ни `ExtendedConfigurationObject`
- `AdoptedStubMetaData` не переигрывает явный soft-exclude: объект из объединенного `excluded`-набора не должен возвращаться только потому, что он перечислен в `CommonTemplate.упо_MetaDataFile`
- для `AdoptedStubExt(Form)` реквизитная часть определяется по полному contract dynamic list, а не только по `MainTable`: учитываются `DataPath`, `Field`, `RowPictureDataPath` и родственные ссылки формы
- для `AdoptedStubExt` не сериализуем `Properties/StandardAttributes`; стандартные поля (`Ссылка`, `Наименование`, `Код`, `Родитель` и т.п.) считаем встроенными, а не заимствованными
- `excluded_objects` и `excluded_subsystems` - мягкие исключения; `forbidden_AdoptedStub_objects` - жесткое исключение
- Единый набор `excluded` сначала собирается из объектов, найденных по `excluded_subsystems`, а затем дополняется объектами из `excluded_objects`; дальше для всего набора используется единая логика soft-exclude и `RefDrivenInclusion`
- Top-level объект, который связан с веткой из `excluded_subsystems` или явно перечислен в `excluded_objects`, должен попадать в `Excluded` раньше primary `Native` по native-prefix
- После такого soft-исключения объект не должен возвращаться в состав, если все его допустимые входящие ref-driven ссылки идут только из `Native`-подсистем
- Если мягко исключенный объект не входит в первичный `Native`, ref-driven возврат в `AdoptedStub` не должен срабатывать от одних только `Native`-подсистем; для такого возврата нужен хотя бы один другой допустимый источник ссылки
- Для восстановления excluded-объекта `Native`-подсистема сама по себе не считается источником `RefDrivenInclusion`; источниками могут быть только `Native`-объекты и их формы
- Ссылки на excluded-объекты в `Role/Ext/Rights.xml` тоже не считаются источником `RefDrivenInclusion`; rights-записи на такие metadata-path нужно вычищать из роли
- BSL-модули нигде не анализируются; работа идет только по XML-метаданным и XML-ссылкам
- для `DefinedType` и `EventSubscription` мягко исключенные ссылки не режутся общим cleanup; их состав сохраняется целиком и дальше живет через `RefDrivenInclusion`
- при замене GUID нельзя менять `ClassId`
- корень конфигурации и `Language.Русский` обрабатываются отдельно
- нужен Go 1.23.x

## Важные ограничения

- [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go) большой и сильно завязан на текущее состояние; локальные правки безопаснее “улучшений структуры”
- зафиксированных проверок от регрессий для XML-логики пока нет
- CI нет, поэтому реальные проверки — это локальные build/test, ручной прогон конвертации и просмотр логов в `output/_log`
- Для расследований полезны `FILES_CONVERTER_KEEP_TMP`, `FILES_CONVERTER_SNAPSHOT_BEFORE_CHANGE` и `FILES_CONVERTER_DEBUG_DECISIONS`

## Термины, которые надо использовать последовательно

- `Native` и `AdoptedStub` — это именно технические режимы, их не стоит каждый раз переводить по-разному
- `AdoptedStubExt` — отдельный полезный термин для частного случая `AdoptedStub`; различаем `AdoptedStubExt(Form)`, `AdoptedStubExt(DefinedType)` и `AdoptedStubExt(EventSubscription)`
- `AdoptedStubMetaData` — отдельный термин для adopted-объектов, которые приходят из `CommonTemplate.упо_MetaDataFile`; их top-level объект остается adopted, retained child metadata ограничен реквизитами с префиксом `упо_`, а сами retained child metadata должны оставаться в native-режиме без `ObjectBelonging=Adopted` и без `ExtendedConfigurationObject`
- “исключенный объект” / `excluded` / “мягко исключенный объект” - объект, снятый с первичного включения; это не всегда окончательный запрет
- “жестко исключенный объект” / `forbidden` - объект из `forbidden_AdoptedStub_objects`
- `RefDrivenInclusion` / `рефдривен` — включение по XML-ссылкам из уже оставленных `Native`-объектов; `AdoptedStubExt(Form)` нужен для динамических списков в формах и вообще для form-linked target objects, а источник этого режима всегда опосредованно `Native`-объект через его форму; `AdoptedStubExt(DefinedType)` и `AdoptedStubExt(EventSubscription)` — для ненативных объектов, которые сами остаются источником ссылок; по ходу может вернуть мягко исключенный объект в `Native` или создать `AdoptedStub` / `AdoptedStubExt`; формы не режем частично: если форма остается в составе, она должна оставаться целой, если не переносится, она не оставляется по кускам
- `RefDrivenInclusion` / `рефдривен` — включение по XML-ссылкам из уже оставленных `Native`-объектов; `AdoptedStubExt(Form)` нужен для динамических списков в формах и вообще для form-linked target objects, а источник этого режима всегда опосредованно `Native`-объект через его форму; `AdoptedStubExt(DefinedType)` и `AdoptedStubExt(EventSubscription)` — для ненативных объектов, которые сами остаются источником ссылок; по ходу может вернуть мягко исключенный объект в `Native` или создать `AdoptedStub` / `AdoptedStubExt`; при этом сама `Native`-подсистема не должна восстанавливать excluded-объект, она считается только группировочным слоем; формы не режем частично: если форма остается в составе, она должна оставаться целой, если не переносится, она не оставляется по кускам
- Форменные XML участвуют в `RefDrivenInclusion` только если владелец формы уже классифицирован как `Native`. Non-native формы могут временно лежать в `_tmp` до удаления, но не являются источником ссылок, не участвуют в dynamic-list contract и не могут порождать `AdoptedStubExt(Form)`.
- Ссылки `style:Имя` внутри `Native`-форм нужно трактовать как ссылки на `StyleItem.Имя`, но только если такой `StyleItem` реально существует в исходном XML-составе. Это отдельный канал form-driven `RefDrivenInclusion`; встроенные style-константы в метаданные не поднимаем.
- Для owner-level операций по `OwnerKey` основной контекст объекта — это его top-level metadata XML. Дочерние XML (`Ext/*`, формы, модули) могут жить рядом с тем же `OwnerKey`, но не должны подменять основной XML в валидации реквизитов и состава.
- В `dynamic list contract` поле считается доступным, если оно есть не только среди `StandardAttributes` и `Attribute`, но и среди field-bearing child objects (`Dimension`, `Resource`, `Measure`) в top-level metadata XML.
- `validate dynamic list contracts` запускается после записи XML и перед `verify old GUID` и cleanup. Этот этап проверяет только form-driven contract dynamic list: объект не исключен, не остался урезанным и реально содержит нужные поля в top-level metadata XML.
  Для `Native` field-level проверка не применяется; она нужна для `AdoptedStubExt`.
- Для standard attributes в `dynamic list contract` используются alias-имена, если форма ссылается на русское имя поля. Текущий минимум: `Recorder -> Регистратор`, `Period -> Период`, `LineNumber -> НомерСтроки`, `Active -> Активность`.
- если регистр остается `Native`, его документы-registrators тоже дотягиваются в `Native`, иначе у документа исчезает `RegisterRecords` и конфигуратор теряет регистрацию
- у `AdoptedStubExt(Form)` есть отдельный инвариант: если `Native`-форма использует dynamic list по ненативному объекту, этот объект обязан пройти валидацию field contract внутри конвертера; не допускаем ситуацию, когда ошибка впервые проявляется уже в Конфигураторе
- `extension_properties` — основной источник имени, префикса и стабильного `identifier` корня; старые `extension` / `prefix` остаются alias совместимости. Имя и префикс должны совпадать одновременно в `Configuration.xml` и `ConfigDumpInfo.xml`, а `identifier` — в `Configuration/@uuid`
- если задан `target.xml_dump`, для `DefinedType` и `EventSubscription` нужно строить `targetCompatibilitySet`: `Native`-объекты расширения допустимы всегда, а adopted-объекты этих kind не должны переживать `RefDrivenInclusion`, если их нет в top-level наборе конфигурации-приемника
- для этой проверки `target.xml_dump` читается не целиком: достаточно каталогов `DefinedTypes/` и `EventSubscriptions/` с их top-level `*.xml`
- у корня `Configuration.xml` обязаны быть `InternalInfo/xr:PropertyState`, `Caption`, `ShortCaption`, `Language.Русский`
- `ChildObjects` корня должны перечислять весь фактический top-level состав расширения, а не только объекты с native-prefix; иначе объект может остаться файлом и записью в `ConfigDumpInfo.xml`, но исчезнуть из состава расширения
- отдельные стандартные объекты тоже могут быть обязательны для загрузки, даже без `native_prefix`; текущий пример — `CommandGroup.Информация`, который нужно удерживать как `AdoptedStub`, если он используется командами
- для top-level adopted metadata нельзя удалять обязательный каркас формата; если исходный объект имел `ChildObjects`, этот контейнер должен остаться даже после очистки состава
- для non-native top-level metadata формы должны исчезать полностью: нельзя оставлять ни XML-файлы форм, ни `Default*Form` / `Auxiliary*Form` в `Properties`, ни дочерние `Metadata` форм в `ConfigDumpInfo.xml`
- у корневого `Ext` не должно быть `ParentConfigurations*`
- для `Subsystem` нельзя вырезать `Content`, если подсистема переносится в расширение
- у `Subsystem` контейнер `Content` должен сохраняться, но ссылки `xr:MDObjectRef` внутри него нужно чистить по фактическому top-level составу; если объект уже не вошел в итоговый дамп, ссылка из `Content` должна исчезнуть
- для `Subsystem` adopted-нормализация не должна срезать верхний `ChildObjects`; иначе Конфигуратор перестает читать состав подсистемы
- для `Subsystem` список имен внутри `ChildObjects` должен совпадать с реально оставленными дочерними подсистемами; нельзя держать там вложенные `Subsystem`, если их XML уже не входит в итоговый состав, включая исключенные ветки внутри native-подсистем
- дочерняя подсистема нативной подсистемы тоже остается `Native`, если цепочка не попала под `excluded_subsystems`; это не должно зависеть от списка special adopted-root
- для вложенных `Subsystem` owner-level ключ должен строиться по полной цепочке, а не по последнему имени; иначе `decision`, поиск контекста и cleanup начинают смешивать одноименные подсистемы из разных веток
- в `Role/Ext/Rights.xml` права на `Configuration.<oldConfigName>` нужно удалять; это хвост исходной конфигурации, а не часть расширения
- в `Role/Ext/Rights.xml` права на дочерние metadata нужно сверять с реально сохраненным child-составом объекта; если `Attribute` / `TabularSection` / `Command` уже не существует в итоговом XML, ссылку из роли нужно удалять
- при проверке существования owner-command metadata-path нельзя опираться только на owner XML: часть команд живет только как `Commands/<Имя>/Ext/CommandModule.bsl` и запись в `ConfigDumpInfo.xml`, без отдельного `Command.xml`; такой путь считается существующим по файловому каркасу
- у adopted top-level объекта child-команду нужно сохранять как metadata-path, если на нее ссылается живой служебный XML (`FunctionalOption.Content`, форма `Native`-объекта и т.п.); при этом текст `CommandModule` по принятому правилу все равно не переносится
- retained child-команды для adopted top-level объекта нужно собирать уже после `RefDrivenInclusion` и других promotion-решений; иначе объект может быть еще `excluded` в момент расчета, и даже живая ссылка из `FunctionalOption.Content` или retained owner-формы не удержит `Command` в `ChildObjects`
- в `Ext/CommandInterface.xml` и `Ext/MainSectionCommandInterface.xml` ссылки `Command name=...` тоже нужно сверять с реально сохраненным metadata-path команды; висячие команды должны удаляться
- в `Form.xml` элементы интерфейса с `<Command>` на metadata-command тоже нужно сверять с реально сохраненным child-path; если команда удалена из adopted/non-native объекта, элемент формы удаляется целиком
- в `Form.xml` для dynamic list нужно убирать поля и связанные controls, если они ссылаются на `CommonAttribute.X`, которого уже нет в итоговом составе, а owner metadata такого поля не содержит; иначе загрузка падает на `Неверный путь к данным`
- в `ConfigDumpInfo.xml` для non-`Native` top-level объекта нужно чистить не только nested `Metadata`, но и плоские top-level child-записи по удаленным child-path
- в child metadata top-level объекта нужно удалять whole child (`Attribute` / `Dimension` / `Resource` / `Measure`), если его `Properties/Type` ссылается на excluded-объект; иначе загрузка падает на `Неизвестное имя типа`
- metadata-ссылки внутри type-bearing XML тоже должны считаться обычными XML-ссылками: `Properties/Type/v8:Type` и `Ext/Predefined.xml` могут использовать current-config namespace alias вроде `d4p1:CatalogRef.X`, и такой слой нельзя оставлять вне cleanup/ref-graph
- у `ChartOfCharacteristicTypes/Ext/Predefined.xml` нужно отдельно сохранять согласованность owner-level qualifier-блоков (`StringQualifiers`, `NumberQualifiers`, `DateQualifiers`, `BinaryDataQualifiers`) с owner `Properties/Type`; это правило применяем и к matching item-типам без собственного scalar `xs:*`, при этом нельзя переписывать корректные current-config alias вроде `d4p1:` в `cfg:`
- если загрузка падает на `Неизвестный объект метаданных` по дочернему metadata-path, сначала проверяй несогласованный хвост в `Role/Ext/Rights.xml` и других служебных XML со ссылками на metadata-path; в первую очередь это `ConfigDumpInfo.xml`, корневые `Ext/*.xml` и прочие XML-слои настроек/прав, где ссылка хранится не как основной состав объекта, а как служебная запись (`name`, `Metadata`, `DataPath`, `Field`, `Command`, `object`). Это частый источник ложного следа, когда сам top-level объект уже приведен к правильному adopted-составу
- “переписывание XML” — общий этап `ChangeFiles`, а не только частная чистка отдельных узлов

## Что помнить при будущих изменениях

- Семантику полей конфига надо держать явной и, где уже есть, обратно совместимой.
- Меняя классификацию или очистку, проверяй не только XML, но и последующую загрузку расширения в 1С.
- Не превращай adopted-объекты в скрытый новый источник `RefDrivenInclusion` без явного правила и записи в документации.
- BSL не является источником правил в этой модели, даже как fallback.
