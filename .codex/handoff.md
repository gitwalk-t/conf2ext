# Передача контекста

Файл предназначен только для временного operational state.

Постоянные XML/classification rules больше не дублируются здесь и живут в:

```text
.codex/context/xml-rules.md
```

Термины:

```text
.codex/context/terms.md
```

## Текущее состояние

- Основной рискованный слой:

```text
internal/utils/xmlutil/change.go
```

- Главная практическая цель:
  - стабильно получать валидную сборку extension из текущего конфига.

- Практический bottleneck:
  - ручные прогоны и анализ `stderr` + `v8_src*`.

## Где продолжать расследование

1. `internal/utils/xmlutil/change.go`
2. `configs/config.json`
3. Последние `run-*.stderr.log`
4. Последний `output/_tmp/v8_src*`
5. `output/_log`

## Рекомендуемый operational cycle

```powershell
go build ./...
go test ./...
go run . --config .\\configs\\config.json
```

Дальше:
- разобрать stderr-log;
- разобрать соответствующий `v8_src*`;
- сделать минимальное XML/classification изменение.

## Что НЕ хранить здесь

- стабильные XML rules;
- glossary/terms;
- orchestration instructions;
- debugging encyclopedia;
- архитектурные описания.
