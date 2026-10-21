"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { Feedback } from "@/components/ui";
import { useAuthSession } from "@/components/session/auth-provider";
import { verifyEmail } from "@/lib/auth/email-verification-api";

type Status = "processing" | "success" | "invalid" | "error";

export function EmailVerification({ token }: { token: string | null }) {
  const [status, setStatus] = useState<Status>("processing");
  const started = useRef(false);
  const router = useRouter();
  const { runAction } = useAuthSession();

  useEffect(() => {
    if (started.current) return;
    let active = true;
    const report = (next: Status) => {
      void Promise.resolve().then(() => {
        if (active) setStatus(next);
      });
    };

    if (!token) {
      report("invalid");
      return () => {
        active = false;
      };
    }

    void Promise.resolve().then(async () => {
      if (!active || started.current) return;
      started.current = true;

      try {
        const result = await runAction(
          () => verifyEmail({ token }),
          "authenticated",
        );
        if (!active) return;
        if (result.status === 200 && result.state === "success") {
          setStatus("success");
          router.replace("/testes");
        } else if (
          result.state === "error" &&
          result.error.code === "invalid_email_confirmation"
        ) {
          setStatus("invalid");
        } else {
          setStatus("error");
        }
      } catch {
        if (active) setStatus("error");
      }
    });

    return () => {
      active = false;
    };
  }, [router, runAction, token]);

  if (status === "processing") {
    return (
      <div role="status" aria-live="polite">
        Confirmando seu e-mail…
      </div>
    );
  }

  if (status === "success") {
    return (
      <Feedback title="E-mail confirmado" variant="success">
        Sua conta está ativa. Estamos abrindo sua área de testes.
      </Feedback>
    );
  }

  if (status === "invalid") {
    return (
      <Feedback title="Link inválido ou expirado" variant="warning">
        Este link não pode ser usado. Solicite uma nova confirmação na tela de
        login.
      </Feedback>
    );
  }

  return (
    <Feedback title="Não foi possível confirmar" variant="error">
      Não conseguimos verificar seu e-mail agora. Tente novamente mais tarde.
    </Feedback>
  );
}
