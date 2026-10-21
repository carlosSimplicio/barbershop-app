import type { components } from "@/types/generated/api";

import { apiFetch } from "@/lib/api/client";
import type { AuthPrincipal } from "./session-api";

export type VerifyEmailRequest =
  components["schemas"]["EmailConfirmationVerifyRequest"];

export function verifyEmail(request: VerifyEmailRequest) {
  return apiFetch<AuthPrincipal>("/api/v1/auth/email-confirmation/verify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}
