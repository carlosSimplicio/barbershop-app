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

import { configureApiSessionObserver, type ApiResult } from "@/lib/api/client";
import { getCurrentSession, type AuthPrincipal } from "@/lib/auth/session-api/session-api";

export type AuthState =
  | { status: "loading" | "anonymous" | "unavailable" }
  | { status: "authenticated"; principal: AuthPrincipal };

type SessionContext = {
  state: AuthState;
  transitioning: boolean;
  reload: () => Promise<void>;
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
  const actionPending = useRef(false);

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
    if (actionPending.current) return;
    setState({ status: "loading" });
    await load();
  }, [load]);

  useEffect(() => {
    let mounted = true;
    configureApiSessionObserver({
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
      configureApiSessionObserver(null);
    };
  }, [load]);

  const runAction = useCallback<SessionContext["runAction"]>(
    async (action, next) => {
      if (actionPending.current)
        throw new Error("Session action already in progress");
      actionPending.current = true;
      setTransitioning(true);
      try {
        const result = await action();
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
      } finally {
        actionPending.current = false;
        setTransitioning(false);
      }
    },
    [load],
  );

  return (
    <AuthSessionContext.Provider
      value={{ state, transitioning, reload, runAction }}
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
