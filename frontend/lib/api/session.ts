import { sendRequest, type ApiResult } from "./transport";

type RecoveryResult = "rotated" | "external" | "failed";
type Recovery = {
  round: number;
  result: RecoveryResult;
  pending: Promise<RecoveryResult> | null;
};

const lockName = "barbershop-auth-refresh";
let recovery: Recovery = { round: 0, result: "rotated", pending: null };
let onInvalidSession: (() => void) | null = null;

export function configureApiSessionRecovery(
  handler: { onInvalidSession: () => void } | null,
) {
  onInvalidSession = handler?.onInvalidSession ?? null;
  if (!handler) recovery = { round: 0, result: "rotated", pending: null };
}

export async function recoverRead<T>(
  path: `/api/v1/${string}`,
  request: () => Promise<ApiResult<T>>,
): Promise<ApiResult<T>> {
  const startedRound = recovery.round;
  const first = await request();
  if (first.status !== 401 || !onInvalidSession) return first;

  let result: RecoveryResult;
  if (recovery.pending) {
    result = await recovery.pending;
  } else if (recovery.round > startedRound) {
    result = recovery.result;
  } else if (recovery.result === "failed") {
    return first;
  } else {
    result = await startRecovery();
  }

  if (
    result !== "rotated" &&
    !(result === "external" && path === "/api/v1/auth/me")
  ) {
    return first;
  }

  const retried = await request();
  if (retried.status === 401) invalidateSession();
  return retried;
}

function startRecovery(): Promise<RecoveryResult> {
  const attempt = async (): Promise<RecoveryResult> => {
    const locks =
      typeof navigator === "undefined" ? undefined : navigator.locks;
    if (!locks) return refresh(false);
    try {
      return await locks.request(lockName, { mode: "exclusive" }, () =>
        refresh(true),
      );
    } catch {
      return "failed";
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

async function refresh(crossTabLocked: boolean): Promise<RecoveryResult> {
  if (crossTabLocked) {
    try {
      const current = await sendRequest<unknown>("/api/v1/auth/me", {}, false);
      if (current.status === 200 && current.state === "success")
        return "external";
      if (current.status !== 401) return "failed";
    } catch {
      return "failed";
    }
  }

  try {
    const result = await sendRequest<void>(
      "/api/v1/auth/refresh",
      { method: "POST" },
      true,
    );
    if (result.status === 204) return "rotated";
    if (result.status === 401) invalidateSession();
  } catch {
    // An ambiguous failure must not reuse a potentially consumed refresh token.
  }
  return "failed";
}

function invalidateSession() {
  if (recovery.result === "failed") return;
  recovery.result = "failed";
  onInvalidSession?.();
}
