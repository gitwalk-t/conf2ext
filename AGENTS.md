# AGENTS.md

## Стартовый маршрут

Всегда сначала читать:

1. `.codex/index.md`
2. `.codex/agent-start.md`

Дальше читать только документы, относящиеся к текущей задаче.

## Быстрый routing

- XML/classification logic:
  - `.codex/context/xml-rules.md`
  - `.codex/context/terms.md`
  - `docs/architecture.md`

- Debugging и ошибки загрузки:
  - `docs/debugging.md`

- Запуск и monitoring:
  - `.codex/skills/run-conversion.md`
  - `.codex/skills/check-run-status.md`
  - `.codex/skills/cleanup-run-tails.md`

- GitHub Issue workflow:
  - `.codex/skills/use-github-issue-task.md`

- Обновление локального репозитория из git:
  - `.codex/skills/update-repo-from-git.md`

- Временный operational context:
  - `.codex/context/current-state.md`
  - `.codex/handoff.md`

## Основные правила

- Предпочитай минимальный diff.
- Не делать широкий рефакторинг `internal/utils/xmlutil/change.go` без прямой необходимости.
- Не добавлять зависимости без необходимости.
- Не менять XML/business rules ради “красоты”.
- `excluded_*` — soft exclude.
- `forbidden_*` — hard exclude.
- BSL не является источником classification rules.
- `configs/config.json` считать активным локальным конфигом по умолчанию.

## Прогоны

- Перед новым запуском использовать:

```text
.codex/skills/cleanup-run-tails.md
```

- Для проверки статуса использовать:

```text
.codex/skills/check-run-status.md
```

- Для orchestration использовать:

```text
.codex/skills/run-conversion.md
```

## Git update

Для любого обновления локального worktree из git использовать:

```text
.codex/skills/update-repo-from-git.md
```

Не дублировать git update orchestration в других skills.

## Что проверять перед завершением

```powershell
go build ./...
go test ./...
```

Если изменились долгоживущие XML/classification rules:
- обновить `.codex/context/xml-rules.md`
- при необходимости обновить `docs/architecture.md`
- при необходимости обновить `.codex/decisions.md`
