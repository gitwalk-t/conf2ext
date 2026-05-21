# Контекст

Этот файл больше не является giant-context dump.

Цель — минимизировать обязательный startup-context агента.

## Читать первым

1. `.codex/index.md`
2. `.codex/agent-start.md`

## Контекст по задачам

### XML/classification logic

Читать:
- `.codex/context/xml-rules.md`
- `.codex/context/terms.md`
- `docs/architecture.md`

### Debugging и ошибки загрузки

Читать:
- `docs/debugging.md`
- затем точечно `.codex/context/xml-rules.md`

### Запуск и monitoring

Читать:
- `.codex/skills/run-conversion.md`
- `.codex/skills/check-run-status.md`
- `.codex/skills/cleanup-run-tails.md`

### Продолжение незавершенного расследования

Читать:
- `.codex/context/current-state.md`
- `.codex/handoff.md`

## Что больше не делать

- Не читать весь historical context для обычной правки.
- Не смешивать:
  - XML rules
  - debugging
  - временный operational state
  - orchestration
- Не использовать `.codex/handoff.md` как обязательный startup-context.

## Источники истины

- XML/classification rules:
  `.codex/context/xml-rules.md`

- Термины:
  `.codex/context/terms.md`

- Architecture:
  `docs/architecture.md`

- Debugging:
  `docs/debugging.md`

- Run orchestration:
  `.codex/skills/*`
