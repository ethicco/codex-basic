---
name: openapi-routes
description: Keep the backend OpenAPI specification current when an HTTP route is created, changed, or removed.
---

# Backend OpenAPI routes

Use this skill whenever a backend HTTP route or its externally observable contract changes, including a route's method, path, parameters, request body, response body, status codes, authentication, or removal.

For every affected handler, update its Swaggo annotations to describe the implemented contract. Include the summary, parameters and request body where applicable, successful response, and every documented error response. Keep the `@Router` path and HTTP method aligned with registration.

After changing a route or its annotations, run:

```bash
.codex/skills/openapi-routes/scripts/sync-openapi.sh
```

The script regenerates the Swaggo contract and then builds ReDoc from the new `swagger.yaml`.

Commit the regenerated OpenAPI contract and documentation alongside the code:

- `backend/docs/docs.go`
- `backend/docs/swagger.json`
- `backend/docs/swagger.yaml`
- `backend/docs/redoc.html`

For a removed route, remove its annotations and ensure the regenerated contract no longer contains that operation. Do not regenerate documentation for backend changes that do not affect an HTTP route or its public contract.
