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
  transitioning: boolean;
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
  const [transitioning, setTransitioning] = useState(false);
  const currentRequest = useRef<AbortController | null>(null);
  const transition = useRef<Promise<unknown> | null>(null);

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
    if (transition.current) {
      try {
        await transition.current;
      } catch {
        // The action reports its own failure.
      }
      return;
    }
    setState({ status: "loading" });
    await load();
  }, [load]);

  const startTransition = useCallback(
    async <T,>(action: () => Promise<T>): Promise<T> => {
      if (transition.current)
        throw new Error("Session action already in progress");
      const pending = action();
      transition.current = pending;
      setTransitioning(true);
      try {
        return await pending;
      } finally {
        transition.current = null;
        setTransitioning(false);
      }
    },
    [],
  );

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
    (action, next) =>
      startTransition(async () => {
        const result = await runSessionAction(action, next);
        if (result.state !== "error") {
          if (next === "anonymous") {
            currentRequest.current?.abort();
            setState({ status: "anonymous" });
          } else if (result.state === "success" && result.data) {
            currentRequest.current?.abort();
            setState({ status: "authenticated", principal: result.data });
          } else {
            setState({ status: "loading" });
            await load();
          }
        }
        return result;
      }),
    [load, startTransition],
  );

  const refresh = useCallback(
    () =>
      startTransition(async () => {
        const result = await runManualRefresh(refreshCurrentSession);
        if (result.state !== "error") {
          setState({ status: "loading" });
          await load();
        }
        return result;
      }),
    [load, startTransition],
  );

  return (
    <AuthSessionContext.Provider
      value={{ state, transitioning, reload, refresh, runAction }}
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
