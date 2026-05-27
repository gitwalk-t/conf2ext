# Skill: GitHub operations

## Purpose
Unified GitHub workflow.

## Rules
- Primary integration: GitHub connector.
- `gh` CLI is optional.
- Prefer connector APIs for:
  - issues
  - issue comments
  - PRs
  - reviews

## Standard flow
1. Read issue/PR.
2. Sync local repo.
3. Implement minimal diff.
4. Run minimal validation.
5. Commit.
6. Push.
7. Create/update PR.
8. Post final issue comment.

## Final issue comment
- summary
- validation result
- branch
- commit/PR
- remaining risks if any

## Fallback
If connector write operations are unavailable:
- do not retry blindly;
- report exact manual external step.
