# Индекс контекста агента

Цель — минимизировать startup-context и направлять агента только к нужным semantic layers.

## Читать первым

1. `.codex/agent-start.md`
2. Этот индекс

## Machine-readable task routes

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

## Stable knowledge

### XML subdomains

- `.codex/context/classification.md`
- `.codex/context/forms.md`
- `.codex/context/target-merge.md`
- `.codex/context/cleanup.md`
- `.codex/context/terms.md`

### Architecture

- `docs/architecture.md`

### Debugging cookbook

- `docs/debugging.md`

## Execution layer

- `.codex/skills/run-conversion.md`
- `.codex/skills/check-run-status.md`
- `.codex/skills/cleanup-run-tails.md`
- `.codex/skills/use-github-issue-task.md`

## Operational state

- `.codex/context/current-state.md`
- `.codex/handoff.md`

## Anti-patterns

- `.codex/patterns/dangerous-refactors.md`

## Правила маршрутизации

- Использовать task-route файл вместо ручного выбора контекста.
- Использовать GitHub Issue как основной контейнер задачи для Codex.
- Не читать весь XML-domain подряд.
- Не читать operational state для обычной задачи.
- Не читать debugging cookbook без debugging-задачи.
- Не читать orchestration skills без run/monitoring-задачи.

## Куда класть новую информацию

- XML/classification rules → `.codex/context/*`
- Task workflow → `.codex/tasks/*`
- Operational state → `.codex/context/current-state.md` или `.codex/handoff.md`
- Anti-patterns/regressions → `.codex/patterns/*`
- Human-facing docs → `README.md` или `docs/*`
