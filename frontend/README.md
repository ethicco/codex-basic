# Frontend

Next.js UI for Uptime authentication. The browser never receives raw JWT values:
local Next.js route handlers proxy authentication calls to the Go backend and
store access and refresh tokens in session-only `HttpOnly`, `SameSite=Lax`
cookies.

## Run

Start the backend first, then run:

```bash
export BACKEND_URL=http://localhost:8080
npm run dev
```

`BACKEND_URL` is server-only and defaults to `http://localhost:8080` for local
development. Open [http://localhost:3000](http://localhost:3000) to register
or sign in.

## Authentication proxy

| Method | Path | Behavior |
| --- | --- | --- |
| POST | `/api/auth/register` | Proxies registration and sets HttpOnly session cookies |
| POST | `/api/auth/login` | Proxies login and sets HttpOnly session cookies |
| POST | `/api/auth/refresh` | Uses refresh cookie and rotates both cookies |
| POST | `/api/auth/logout` | Revokes backend session and clears cookies |
| GET | `/api/auth/session` | Uses access cookie to load the profile and refreshes only when needed |
| GET/POST/PATCH | `/api/profile` | Proxies profile read, creation, and name updates using the access cookie |

Responses from register/login/refresh include only the public user object and
access-token lifetime; token strings are not returned to browser JavaScript.
After a successful login or registration, the user is redirected to `/dashboard`.
The profile menu opens `/profile`; the edit screen at `/profile/edit` currently
allows changing only the profile name.

## Validation

```bash
npm run lint
npm run build
```
