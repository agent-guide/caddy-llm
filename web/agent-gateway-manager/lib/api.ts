import { API_BASE_URL, clearSession, getToken } from "./auth";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/**
 * Fetch wrapper for authenticated admin API calls.
 * Automatically injects the Bearer session token.
 * On 401, clears the session and redirects to /login.
 */
export async function adminFetch<T = unknown>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE_URL}${path}`, { ...options, headers });

  if (res.status === 401 && token) {
    // Session expired — clear local state and redirect.
    clearSession();
    if (typeof window !== "undefined") {
      window.location.replace("/login");
    }
    throw new ApiError(401, "Session expired");
  }

  if (!res.ok) {
    let msg = res.statusText;
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      // ignore parse errors
    }
    throw new ApiError(res.status, msg);
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

/**
 * Unauthenticated POST for the login endpoint.
 * Does not inject a token and does not redirect on 401.
 */
export async function login(
  username: string,
  password: string,
): Promise<{ token: string; username: string }> {
  const res = await fetch(`${API_BASE_URL}/admin/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });

  let msg = res.statusText;
  if (!res.ok) {
    try {
      const body = (await res.json()) as { error?: string };
      if (body.error) msg = body.error;
    } catch {
      // ignore parse errors
    }
    throw new ApiError(res.status, msg);
  }

  return res.json() as Promise<{ token: string; username: string }>;
}
