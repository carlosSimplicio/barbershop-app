import type { components } from "@/types/generated/api";

import { apiFetch } from "@/lib/api/client";

export type EmailConfirmationRequest =
  components["schemas"]["EmailConfirmationRequest"];

export function requestEmailConfirmation(request: EmailConfirmationRequest) {
  return apiFetch<void>("/api/v1/auth/email-confirmation", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}
