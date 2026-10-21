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
  runSessionAction,
  type ApiResult,
} from "@/lib/api/client";
import { getCurrentSession, type AuthPrincipal } from "@/lib/auth/session-api";

export type AuthState =
  | { status: "loading" | "anonymous" | "unavailable" }
  | { status: "authenticated"; principal: AuthPrincipal };

type SessionContext = {
  state: AuthState;
  reload: () => Promise<void>;
  runAction: <T>(
    action: () => Promise<ApiResult<T>>,
    next: "authenticated" | "anonymous",
  ) => Promise<ApiResult<T>>;
};

const AuthSessionContext = createContext<SessionContext | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: "loading" });
  const currentRequest = useRef<AbortController | null>(null);

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
        } else {
          await reload();
        }
      }
      return result;
    },
    [reload],
  );

  return (
    <AuthSessionContext.Provider value={{ state, reload, runAction }}>
      {children}
    </AuthSessionContext.Provider>
  );
}

export function useAuthSession(): SessionContext {
  const session = useContext(AuthSessionContext);
  if (!session) throw new Error("AuthProvider is required");
  return session;
}
