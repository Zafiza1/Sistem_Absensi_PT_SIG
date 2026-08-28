"use client";

import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";

import { api, tokenStorage } from "./api-client";
import type { CurrentUser } from "./types";

interface AuthContextValue {
  user: CurrentUser | null;
  // True only while the very first "am I already logged in" check (via
  // /auth/me on mount) is in flight — lets the dashboard layout show a
  // loading state instead of redirecting to /login for a split second on
  // every page load/refresh.
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    let cancelled = false;

    async function loadUser() {
      if (!tokenStorage.getAccessToken()) {
        if (!cancelled) setLoading(false);
        return;
      }
      try {
        const me = await api.get<CurrentUser>("/auth/me");
        if (!cancelled) setUser(me);
      } catch {
        if (!cancelled) setUser(null);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    loadUser();
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const data = await api.post<{ access_token: string; refresh_token: string; user: CurrentUser }>(
      "/auth/login",
      { email, password },
      { skipAuth: true },
    );
    tokenStorage.setTokens(data.access_token, data.refresh_token);
    setUser(data.user);
  }, []);

  const logout = useCallback(async () => {
    const refreshToken = tokenStorage.getRefreshToken();
    tokenStorage.clear();
    setUser(null);
    if (refreshToken) {
      // Best-effort: the user is logged out client-side regardless of
      // whether the server-side revocation call succeeds.
      api.post("/auth/logout", { refresh_token: refreshToken }, { skipAuth: true }).catch(() => {});
    }
    router.push("/login");
  }, [router]);

  return <AuthContext.Provider value={{ user, loading, login, logout }}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth() must be used within <AuthProvider>");
  return ctx;
}
