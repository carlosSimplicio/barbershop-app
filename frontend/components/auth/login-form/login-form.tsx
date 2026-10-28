"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import type { FormEvent } from "react";

import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import { Button, Feedback, PasswordField, TextField } from "@/components/ui";
import {
  EMAIL_MAX_LENGTH,
  INVALID_EMAIL_MESSAGE,
  validateEmail,
} from "@/lib/auth/email-validation/email-validation";
import { login, type LoginRequest } from "@/lib/auth/login-api/login-api";
import {
  INVALID_PASSWORD_MESSAGE,
  validatePassword,
} from "@/lib/auth/password-validation/password-validation";

import { ConfirmationResend } from "../confirmation-resend/confirmation-resend";
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
        router.replace(result.data.role === "owner" ? "/painel" : "/testes");
        return;
      }
      if (result.state === "error") {
        const fieldErrors: FieldErrors = {};
        for (const issue of result.error.fields ?? []) {
          if (issue.field === "email" && issue.code === "invalid_email") {
            fieldErrors.email = INVALID_EMAIL_MESSAGE;
          }
          if (issue.field === "password" && issue.code === "invalid_password") {
            fieldErrors.password = INVALID_PASSWORD_MESSAGE;
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
        maxLength={EMAIL_MAX_LENGTH}
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
      <Link className={styles.recoveryLink} href="/recuperar-senha">
        Esqueci minha senha
      </Link>
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
  errors.email = validateEmail(values.email);
  errors.password = validatePassword(values.password, "Informe sua senha.");
  return errors;
}
