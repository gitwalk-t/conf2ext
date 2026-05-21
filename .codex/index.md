# Индекс контекста агента

Цель файла — не расширять обязательный контекст, а сократить его. Перед началом задачи выбери минимальный набор документов ниже.

## Всегда читать первым

1. `.codex/agent-start.md`
2. Этот индекс

## Маршруты по типам задач

| Задача | Читать | Не читать без необходимости |
|---|---|---|
| Быстро понять проект | `.codex/agent-start.md` | `docs/debugging.md`, полный `.codex/context.md`, handoff |
| Ревью коммита или архитектурная оценка | `.codex/agent-start.md`, `docs/architecture.md`, при необходимости `.codex/context/terms.md` | run skills, debugging |
| Правка XML/classification-логики | `.codex/agent-start.md`, `.codex/context/xml-rules.md`, `.codex/context/terms.md`, релевантный фрагмент `docs/architecture.md` | handoff целиком, debugging целиком |
| Ошибка загрузки расширения | `.codex/agent-start.md`, `docs/debugging.md`, затем точечно `.codex/context/xml-rules.md` | architecture целиком |
| Запуск или проверка прогона | `.codex/skills/run-conversion.md` или `.codex/skills/check-run-status.md` | общий context, architecture |
| Продолжение незавершенной отладки | `.codex/context/current-state.md`, `.codex/handoff.md`, затем релевантный skill/debugging | README целиком |
| Обновление терминологии | `.codex/context/terms.md` | debugging, run skills |
| Обновление пользовательской документации | `README.md`, `docs/technical-spec.md`, релевантные `docs/*` | `.codex/handoff.md` |

## Правило добавления новой информации

- Долгоживущее правило XML/classification — в `.codex/context/xml-rules.md`.
- Термин или словарь — в `.codex/context/terms.md`.
- Текущий временный статус расследования — в `.codex/context/current-state.md` или `.codex/handoff.md`.
- Порядок запуска/мониторинга — в `.codex/skills/*`.
- Человеко-читаемое описание проекта — в `README.md` или `docs/*`.

## Правило экономии токенов

Не читай все документы подряд. Если задача не требует исторического статуса, не открывай `.codex/handoff.md`. Если задача не связана с запуском 1С, не открывай run skills. Если задача не связана с XML/classification, не открывай полный набор XML-правил.
