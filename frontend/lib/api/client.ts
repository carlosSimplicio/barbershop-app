import { recoverRead } from "./session";
import { sendRequest, type ApiResult } from "./transport";

export type { ApiError, ApiResult } from "./transport";
export { configureApiSessionRecovery } from "./session";

export type ApiFetchOptions = RequestInit & {
  authenticated?: boolean;
  csrf?: boolean;
};

export function apiFetch<T>(
  path: `/api/v1/${string}`,
  options: ApiFetchOptions = {},
): Promise<ApiResult<T>> {
  const { authenticated = false, csrf = false, ...init } = options;
  const request = () => sendRequest<T>(path, init, csrf);
  const method = init.method?.toUpperCase() ?? "GET";
  return authenticated && (method === "GET" || method === "HEAD")
    ? recoverRead(path, request)
    : request();
}
