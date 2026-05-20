# Соглашения

## Нейминг

- Публичные точки входа для встраивания лежат в `pkg/*`
- Внутреннее поведение лежит в `internal/*`
- JSON-поля конфига используют snake_case
- Ключи объектов метаданных в XML-логике имеют вид `Kind.Name`, например `Catalog.Номенклатура`

## Словарь терминов

- `Native`:
  технический режим объекта, который переносится как нативный по текущим правилам классификации. Термин не переводим, чтобы не путать его с произвольными формулировками вроде “полный” или “родной”.
  В XML для обычного `Native`-объекта не пишем явный `<ObjectBelonging>Native</ObjectBelonging>`: это дефолтный режим. Явно сериализуются только adopted-режимы.
- Объекты из `included_Native_objects` должны участвовать в helper-ветках так же, как `Native` по префиксу, но в самой классификации это отдельный explicit include: он сильнее soft-exclude, но слабее `forbidden_*`.
- `AdoptedStub`:
  технический режим урезанного заимствованного объекта. Термин не переводим и не подменяем на просто `Adopted`, потому что в коде и текущей модели это не одно и то же.
- `Use_упо_SearchResult`:
  дополнительный overlay `AdoptedStubCode` поверх уже принятого adopted-решения. Отдельного нового `ObjectBelonging` для него нет: базовая классификация остается в рамках `Native`, `AdoptedStub`, `AdoptedStubExt` и `AdoptedStubMetaData`. Если переносимый SearchResult-метод уже начинается с `упо_` или `Подключаемый_упо`, его переносим как есть, без добавления `&После`.
- У `AdoptedStub` не переносим тексты `ManagerModule` и `ObjectModule`; для adopted-констант по тому же правилу не переносим `ValueManagerModule`; для adopted общих модулей не переносим `Module`; для adopted команд не переносим `CommandModule`. Если adopted-объект уже не `Native`, такие child-записи нужно вычищать и из `ConfigDumpInfo.xml`; для `CommonModule` еще и удалять `CommonModules/<Имя>/Ext/Module.bsl`, а для команд — `Ext/CommandModule.bsl`.
- `AdoptedStubMetaData`:
частный случай `AdoptedStub`, который включается флагом `AdditionalProcessing.Use_MetaDataFile`. Объект попадает в adopted-режим по списку из `CommonTemplate.упо_MetaDataFile`, а child metadata из множеств `Реквизиты` и `ТабличныеЧасти` сохраняются только для реквизитов с префиксом `упо_`.
- У retained child metadata в этом режиме должен оставаться `Native`-статус: top-level объект остается `Adopted`, а сохраненные реквизиты и табличные части не должны нести ни `ObjectBelonging=Adopted`, ни `ExtendedConfigurationObject`.
- Если объект уже попал в единый `excluded`-набор по `excluded_subsystems` или `excluded_objects`, `AdoptedStubMetaData` не должен возвращать его обратно в состав.
- `AdoptedStubExt`:
  заимствованный объект без форм и кода, но с сохраненным реквизитным составом. Это общий термин для двух частных случаев:
- `AdoptedStubExt(Form)` — нужен, когда форма `Native`-объекта тянет не-native ссылку через dynamic list. В этом случае сохраняется не просто сам объект по `MainTable`, а полный field contract списка: `MainTable` плюс используемые `DataPath` / `Field` / `RowPictureDataPath` и родственные поля. Если dynamic list требует реквизит, он должен остаться доступным в итоговом XML; если реквизит не поддержан, конвертер должен падать своей ошибкой, а не оставлять проблему Конфигуратору.
- `AdoptedStubExt(DefinedType)` — нужен, когда ненативный `DefinedType` сам остается источником ссылок и его состав должен дотягиваться по обычному правилу `AdoptedStub`
- `AdoptedStubExt(EventSubscription)` — нужен, когда ненативная подписка на событие сама остается источником ссылок и переносится с полным составом по `RefDrivenInclusion`
- `AdoptedStubExtMetaData`:
  специальный adopted metadata merge object из `CommonTemplate.упо_MetaDataFile`. Используется только для `DefinedType`, `ExchangePlan`, `EventSubscription`, не является alias обычного `AdoptedStub`, сохраняет `Type` / `Content` / `Source` для target-merge и target-ref-driven ссылок, но не должен сохранять формы и BSL.
- Для `AdoptedStubExt` не заимствуем `Properties/StandardAttributes`. Стандартные поля вроде `Ссылка`, `Наименование`, `Код`, `Родитель` считаем встроенными возможностями платформы, а не частью сериализуемого adopted-состава.
- Для `AdoptedStubExt(Form)` у `Catalog` недостаточно только сохранить нужные child objects; top-level и child `Properties` тоже нужно приводить к минимальному stub-виду, близкому к выгрузке Конфигуратора. Нельзя оставлять полный исходный набор свойств каталога (`InputByString`, `ChoiceMode`, `DataHistory` и т.п.), если объект уже переведен в retained adopted-stub для формы.
  В обоих случаях формы и код не переносятся; для `DefinedType` и `EventSubscription` состав не режется.
- `excluded` / исключенный объект / мягко исключенный объект:
  объект, снятый с первичного включения. В текущей модели это мягкое исключение: такой объект может вернуться в `Native`, если на него есть ссылка из допустимого `Native`-объекта.
- Единый набор `excluded` сначала собирается из объектов, найденных по `excluded_subsystems`, а затем дополняется единичными объектами из `excluded_objects`; после этого для всего набора действует одна и та же логика.
- Top-level объект, который связан с веткой из `excluded_subsystems` или явно перечислен в `excluded_objects`, должен попадать в `Excluded` раньше обычного `Native` по native-prefix, но не раньше explicit include из `included_Native_objects`.
- После такого soft-исключения объект не должен возвращаться в состав, если все его допустимые входящие ref-driven ссылки идут только из `Native`-подсистем.
- Если мягко исключенный объект не входил в первичный `Native`, но на него ссылаются только `Native`-подсистемы, этого недостаточно для возврата в `AdoptedStub`: подсистема в таком случае считается только группировочным владельцем ссылки, а не достаточным источником для восстановления объекта.
- Для восстановления excluded-объекта по `RefDrivenInclusion` `Native`-подсистема вообще не считается источником. Источником могут быть только `Native`-объекты и их формы.
- Ссылки на excluded-объекты в `Role/Ext/Rights.xml` не должны восстанавливать объект по `RefDrivenInclusion`; права на такие metadata-path нужно удалять из роли.
- `forbidden_*`:
  жесткое исключение / forbidden. Объект из `forbidden_AdoptedStub_objects` не должен попадать в расширение ни в каком режиме, а ссылка на него очищается.
- `RefDrivenInclusion` / `рефдривен`:
    включение по XML-ссылкам из уже оставленных в составе `Native`-объектов. Также допускает ненативные `DefinedType` и `EventSubscription`, если они уже перенесены как `AdoptedStubExt`: в этом случае их состав остается источником ссылок, а вложенные типы дотягиваются по обычному правилу `AdoptedStub`. Для форм `Native`-объектов источником `AdoptedStubExt` считается не только `MainTable`, но и все XML-ссылки формы на нужные объекты. `AdoptedStubExt(Form)` может появляться только опосредованно через форму `Native`-объекта; у non-native-объектов формы не переносятся, поэтому они не могут быть источником `AdoptedStubExt(Form)`. Может вернуть мягко исключенный объект в `Native`, добавить `AdoptedStub` или `AdoptedStubExt`, но для восстановления excluded-объекта сама по себе `Native`-подсистема источником не считается. На `forbidden_*` не распространяется. BSL не участвует.
    Для `DefinedType` и `EventSubscription` общий cleanup мягко исключенных ссылок не применяется: их состав сохраняется целиком и не режется по `excluded_*`.
    формы не режем частично: если форма остается в составе, она должна быть целой; если форма не переносится, она не оставляется по кускам
    Для dynamic list это означает отдельный инвариант: если `Native`-форма оставляет ссылку на ненативный объект, то target должен быть доведен до `AdoptedStubExt(Form)` по полному field contract списка.
    Формы участвуют в `RefDrivenInclusion` и в сборе dynamic-list contract только у `Native`-владельцев. Если форма принадлежит non-native объекту, она может временно лежать в `_tmp` до финальной очистки, но не является источником ссылок, не участвует в dynamic-list contract и не может порождать `AdoptedStubExt(Form)`.
    Ссылки `style:Имя` внутри `Native`-форм тоже считаем XML-ссылками на `StyleItem.Имя`, но только если такой `StyleItem` реально существует в исходном XML-составе. Встроенные style-константы вроде `style:ButtonTextColor` метаданными не считаем.
  - “переписывание XML”:
    предпочтительная формулировка для XML rewrite.
- `extension_properties`:
  основной JSON-блок для имени, префикса и стабильного `identifier` корня расширения. Старые `extension` и `prefix` остаются backward-compatible alias. Имя и префикс нужно синхронно отражать в `Configuration.xml` и `ConfigDumpInfo.xml`, а `identifier` — в `Configuration/@uuid`.
- `target.xml_dump`:
  не глобальный source graph. Для `DefinedType`, `ExchangePlan`, `EventSubscription` сначала формируем `targetCompatibilitySet`: читаем только top-level XML `DefinedTypes/*.xml`, `EventSubscriptions/*.xml`, `ExchangePlans/*.xml`; `Native`-объекты этих типов допустимы всегда, adopted-объекты допустимы только при наличии top-level XML в target.
- `targetCompatibilitySet`:
  compatibility filter для target-sensitive объектов `DefinedType`, `EventSubscription`, `ExchangePlan`. Он применяется до promotion, в promotion guard и после promotion; не переигрывает `forbidden_*` и не делает `target.xml_dump` самостоятельным механизмом возврата soft-excluded объектов.
- post-promotion merge `target.xml_dump`:
  отдельный источник только для merge-объектов из `CommonTemplate.упо_MetaDataFile`: `DefinedType`, `ExchangePlan`, `EventSubscription`. Эти объекты идут в режиме `AdoptedStubExtMetaData`: source-ссылки до merge не становятся обычным `RefDrivenInclusion`, а target-ссылки из сохраненного `Type` / `Content` / `Source` могут дотянуть отсутствующий top-level metadata-объект как обычный `AdoptedStub`. `forbidden_*` сильнее такого target-ref-driven merge, а сам merge не должен переигрывать `targetCompatibilitySet`.
- У корня `Configuration.xml` обязаны быть `InternalInfo/xr:PropertyState`, `Caption`, `ShortCaption`, `Language.Русский`.
- `ChildObjects` корня должны отражать весь фактический top-level состав расширения:
  все top-level `Native` и `AdoptedStub`-объекты, которые не исключены итоговым решением. Нельзя чистить корень по правилу "оставить только native-prefix".
- часть стандартных служебных объектов тоже может входить в этот состав, если на них завязаны `Native`-команды; текущий пример — `CommandGroup.Информация`, который должен оставаться в составе как `AdoptedStub`
- При нормализации adopted-композиции нельзя удалять обязательные структурные контейнеры формата, если они были в исходном XML. Минимум `ChildObjects` должен сохраняться пустым, если платформа ждет его у данного типа метаданных.
- Для non-native top-level metadata нельзя оставлять ссылки на формы в `Properties` (`DefaultObjectForm`, `DefaultListForm`, `DefaultChoiceForm`, `Auxiliary*Form`) и дочерние `Metadata` форм в `ConfigDumpInfo.xml`; если форма не переносится, ее не должно быть ни как файла, ни как ссылки, ни как записи в реестре.
- Для owner-level проверок и promotion по `OwnerKey` источником истины считается top-level metadata XML объекта. `Ext/*`, формы и другие дочерние XML того же владельца не должны подменять основной XML при валидации состава и реквизитов.
- Валидация `dynamic list contract` должна учитывать не только `StandardAttributes` и `Attribute`, но и field-bearing children объекта, например `Dimension`, `Resource`, `Measure`; иначе конвертер начинает ложноположительно ругаться на реально сохраненные поля регистра.
- Для `dynamic list contract` declared-поля из `Settings/Field` и пути из `DataPath` нужно хранить полным путем, без усечения после первого `.`; nested-path вроде `Состав.Пользователь` — это обязательный retained-состав `AdoptedStubExt(Form)`, а не виртуальный alias.
- Для nested-path табличной части стандартные поля строки (`Ссылка`, `НомерСтроки`) считаются встроенными платформенными и не требуют явной сериализации в adopted-составе.
- Для manual-query `DynamicList`, у которого после переписывания XML больше нет `MainTable`, прямые select-поля вида `Источник.Поле` должны быть явно aliased как `КАК Поле`, если форма ссылается на них через `DataPath` / `Field`. Иначе Конфигуратор может перестать связывать form-driven пути даже при корректном adopted-составе target-объекта.
- Для такого же manual-query списка без `MainTable` нельзя оставлять `ExcludedCommand` со стандартными row-командами `Change`, `ChangeHistory`, `GetURL`, `LevelDown`, `LevelUp`, а `RowPictureDataPath=<Атрибут>.DefaultPicture` нужно убирать: без `MainTable` эти элементы перестают быть валидными.
- Этап `validate dynamic list contracts` выполняется после записи переписанных XML и до `verify old GUID` и финальной cleanup-фазы.
- На этом этапе проверяем только form-driven contract dynamic list:
    target-объект не исключен, не остался урезанным и для non-Native содержит все требуемые поля в top-level metadata XML.
  - Имена полей на этом этапе сравниваются с учетом alias-имен стандартных атрибутов; текущий минимум:
  `Recorder -> Регистратор`, `Period -> Период`, `LineNumber -> НомерСтроки`, `Active -> Активность`.
  - Для `Native` field-level проверка не выполняется; этот слой нужен для `AdoptedStubExt`, а не для полных native-объектов.
- Этот этап не является полной валидацией XML-выгрузки и не заменяет загрузку расширения в 1С.
- У корневого `Ext` не должно быть `ParentConfigurations*`.
- `Ext/CommandInterface.xml` должен быть командным зеркалом `Ext/MainSectionCommandInterface.xml`: в нем сохраняются `CommandsVisibility`, `CommandsPlacement`, `CommandsOrder` и `GroupsOrder`, а `SubsystemsVisibility` / `SubsystemsOrder` туда не пишутся.
- Для `Subsystem` нельзя вырезать `Content`, если подсистема переносится в расширение.
- Для `Subsystem` контейнер `Content` должен оставаться, но ссылки `xr:MDObjectRef` внутри него нужно сверять с реально сохраненным top-level metadata-составом; висячие ссылки на отсутствующие объекты удаляются.
- Для `Subsystem` adopted-нормализация не должна срезать верхний `ChildObjects`; иначе загрузка падает на чтении состава подсистемы.
- Для `Subsystem` верхний контейнер `ChildObjects` сохраняется, но список вложенных `Subsystem` внутри него должен совпадать с реально оставленными дочерними подсистемами. Нельзя оставлять имена дочерних подсистем, если их XML уже не входит в итоговый состав, в том числе для веток, исключенных через `excluded_subsystems`.
- Ненативная top-level `Subsystem` не переносится сама по себе: она может остаться только как adopted-предок реально вложенной native-подсистемы. Отдельных hardcoded special adopted-root для `СтандартныеПодсистемы`, `Администрирование` или `ПодключаемыеОтчетыИОбработки` быть не должно.
- Дочерняя подсистема нативной подсистемы тоже должна оставаться `Native`, если ветка не исключена правилом `excluded_subsystems`. Это правило не зависит от списка special adopted-root: нативная ветка подсистем должна попадать в subsystem-decision целиком.
- Вложенные `Subsystem` нельзя идентифицировать только по последнему имени. Для owner-level логики, `decision` и cleanup подсистема должна иметь ключ полной цепочки, например `Subsystem.Администрирование.Subsystem.КонтрольРаботыПользователей`; иначе разные ветки с одинаковым хвостовым именем начинают склеиваться.
- В `Role/Ext/Rights.xml` нельзя оставлять права на `Configuration.<старое_имя_конфигурации>`. Это хвост исходной конфигурации, а не объект расширения; такие `object`-узлы нужно удалять целиком.
- В `Role/Ext/Rights.xml` нельзя оставлять права на дочерние metadata (`...Attribute...`, `...TabularSection...`, `...Command...`), если соответствующий путь уже не существует в итоговом XML объекта. Права ролей должны совпадать с реально сохраненным составом metadata.
- В `Ext/CommandInterface.xml` и `Ext/MainSectionCommandInterface.xml` нельзя оставлять `Command name=...`, если соответствующая команда уже не существует в итоговом XML объекта. Интерфейс команд должен совпадать с реально сохраненным составом команд.
- В `Form.xml` нельзя оставлять элементы формы с `<Command>Catalog.X.Command.Y</Command>` или другой metadata-command ссылкой, если такой команды уже нет в итоговом XML объекта. Такой элемент формы нужно удалять целиком, а не возвращать команду в adopted-состав без отдельного правила.
- В `ConfigDumpInfo.xml` для non-`Native` top-level объекта нельзя оставлять плоские child-записи (`...Command...`, `...Attribute...`, `...TabularSection...`), если такой child-path уже удален из итогового XML объекта.
- В child metadata top-level объекта нельзя оставлять `Attribute` / `Dimension` / `Resource` / `Measure`, если их `Properties/Type` ссылается на excluded-объект. В таком случае нужно удалять весь child-узел, а не только отдельное значение типа.
- Ссылки на metadata внутри type-bearing XML тоже должны участвовать в cleanup: как минимум `Properties/Type/v8:Type` и `Ext/Predefined.xml` с current-config namespace alias вроде `d4p1:CatalogRef.X`. Иначе в дампе остаются висячие типы и предопределенные элементы, хотя основной состав уже очищен.

Если термин совпадает с именем поля, режима или значением в коде/конфиге, оставляй техническое имя как есть.

## Стиль изменений

- Предпочитай небольшие локальные правки
- Сохраняй существующую русскоязычную формулировку логов и ошибок, если нет причины стандартизировать затронутую область
- Не делай попутную “косметическую” очистку в `change.go`
- Не переименовывай устоявшиеся технические режимы (`Native`, `AdoptedStub`) в документации и комментариях без явной причины
- Если нужен `AdoptedStubExt` или `AdoptedStubExtMetaData`, называй его именно так; не подменяй формулировками вроде “почти полный adopted” или “stub с полями”

## Тесты

- Сейчас в репозитории нет зафиксированных `*_test.go`
- Базовая проверка — `go build ./...` и `go test ./...`
- Для XML-поведения практическая проверка включает реальный прогон конвертации на активном конфиге

## Логирование

- Используй `log.Printf`/`log.Fatalf` для журналирования стадий процесса и ошибок
- Держи лог стадийным: `step: ...`, `xml step: ...`, `decision debug: ...`
- Не оставляй шумный отладочный вывод постоянно, если он не помогает будущим расследованиям

## Документация

- Пиши компактно и только по реально подтвержденному поведению репозитория
- Если что-то нельзя надежно вывести, помечай как `assumption` или `TODO`
- При смене долгоживущего поведения обновляй `.codex/decisions.md`

## Конфиги и секреты

- Активный локальный конфиг — `configs/config.json`
- Embedded defaults лежат в `internal/config/default.json`
- Пути нормализуются в абсолютные на этапе загрузки конфига
- Управления секретами в репозитории нет; текущие конфиги содержат только локальные файловые пути
