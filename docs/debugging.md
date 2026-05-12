# Отладка

## Базовый цикл

1. Собрать проект:

```powershell
go build ./...
```

2. Запустить конвертацию на активном локальном конфиге:

```powershell
go run . --config .\configs\config.json
```

3. Если конвертация упала, сначала смотреть stderr-log текущего прогона.

4. Затем смотреть соответствующую выгрузку в `output/_tmp\...`, обычно каталог `v8_src*`.

5. После правки повторять тот же цикл.

Если прогон уже дошел до `xml step: validate dynamic list contracts` и упал на этом этапе, можно продолжить на текущем `v8_src*`, не копируя исходники заново:

```powershell
go run .\cmd\changefiles\main.go .\configs\config.json D:\Codex\files-converter_ver2\output\_tmp\v8_src1044738853 --resume-from-validation
```

Этот режим:
- не переписывает XML заново
- повторяет расчет `decisions` и form-contract
- начинает с `validate dynamic list contracts`
- затем выполняет финальную cleanup-фазу
- не повторяет проверку old GUID, потому что на уже переписанном temp исходная карта замен недоступна

Перед новым прогоном:
- сначала остановить старые `go` и связанные с ними `1cv8.exe`
- потом запускать новый сборочный цикл
- сразу поставить автоматизацию проверки состояния раз в 10 минут, по образцу `Files-converter run check (silent go, 10m)` / `check-run`
- так проще не перепутать хвосты старого процесса с результатом текущего прогона

Важно:
- каталог `v8_src*` сам по себе еще не гарантирует, что это уже дамп расширения
- если нужно назвать пользователю "свежий дамп", сначала проверь `Configuration.xml`
- корректный дамп расширения должен иметь у корня свойства расширения (`ObjectBelonging=Adopted`, имя и префикс из конфига), а не исходные `Name` / `Synonym` / пустой `NamePrefix` основной конфигурации

## Типовые ошибки загрузки расширения

### Неверный путь к данным

- Пример: `DataProcessors/упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов/Forms/Форма/Ext/Form.xml: "СписокСтатейБюджетов.ФинансоваяСтатья"`
- С высокой вероятностью причина в одном из двух мест:
  - конвертер сам испортил `Form.xml` при нормализации `DataPath`
  - в дампе вообще остался объект, который должен был быть `excluded`, и платформа уже падает на его форме
- В первую очередь смотреть:
  - diff `input/.../Ext/Form.xml` против `output/_log/xml_dumps/.../Ext/Form.xml`
  - [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go), функции form-cleanup и form-normalization
  - присутствует ли сам объект в итоговом дампе как top-level metadata
- Что уже исправлялось на таких кейсах:
  - убрать агрессивное переписывание `ChildItems//DataPath` и `RowPictureDataPath` у manual-query dynamic list
  - удалить из формы control, если его `DataPath` ссылается на `НаборКонстант.<ИмяКонстанты>`, а такой `Constant.*` не вошел в итоговый состав
  - удалять из dynamic list формы поля и связанные controls, если они ссылаются на `CommonAttribute.<Имя>`, которого уже нет в итоговом составе, а самого поля нет в owner metadata
  - если проблемный объект должен быть `excluded`, сначала чинить его удержание в составе, а не саму форму

### Неизвестный объект метаданных

- Пример: `Неизвестный объект метаданных - DataProcessor.ОперацииЗакрытияМесяца`
- Обычно сам top-level объект уже вырезан правильно, а в одном из служебных XML осталась висячая ссылка на него
- Если ошибка звучит как `Неизвестный объект метаданных` по `Attribute` / `TabularSection` / `Command`, сначала проверяй именно несогласованный хвост в служебных XML со ссылками на metadata-path: часто сам top-level объект уже корректно урезан, а падает именно висячая ссылка в таком платформенном слое
- Для `Command` отдельно проверяй не только owner XML, но и файловый каркас команды: часть owner-команд живет только как каталог `Commands/<Имя>` с `Ext/CommandModule.bsl` и записью в `ConfigDumpInfo.xml`, без отдельного `Command.xml`; такой metadata-path нельзя считать отсутствующим только потому, что его нет в основном XML владельца
- Если неизвестным остается `Catalog.X.Command.Y` у adopted-владельца, проверь еще один сценарий: ссылка на child-команду могла остаться в `FunctionalOption.Content`, форме или другом служебном XML, а adopted-cleanup вырезал сам `Command` из `ChildObjects`; в таком случае нужно удержать сам metadata-path команды, но не возвращать текст `CommandModule`
- Если команда должна удерживаться у adopted-владельца, проверь и порядок вычисления retained-команд: такой список нужно собирать уже после `RefDrivenInclusion` и других promotion-решений, иначе объект может еще считаться `excluded` в момент расчета и child-команда снова пропадет из `ChildObjects`
- В первую очередь смотреть:
  - `Role/Ext/Rights.xml`
  - `Subsystem/.../Content`
  - `ConfigDumpInfo.xml`
  - корневые `Ext/*.xml`
  - другие служебные XML, где ссылка живет в `name` / `Metadata` / `DataPath` / `Field` / `Command` / `object`
- Что уже исправлялось на таких кейсах:
  - чистка `Role/Ext/Rights.xml` от прав на excluded metadata и отсутствующие child metadata-path
  - чистка `Subsystem/Content` от `xr:MDObjectRef` на top-level metadata, которого уже нет в итоговом составе

### Ошибка преобразования данных XDTO

- Пример: `ChartsOfCharacteristicTypes/.../Ext/Predefined: Значение cfg:CatalogRef.упо_Планы, тип QName`
- Чаще всего это признак того, что конвертер испортил type-bearing XML: `Properties/Type`, `Ext/Predefined.xml`, `v8:Type`, `TypeSet`, namespace alias
- В первую очередь смотреть:
  - diff `input` vs `dump` для `ChartsOfCharacteristicTypes/.../Ext/Predefined.xml`
  - current-config alias вида `d4p1:CatalogRef...`
  - [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go), нормализацию `ChartOfCharacteristicTypes` и cleanup type-bearing XML
- Что уже исправлялось на таких кейсах:
  - не переписывать корректный current-config alias `d4p1:` в `cfg:`
  - не переносить owner-level qualifiers в predefined item без отдельного основания
  - оставлять только безопасную проверку совместимости типа predefined item с owner type-set

### Тип предопределенного вида характеристики не соответствует типу плана вида характеристики

- Пример: `ПланВидовХарактеристик.упо_СвойстваАгрегированныхХарактеристикТабличныхЧастей ...(СписокРоль)`
- Это почти всегда соседний кейс к XDTO-ошибкам: нарушена согласованность между `Properties/Type` у самого `ChartOfCharacteristicTypes` и типами в `Ext/Predefined.xml`
- В первую очередь смотреть:
  - `ChartsOfCharacteristicTypes/<объект>.xml`
  - `ChartsOfCharacteristicTypes/<объект>/Ext/Predefined.xml`
  - cleanup ссылок в `Properties/Type` и `Predefined`
- Отдельно сверяй qualifier-блоки (`StringQualifiers`, `NumberQualifiers`, `DateQualifiers`, `BinaryDataQualifiers`) у owner `Properties/Type` и у `Type` конкретного predefined item: в extension-дампе платформа может требовать owner-level qualifiers даже у matching item-типа без собственного scalar `xs:*`, поэтому расхождение здесь важно не только для явных string/number/date item-типов
- Что уже исправлялось на таких кейсах:
  - добавить распознавание metadata refs с alias вроде `d4p1:CatalogRef...`
  - включить type-bearing XML в cleanup висячих metadata-ссылок
  - затем сузить нормализацию `Predefined`, чтобы она не портила QName и qualifiers
  - синхронизировать owner-level qualifier-блоки predefined item с owner `Properties/Type`, не переписывая QName и current-config alias

### Excluded-объект остался в расширении

- Пример: `DataProcessor.упо_СопоставлениеФинансовыхСтатейСтатьямБюджетов` остался в `v8_src*`, хотя должен был уйти по `excluded_subsystems`
- Почти всегда проблема в порядке ref-driven восстановления: объект сначала попал в `excluded`, а потом был возвращен обратно ролью, подсистемой, metadata-file include или веткой `primaryNativeObjects`
- В первую очередь смотреть:
  - [configs/config.json](D:\Codex\files-converter_ver2\configs\config.json): `excluded_subsystems`, `excluded_objects`, `forbidden_*`
  - `Roles/.../Ext/Rights.xml`
  - `Subsystems/.../Content`
  - [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go), порядок веток в `decideObject(...)` и `promoteReferencedObjectsToAdoptedStub(...)`
- Что уже исправлялось на таких кейсах:
  - native роли не являются источником ref-driven восстановления excluded-объектов
  - native подсистемы тоже не являются таким источником, даже если объект имеет native prefix и попадает в `primaryNativeObjects`
  - `AdoptedStubMetaData` из `Use_MetaDataFile=true` не должен переигрывать уже собранный soft-exclude

## Как отслеживать прогон

- Для долгих прогонов должна быть активна 10-минутная автоматизация мониторинга: она читает `output/_log/last_run.txt`, связанные stdout/stderr, свежий temp в `output/_tmp` и свежий дамп в `output/_log/xml_dumps`, но не перезапускает процесс и не трогает чужие `1cv8.exe`.
- Сначала смотреть `output/_log`, потом `output/_tmp`, потом `output/demo.cfe`.
- Новый прогон 1С отличать по более позднему `StartTime`, `CommandLine` с текущим `--config` и связке с новым `go run`.
- Старые `1cv8.exe`, не связанные с текущим запуском, считать фоновыми.
- Если `go` уже завершился, а `1cv8.exe` жив, прогон еще в 1С-фазе.

## Полезные переменные окружения

- `FILES_CONVERTER_KEEP_TMP=1`: оставить временные каталоги и временную ИБ для расследования
- `FILES_CONVERTER_SNAPSHOT_BEFORE_CHANGE=1`: сохранить снимок выгрузки до переписывания XML
- `FILES_CONVERTER_DEBUG_DECISIONS=1`: включить выборочный `decision debug` в [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go)
- `CONFIG_PATH`: использовать конфиг из переменной окружения, если не передан `--config`

## Полезный флаг конфига

- `keep_xml_dump=true`: сохранить временную XML-выгрузку и временную ИБ после прогона
- при `keep_xml_dump=true` дополнительно сохраняется копия XML-дампа в `output/_log/xml_dumps/<v8_src*>`
- `stop_after_xml_dump=true`: остановить прогон сразу после `ChangeFiles`, чтобы проверить только XML-выгрузку
- `enable_form_validation=false`: временно отключить `validate dynamic list contracts`, если нужно довести XML-дамп до стадии загрузки без этой проверки

## Что считать источником истины

- Код и текущий конфиг: [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go), [configs/config.json](D:\Codex\files-converter_ver2\configs\config.json)
- Рабочие артефакты расследования: `output/_tmp`, `output/_log`
- Старые `run-*.log` в корне: временный хвост, а не желаемое место хранения
