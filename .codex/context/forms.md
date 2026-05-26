# Form and dynamic-list rules

## General form rules

- Формы не режутся частично.
- Non-native формы не являются источником `RefDrivenInclusion`.
- Если форма не должна переноситься — удаляется целиком.

## Native form cleanup boundary

Для `Native` forms:
- aggressive dynamic-list cleanup запрещен;
- aggressive command cleanup запрещен;
- нельзя менять standard command flags (`Create`/`Copy`/`Delete`/`Change` и аналоги) generic cleanup-ом;
- нельзя удалять native form command references только потому, что generic resolver не доказал их existence;
- нельзя удалять `MainTable`;
- нельзя удалять custom/manual-query dynamic-list fields только потому, что они отсутствуют в metadata owner;
- нельзя массово удалять `DataPath` через generic missing-field cleanup;
- cleanup должен удалять только доказанно dangling/non-native references.

Aggressive cleanup допускается только для non-Native forms (`Adopted` / `AdoptedStub*`).

## AdoptedStubExt(Form)

Используется для form-driven target objects.

Field contract включает:
- `MainTable`
- `DataPath`
- `Field`
- `RowPictureDataPath`
- related dynamic-list references

## Dynamic list validation

`validate dynamic list contracts`:
- выполняется после XML rewrite;
- проверяет только form-driven contracts;
- нужен для non-Native target objects;
- не является full platform validation.

## Field sources

Field считается доступным через:
- `StandardAttributes`
- `Attribute`
- `Dimension`
- `Resource`
- `Measure`

## Standard attribute aliases

Минимальный набор:
- `Recorder -> Регистратор`
- `Period -> Период`
- `LineNumber -> НомерСтроки`
- `Active -> Активность`

## Dynamic-list virtual fields

Стандартные/virtual list fields должны сохраняться даже если они не найдены через обычный metadata lookup.

Минимальный allowlist:
- `Ссылка`
- `Наименование`
- `Код`
- `ПометкаУдаления`
- `Представление`

## Common failure patterns

- broken `DataPath`
- removed `CommonAttribute`
- dangling metadata-command
- partial form cleanup
- aggressive cleanup applied to Native forms
