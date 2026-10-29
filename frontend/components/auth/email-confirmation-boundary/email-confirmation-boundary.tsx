"use client";

import { useTransactionalLinkToken } from "@/lib/auth/use-transactional-link-token/use-transactional-link-token";

import { AuthProvider } from "@/components/session/auth-provider/auth-provider";
import { Feedback } from "@/components/ui";

import { EmailVerification } from "../email-verification/email-verification";

export function EmailConfirmationBoundary() {
  const capture = useTransactionalLinkToken();

  if (capture.status === "error") {
    return (
      <Feedback title="Não foi possível preparar a confirmação" variant="error">
        Não conseguimos abrir este link com segurança. Tente novamente mais
        tarde.
      </Feedback>
    );
  }

  if (capture.status === "capturing") {
    return <div role="status">Preparando confirmação…</div>;
  }

  return (
    <AuthProvider>
      <EmailVerification token={capture.token} />
    </AuthProvider>
  );
}
