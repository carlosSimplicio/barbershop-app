import type { components } from "@/types/generated/api";

import { apiFetch } from "@/lib/api/client";
import type { AuthPrincipal } from "./session-api";

export type LoginRequest = components["schemas"]["LoginRequest"];

export function login(request: LoginRequest) {
  return apiFetch<AuthPrincipal>("/api/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}
