import { useQueryClient } from '@tanstack/react-query';
import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import * as authApi from '../api/auth';
import { getToken, setToken, UNAUTHORIZED_EVENT } from '../api/client';
import type { User } from '../api/types';
import { setCurrency } from '../lib/money';

interface AuthState {
  user: User | null;
  /** True while we check a stored token on first load. */
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (input: { name: string; email: string; password: string }) => Promise<void>;
  logout: () => void;
  /** Replaces the user after their settings change. */
  updateUser: (user: User) => void;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [user, setUserState] = useState<User | null>(null);
  const setUser = useCallback((u: User | null) => {
    setCurrency(u?.currency ?? 'USD');
    setUserState(u);
  }, []);
  const [loading, setLoading] = useState(() => getToken() !== null);

  useEffect(() => {
    if (!getToken()) return;
    let cancelled = false;
    authApi
      .me()
      .then((u) => {
        if (!cancelled) setUser(u);
      })
      .catch(() => {
        if (!cancelled) setToken(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [setUser]);

  const logout = useCallback(() => {
    setToken(null);
    setUser(null);
    queryClient.clear();
  }, [queryClient, setUser]);

  useEffect(() => {
    const onUnauthorized = () => logout();
    window.addEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
    return () => window.removeEventListener(UNAUTHORIZED_EVENT, onUnauthorized);
  }, [logout]);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await authApi.login(email, password);
      setToken(res.token);
      setUser(res.user);
    },
    [setUser],
  );

  const register = useCallback(
    async (input: { name: string; email: string; password: string }) => {
      const res = await authApi.register(input);
      setToken(res.token);
      setUser(res.user);
    },
    [setUser],
  );

  const value = useMemo(
    () => ({ user, loading, login, register, logout, updateUser: setUser }),
    [user, loading, login, register, logout, setUser],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider');
  return ctx;
}
