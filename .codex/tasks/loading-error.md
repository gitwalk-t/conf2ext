# Task route: loading/XDTO error

## When to use

Use when:
- extension loading fails
- XDTO error appears
- QName/type mismatch appears
- `Неизвестный объект метаданных`
- predefined/type-bearing XML is suspicious

## Required docs

1. `.codex/agent-start.md`
2. `.codex/context/cleanup.md`
3. optionally `.codex/context/forms.md`
4. `docs/debugging.md` for known patterns

## Do not read by default

- classification rules
- target merge rules
- orchestration skills
- operational handoff

## Expected work pattern

1. Identify failing XML path.
2. Determine whether problem is:
   - dangling metadata reference;
   - broken type-bearing XML;
   - command cleanup issue;
   - subsystem cleanup issue.
3. Compare source XML vs dump XML.
4. Make minimal cleanup fix.

## Output checklist

- failing XML file
- broken metadata/type path
- cleanup stage responsible
- minimal cleanup fix
