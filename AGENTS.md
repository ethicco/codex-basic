# Repository Guidelines

## Project Layout

This repository contains two independently runnable applications: `frontend/` and `backend/`. Their application-specific conventions are defined in the `AGENTS.md` file within each directory.

Keep dependencies, tooling, build output, and source changes scoped to the application they belong to. Do not commit generated build artifacts or dependency directories.

## Commit & Pull Request Guidelines

The existing history is minimal and uses a concise, sentence-style initial commit. Keep commits short and action-oriented (for example, `Add uptime check endpoint`). Pull requests should explain the change, identify affected areas (`frontend` or `backend`), include validation commands and results, link related issues when applicable, and attach screenshots or recordings for visible UI changes.

### Pull Request Workflow

After completing work on a feature or fix branch, create a pull request into
`main`. Before creating it:

1. Verify the working tree is clean and review the branch diff against `main`.
2. Run the relevant validation commands for every affected application and
   record their results.
3. Push the current branch and create a PR with a Conventional Commits title
   matching the primary change (for example, `feat: add monitor creation`).
4. Use a detailed PR body containing these sections: `Summary`, `Changes`,
   `Validation`, `Security and architecture`, `Affected areas`, and
   `Screenshots` (or state that no visual evidence is applicable).
5. Include links to related issues when they exist. Do not create, merge, or
   close a PR without the user's explicit request.

For visible UI changes, attach screenshots or a recording to the PR. If the
repository has no configured GitHub remote or the GitHub account is not
connected, report that blocker and preserve the prepared title and body for
publication after access is available.

## GitHub Flow

- Перед началом каждой задачи создавайте отдельную ветку от актуальной `main`.
- Имена веток: `feat/<nazvanie-fichi-translitom>` для новой функциональности и `fix/<opisanie-ispravleniya-translitom>` для исправлений. Используйте нижний регистр и дефисы между словами.
- После завершения работы создавайте Pull Request в `main`; изменения вливаются только через PR.

## Security & Configuration

Keep secrets and local configuration out of Git; use environment variables and document required names in the relevant README. Review dependency-lockfile changes carefully, and never expose credentials in client-side code or committed logs.

## Коммиты

- Формат: Conventional Commits (feat, fix, refactor, test, docs, chore)
- Заголовок до 72 символов, в повелительном наклонении
- Без эмодзи, без "significantly improved" и прочей воды
- Тело - только если нужно объяснить "почему", а не "что"
- Один логический шаг - один коммит
