import { hasCsrfCookie, sendRequest, type ApiResult } from "./transport";

type RecoveryResult = "rotated" | "external" | "invalid" | "unavailable";
type Recovery = {
  round: number;
  result: RecoveryResult;
  pending: Promise<RecoveryResult> | null;
};

const lockName = "barbershop-auth-refresh";
const recovery: Recovery = { round: 0, result: "rotated", pending: null };
let onInvalidSession: (() => void) | null = null;
let changingSession = false;

export function configureApiSessionRecovery(
  handler: { onInvalidSession: () => void } | null,
) {
  onInvalidSession = handler?.onInvalidSession ?? null;
}

export async function runSessionAction<T>(
  action: () => Promise<ApiResult<T>>,
  next: "authenticated" | "anonymous",
): Promise<ApiResult<T>> {
  if (changingSession) throw new Error("Session action already in progress");
  changingSession = true;
  try {
    await recovery.pending;
    const result = await withRefreshLock(action);
    if (result.state !== "error") {
      recovery.round += 1;
      recovery.result = next === "authenticated" ? "external" : "invalid";
    }
    return result;
  } finally {
    changingSession = false;
  }
}

export async function recoverRead<T>(
  path: `/api/v1/${string}`,
  request: () => Promise<ApiResult<T>>,
): Promise<ApiResult<T>> {
  const startedRound = recovery.round;
  const first = await request();
  if (first.status !== 401 || !onInvalidSession || changingSession)
    return first;

  let result: RecoveryResult;
  if (recovery.pending) {
    result = await recovery.pending;
  } else if (recovery.round > startedRound) {
    result = recovery.result;
  } else if (
    recovery.result === "invalid" ||
    recovery.result === "unavailable"
  ) {
    result = recovery.result;
  } else {
    result = await startRecovery();
  }

  if (result === "unavailable") {
    return {
      state: "error",
      status: 503,
      error: {
        code: "session_recovery_unavailable",
        message: "Não foi possível verificar sua sessão.",
        request_id: "unknown",
      },
    };
  }

  if (
    result !== "rotated" &&
    !(result === "external" && path === "/api/v1/auth/me")
  ) {
    return first;
  }

  const retryRound = recovery.round;
  const retried = await request();
  if (recovery.round !== retryRound) return first;
  if (retried.status === 401) invalidateSession();
  return retried;
}

function startRecovery(): Promise<RecoveryResult> {
  const attempt = async (): Promise<RecoveryResult> => {
    try {
      return await withRefreshLock(refresh);
    } catch {
      return "unavailable";
    }
  };

  recovery.pending = attempt().then((result) => {
    recovery.round += 1;
    recovery.result = result;
    recovery.pending = null;
    return result;
  });
  return recovery.pending;
}

async function withRefreshLock<T>(
  action: (locked: boolean) => Promise<T>,
): Promise<T> {
  const locks = typeof navigator === "undefined" ? undefined : navigator.locks;
  return locks
    ? await locks.request(lockName, { mode: "exclusive" }, () => action(true))
    : action(false);
}

async function refresh(crossTabLocked: boolean): Promise<RecoveryResult> {
  if (crossTabLocked) {
    try {
      const current = await sendRequest<unknown>("/api/v1/auth/me", {}, false);
      if (current.status === 200 && current.state === "success")
        return "external";
      if (current.status !== 401) return "unavailable";
    } catch {
      return "unavailable";
    }
  }

  if (!hasCsrfCookie()) {
    invalidateSession();
    return "invalid";
  }

  try {
    const result = await sendRequest<void>(
      "/api/v1/auth/refresh",
      { method: "POST" },
      true,
    );
    if (result.status === 204) return "rotated";
    if (result.status === 401) {
      invalidateSession();
      return "invalid";
    }
  } catch {
    // An ambiguous failure must not reuse a potentially consumed refresh token.
  }
  return "unavailable";
}

function invalidateSession() {
  if (recovery.result === "invalid") return;
  recovery.result = "invalid";
  onInvalidSession?.();
}
