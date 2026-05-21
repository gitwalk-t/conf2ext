# Термины

## Основные режимы

- `Native` — объект, остающийся нативным.
- `AdoptedStub` — adopted-объект в урезанном виде.
- `AdoptedStubExt` — adopted-объект с сохраненным metadata composition, но без форм и BSL.
- `AdoptedStubMetaData` — special adopted-mode через `CommonTemplate.упо_MetaDataFile`.
- `AdoptedStubExtMetaData` — merge-mode для `DefinedType`, `ExchangePlan`, `EventSubscription`.

## Исключения

- `excluded` / soft exclude — объект снят с первичного включения, но может вернуться через `RefDrivenInclusion`.
- `forbidden` / hard exclude — объект запрещен полностью.
- `included_Native_objects` — explicit native override над soft exclude.

## RefDrivenInclusion

`RefDrivenInclusion` / `рефдривен` — включение по XML-ссылкам из допустимых `Native`-объектов.

Правила:
- BSL не участвует.
- `Native`-подсистема сама по себе не является источником восстановления excluded-объекта.
- формы не режутся частично.
- form-driven links работают только для форм native-владельцев.

## Target merge

- `targetCompatibilitySet` — compatibility filter для `DefinedType`, `EventSubscription`, `ExchangePlan` при наличии `target.xml_dump`.
- target merge не делает `target.xml_dump` глобальным source graph.

## XML pipeline

- `ChangeFiles` — общий этап переписывания XML.
- `dynamic list contract` — проверка field contract для form-driven dynamic list.
- `OwnerKey` — owner-level identity top-level metadata object.
