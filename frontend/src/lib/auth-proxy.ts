import { NextRequest, NextResponse } from "next/server";

export const ACCESS_COOKIE = "uptime_access_token";
export const REFRESH_COOKIE = "uptime_refresh_token";

type AuthUser = { id: string; email: string; created_at: string };
type TokenPayload = { user: AuthUser; access_token: string; refresh_token: string; expires_in: number };
type SessionPayload = { user: AuthUser };
type ErrorPayload = { error?: { code?: string; message?: string } };

const backendURL = (process.env.BACKEND_URL ?? "http://localhost:8080").replace(/\/$/, "");

function cookieOptions() {
  return { httpOnly: true, path: "/", sameSite: "lax" as const, secure: process.env.NODE_ENV === "production" };
}

function clearAuthCookies(response: NextResponse) {
  response.cookies.set(ACCESS_COOKIE, "", { ...cookieOptions(), maxAge: 0 });
  response.cookies.set(REFRESH_COOKIE, "", { ...cookieOptions(), maxAge: 0 });
}

function setAuthCookies(response: NextResponse, payload: TokenPayload) {
  // No maxAge/expires: both cookies are discarded when the browser session ends.
  response.cookies.set(ACCESS_COOKIE, payload.access_token, cookieOptions());
  response.cookies.set(REFRESH_COOKIE, payload.refresh_token, cookieOptions());
}

async function backendRequest(path: string, body: unknown) {
  try {
    const response = await fetch(`${backendURL}${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      cache: "no-store",
    });
    const payload: unknown = await response.json().catch(() => ({}));
    return { response, payload };
  } catch {
    return null;
  }
}

async function backendSessionRequest(accessToken: string) {
  try {
    const response = await fetch(`${backendURL}/api/auth/me`, {
      headers: { Authorization: `Bearer ${accessToken}` },
      cache: "no-store",
    });
    const payload: unknown = await response.json().catch(() => ({}));
    return { response, payload };
  } catch {
    return null;
  }
}

type AuthorizedMethod = "GET" | "POST" | "PATCH" | "DELETE";

async function backendAuthorizedRequest(path: string, method: AuthorizedMethod, accessToken: string, body?: unknown) {
  try {
    const response = await fetch(`${backendURL}${path}`, {
      method,
      headers: {
        Authorization: `Bearer ${accessToken}`,
        ...(body === undefined ? {} : { "Content-Type": "application/json" }),
      },
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
      cache: "no-store",
    });
    const payload: unknown = await response.json().catch(() => ({}));
    return { response, payload };
  } catch {
    return null;
  }
}

async function backendAvatarRequest(method: "GET" | "POST", accessToken: string, body?: FormData) {
  try {
    return await fetch(`${backendURL}/api/profile/avatar`, {
      method,
      headers: { Authorization: `Bearer ${accessToken}` },
      ...(body === undefined ? {} : { body }),
      cache: "no-store",
    });
  } catch {
    return null;
  }
}

async function refreshTokenPair(request: NextRequest) {
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
  if (!refreshToken) return null;
  const result = await backendRequest("/api/auth/refresh", { refresh_token: refreshToken });
  if (!result || !result.response.ok || !isTokenPayload(result.payload)) return null;
  return result.payload;
}

function backendError(status: number, payload: unknown) {
  const error = payload as ErrorPayload;
  return NextResponse.json(
    { error: error.error ?? { code: "backend_unavailable", message: "Сервис авторизации временно недоступен" } },
    { status },
  );
}

function isTokenPayload(payload: unknown): payload is TokenPayload {
  if (!payload || typeof payload !== "object") return false;
  const candidate = payload as Partial<TokenPayload>;
  return typeof candidate.access_token === "string" && typeof candidate.refresh_token === "string" && typeof candidate.expires_in === "number" && Boolean(candidate.user && typeof candidate.user.id === "string" && typeof candidate.user.email === "string");
}

function isSessionPayload(payload: unknown): payload is SessionPayload {
  if (!payload || typeof payload !== "object") return false;
  const candidate = payload as Partial<SessionPayload>;
  return Boolean(candidate.user && typeof candidate.user.id === "string" && typeof candidate.user.email === "string");
}

export async function handleCredentials(request: NextRequest, backendPath: "/api/auth/register" | "/api/auth/login") {
  const credentials: unknown = await request.json().catch(() => null);
  if (!credentials || typeof credentials !== "object") return NextResponse.json({ error: { code: "invalid_request", message: "Введите email и пароль" } }, { status: 400 });
  const result = await backendRequest(backendPath, credentials);
  if (!result) return backendError(503, {});
  if (!result.response.ok) return backendError(result.response.status, result.payload);
  if (!isTokenPayload(result.payload)) return backendError(502, {});
  const response = NextResponse.json({ user: result.payload.user, expires_in: result.payload.expires_in }, { status: result.response.status });
  setAuthCookies(response, result.payload);
  return response;
}

export async function handleRefresh(request: NextRequest) {
  return renewSession(request);
}

async function renewSession(request: NextRequest) {
  const tokenPair = await refreshTokenPair(request);
  if (!tokenPair) {
    const response = NextResponse.json({ error: { code: "unauthenticated", message: "Сессия не найдена" } }, { status: 401 });
    clearAuthCookies(response);
    return response;
  }
  const response = NextResponse.json({ user: tokenPair.user, expires_in: tokenPair.expires_in });
  setAuthCookies(response, tokenPair);
  return response;
}

export async function handleSession(request: NextRequest) {
  const accessToken = request.cookies.get(ACCESS_COOKIE)?.value;
  if (accessToken) {
    const result = await backendSessionRequest(accessToken);
    if (result?.response.ok && isSessionPayload(result.payload)) {
      return NextResponse.json({ user: result.payload.user });
    }
    if (result && result.response.status !== 401) {
      return backendError(result.response.status, result.payload);
    }
  }
  return renewSession(request);
}

export async function handleLogout(request: NextRequest) {
  const refreshToken = request.cookies.get(REFRESH_COOKIE)?.value;
  if (refreshToken) await backendRequest("/api/auth/logout", { refresh_token: refreshToken });
  const response = new NextResponse(null, { status: 204 });
  clearAuthCookies(response);
  return response;
}

export async function handleProfile(request: NextRequest) {
  return handleAuthorizedJSON(request, "/api/profile", "Некорректные данные профиля");
}

export async function handleMonitors(request: NextRequest, path = "/api/monitors") {
  const backendPath = path === "/api/monitors" ? `${path}${request.nextUrl.search}` : path;
  return handleAuthorizedJSON(request, backendPath, "Некорректные данные монитора");
}

async function handleAuthorizedJSON(request: NextRequest, path: string, invalidBodyMessage: string) {
  const accessToken = request.cookies.get(ACCESS_COOKIE)?.value;
  if (!accessToken) {
    return NextResponse.json({ error: { code: "unauthenticated", message: "Сессия не найдена" } }, { status: 401 });
  }
  const hasBody = request.method !== "GET" && request.method !== "DELETE";
  const body = hasBody ? await request.json().catch(() => null) : undefined;
  if (hasBody && !body) {
    return NextResponse.json({ error: { code: "invalid_request", message: invalidBodyMessage } }, { status: 400 });
  }
  const method = request.method as AuthorizedMethod;
  let result = await backendAuthorizedRequest(path, method, accessToken, body);
  let refreshedTokens: TokenPayload | null = null;
  if (result?.response.status === 401) {
    refreshedTokens = await refreshTokenPair(request);
    if (refreshedTokens) {
      result = await backendAuthorizedRequest(path, method, refreshedTokens.access_token, body);
    }
  }
  if (!result) return backendError(503, {});
  if (!result.response.ok) {
    const response = backendError(result.response.status, result.payload);
    if (result.response.status === 401) clearAuthCookies(response);
    return response;
  }
  const response = result.response.status === 204
    ? new NextResponse(null, { status: 204 })
    : NextResponse.json(result.payload, { status: result.response.status });
  if (refreshedTokens) setAuthCookies(response, refreshedTokens);
  return response;
}

export async function handleAvatar(request: NextRequest) {
  const accessToken = request.cookies.get(ACCESS_COOKIE)?.value;
  if (!accessToken) {
    return NextResponse.json({ error: { code: "unauthenticated", message: "Сессия не найдена" } }, { status: 401 });
  }
  const body = request.method === "POST" ? await request.formData().catch(() => null) : undefined;
  if (request.method === "POST" && !body) {
    return NextResponse.json({ error: { code: "invalid_avatar", message: "Некорректное изображение" } }, { status: 400 });
  }
  let result = await backendAvatarRequest(request.method as "GET" | "POST", accessToken, body ?? undefined);
  let refreshedTokens: TokenPayload | null = null;
  if (result?.status === 401) {
    refreshedTokens = await refreshTokenPair(request);
    if (refreshedTokens) result = await backendAvatarRequest(request.method as "GET" | "POST", refreshedTokens.access_token, body ?? undefined);
  }
  if (!result) return backendError(503, {});
  if (!result.ok) {
    const payload: unknown = await result.json().catch(() => ({}));
    const response = backendError(result.status, payload);
    if (result.status === 401) clearAuthCookies(response);
    return response;
  }
  if (request.method === "POST") {
    const payload: unknown = await result.json().catch(() => ({}));
    const response = NextResponse.json(payload, { status: result.status });
    if (refreshedTokens) setAuthCookies(response, refreshedTokens);
    return response;
  }
  const response = new NextResponse(result.body, { status: result.status });
  const contentType = result.headers.get("Content-Type");
  const cacheControl = result.headers.get("Cache-Control");
  if (contentType) response.headers.set("Content-Type", contentType);
  if (cacheControl) response.headers.set("Cache-Control", cacheControl);
  if (refreshedTokens) setAuthCookies(response, refreshedTokens);
  return response;
}
