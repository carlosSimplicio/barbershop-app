"use client";

import { useTransactionalLinkToken } from "@/lib/auth/use-transactional-link-token/use-transactional-link-token";

import { ActionLink, Feedback } from "@/components/ui";

import { PasswordRecoveryResetForm } from "../password-recovery-reset-form/password-recovery-reset-form";

export function PasswordRecoveryBoundary() {
  const capture = useTransactionalLinkToken();

  if (capture.status === "capturing") {
    return <div role="status">Preparando redefinição…</div>;
  }
  if (capture.status === "error") {
    return (
      <Feedback title="Não foi possível preparar a redefinição" variant="error">
        Não foi possível abrir este link com segurança. Tente novamente mais
        tarde.
      </Feedback>
    );
  }
  if (!capture.token) {
    return (
      <Feedback
        action={
          <ActionLink href="/recuperar-senha">Solicitar outro link</ActionLink>
        }
        title="Link inválido ou expirado"
        variant="warning"
      >
        Solicite um novo link para redefinir sua senha.
      </Feedback>
    );
  }
  return <PasswordRecoveryResetForm token={capture.token} />;
}
