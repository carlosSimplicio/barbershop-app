"use client";

import { useRef, useState } from "react";

import { Button, Feedback } from "@/components/ui";
import { requestEmailConfirmation } from "@/lib/auth/confirmation-api/confirmation-api";

import styles from "./confirmation-resend.module.css";

type Status = "idle" | "sending" | "success" | "rate_limited" | "error";

export function ConfirmationResend({ email }: { email: string }) {
  const [status, setStatus] = useState<Status>("idle");
  const sending = useRef(false);

  async function resend() {
    if (sending.current) return;
    sending.current = true;
    setStatus("sending");
    try {
      const result = await requestEmailConfirmation({ email });
      if (result.status === 202 && result.state !== "error") {
        setStatus("success");
      } else if (
        result.state === "error" &&
        result.error.code === "email_confirmation_rate_limited"
      ) {
        setStatus("rate_limited");
      } else {
        setStatus("error");
      }
    } catch {
      setStatus("error");
    } finally {
      sending.current = false;
    }
  }

  return (
    <div className={styles.resend}>
      <Button disabled={status === "sending"} onClick={() => void resend()}>
        {status === "sending" ? "Enviando…" : "Reenviar link de confirmação"}
      </Button>
      {status === "success" ? (
        <Feedback title="Solicitação recebida" variant="success">
          Se houver uma conta pendente para este e-mail, enviaremos um novo link
          de confirmação.
        </Feedback>
      ) : null}
      {status === "rate_limited" ? (
        <Feedback title="Aguarde para tentar novamente" variant="warning">
          Muitas solicitações. Tente novamente mais tarde.
        </Feedback>
      ) : null}
      {status === "error" ? (
        <Feedback title="Não foi possível enviar" variant="error">
          Não foi possível solicitar outro link. Tente novamente.
        </Feedback>
      ) : null}
    </div>
  );
}
