import type { components } from "@/types/generated/api";

import { apiFetch } from "@/lib/api/client";

export type PasswordRecoveryRequest =
  components["schemas"]["PasswordRecoveryRequest"];

export function requestPasswordRecovery(request: PasswordRecoveryRequest) {
  return apiFetch<void>("/api/v1/auth/password/recovery", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}
