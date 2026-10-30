"use client";

import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { Button, Feedback, PasswordField, TextField } from "@/components/ui";
import {
  EMAIL_MAX_LENGTH,
  validateEmail,
} from "@/lib/auth/email-validation/email-validation";
import { validatePassword } from "@/lib/auth/password-validation/password-validation";
import {
  createProfessional,
  type CreateProfessionalRequest,
} from "@/lib/professionals/professionals-api";
import {
  validateProfessionalName,
  PROFESSIONAL_NAME_HINT,
  TEMPORARY_PASSWORD_HINT,
} from "@/lib/professionals/professional-form-model";
import { classifyProfessionalOperation } from "@/lib/professionals/professional-operation-result";

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
  canWrite = true,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
}: {
  onCreated: () => void;
  canWrite?: boolean;
  onReadOnly?: () => void;
  onUnauthorized?: () => void;
  onBusyChange?: (busy: boolean) => void;
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
  const controller = useRef<AbortController | null>(null);
  const [uncertain, setUncertain] = useState(false);
  const [verification, setVerification] = useState(false);
  const focusAfterResponse = useRef(false);
  useEffect(() => () => controller.current?.abort(), []);
  useEffect(() => {
    if (!submitting && focusAfterResponse.current) {
      focusAfterResponse.current = false;
      focusFirstError(errors);
    }
  }, [submitting, errors]);
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
    if (submittingRef.current || !canWrite || uncertain) return;

    const nextErrors = validate(values);
    setErrors(nextErrors);
    if (fieldOrder.some((field) => nextErrors[field])) {
      focusFirstError(nextErrors);
      return;
    }

    submittingRef.current = true;
    onBusyChange?.(true);
    const requestController = new AbortController();
    controller.current = requestController;
    setSubmitting(true);
    setFeedback(null);
    try {
      const result = await createProfessional(
        {
          ...values,
          name: values.name.trim(),
          email: values.email.trim(),
        },
        requestController.signal,
      );
      if (requestController.signal.aborted) return;
      const outcome = classifyProfessionalOperation(result, 201);
      if (outcome.state === "unauthorized") {
        onUnauthorized?.();
        return;
      }
      if (outcome.state === "confirmed") {
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
      if (outcome.state === "read_only") {
        onReadOnly?.();
        setFeedback({
          variant: "warning",
          title: "Cadastro indisponível",
          message:
            "O período de avaliação não está ativo. Você ainda pode consultar a lista, mas novos cadastros estão bloqueados.",
        });
        return;
      }
      if (outcome.state === "validation") {
        setErrors(outcome.fields);
        focusAfterResponse.current = true;
        return;
      }
      if (outcome.state === "uncertain") {
        setUncertain(true);
        setFeedback({
          variant: "warning",
          title: "Não foi possível confirmar o cadastro",
          message:
            "Atualize a lista para conferir se o profissional foi criado antes de tentar novamente.",
        });
        return;
      }
      setFeedback({
        variant: "error",
        title: "Não foi possível cadastrar o profissional",
        message: "Tente novamente mais tarde.",
      });
    } catch {
      if (!requestController.signal.aborted) {
        setUncertain(true);
        setFeedback({
          variant: "warning",
          title: "Não foi possível confirmar o cadastro",
          message:
            "Verifique sua conexão e atualize a lista para conferir o cadastro antes de tentar novamente.",
        });
      }
    } finally {
      onBusyChange?.(false);
      if (!requestController.signal.aborted) {
        submittingRef.current = false;
        setSubmitting(false);
      }
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
      {uncertain ? (
        <Button
          disabled={!canWrite}
          onClick={() => {
            onCreated();
            setVerification(true);
          }}
        >
          Conferir lista antes de novo cadastro
        </Button>
      ) : null}
      {uncertain && verification && canWrite ? (
        <Button
          onClick={() => {
            setUncertain(false);
            setVerification(false);
            setFeedback(null);
          }}
        >
          Conferi a lista e desejo preencher outro cadastro
        </Button>
      ) : null}
      <form
        className={styles.form}
        noValidate
        onSubmit={(event) => void submit(event)}
      >
        <TextField
          autoComplete="name"
          error={errors.name}
          hint={PROFESSIONAL_NAME_HINT}
          label="Nome"
          onChange={(event) => update("name", event.target.value)}
          disabled={submitting || !canWrite || uncertain}
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
          disabled={submitting || !canWrite || uncertain}
          ref={emailRef}
          required
          spellCheck={false}
          type="email"
          value={values.email}
        />
        <PasswordField
          autoComplete="new-password"
          error={errors.temporary_password}
          hint={TEMPORARY_PASSWORD_HINT}
          label="Senha temporária"
          onChange={(event) => update("temporary_password", event.target.value)}
          disabled={submitting || !canWrite || uncertain}
          ref={passwordRef}
          required
          value={values.temporary_password}
        />
        <Button disabled={submitting || !canWrite || uncertain} type="submit">
          {submitting ? "Cadastrando…" : "Cadastrar profissional"}
        </Button>
      </form>
    </section>
  );
}

function validate(values: CreateProfessionalRequest): FieldErrors {
  const errors: FieldErrors = {};
  const nameError = validateProfessionalName(values.name);
  if (nameError) errors.name = nameError;
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
