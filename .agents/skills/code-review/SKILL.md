---
name: code-review
description: Review a repository diff for the Go backend and Next.js frontend, with targeted static checks and prioritized findings.
---

# Project code review

Review the requested commit, range, branch, or working-tree diff. Do not change production code as part of review.

## Establish the review target

Use the range supplied by the user. For a feature branch, compare it with `main` using its merge base. For uncommitted work, inspect `git status --short`, `git diff HEAD`, and untracked files. Record the commits with `git log --oneline <base>..HEAD`, summarize changed paths with `git diff --stat`, then read the full diff with rename detection.

Before inspecting the diff, run:

```bash
.agents/skills/code-review/scripts/run-static-checks.sh "<base>...<head>"
```

Omit the argument for the current working tree. The script runs checks only for applications whose files occur in the selected diff. Report its results even when a check fails; continue the review unless the diff cannot be read.

## Architecture to preserve

- `backend/cmd/server` composes the Go application. `backend/internal/server` is the HTTP transport layer: it registers routes, translates requests and responses, and depends on domain-facing interfaces. Authentication, configuration, monitor validation, database migrations, and user persistence belong in their focused `internal/*` packages. Keep reusable code in `backend/pkg` only when it is genuinely public to other modules.
- `frontend/src/app` is the Next.js App Router surface: pages, layouts, UI, and browser-facing route handlers. `frontend/src/lib` contains focused shared helpers and server-side backend access. The auth proxy is the trust boundary between the browser and Go API: tokens stay in `HttpOnly` cookies and must not reach client components or browser storage.
- Browser UI calls Next.js route handlers; they validate and proxy requests to the Go API. When this contract changes, check both sides and the backend OpenAPI annotations and generated artifacts.

## Review procedure

Read [the checklist](references/review-checklist.md) in full. Apply the universal, security, and test sections to every review; apply backend, frontend, API, migration, or dependency sections only when matching files or behavior are affected. Mark irrelevant sections as `N/A` in the review summary.

Report only concrete findings introduced by the review target. For each finding, include priority, file and line, evidence, impact, and the smallest safe correction. Do not turn personal preferences or pre-existing code into findings.

Use these priorities:

- `P0` — exploitable secret or security flaw, data loss/corruption, outage, or a broken critical flow; fix before merge.
- `P1` — likely functional, authorization, contract, or data-integrity defect; fix before merge.
- `P2` — meaningful reliability, maintainability, test, or architectural issue; fix in this change when practical.
- `P3` — low-risk quality issue with a clear benefit; schedule as follow-up if needed.
- `PX` — non-blocking suggestion or question; never present it as a required fix.

Finish with: reviewed range and changed behavior, static-check results, checklist sections applied or `N/A`, findings ordered P0 through PX, and remaining risks or `No findings`.
