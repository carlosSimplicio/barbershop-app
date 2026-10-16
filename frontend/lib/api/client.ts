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
  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers: {
      Accept: "application/json",
      ...init.headers,
    },
  });

  if (response.status === 204) {
    return { state: "empty" };
  }

  if (!response.ok) {
    const error = await response
      .json()
      .then((body: unknown) => body as ApiError)
      .catch(() => fallbackError(response.status));
    return { state: "error", error };
  }

  return { state: "success", data: (await response.json()) as T };
}
