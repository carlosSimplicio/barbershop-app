"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import type { ReactNode } from "react";

import {
  configureApiSessionRecovery,
  runManualRefresh,
  runSessionAction,
  type ApiResult,
} from "@/lib/api/client";
import {
  getCurrentSession,
  refreshCurrentSession,
  type AuthPrincipal,
} from "@/lib/auth/session-api";

export type AuthState =
  | { status: "loading" | "anonymous" | "unavailable" }
  | { status: "authenticated"; principal: AuthPrincipal };

type SessionContext = {
  state: AuthState;
  refreshing: boolean;
  reload: () => Promise<void>;
  refresh: () => Promise<ApiResult<void>>;
  runAction: <T extends AuthPrincipal | void>(
    action: () => Promise<ApiResult<T>>,
    next: "authenticated" | "anonymous",
  ) => Promise<ApiResult<T>>;
};

const AuthSessionContext = createContext<SessionContext | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: "loading" });
  const [refreshing, setRefreshing] = useState(false);
  const currentRequest = useRef<AbortController | null>(null);
  const manualRefresh = useRef<Promise<ApiResult<void>> | null>(null);

  const load = useCallback(async () => {
    currentRequest.current?.abort();
    const controller = new AbortController();
    currentRequest.current = controller;
    try {
      const result = await getCurrentSession(controller.signal);
      if (controller.signal.aborted) return;
      if (result.status === 200 && result.state === "success") {
        setState({ status: "authenticated", principal: result.data });
      } else if (result.status === 401) {
        setState({ status: "anonymous" });
      } else {
        setState({ status: "unavailable" });
      }
    } catch {
      if (!controller.signal.aborted) setState({ status: "unavailable" });
    } finally {
      if (currentRequest.current === controller) currentRequest.current = null;
    }
  }, []);

  const reload = useCallback(async () => {
    if (manualRefresh.current) {
      try {
        await manualRefresh.current;
      } catch {
        // The refresh action reports its own failure.
      }
      return;
    }
    setState({ status: "loading" });
    await load();
  }, [load]);

  useEffect(() => {
    let mounted = true;
    configureApiSessionRecovery({
      onInvalidSession: () => {
        currentRequest.current?.abort();
        setState({ status: "anonymous" });
      },
    });
    void Promise.resolve().then(() => {
      if (mounted) void load();
    });
    return () => {
      mounted = false;
      currentRequest.current?.abort();
      configureApiSessionRecovery(null);
    };
  }, [load]);

  const runAction = useCallback<SessionContext["runAction"]>(
    async (action, next) => {
      const result = await runSessionAction(action, next);
      if (result.state !== "error") {
        if (next === "anonymous") {
          currentRequest.current?.abort();
          setState({ status: "anonymous" });
        } else if (result.state === "success" && result.data) {
          currentRequest.current?.abort();
          setState({ status: "authenticated", principal: result.data });
        } else {
          await reload();
        }
      }
      return result;
    },
    [reload],
  );

  const refresh = useCallback(async () => {
    if (manualRefresh.current) return manualRefresh.current;
    setRefreshing(true);
    const operation = (async () => {
      const result = await runManualRefresh(refreshCurrentSession);
      if (result.state !== "error") {
        setState({ status: "loading" });
        await load();
      }
      return result;
    })();
    manualRefresh.current = operation;
    try {
      return await operation;
    } finally {
      manualRefresh.current = null;
      setRefreshing(false);
    }
  }, [load]);

  return (
    <AuthSessionContext.Provider
      value={{ state, refreshing, reload, refresh, runAction }}
    >
      {children}
    </AuthSessionContext.Provider>
  );
}

export function useAuthSession(): SessionContext {
  const session = useContext(AuthSessionContext);
  if (!session) throw new Error("AuthProvider is required");
  return session;
}
