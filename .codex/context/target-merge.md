# Target merge rules

## Target-sensitive object types

- `DefinedType`
- `EventSubscription`
- `ExchangePlan`

## targetCompatibilitySet

Если задан `target.xml_dump`:

- `Native` target-sensitive objects допустимы всегда;
- adopted target-sensitive objects допустимы только если top-level XML существует в target.

`targetCompatibilitySet`:
- не делает target глобальным source graph;
- не заменяет обычный `RefDrivenInclusion`.
- применяется только к top-level metadata objects.

## AdoptedStubExtMetaData

Используется для merge-объектов из:

```text
CommonTemplate.упо_MetaDataFile
```

Сохраняются:
- `Properties/Type`
- `Ext/Content.xml`
- `Properties/Source`

Не переносятся:
- формы
- BSL

## Canonical naming

Если source и target совпадают по `id/uuid`:
- canonical name берется из target;
- stale source-name не должен создавать duplicate adopted-object;
- UUID-aware matching должен работать даже если persisted extension id устарел.

## Merge constraints

- target merge не переигрывает `forbidden_*`;
- target merge не возвращает excluded adopted-object вне `targetCompatibilitySet`;
- target merge не делает target глобальным dependency source.

## Adopted form borrowing

`target.xml_dump` также используется как canonical source для Adopted object forms, но не глобально для всех форм.

Правило:
- borrowed form переносится только если форма явно сохранена search-result overlay;
- и только если в target найден соответствующий form template.

Matching order:
- сначала по form identifier/uuid;
- затем по полному metadata name.

После match:
- borrowed `Ext/Form.xml` переписывается по configurator borrowing pattern;
- borrowed form не уходит в generic partial cleanup/adopted excluded-reference cleanup.

Borrowed adopted normalization:
- после target-based rewrite borrowed `Adopted` form проходит только deterministic cleanup, выведенный из configurator-borrowed эталонов в `input/comerr`;
- cleanup не должен использовать per-form hardcode, только общие borrowed `Adopted` rules;
- cleanup удаляет stray `ExcludedCommand`, которые не подтверждены configurator borrowing pattern;
- root `CommandSet` и nested/table `CommandSet` нормализуются раздельно;
- root-level `SetDeletionMark` может сохраняться, если он есть в configurator-borrowed форме;
- nested/table `SetDeletionMark` должен удаляться.
