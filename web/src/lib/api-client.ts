// Thin fetch wrapper around the Go backend's single response envelope
// (`{success, message, data, errors}` — see backend/pkg/response). Runs
// entirely in the browser: the dashboard is a client-rendered SPA against
// an external API, not Next's own server-side data layer, so tokens live
// in localStorage and every call happens from client components.

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1";

const ACCESS_TOKEN_KEY = "absensi_access_token";
const REFRESH_TOKEN_KEY = "absensi_refresh_token";

export const tokenStorage = {
  getAccessToken(): string | null {
    if (typeof window === "undefined") return null;
    return localStorage.getItem(ACCESS_TOKEN_KEY);
  },
  getRefreshToken(): string | null {
    if (typeof window === "undefined") return null;
    return localStorage.getItem(REFRESH_TOKEN_KEY);
  },
  setTokens(access: string, refresh: string) {
    localStorage.setItem(ACCESS_TOKEN_KEY, access);
    localStorage.setItem(REFRESH_TOKEN_KEY, refresh);
  },
  clear() {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  },
};

export class ApiError extends Error {
  status: number;
  fieldErrors?: Record<string, string>;

  constructor(message: string, status: number, fieldErrors?: Record<string, string>) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.fieldErrors = fieldErrors;
  }
}

interface Envelope<T> {
  success: boolean;
  message: string;
  data?: T;
  errors?: unknown;
}

interface RequestOptions {
  method?: "GET" | "POST" | "PUT" | "DELETE";
  body?: unknown;
  query?: Record<string, string | number | boolean | undefined>;
  // Only /auth/login and /auth/refresh themselves call with this — every
  // other request should carry whatever access token is on hand.
  skipAuth?: boolean;
}

// Refreshing is coalesced into a single in-flight promise: several calls
// racing to the API at the moment an access token expires must not each
// fire their own /auth/refresh (the backend rotates refresh tokens on use,
// so a second concurrent refresh would invalidate the first).
let refreshPromise: Promise<boolean> | null = null;

async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = tokenStorage.getRefreshToken();
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${API_URL}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    const body: Envelope<{ access_token: string; refresh_token: string }> = await res.json();
    if (!res.ok || !body.success || !body.data) return false;
    tokenStorage.setTokens(body.data.access_token, body.data.refresh_token);
    return true;
  } catch {
    return false;
  }
}

function buildUrl(path: string, query?: RequestOptions["query"]): string {
  const url = new URL(`${API_URL}${path}`);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined && value !== "") url.searchParams.set(key, String(value));
    }
  }
  return url.toString();
}

async function request<T>(path: string, options: RequestOptions = {}, isRetry = false): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (!options.skipAuth) {
    const token = tokenStorage.getAccessToken();
    if (token) headers.Authorization = `Bearer ${token}`;
  }

  const res = await fetch(buildUrl(path, options.query), {
    method: options.method ?? "GET",
    headers,
    body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
  });

  // A 401 on anything other than login/refresh itself means the access
  // token (15-minute TTL) expired mid-session. Try exactly one silent
  // refresh-and-retry before forcing the user back to /login, so a normal
  // session doesn't get interrupted every 15 minutes.
  if (res.status === 401 && !options.skipAuth && !isRetry) {
    refreshPromise ??= refreshAccessToken().finally(() => {
      refreshPromise = null;
    });
    const refreshed = await refreshPromise;
    if (refreshed) {
      return request<T>(path, options, true);
    }
    tokenStorage.clear();
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      // A deliberate hard navigation, not an internal Next.js Link/router
      // transition: this runs from plain module code with no access to
      // useRouter(), and a full reload is exactly what's wanted here —
      // it wipes every component's in-memory state along with the
      // expired session instead of leaving stale, half-authenticated UI
      // mounted underneath the redirect.
      // eslint-disable-next-line @next/next/no-location-assign-relative-destination
      window.location.href = "/login";
    }
    throw new ApiError("Sesi berakhir, silakan login kembali", 401);
  }

  let body: Envelope<T> | null = null;
  try {
    body = await res.json();
  } catch {
    // No/invalid JSON body (e.g. a 204 or a proxy error page) — fall
    // through to the generic status-code error below.
  }

  if (!res.ok || !body?.success) {
    const fieldErrors =
      body?.errors && typeof body.errors === "object" && !Array.isArray(body.errors)
        ? (body.errors as Record<string, string>)
        : undefined;
    throw new ApiError(body?.message ?? `Permintaan gagal (${res.status})`, res.status, fieldErrors);
  }

  return body.data as T;
}

export const api = {
  get: <T,>(path: string, query?: RequestOptions["query"]) => request<T>(path, { method: "GET", query }),
  post: <T,>(path: string, body?: unknown, options?: Pick<RequestOptions, "skipAuth">) =>
    request<T>(path, { method: "POST", body, ...options }),
  put: <T,>(path: string, body?: unknown) => request<T>(path, { method: "PUT", body }),
  delete: <T,>(path: string) => request<T>(path, { method: "DELETE" }),
};
