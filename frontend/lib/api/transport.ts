import type { components } from "@/types/generated/api";

export type ApiError = components["schemas"]["ErrorResponse"];

export type ApiResult<T> =
  | { state: "success"; data: T; status: number }
  | { state: "empty"; status: number }
  | { state: "error"; error: ApiError; status: number };

const csrfCookieName = "__Secure-barbershop_csrf";

export function hasCsrfCookie() {
  return Boolean(readCookie(csrfCookieName));
}

export async function sendRequest<T>(
  path: `/api/v1/${string}`,
  init: RequestInit,
  csrf: boolean,
): Promise<ApiResult<T>> {
  const headers = new Headers(init.headers);
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }
  if (csrf) {
    const token = readCookie(csrfCookieName);
    if (token) {
      headers.set("X-CSRF-Token", token);
    } else {
      headers.delete("X-CSRF-Token");
    }
  }

  const response = await fetch(path, {
    ...init,
    credentials: "include",
    headers,
  });

  if (response.status === 204 || response.status === 205) {
    return { state: "empty", status: response.status };
  }

  const body = await response.text();
  if (!response.ok) {
    return {
      state: "error",
      error: parseApiError(body, response.status),
      status: response.status,
    };
  }

  if (body === "") {
    return { state: "empty", status: response.status };
  }

  try {
    return {
      state: "success",
      data: JSON.parse(body) as T,
      status: response.status,
    };
  } catch {
    return {
      state: "error",
      error: fallbackError(response.status),
      status: response.status,
    };
  }
}

const fallbackError = (status: number): ApiError => ({
  code: "unexpected_response",
  message: `Não foi possível concluir a solicitação (${status}).`,
  request_id: "unknown",
});

function parseApiError(body: string, status: number): ApiError {
  try {
    const value: unknown = JSON.parse(body);
    if (isApiError(value)) {
      return value;
    }
  } catch {
    // Keep malformed upstream responses out of the UI.
  }
  return fallbackError(status);
}

function isApiError(value: unknown): value is ApiError {
  return (
    typeof value === "object" &&
    value !== null &&
    "code" in value &&
    typeof value.code === "string" &&
    "message" in value &&
    typeof value.message === "string" &&
    "request_id" in value &&
    typeof value.request_id === "string" &&
    (!("fields" in value) ||
      value.fields === undefined ||
      (Array.isArray(value.fields) && value.fields.every(isFieldError)))
  );
}

function isFieldError(value: unknown) {
  return (
    typeof value === "object" &&
    value !== null &&
    "field" in value &&
    typeof value.field === "string" &&
    "code" in value &&
    typeof value.code === "string" &&
    "message" in value &&
    typeof value.message === "string"
  );
}

function readCookie(name: string): string | null {
  if (typeof document === "undefined") {
    return null;
  }

  const prefix = `${name}=`;
  for (const part of document.cookie.split(";")) {
    const cookie = part.trim();
    if (cookie.startsWith(prefix)) {
      const value = cookie.slice(prefix.length);
      try {
        return decodeURIComponent(value);
      } catch {
        return value;
      }
    }
  }
  return null;
}
