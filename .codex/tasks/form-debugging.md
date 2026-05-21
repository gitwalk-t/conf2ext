# Task route: form debugging

## When to use

Use when the issue mentions:
- `Неверный путь к данным`
- `DataPath`
- dynamic list
- form field
- form command
- `Form.xml`

## Required docs

1. `.codex/agent-start.md`
2. `.codex/context/forms.md`
3. `.codex/context/cleanup.md`
4. `docs/debugging.md` only for troubleshooting pattern lookup

## Do not read by default

- `.codex/handoff.md`
- full architecture
- target merge rules
- run orchestration skills

## Expected work pattern

1. Identify owner metadata object.
2. Compare source `Form.xml` vs dump `Form.xml`.
3. Check whether referenced metadata path exists in final composition.
4. Check dynamic-list contract if target object is non-Native.
5. Make minimal cleanup/classification change.

## Output checklist

- affected form path
- missing/broken `DataPath` or command path
- decision: form cleanup bug vs classification bug
- minimal code/documentation change
