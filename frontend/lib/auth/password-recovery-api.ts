import type { components } from "@/types/generated/api";

import { apiFetch } from "@/lib/api/client";

export type PasswordRecoveryRequest =
  components["schemas"]["PasswordRecoveryRequest"];
export type PasswordRecoveryVerifyRequest =
  components["schemas"]["PasswordRecoveryVerifyRequest"];

export function requestPasswordRecovery(request: PasswordRecoveryRequest) {
  return apiFetch<void>("/api/v1/auth/password/recovery", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}

export function verifyPasswordRecovery(request: PasswordRecoveryVerifyRequest) {
  return apiFetch<void>("/api/v1/auth/password/recovery/verify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}
