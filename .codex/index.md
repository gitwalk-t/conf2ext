# Индекс контекста агента

Цель:
- минимальный startup-context;
- deterministic routing;
- selective context loading.

## Читать первым

1. `.codex/agent-start.md`
2. `AGENTS.md`
3. Этот индекс

---

## Task routes

### Form debugging

```text
.codex/tasks/form-debugging.md
```

### Classification bug

```text
.codex/tasks/classification-bug.md
```

### Loading/XDTO error

```text
.codex/tasks/loading-error.md
```

---

## Stable domain context

### XML domains

- `.codex/context/classification.md`
- `.codex/context/forms.md`
- `.codex/context/target-merge.md`
- `.codex/context/cleanup.md`
- `.codex/context/terms.md`

### Architecture

- `docs/architecture.md`

### Debugging

- `docs/debugging.md`

### Shared agent policies

- `docs/agent-core/documentation-rules.md`

---

## Execution layer

- `.codex/skills/run-conversion.md`
- `.codex/skills/check-run-status.md`
- `.codex/skills/cleanup-run-tails.md`
- `.codex/skills/use-github-issue-task.md`
- `.codex/skills/update-repo-from-git.md`

---

## Operational state

Читать только при debugging/handoff:

- `.codex/context/current-state.md`
- `.codex/handoff.md`

---

## Anti-patterns

- читать весь `.codex/context/*` подряд;
- reread всей документации;
- дублирование workflow logic;
- смешивание stable docs и operational state.

---

## Context placement

- XML/classification rules → `.codex/context/*`
- Workflow/process → `.codex/skills/*`
- Shared policies → `docs/agent-core/*`
- Operational state → `.codex/context/current-state.md`
- Human-facing docs → `README.md`, `docs/*`
