"use client";

import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { Button, Feedback, TextField } from "@/components/ui";
import {
  EMAIL_MAX_LENGTH,
  INVALID_EMAIL_MESSAGE,
  validateEmail,
} from "@/lib/auth/email-validation";
import { requestPasswordRecovery } from "@/lib/auth/password-recovery-api";

import styles from "./password-recovery-request-form.module.css";

type FeedbackKind = "success" | "rate_limited" | "error" | "network" | null;

export function PasswordRecoveryRequestForm() {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | undefined>();
  const [feedback, setFeedback] = useState<FeedbackKind>(null);
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);
  const emailRef = useRef<HTMLInputElement>(null);
  const successRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (feedback === "success") successRef.current?.focus();
  }, [feedback]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submittingRef.current || feedback === "success") return;
    const normalized = email.trim();
    const emailError = validateEmail(email);
    if (emailError) {
      setError(emailError);
      emailRef.current?.focus();
      return;
    }

    submittingRef.current = true;
    setSubmitting(true);
    setError(undefined);
    setFeedback(null);
    try {
      const result = await requestPasswordRecovery({ email: normalized });
      if (result.status === 202 && result.state !== "error") {
        setFeedback("success");
      } else if (result.status === 429) {
        setFeedback("rate_limited");
      } else if (
        result.state === "error" &&
        result.error.fields?.some(
          (issue) => issue.field === "email" && issue.code === "invalid_email",
        )
      ) {
        setError(INVALID_EMAIL_MESSAGE);
        emailRef.current?.focus();
      } else {
        setFeedback("error");
      }
    } catch {
      setFeedback("network");
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  }

  if (feedback === "success") {
    return (
      <Feedback
        ref={successRef}
        tabIndex={-1}
        title="Solicitação recebida"
        variant="success"
      >
        Se houver uma conta apta para este e-mail, enviaremos um link para
        redefinir a senha.
      </Feedback>
    );
  }

  return (
    <form
      className={styles.form}
      noValidate
      onSubmit={(event) => void submit(event)}
    >
      {feedback === "rate_limited" ? (
        <Feedback title="Aguarde para tentar novamente" variant="warning">
          Muitas solicitações. Tente novamente mais tarde.
        </Feedback>
      ) : null}
      {feedback === "error" || feedback === "network" ? (
        <Feedback title="Não foi possível solicitar o link" variant="error">
          {feedback === "network"
            ? "Verifique sua conexão e tente novamente."
            : "Tente novamente mais tarde."}
        </Feedback>
      ) : null}
      <TextField
        autoCapitalize="none"
        autoComplete="email"
        error={error}
        label="E-mail"
        maxLength={EMAIL_MAX_LENGTH}
        onChange={(event) => {
          setEmail(event.target.value);
          setError(undefined);
          setFeedback(null);
        }}
        placeholder="voce@exemplo.com"
        readOnly={submitting}
        ref={emailRef}
        required
        spellCheck={false}
        type="email"
        value={email}
      />
      <Button className={styles.submit} disabled={submitting} type="submit">
        {submitting ? "Enviando…" : "Solicitar link"}
      </Button>
    </form>
  );
}
