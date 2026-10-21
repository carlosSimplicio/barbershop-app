import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type RegisterRequest = components["schemas"]["RegisterRequest"];

export function registerOwner(request: RegisterRequest) {
  return apiFetch<void>("/api/v1/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}
