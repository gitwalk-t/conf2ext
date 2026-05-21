# Cleanup rules

## Adopted cleanup

Для `AdoptedStub` не переносить:
- `ManagerModule`
- `ObjectModule`
- `ValueManagerModule`
- `Module`
- `CommandModule`

Удалять:
- child metadata-записи этих модулей из `ConfigDumpInfo.xml`;
- соответствующие `Ext/*.bsl`.

## Root cleanup

- У root `Ext` не должно быть `ParentConfigurations*`.
- `Configuration.xml/ChildObjects` должен соответствовать фактическому top-level составу.
- `Language.Русский` и `InternalInfo/xr:PropertyState` обязательны.

## Role cleanup

`Role/Ext/Rights.xml` очищать от:
- `Configuration.<oldConfigName>`;
- dangling metadata references;
- dangling child metadata references.

## Subsystem cleanup

- `Subsystem/Content` сохраняется.
- dangling `xr:MDObjectRef` удаляются.
- adopted subsystem не должен терять верхний `ChildObjects`.

## Command cleanup

Из:
- `CommandInterface.xml`
- `MainSectionCommandInterface.xml`
- `Form.xml`

удалять ссылки на отсутствующие metadata commands.

## Type-bearing cleanup

Учитывать:
- `Properties/Type`
- `Ext/Predefined.xml`
- QName aliases
- qualifier synchronization

Не переписывать корректные current-config aliases.
