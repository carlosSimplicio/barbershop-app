"use client";

import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import type { FormEvent } from "react";

import { useAuthSession } from "@/components/session/auth-provider";
import { Button, Feedback, PasswordField, TextField } from "@/components/ui";
import { login, type LoginRequest } from "@/lib/auth/login-api";

import { ConfirmationResend } from "./confirmation-resend";
import styles from "./login-form.module.css";

type FieldErrors = Partial<Record<keyof LoginRequest, string>>;
type FeedbackKind =
  "credentials" | "confirmation" | "rate_limited" | "error" | "network" | null;

export function LoginForm() {
  const [values, setValues] = useState<LoginRequest>({
    email: "",
    password: "",
  });
  const [errors, setErrors] = useState<FieldErrors>({});
  const [feedback, setFeedback] = useState<FeedbackKind>(null);
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);
  const emailRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);
  const router = useRouter();
  const { runAction } = useAuthSession();

  function update(field: keyof LoginRequest, value: string) {
    setValues((current) => ({ ...current, [field]: value }));
    setErrors((current) => ({ ...current, [field]: undefined }));
    setFeedback(null);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submittingRef.current) return;
    const nextErrors = validate(values);
    setErrors(nextErrors);
    if (nextErrors.email || nextErrors.password) {
      if (nextErrors.email) emailRef.current?.focus();
      else passwordRef.current?.focus();
      return;
    }

    submittingRef.current = true;
    setSubmitting(true);
    setFeedback(null);
    try {
      const result = await runAction(
        () => login({ email: values.email.trim(), password: values.password }),
        "authenticated",
      );
      if (result.status === 200 && result.state === "success") {
        router.replace("/testes");
        return;
      }
      if (result.state === "error") {
        const fieldErrors: FieldErrors = {};
        for (const issue of result.error.fields ?? []) {
          if (issue.field === "email" && issue.code === "invalid_email") {
            fieldErrors.email = "Digite um e-mail válido.";
          }
          if (issue.field === "password" && issue.code === "invalid_password") {
            fieldErrors.password = "Use de 8 a 32 caracteres, sem espaços.";
          }
        }
        if (fieldErrors.email || fieldErrors.password) {
          setErrors(fieldErrors);
          if (fieldErrors.email) emailRef.current?.focus();
          else passwordRef.current?.focus();
          return;
        }
        if (result.error.code === "invalid_credentials") {
          setFeedback("credentials");
          return;
        }
        if (result.error.code === "email_confirmation_required") {
          setFeedback("confirmation");
          return;
        }
        if (result.error.code === "authentication_rate_limited") {
          setFeedback("rate_limited");
          return;
        }
      }
      setFeedback("error");
    } catch {
      setFeedback("network");
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  }

  return (
    <form
      className={styles.form}
      noValidate
      onSubmit={(event) => void submit(event)}
    >
      {feedback === "credentials" ? (
        <Feedback title="Não foi possível entrar" variant="error">
          E-mail ou senha inválidos.
        </Feedback>
      ) : null}
      {feedback === "confirmation" ? (
        <Feedback title="Confirme seu e-mail" variant="warning">
          Confirme seu e-mail para entrar. Se precisar, solicite um novo link
          abaixo.
        </Feedback>
      ) : null}
      {feedback === "rate_limited" ? (
        <Feedback title="Aguarde para tentar novamente" variant="warning">
          Muitas tentativas. Tente novamente mais tarde.
        </Feedback>
      ) : null}
      {feedback === "error" || feedback === "network" ? (
        <Feedback title="Não foi possível entrar" variant="error">
          {feedback === "network"
            ? "Verifique sua conexão e tente novamente."
            : "Tente novamente mais tarde."}
        </Feedback>
      ) : null}
      <TextField
        autoCapitalize="none"
        autoComplete="username"
        error={errors.email}
        label="E-mail"
        maxLength={254}
        onChange={(event) => update("email", event.target.value)}
        placeholder="voce@exemplo.com"
        readOnly={submitting}
        ref={emailRef}
        required
        spellCheck={false}
        type="email"
        value={values.email}
      />
      <PasswordField
        autoComplete="current-password"
        error={errors.password}
        label="Senha"
        onChange={(event) => update("password", event.target.value)}
        readOnly={submitting}
        ref={passwordRef}
        required
        value={values.password}
      />
      <Button className={styles.submit} disabled={submitting} type="submit">
        {submitting ? "Entrando…" : "Entrar"}
      </Button>
      {feedback === "confirmation" ? (
        <ConfirmationResend email={values.email.trim()} />
      ) : null}
    </form>
  );
}

function validate(values: LoginRequest): FieldErrors {
  const errors: FieldErrors = {};
  const email = values.email.trim();
  if (!email) errors.email = "Informe seu e-mail.";
  else if (email.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    errors.email = "Digite um e-mail válido.";
  }
  if (!values.password) errors.password = "Informe sua senha.";
  else if (
    [...values.password].length < 8 ||
    [...values.password].length > 32 ||
    /\s/.test(values.password)
  ) {
    errors.password = "Use de 8 a 32 caracteres, sem espaços.";
  }
  return errors;
}
