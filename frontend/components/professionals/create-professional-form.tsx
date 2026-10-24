"use client";

import { useRef, useState } from "react";
import type { FormEvent } from "react";

import { Button, Feedback, PasswordField, TextField } from "@/components/ui";
import {
  EMAIL_MAX_LENGTH,
  INVALID_EMAIL_MESSAGE,
  validateEmail,
} from "@/lib/auth/email-validation";
import {
  INVALID_PASSWORD_MESSAGE,
  validatePassword,
} from "@/lib/auth/password-validation";
import {
  createProfessional,
  type CreateProfessionalRequest,
} from "@/lib/professionals/professionals-api";

import styles from "./create-professional-form.module.css";

type FieldName = keyof CreateProfessionalRequest;
type FieldErrors = Partial<Record<FieldName, string>>;
type FeedbackState = {
  variant: "error" | "success" | "warning";
  title: string;
  message: string;
};

const fieldOrder: FieldName[] = ["name", "email", "temporary_password"];

export function CreateProfessionalForm({
  onCreated,
}: {
  onCreated: () => void;
}) {
  const [values, setValues] = useState<CreateProfessionalRequest>({
    name: "",
    email: "",
    temporary_password: "",
  });
  const [errors, setErrors] = useState<FieldErrors>({});
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const submittingRef = useRef(false);
  const nameRef = useRef<HTMLInputElement>(null);
  const emailRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);

  function update(field: FieldName, value: string) {
    setValues((current) => ({ ...current, [field]: value }));
    setErrors((current) => ({ ...current, [field]: undefined }));
    setFeedback(null);
  }

  function focusFirstError(fieldErrors: FieldErrors) {
    const refs = {
      name: nameRef,
      email: emailRef,
      temporary_password: passwordRef,
    };
    const first = fieldOrder.find((field) => fieldErrors[field]);
    if (first) refs[first].current?.focus();
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submittingRef.current) return;

    const nextErrors = validate(values);
    setErrors(nextErrors);
    if (fieldOrder.some((field) => nextErrors[field])) {
      focusFirstError(nextErrors);
      return;
    }

    submittingRef.current = true;
    setSubmitting(true);
    setFeedback(null);
    try {
      const result = await createProfessional({
        ...values,
        name: values.name.trim(),
        email: values.email.trim(),
      });
      if (result.status === 201 && result.state === "success") {
        setValues((current) => ({ ...current, temporary_password: "" }));
        setErrors({});
        setFeedback({
          variant: "success",
          title: "Profissional cadastrado",
          message:
            "A senha temporária não poderá ser recuperada. Compartilhe-a com o profissional por um canal seguro.",
        });
        onCreated();
        return;
      }
      if (result.state === "error") {
        if (result.error.code === "read_only") {
          setFeedback({
            variant: "warning",
            title: "Cadastro indisponível",
            message:
              "O período de avaliação não está ativo. Você ainda pode consultar a lista, mas novos cadastros estão bloqueados.",
          });
          return;
        }
        const serverErrors: FieldErrors = {};
        for (const issue of result.error.fields ?? []) {
          if (!fieldOrder.includes(issue.field as FieldName)) continue;
          const message = serverMessages[issue.code];
          if (message) serverErrors[issue.field as FieldName] = message;
        }
        if (fieldOrder.some((field) => serverErrors[field])) {
          setErrors(serverErrors);
          focusFirstError(serverErrors);
          return;
        }
      }
      setFeedback({
        variant: "error",
        title: "Não foi possível cadastrar o profissional",
        message: "Tente novamente mais tarde.",
      });
    } catch {
      setFeedback({
        variant: "error",
        title: "Não foi possível cadastrar o profissional",
        message: "Verifique sua conexão e tente novamente.",
      });
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  }

  return (
    <section
      aria-labelledby="create-professional-title"
      className={styles.panel}
    >
      <div className={styles.heading}>
        <h2 id="create-professional-title">Cadastrar profissional</h2>
        <p>Crie o acesso inicial para quem trabalha no estabelecimento.</p>
      </div>
      {feedback ? (
        <Feedback title={feedback.title} variant={feedback.variant}>
          {feedback.message}
        </Feedback>
      ) : null}
      <form
        className={styles.form}
        noValidate
        onSubmit={(event) => void submit(event)}
      >
        <TextField
          autoComplete="name"
          error={errors.name}
          label="Nome"
          onChange={(event) => update("name", event.target.value)}
          readOnly={submitting}
          ref={nameRef}
          required
          value={values.name}
        />
        <TextField
          autoCapitalize="none"
          autoComplete="email"
          error={errors.email}
          label="E-mail"
          maxLength={EMAIL_MAX_LENGTH}
          onChange={(event) => update("email", event.target.value)}
          readOnly={submitting}
          ref={emailRef}
          required
          spellCheck={false}
          type="email"
          value={values.email}
        />
        <PasswordField
          autoComplete="new-password"
          error={errors.temporary_password}
          hint="Use de 8 a 32 caracteres, sem espaços. A troca será obrigatória no primeiro acesso."
          label="Senha temporária"
          onChange={(event) => update("temporary_password", event.target.value)}
          readOnly={submitting}
          ref={passwordRef}
          required
          value={values.temporary_password}
        />
        <Button disabled={submitting} type="submit">
          {submitting ? "Cadastrando…" : "Cadastrar profissional"}
        </Button>
      </form>
    </section>
  );
}

function validate(values: CreateProfessionalRequest): FieldErrors {
  const errors: FieldErrors = {};
  const name = values.name.trim();
  if (!name || [...name].length > 120) {
    errors.name = name
      ? "Informe um nome com até 120 caracteres."
      : "Informe o nome do profissional.";
  }
  const email = values.email.trim()
    ? validateEmail(values.email)
    : "Informe o e-mail do profissional.";
  if (email) errors.email = email;
  const password = validatePassword(
    values.temporary_password,
    "Informe uma senha temporária.",
  );
  if (password) errors.temporary_password = password;
  return errors;
}

const serverMessages: Record<string, string> = {
  invalid_name: "Informe um nome com até 120 caracteres.",
  invalid_email: INVALID_EMAIL_MESSAGE,
  invalid_password: INVALID_PASSWORD_MESSAGE,
  email_unavailable: "Este e-mail já está cadastrado.",
};
