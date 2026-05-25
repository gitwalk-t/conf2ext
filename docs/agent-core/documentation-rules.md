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

---

### docs/agent-core/*

Общие stable policies:

- review rules;
- conversion rules;
- documentation rules;
- issue lifecycle.

---

## Context tiers

### Tier 0

Always-read:

- `AGENTS.md`

### Tier 1

Task bootstrap:

- relevant skill;
- required docs only.

### Tier 2

On-demand domain context.

---

## Anti-patterns

- recursive reread docs;
- duplicated workflow text;
- copying same instructions into multiple skills;
- mixing operational state with stable architecture;
- large prose-oriented skills.
