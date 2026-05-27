# AGENTS.md

## Bootstrap

Всегда читать только:

1. `.codex/agent-start.md`
2. `.codex/index.md`

Дальше читать только task-specific context.

---

## Routing

### XML/classification

- `.codex/context/xml-rules.md`
- `.codex/context/terms.md`
- `docs/architecture.md`

### Debugging/loading errors

- `docs/debugging.md`

### Conversion orchestration

- `.codex/skills/run-conversion.md`
- `.codex/skills/check-run-status.md`
- `.codex/skills/cleanup-run-tails.md`

### GitHub Issue workflow

- `.codex/skills/use-github-issue-task.md`
- `.codex/skills/github-operations.md`

### Repository update

- `.codex/skills/update-repo-from-git.md`

### Operational context

Только при handoff/debugging:

- `.codex/context/current-state.md`
- `.codex/handoff.md`

---

## Hard constraints

- Предпочитай minimal diff.
- Не делать широкий рефакторинг `internal/utils/xmlutil/change.go`.
- Не добавлять зависимости без необходимости.
- Не менять XML/business rules ради cleanup.
- Не читать весь `.codex/context/*` подряд.
- Не читать debugging docs без debugging-задачи.
- Не дублировать workflow logic между skills.

---

## Stable invariants

- `excluded_*` — soft exclude.
- `forbidden_*` — hard exclude.
- BSL не источник classification rules.
- `configs/config.json` — основной локальный config.
- GitHub connector — primary GitHub integration.
- `gh` CLI не гарантирован.

---

## Validation

```powershell
go build ./...
go test ./...
```

Если изменены XML/classification rules:

- обновить `.codex/context/xml-rules.md`
- обновить `docs/architecture.md` при необходимости