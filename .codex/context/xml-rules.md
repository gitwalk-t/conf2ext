# XML/classification rules

Этот файл больше не содержит полный XML knowledge dump.

Цель — routing по XML-subdomains.

## Что читать по типу задачи

### Classification / promotion

Читать:

```text
.codex/context/classification.md
```

Темы:
- priority order
- soft/hard exclude
- `RefDrivenInclusion`
- adopted modes
- registrator propagation

### Forms / dynamic list

Читать:

```text
.codex/context/forms.md
```

Темы:
- `AdoptedStubExt(Form)`
- field contract
- dynamic list validation
- `DataPath`
- standard attribute aliases

### Target merge / target.xml_dump

Читать:

```text
.codex/context/target-merge.md
```

Темы:
- `targetCompatibilitySet`
- `AdoptedStubExtMetaData`
- canonical naming
- merge constraints

### Cleanup / normalization

Читать:

```text
.codex/context/cleanup.md
```

Темы:
- adopted cleanup
- root cleanup
- role cleanup
- subsystem cleanup
- command cleanup
- type-bearing cleanup

### Термины

Читать:

```text
.codex/context/terms.md
```

## Правило экономии контекста

Не читать все XML-subdomains подряд.

Для обычной form-debugging задачи обычно достаточно:
- `forms.md`
- иногда `cleanup.md`

Для classification bug:
- `classification.md`
- иногда `target-merge.md`

Для loading/XDTO problem:
- `cleanup.md`
- иногда `forms.md`
