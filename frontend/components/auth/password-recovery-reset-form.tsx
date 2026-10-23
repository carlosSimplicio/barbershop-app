"use client";

import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { ActionLink, Button, Feedback, PasswordField } from "@/components/ui";
import { verifyPasswordRecovery } from "@/lib/auth/password-recovery-api";
import {
  INVALID_PASSWORD_MESSAGE,
  validatePassword,
} from "@/lib/auth/password-validation";

import styles from "./password-recovery-reset-form.module.css";

type Result = "form" | "success" | "invalid_link";
type FeedbackKind = "error" | "network" | null;

export function PasswordRecoveryResetForm({ token }: { token: string }) {
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [feedback, setFeedback] = useState<FeedbackKind>(null);
  const [result, setResult] = useState<Result>("form");
  const [submitting, setSubmitting] = useState(false);
  const pending = useRef(false);
  const passwordRef = useRef<HTMLInputElement>(null);
  const resultRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (result !== "form") resultRef.current?.focus();
  }, [result]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending.current || result !== "form") return;
    const validationError = validatePassword(
      password,
      "Informe uma nova senha.",
    );
    if (validationError) {
      setError(validationError);
      passwordRef.current?.focus();
      return;
    }

    pending.current = true;
    setSubmitting(true);
    setError(undefined);
    setFeedback(null);
    try {
      const response = await verifyPasswordRecovery({
        token,
        new_password: password,
      });
      if (response.status === 204 && response.state !== "error") {
        setPassword("");
        setResult("success");
      } else if (
        response.state === "error" &&
        response.error.code === "invalid_password_recovery"
      ) {
        setPassword("");
        setResult("invalid_link");
      } else if (response.state === "error") {
        const fieldError = response.error.fields?.find(
          (issue) => issue.field === "new_password",
        );
        if (fieldError?.code === "invalid_password") {
          setError(INVALID_PASSWORD_MESSAGE);
          passwordRef.current?.focus();
        } else if (fieldError?.code === "password_unchanged") {
          setError("Escolha uma senha diferente da atual.");
          passwordRef.current?.focus();
        } else {
          setFeedback("error");
        }
      } else {
        setFeedback("error");
      }
    } catch {
      setFeedback("network");
    } finally {
      pending.current = false;
      setSubmitting(false);
    }
  }

  if (result === "success") {
    return (
      <Feedback
        action={<ActionLink href="/login">Ir para o login</ActionLink>}
        ref={resultRef}
        tabIndex={-1}
        title="Senha redefinida"
        variant="success"
      >
        Suas sessões anteriores foram encerradas. Entre com a nova senha.
      </Feedback>
    );
  }
  if (result === "invalid_link") {
    return (
      <Feedback
        action={
          <ActionLink href="/recuperar-senha">Solicitar outro link</ActionLink>
        }
        ref={resultRef}
        tabIndex={-1}
        title="Link inválido ou expirado"
        variant="warning"
      >
        Este link não pode ser usado. Solicite um novo link de recuperação.
      </Feedback>
    );
  }

  return (
    <form
      className={styles.form}
      noValidate
      onSubmit={(event) => void submit(event)}
    >
      {feedback ? (
        <Feedback title="Não foi possível redefinir a senha" variant="error">
          {feedback === "network"
            ? "Verifique sua conexão e tente novamente."
            : "Tente novamente mais tarde."}
        </Feedback>
      ) : null}
      <PasswordField
        autoComplete="new-password"
        error={error}
        label="Nova senha"
        onChange={(event) => {
          setPassword(event.target.value);
          setError(undefined);
          setFeedback(null);
        }}
        readOnly={submitting}
        ref={passwordRef}
        required
        spellCheck={false}
        value={password}
      />
      <Button disabled={submitting} type="submit">
        {submitting ? "Redefinindo senha…" : "Redefinir senha"}
      </Button>
    </form>
  );
}
