# Skill: обновление репозитория из git

## Когда использовать

Используй этот skill, когда пользователь просит:
- обновить репозиторий из git;
- подтянуть последние изменения;
- выполнить `git pull`;
- синхронизировать локальный worktree с remote;
- проверить, что локальная копия актуальна.

## Назначение

Skill отвечает только за безопасное обновление текущего worktree из remote git.

Skill не запускает конвертацию.
Skill не делает cleanup процессов.
Skill не меняет бизнес-логику.
Skill не исправляет merge conflicts без отдельного запроса.

## Обязательный контекст

Перед началом определить:
- текущий worktree / `cwd`;
- текущую ветку;
- remote tracking branch;
- наличие незакоммиченных изменений;
- наличие untracked файлов.

Если рядом есть несколько worktree одного репозитория, нельзя молча обновлять “похожую” копию. Сначала зафиксируй, какой `cwd` обновляется.

## Проверки перед обновлением

Выполнить:

```powershell
git status --short
git branch --show-current
git remote -v
git rev-parse --abbrev-ref --symbolic-full-name @{u}
```

Если upstream отсутствует, не делать `git pull` вслепую. Сообщить пользователю, что upstream не настроен.

## Dirty worktree policy

Если есть незакоммиченные изменения:

1. Не выполнять `git pull` автоматически.
2. Сообщить список измененных файлов.
3. Предложить безопасные варианты:
   - закоммитить изменения;
   - stash;
   - отменить локальные изменения;
   - обновить только после ручного решения пользователя.

Запрещено автоматически делать:
- `git reset --hard`;
- `git clean -fd`;
- `git stash` без явного согласия;
- force checkout.

## Стандартный safe update flow

Если worktree clean:

```powershell
git fetch --all --prune
git status --short
git pull --ff-only
```

Почему `--ff-only`:
- не создает merge commit;
- не смешивает update с конфликтным merge;
- безопасно показывает, если локальная ветка разошлась с remote.

## Если pull невозможен

Если `git pull --ff-only` не проходит:

- не делать merge/rebase автоматически;
- показать причину;
- вывести:

```powershell
git status --short
git log --oneline --decorate --graph --max-count=20 --all
```

Затем сообщить пользователю варианты:
- rebase;
- merge;
- reset to remote;
- manual conflict resolution.

## После успешного обновления

Выполнить:

```powershell
git status --short
git log --oneline --decorate --max-count=5
```

Если проектные инструкции требуют проверки после обновления, выполнить минимум:

```powershell
go build ./...
go test ./...
```

Если пользователь просил только синхронизировать репозиторий, build/test можно не запускать без отдельной необходимости.

## Финальный ответ пользователю

Указать:
- worktree / `cwd`;
- ветку;
- upstream;
- результат `fetch`;
- результат `pull --ff-only`;
- old HEAD;
- new HEAD;
- изменился ли worktree;
- есть ли незакоммиченные изменения после обновления;
- запускались ли `go build ./...` и `go test ./...`.

## Запреты

- Не делать destructive commands без явного запроса.
- Не обновлять не тот worktree.
- Не скрывать dirty state.
- Не решать merge conflicts автоматически.
- Не делать force push/fetch/reset.
- Не запускать долгий прогон конвертации как часть git update.
