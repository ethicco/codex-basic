# Repository Guidelines

## Project Layout

This repository contains two independently runnable applications:

- `frontend/`
- `backend/`

Their application-specific conventions are defined in the `AGENTS.md` file within each directory.

Keep dependencies, tooling, build output, and source changes scoped to the application they belong to. Do not commit generated build artifacts or dependency directories.

## Commit & Pull Request Guidelines

The existing history is minimal and uses a concise, sentence-style initial commit. Keep commits short and action-oriented (for example, `Add uptime check endpoint`). Pull requests should explain the change, identify affected areas (`frontend` or `backend`), include validation commands and results, link related issues when applicable, and attach screenshots or recordings for visible UI changes.

### Pull Request Workflow

Details in file: docs/rules/pr.md

## When starting a new task use GitHub Flow

Details in file: docs/rules/github-flow.md

## Security & Configuration

Keep secrets and local configuration out of Git; use environment variables and document required names in the relevant README. Review dependency-lockfile changes carefully, and never expose credentials in client-side code or committed logs.

## Коммиты

Use rules in file: docs/rules/commit.md
