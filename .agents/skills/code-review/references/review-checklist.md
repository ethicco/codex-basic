# Review checklist

Read every section. Apply **Universal**, **Security**, and **Tests** to every diff. Apply the remaining sections only when their file types or behavior are affected, and mark skipped sections `N/A` in the final review.

## Universal

- Good: the diff is scoped to the stated behavior, removes superseded code, uses clear domain names, and keeps functions and modules focused. Public Go symbols have useful doc comments; comments explain decisions rather than restating code.
- Bad: unrelated refactors obscure behavior; duplicated or unreachable code remains; names hide units, ownership, or side effects; comments are stale or explain the obvious.
- Check error paths, resource cleanup, boundaries, concurrency assumptions, and whether the change preserves existing callers.

## Security

- Good: secrets come from server-side environment variables; credentials and refresh tokens remain out of source, logs, URLs, client components, browser storage, and `NEXT_PUBLIC_*`; authorization is verified for every protected resource; untrusted input is validated at its boundary.
- Bad: committed keys, passwords, tokens, connection strings, or private URLs; bypassable authorization; reflected unsanitized input; raw database queries built from request values; user-controlled filesystem paths; verbose internal errors exposed to clients.
- For backend monitor, URL, redirect, network, upload, or database changes, verify SSRF defenses, file size/type/path restrictions, parameterized queries, and least-privilege behavior. For browser mutations, verify authentication, CSRF posture, cookie flags, and that sensitive responses are not cached.

## Go backend

- Good: `cmd/server` only composes dependencies; `internal/server` owns HTTP decoding, status mapping, and response shaping; domain and persistence concerns remain in focused `internal` packages behind interfaces where the transport layer needs them. Handlers pass `r.Context()` through I/O, validate requests before side effects, and preserve meaningful errors internally while returning safe responses.
- Bad: SQL, configuration lookup, or token implementation leaks into handlers; HTTP types leak into domain/persistence packages; global mutable state replaces dependency injection; contexts are discarded; status codes or error envelopes drift across handlers.
- For changed routes, verify registration, method/path, authentication and ownership checks, request limits, all success and error statuses, Swaggo annotations, and regenerated `backend/docs/{docs.go,swagger.json,swagger.yaml,redoc.html}`.
- For migrations and stores, verify forward compatibility, constraints and indexes, transaction/error handling, pagination order, and tests covering the new persistence behavior.

## Next.js frontend

- Good: App Router routes/layouts/components live in `src/app`; shared helpers live in focused `src/lib` modules; client components are marked only when browser state or effects are required; component names are PascalCase, variables/functions camelCase, and route folders lowercase. UI uses CSS Modules and preserves semantic HTML and accessible labels/errors.
- Bad: server secrets or backend tokens cross into client code; browser code calls the Go service with credentials directly; route handlers blindly forward malformed input; client/server boundaries are blurred; duplicated proxy/auth logic diverges; `dangerouslySetInnerHTML` receives untrusted content.
- For route handlers and auth changes, check that the auth proxy keeps tokens in `HttpOnly`, `Secure`-in-production, `SameSite` cookies; validates backend payloads; handles expired sessions and 4xx/5xx outcomes safely; and uses `no-store` for sensitive data.

## API contract and cross-application behavior

- Good: frontend request/response shapes, status handling, authentication, and error envelopes agree with the Go route and OpenAPI contract. Contract changes update both consumers and documentation.
- Bad: one side accepts or emits a different field/type/status; UI assumes success or leaks backend implementation errors; a backend route changes without Swaggo and generated OpenAPI updates.

## Tests

- Good: changed behavior has focused tests beside the affected Go package or frontend behavior; tests cover the happy path and relevant invalid, authorization, and failure paths. Existing tests explain intent and are deterministic.
- Bad: only compilation is tested; a new branch, validator, authorization rule, or error mapping has no behavioral test; tests depend on time, network, or shared state without control.
- The frontend currently has no configured test command. Treat a missing focused frontend test for non-trivial changed behavior as review evidence to assess, not a silent pass.

## Dependencies and configuration

- Good: dependency and lockfile changes are intentional, minimal, and compatible with the affected application; configuration names and defaults are documented; generated artifacts are updated only when their inputs change.
- Bad: broad version churn, unused dependencies, secrets in configuration examples, unchecked runtime configuration, committed build outputs or dependency directories.
