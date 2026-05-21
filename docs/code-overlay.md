# Code Overlay Architecture

## Goal

The project currently generates a template extension that contains generated code blocks.
Those blocks are later manually adapted by users in the resulting extension.

The purpose of Code Overlay is to support transferring adapted code blocks from a reference extension back into a newly generated extension template.

This mechanism must preserve:

- reproducibility;
- deterministic generation;
- generated object structure;
- compatibility with future source configuration updates.

The reference extension must NOT become a second configuration source.
It must act only as an overlay source for adapted code blocks.

---

# High-Level Architecture

Generation pipeline:

1. Generate extension using current pipeline.
2. Build generated code blocks.
3. Load overlay file.
4. Match overlay blocks against generated blocks.
5. Apply override content.
6. Save resulting extension.

The overlay layer must be applied AFTER:

- Native/Adopted processing;
- merge processing;
- module cleanup;
- generated module creation.

The overlay layer must be applied BEFORE:

- final export/save.

---

# Overlay Source

Overlay data is stored in a standalone service file:

```text
code_overlay.json
```

The file is generated from a reference extension using a dedicated extraction tool.

The overlay file must be committed into Git and reviewed like regular source code.

---

# Overlay Modes

Configuration:

```json
{
  "code_source": "generated"
}
```

Supported values:

| Value | Behavior |
|---|---|
| generated | Current behavior. Ignore overlay. |
| overlay | Apply overlay when matching blocks are found. |
| overlay_strict | Same as overlay, but missing blocks are treated as errors. |

---

# Overlay Extraction Tool

Example CLI:

```bash
extract_code_overlay --extension-path path/to/ext --output code_overlay.json
```

Responsibilities:

- traverse extension objects;
- extract supported code blocks;
- generate stable identifiers;
- serialize overlay data;
- store metadata and hashes.

---

# Supported Block Types

Initial scope:

- ObjectModule
- ManagerModule
- FormModule
- CommandModule
- CommonModule

Future extensions are allowed.

---

# Overlay Block Identity

Block identity must be deterministic and stable.

Examples:

```text
Catalog.Номенклатура:ObjectModule
Catalog.Номенклатура.Form.ФормаЭлемента:FormModule
CommonModule.упо_Utils:CommonModule
```

Matching rules must be centralized and covered by tests.

---

# Overlay File Structure

Example:

```json
{
  "version": 1,
  "source": "reference_extension",
  "blocks": [
    {
      "id": "Catalog.Номенклатура:ObjectModule",
      "object": "Catalog.Номенклатура",
      "kind": "ObjectModule",
      "path": "Catalogs/Номенклатура/Ext/ObjectModule.bsl",
      "hash": "...",
      "content": "..."
    }
  ]
}
```

---

# Overlay Diagnostics

The pipeline must generate diagnostics:

```text
loaded: N
applied: N
skipped: N
missing: N
conflicted: N
```

Optional output:

```text
code_overlay_report.json
```

---

# Design Constraints

## Overlay must NOT:

- create new metadata objects;
- affect object inclusion rules;
- modify merge priority logic;
- change Native/Adopted selection.

## Overlay must:

- override only code content;
- preserve generated structure;
- support forward regeneration;
- support Git diff/review workflow.

---

# Recommended Implementation Order

1. Overlay extraction tool.
2. Matching rules.
3. Overlay application pipeline.
4. Diagnostics.
5. Regression tests.

---

# Regression Coverage

Minimum regression scenarios:

- modified form module transfer;
- generated fallback when overlay block is missing;
- strict mode failure;
- renamed object behavior;
- removed form behavior;
- merge target object support.
