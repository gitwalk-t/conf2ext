# Передача контекста

## Что уже сделано

- Добавлены публичные обертки в `pkg/config` и `pkg/converter`
- CLI по умолчанию использует `./configs/config.json`
- Текущая локальная работа сосредоточена на `cfConvert` в `output/demo.cfe`
- `keep_xml_dump=true` в активном конфиге удерживает временную XML-выгрузку и временную ИБ после прогона
- `keep_xml_dump=true` также сохраняет копию XML-дампа в `output/_log/xml_dumps/<v8_src*>`, если временный каталог потом исчезает
- `stop_after_xml_dump=true` останавливает прогон после `ChangeFiles`, не доходя до загрузки расширения и `.cfe`
- перед новым прогоном сначала завершаем старые `go` и соответствующие `1cv8.exe`, иначе можно перепутать новый запуск со старым хвостом
- Примеры для расследований и сравнения лежат в `D:\Codex\files-converter_ver2\Artefacts`; фраза "пример в артефактах" означает сначала искать там
- если в `Artefacts/` лежит `.cfe`, сначала получаем из него XML-дамп и только потом сравниваем с текущим состоянием
- XML-логика уже сильно расширена вокруг:
  - классификации объектов
  - мягко исключаемых объектов и подсистем
  - жестко исключаемых объектов из `forbidden_AdoptedStub_objects`
  - обработка `AdoptedStub` и `AdoptedStubExt`
  - замены GUID у adopted metadata
  - очистки forbidden references и движений по регистрам

## Что осталось

- Главная незавершенная ветка — получить валидную сборку расширения из текущего конфига
- Основное узкое место — [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go), особенно правила `RefDrivenInclusion` и очистки XML для `Native` vs `AdoptedStub`
- Ручные прогоны долгие: обычно надо читать stderr-log вместе с соответствующим `_tmp/v8_src*`

## Где риски

- В `change.go` накопились экспериментальные вспомогательные функции от итеративной отладки; не все они реально участвуют в основном пути выполнения
- Маленькое изменение правила может затронуть сразу много классов метаданных
- Из-за отсутствия проверок от регрессий иногда опаснее “аккуратный рефакторинг”, чем локальная точечная правка
- Термины `Native`, `AdoptedStub`, `AdoptedStubExt(Form)`, `AdoptedStubExt(DefinedType)`, `AdoptedStubExt(EventSubscription)`, “мягко исключенный” и “жестко исключенный” важно не смешивать: часть прошлых ошибок рождалась именно из размывания этих границ
- `excluded` и “мягко исключенный объект” — это один и тот же смысл; `forbidden` и “жестко исключенный объект” — тоже один и тот же смысл
- Если в правилах появляется `AdoptedStubExt`, его нужно трактовать как частный случай `AdoptedStub`, а не как возвращение полноценного режима `Adopted`; обязательно различать форменный, типовой и событийный случаи
- `RefDrivenInclusion` должно учитывать только XML-ссылки из допустимых `Native`-объектов, а для ненативных `DefinedType` и `EventSubscription` в статусе `AdoptedStubExt` еще и сохранять их состав как источник ссылок; BSL тут не участвует; формы не режем частично, это всегда целый объект или полный отказ от формы
- если задан `target.xml_dump`, для `DefinedType` / `EventSubscription` / `ExchangePlan` теперь есть отдельный `targetCompatibilitySet`: lightweight-коллектор читает только `DefinedTypes/*.xml`, `EventSubscriptions/*.xml`, `ExchangePlans/*.xml`; `Native`-объекты этих типов допустимы всегда, adopted-объекты допустимы только при наличии top-level XML в target; этот слой не заменяет template-driven target-merge из `CommonTemplate.упо_MetaDataFile`
- для `AdoptedStubExt(Form)` нельзя ограничиваться одним `MainTable`: dynamic list требует полного field contract (`DataPath`, `Field`, `RowPictureDataPath` и родственные поля). Если контракт не выполнен, ошибка должна проявляться в конвертере
- `extension_properties` — основной источник имени, префикса и стабильного `identifier` корня; старые `extension` / `prefix` остаются только backward-compatible alias и тоже должны сходиться с `Configuration.xml` и `ConfigDumpInfo.xml`
- у корня `Configuration.xml` обязаны быть `InternalInfo/xr:PropertyState`, `Caption`, `ShortCaption`, `Language.Русский`
- `Configuration.xml/ChildObjects` должен перечислять весь итоговый top-level состав расширения, включая `AdoptedStub`; нельзя чистить его только по `native_prefixes`
- при очистке top-level adopted metadata нужно сохранять обязательный контейнер `ChildObjects`, если он был в исходном XML; иначе Конфигуратор падает на ошибке структуры документа
- у корневого `Ext` не должно быть `ParentConfigurations*`
- для `Subsystem` нельзя вырезать `Content`, если подсистема переносится в расширение
- для `Subsystem` adopted-нормализация не должна срезать верхний `ChildObjects`; иначе Конфигуратор перестает читать состав подсистемы

## С каких файлов продолжать

1. [internal/utils/xmlutil/change.go](D:\Codex\files-converter_ver2\internal\utils\xmlutil\change.go)
2. [configs/config.json](D:\Codex\files-converter_ver2\configs\config.json)
3. Последние `run-*.stderr.log` в корне
 4. Последняя выгрузка в `output/_tmp/v8_src*`
 5. Логи расследований в `output/_log`

## Рекомендуемый цикл продолжения

1. `go build ./...`
2. `go test ./...`
3. `go run . --config .\configs\config.json`
4. Разобрать stderr-log
 5. Разобрать соответствующий `output/_tmp/v8_src*`
 6. При необходимости посмотреть связанные логи в `output/_log`
 7. Сделать наименьшее возможное изменение XML-правила
