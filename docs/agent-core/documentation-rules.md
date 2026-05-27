# Documentation rules

## Goals

- Минимизировать startup-context агента.
- Исключить duplication между docs.
- Поддерживать deterministic routing.
- Разделять human docs и agent docs.

---

## Single source of truth

Каждое правило должно иметь один canonical document.

Другие документы:
- не дублируют правило;
- только ссылаются.

GitHub workflow policy:

```text
.codex/skills/github-operations.md
```

---

## Layering

### README.md

Только human-facing информация:
- назначение проекта;
- quick start;
- high-level architecture;
- ограничения проекта.

README не должен содержать:

- agent workflow;
- orchestration;
- issue lifecycle;
- Codex prompts;
- operational state.

---

### AGENTS.md

Только:

- bootstrap;
- routing;
- hard constraints;
- minimal invariants.

AGENTS.md должен оставаться коротким.

---

### .codex/skills/*

Только task execution.

Skills не должны:

- дублировать policy;
- содержать архитектурный обзор;
- повторять git/update workflow.

GitHub workflow details не копировать между skills.

Использовать:

```text
.codex/skills/github-operations.md
```

---

### docs/agent-core/*

Общие stable policies:

- review rules;
- conversion rules;
- documentation rules;
- issue lifecycle.

---

# Strict context tiers

## Tier 0 — always-read

Читается всегда:

- `AGENTS.md`
- `.codex/agent-start.md`
- `.codex/index.md`

Tier 0 должен оставаться:
- коротким;
- стабильным;
- без operational history.

---

## Tier 1 — task bootstrap

Читается только после выбора task route.

Разрешено:
- один task-route;
- минимальный набор domain docs.

Примеры:

### Classification bug

```text
.codex/tasks/classification-bug.md
.codex/context/classification.md
.codex/context/terms.md
```

### Form debugging

```text
.codex/tasks/form-debugging.md
.codex/context/forms.md
.codex/context/cleanup.md
```

Tier 1 routes должны явно запрещать unrelated domains.

---

## Tier 2 — on-demand context

Читается только при подтвержденной необходимости.

Примеры:
- `target-merge.md`
- дополнительные debugging docs
- deep architecture docs

Tier 2 нельзя читать proactively.

---

## Tier 3 — operational state

Не читать по умолчанию.

Использовать только для:
- handoff;
- resume;
- debugging active run.

Примеры:

```text
.codex/context/current-state.md
.codex/handoff.md
```

---

## Required loading flow

```text
Tier 0
  ↓
ONE task route
  ↓
Tier 1 only
  ↓
Tier 2 only if required
  ↓
Tier 3 almost never
```

---

## Anti-patterns

- recursive reread docs;
- duplicated workflow text;
- copying same instructions into multiple skills;
- mixing operational state with stable architecture;
- large prose-oriented skills;
- loading multiple unrelated routes simultaneously;
- reading full `.codex/context/*` without task justification.