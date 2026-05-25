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
- нативные template-content (`*/Templates/*/Ext/Template.xml`) не являются источником восстановления excluded-object.
- `Native`-подсистема сама по себе не является источником восстановления excluded-object.
- `Role/Ext/Rights.xml` не должен восстанавливать excluded-object.

## BSL reachability

BSL reachability — отдельный механизм от XML RefDrivenInclusion.

Правила:

- BSL-зависимости не должны участвовать в metadata promotion.
- BSL-зависимости не должны восстанавливать excluded-object.
- `Native` event subscriptions являются source root для дотягивания BSL-кода.
- Если handler Native event subscription вызывает метод Adopted-модуля, полный текст метода должен переноситься даже без `упо_SearchResult`.
- Это правило ограничено переносом текста методов и не меняет XML classification.

## Adopted modes

- `AdoptedStub` — базовый adopted-mode.
- `AdoptedStubExt` — adopted с metadata composition.
- `AdoptedStubMetaData` — mode через `Use_MetaDataFile`.
- `AdoptedStubExtMetaData` — target-sensitive merge mode.

## Use_MetaDataFile semantics

`CommonTemplate.упо_MetaDataFile` не должен восстанавливать excluded owner object с нуля.

Правила:
- owner object должен сохраняться в extension;
- added leaf-fields/requisites могут оставаться `Native`;
- leaf-field promotion не должен удалять owner XML;
- `CommonTemplate.упо_MetaDataFile` может только расширять уже сохраненный owner object;
- `forbidden_*` остается сильнее `Use_MetaDataFile`;
- `included_Native_objects` остается сильнее soft-exclude;
- BSL не используется как source classification.

## Registrator rule

Если register остается `Native`, его documents-registrators тоже должны остаться `Native`.

Ограничение:
- registrator rule не должен переопределять `excluded_subsystems` или `excluded_objects`;
- explicit `included_Native_objects` остается сильнее soft-exclude и срабатывает раньше registrator rule.
