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
