---
name: Codex task
about: Задача для Codex-агента
title: "[codex] "
labels: codex
assignees: ""
---

## Цель

Кратко опишите, какой результат нужен.

## Контекст

Минимальный список файлов, документов, логов или ссылок, которые Codex должен прочитать.

Пример:

```text
.codex/index.md
.codex/tasks/classification-bug.md
internal/utils/xmlutil/change.go
output/_log/<run>/errors.log
```

## Тип задачи

Выберите один вариант:

- [ ] bugfix
- [ ] performance-review
- [ ] performance-fix
- [ ] docs
- [ ] refactoring
- [ ] run/debug
- [ ] architecture-review

## Ограничения

Что нельзя менять, какие инварианты сохранить.

Пример:

```text
- Минимальный diff.
- Не делать широкий refactoring internal/utils/xmlutil/change.go.
- Не менять XML/business rules без явного обоснования.
- Не добавлять зависимости без необходимости.
```

## Ожидаемый план

1. Что проверить.
2. Что изменить.
3. Какие тесты или прогоны выполнить.

## Definition of Done

- [ ] изменения внесены минимальным diff
- [ ] `go build ./...` выполнен или причина пропуска указана
- [ ] `go test ./...` выполнен или причина пропуска указана
- [ ] документация обновлена, если изменились долгоживущие правила
- [ ] результат зафиксирован commit/PR
- [ ] в issue оставлен финальный отчет

## Дополнительные заметки

Любые дополнительные условия, ссылки на связанные issue/PR/commits или known risks.
