# Task route: classification bug

## When to use

Use when:
- excluded-object remained in extension
- object disappeared unexpectedly
- `RefDrivenInclusion` behaves incorrectly
- `Native` vs `AdoptedStub` decision is wrong
- subsystem/include/exclude priority is suspicious

## Required docs

1. `.codex/agent-start.md`
2. `.codex/context/classification.md`
3. `.codex/context/terms.md`
4. optionally `.codex/context/target-merge.md`

## Do not read by default

- forms rules
- debugging cookbook
- run orchestration
- handoff history

## Expected work pattern

1. Determine top-level decision path.
2. Check soft-exclude vs hard-exclude.
3. Check promotion source.
4. Verify whether promotion source is legal.
5. Verify target-sensitive constraints if `target.xml_dump` exists.

## Output checklist

- incorrect decision point
- incorrect promotion source
- violated priority rule
- minimal classification fix
