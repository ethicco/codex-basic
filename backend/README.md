# Backend

Go HTTP service with PostgreSQL-backed email/password authentication, JWT access
tokens, and rotating refresh tokens.

## Run

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/uptime?sslmode=disable'
export AUTH_JWT_SECRET="$(openssl rand -base64 48)"
go run ./cmd/server
```

The service applies embedded SQL migrations at startup and listens on `:8080`.

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | required | PostgreSQL connection string |
| `AUTH_JWT_SECRET` | required | Signing key, at least 32 bytes |
| `AUTH_ACCESS_TOKEN_TTL` | `15m` | Access-token lifetime |
| `AUTH_REFRESH_TOKEN_TTL` | `720h` | Refresh-token lifetime; must exceed access TTL |
| `SERVER_ADDR` | `:8080` | Listen address |

## Authentication API

All endpoints accept `application/json` and return JSON errors in the form
`{"error":{"code":"…","message":"…"}}`.

| Method | Path | Request | Success |
| --- | --- | --- | --- |
| POST | `/api/auth/register` | `{"email":"person@example.com","password":"at-least-12-characters"}` | `201` and token pair |
| POST | `/api/auth/login` | same as registration | `200` and token pair |
| POST | `/api/auth/refresh` | `{"refresh_token":"<JWT>"}` | `200` and a new token pair |
| POST | `/api/auth/logout` | `{"refresh_token":"<JWT>"}` | `204` |
| GET | `/api/auth/me` | `Authorization: Bearer <access JWT>` | `200` and public user profile |
| GET | `/api/profile` | `Authorization: Bearer <access JWT>` | `200` and profile, or `404` if absent |
| POST | `/api/profile` | Bearer access token and `{"name":"…"}` | `201` and created profile |
| PATCH | `/api/profile` | Bearer access token and `{"name":"…"}` | `200` and updated profile |
| GET | `/api/monitors` | Bearer access token | `200` and the current user's monitors |
| POST | `/api/monitors` | Bearer access token and `{"url":"https://example.com","interval_seconds":60}` | `201` and created monitor |

A token response contains `user`, `access_token`, `refresh_token`,
`token_type` (`Bearer`), and `expires_in`. Each refresh rotates the stored
refresh-token identifier. Reusing a previous refresh token revokes only that
session. Password recovery and CORS are not part of this version.

## Validation

```bash
go test ./...
go vet ./...
go build ./...
```
