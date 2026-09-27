# Repository Guidelines

## Project Layout

This repository contains two independently runnable applications: `frontend/` and `backend/`. Their application-specific conventions are defined in the `AGENTS.md` file within each directory.

Keep dependencies, tooling, build output, and source changes scoped to the application they belong to. Do not commit generated build artifacts or dependency directories.

## Commit & Pull Request Guidelines

The existing history is minimal and uses a concise, sentence-style initial commit. Keep commits short and action-oriented (for example, `Add uptime check endpoint`). Pull requests should explain the change, identify affected areas (`frontend` or `backend`), include validation commands and results, link related issues when applicable, and attach screenshots or recordings for visible UI changes.

## Security & Configuration

Keep secrets and local configuration out of Git; use environment variables and document required names in the relevant README. Review dependency-lockfile changes carefully, and never expose credentials in client-side code or committed logs.

## Коммиты

- Формат: Conventional Commits (feat, fix, refactor, test, docs, chore)
- Заголовок до 72 символов, в повелительном наклонении
- Без эмодзи, без "significantly improved" и прочей воды
- Тело - только если нужно объяснить "почему", а не "что"
- Один логический шаг - один коммит
