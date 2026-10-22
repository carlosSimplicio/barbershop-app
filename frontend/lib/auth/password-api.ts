import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type ChangePasswordRequest =
  components["schemas"]["ChangePasswordRequest"];

export function changeTemporaryPassword(request: ChangePasswordRequest) {
  return apiFetch<void>("/api/v1/auth/password/change", {
    method: "POST",
    csrf: true,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}
