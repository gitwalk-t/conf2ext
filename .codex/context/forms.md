# Form and dynamic-list rules

## General form rules

- Формы не режутся частично.
- Non-native формы не являются источником `RefDrivenInclusion`.
- Если форма не должна переноситься — удаляется целиком.

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

## Common failure patterns

- broken `DataPath`
- removed `CommonAttribute`
- dangling metadata-command
- partial form cleanup
