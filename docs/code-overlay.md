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

The reference extension XML dump must NOT become a runtime source for the main generation pipeline.
It is used only by a standalone extraction tool to build a reusable overlay artifact.

---

# High-Level Architecture

Two separate stages are required.

## Stage 1: Extract overlay artifact

Input:

```text
input/etalonCode
```

Output:

```text
configs/code_overlay.json
```

The `input/etalonCode` directory contains a full XML dump of the reference extension.
It is consumed only by the overlay extraction tool.

## Stage 2: Apply overlay artifact during generation

Generation pipeline:

1. Generate extension using current pipeline.
2. Build generated code blocks.
3. Load overlay artifact from `configs/code_overlay.json`.
4. Match overlay blocks against generated blocks.
5. Apply override content.
6. Save resulting extension.

The main generation pipeline must not read `input/etalonCode` directly.
It must read only the prepared overlay artifact.

The overlay layer must be applied AFTER:

- Native/Adopted processing;
- merge processing;
- module cleanup;
- generated module creation.

The overlay layer must be applied BEFORE:

- final export/save.

---

# Reference Extension XML Dump Location

Default extraction input:

```text
input/etalonCode
```

This path belongs to the standalone extraction tool, not to the main runtime generation config.

The extraction tool may accept this path as a CLI argument, for example:

```bash
extract_code_overlay --extension-path input/etalonCode --output configs/code_overlay.json
```

If omitted, the extraction tool should use `input/etalonCode` as the default convention.

---

# Overlay Artifact Location

Overlay data is stored in a standalone service file:

```text
configs/code_overlay.json
```

The file is generated from the reference extension XML dump using the dedicated extraction tool.

The overlay artifact must be committed into Git and reviewed like regular source code.

`configs/code_overlay.json` should be placed near other stable configuration artifacts, such as bindings.

---

# Overlay Configuration MVP

The main generation config should reference the prepared overlay artifact, not the reference XML dump.

MVP configuration:

```json
{
  "code_overlay": {
    "enabled": true,
    "overlay_file": "configs/code_overlay.json"
  }
}
```

If `code_overlay.enabled` is absent or false, current generated behavior must be preserved.

For MVP there is no strict mode. Overlay application is relaxed by default:

- apply overlay when matching blocks are found;
- fallback to generated code when overlay block is missing;
- write warnings/diagnostics for unresolved overlay blocks;
- do not fail generation only because an overlay block was not applied.

A strict/fail-on-missing mode may be added later as a separate task if needed.

---

# Overlay Extraction Tool

Example CLI:

```bash
extract_code_overlay --extension-path input/etalonCode --output configs/code_overlay.json
```

Responsibilities:

- traverse reference extension XML dump;
- load active project config from `configs/config.json`;
- read `CommonTemplates/упо_SearchResult/Ext/Template.txt` under configured `input_path`;
- export code only for objects requested by that SearchResult template;
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
- SessionModule

Future extensions are allowed.

---

# Overlay Block Identity

Block identity must be deterministic and stable.

Examples:

```text
Catalog.Номенклатура:ObjectModule
Catalog.Номенклатура.Form.ФормаЭлемента:FormModule
CommonModule.упо_Utils:CommonModule
Session:SessionModule
```

Matching rules must be centralized and covered by tests.

---

# Overlay File Structure

Example:

```json
{
  "version": 1,
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
- change Native/Adopted selection;
- read the full reference XML dump during main generation;
- fail generation only because an overlay block was not applied in MVP.

## Overlay must:

- override only code content;
- preserve generated structure;
- support forward regeneration;
- support Git diff/review workflow;
- use `configs/code_overlay.json` as the prepared runtime artifact;
- fallback to generated code for unresolved overlay blocks in MVP.

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

- extraction from default `input/etalonCode` dump path;
- overlay artifact written to `configs/code_overlay.json`;
- main pipeline reads `configs/code_overlay.json`;
- main pipeline does not read `input/etalonCode` directly;
- modified form module transfer;
- generated fallback when overlay block is missing;
- warning/diagnostic when overlay block is unresolved;
- generation does not fail only because an overlay block is unresolved;
- renamed object behavior;
- removed form behavior;
- merge target object support.
