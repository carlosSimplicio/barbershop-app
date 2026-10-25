"use client";

import { useEffect, useRef, useState } from "react";

import { ActionLink, Feedback } from "@/components/ui";

import { PasswordRecoveryResetForm } from "../password-recovery-reset-form/password-recovery-reset-form";

type Capture =
  | { status: "capturing" }
  | { status: "error" }
  | { status: "ready"; token: string | null };

export function PasswordRecoveryBoundary() {
  const [capture, setCapture] = useState<Capture>({ status: "capturing" });
  const token = useRef<string | null>(null);

  useEffect(() => {
    let active = true;
    if (token.current === null) {
      const fragment = window.location.hash;
      try {
        window.history.replaceState(
          window.history.state,
          "",
          window.location.pathname,
        );
      } catch {
        void Promise.resolve().then(() => {
          if (active) setCapture({ status: "error" });
        });
        return () => {
          active = false;
        };
      }
      token.current = new URLSearchParams(fragment.slice(1)).get("token") || "";
    }

    void Promise.resolve().then(() => {
      if (active) setCapture({ status: "ready", token: token.current || null });
    });
    return () => {
      active = false;
    };
  }, []);

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
