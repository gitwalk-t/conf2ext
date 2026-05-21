# Dangerous refactors

## Broad rewrite of change.go

Avoid:
- large-scale cleanup of `change.go` during active debugging;
- moving multiple decision branches simultaneously;
- mixing classification and cleanup refactor.

Reason:
- tiny XML/classification changes create wide regressions.

## Mixing stable rules and temporary investigation

Do not mix:
- stable XML rules;
- operational debugging notes;
- temporary workaround.

Use separate context layers.

## Making BSL a classification source

BSL is not part of:
- `RefDrivenInclusion`;
- classification;
- dependency graph.

## Partial form preservation

Forms are not partially preserved.

Either:
- form remains fully;
- or form is removed fully.

## Target.xml_dump as global graph

`target.xml_dump`:
- is not a global dependency graph;
- is not a universal promotion source.

Use only target-sensitive merge rules.

## Ref-driven promotion from subsystems/roles

Do not allow:
- subsystem-only promotion;
- role-only promotion.

Promotion source must be a legal metadata source.
