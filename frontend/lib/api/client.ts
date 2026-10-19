import type { components } from "@/types/generated/api";

type ApiError = components["schemas"]["ErrorResponse"];

export type ApiResult<T> =
  | { state: "success"; data: T }
  | { state: "empty" }
  | { state: "error"; error: ApiError };

const fallbackError = (status: number): ApiError => ({
  code: "unexpected_response",
  message: `Não foi possível concluir a solicitação (${status}).`,
  request_id: "unknown",
});

export async function apiFetch<T>(
  path: `/api/v1/${string}`,
  init: RequestInit = {},
): Promise<ApiResult<T>> {
  const headers = new Headers(init.headers);
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }

  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers,
  });

  if (response.status === 204 || response.status === 205) {
    return { state: "empty" };
  }

  if (!response.ok) {
    const error = await response
      .json()
      .then((body: unknown) => body as ApiError)
      .catch(() => fallbackError(response.status));
    return { state: "error", error };
  }

  const body = await response.text();
  if (body === "") {
    return { state: "empty" };
  }

  return { state: "success", data: JSON.parse(body) as T };
}
