# Classification rules

## Priority order

1. `forbidden_*`
2. `included_Native_objects`
3. `excluded_subsystems` + `excluded_objects`
4. native-prefix classification
5. ref-driven promotion

## Soft vs hard exclude

- `excluded_*` — soft exclude.
- `forbidden_*` — hard exclude.
- `included_Native_objects` overrides soft exclude, but not hard exclude.

## RefDrivenInclusion

- Работает только по XML-ссылкам.
- BSL не участвует.
- Источники:
  - допустимые `Native`-объекты;
  - их формы;
  - отдельные `AdoptedStubExt(DefinedType/EventSubscription)`.
- `Native`-подсистема сама по себе не является источником восстановления excluded-object.
- `Role/Ext/Rights.xml` не должен восстанавливать excluded-object.

## Adopted modes

- `AdoptedStub` — базовый adopted-mode.
- `AdoptedStubExt` — adopted с metadata composition.
- `AdoptedStubMetaData` — mode через `Use_MetaDataFile`.
- `AdoptedStubExtMetaData` — target-sensitive merge mode.

## Registrator rule

Если register остается `Native`, его documents-registrators тоже должны остаться `Native`.
