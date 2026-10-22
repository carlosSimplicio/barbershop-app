import { sendRequest, type ApiResult } from "./transport";

export type { ApiError, ApiResult } from "./transport";

let onInvalidSession: (() => void) | null = null;

export function configureApiSessionObserver(
  handler: { onInvalidSession: () => void } | null,
) {
  onInvalidSession = handler?.onInvalidSession ?? null;
}

export type ApiFetchOptions = RequestInit & {
  csrf?: boolean;
};

export async function apiFetch<T>(
  path: `/api/v1/${string}`,
  options: ApiFetchOptions = {},
): Promise<ApiResult<T>> {
  const { csrf = false, ...init } = options;
  const result = await sendRequest<T>(path, init, csrf);
  if (result.status === 401 && !init.signal?.aborted) {
    onInvalidSession?.();
  }
  return result;
}
