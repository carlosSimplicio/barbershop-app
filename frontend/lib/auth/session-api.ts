import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type AuthPrincipal = components["schemas"]["AuthPrincipal"];

export function getCurrentSession(signal?: AbortSignal) {
  return apiFetch<AuthPrincipal>("/api/v1/auth/me", {
    authenticated: true,
    signal,
  });
}

export function refreshCurrentSession() {
  return apiFetch<void>("/api/v1/auth/refresh", {
    csrf: true,
    method: "POST",
  });
}
