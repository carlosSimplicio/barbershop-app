"use client";

import { useMutationRequest } from "@/lib/api/use-mutation-request/use-mutation-request";
import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import {
  Button,
  Feedback,
  ModalActions,
  PasswordField,
  TextField,
} from "@/components/ui";
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
  const { submitting, begin, isPending } = useMutationRequest(onBusyChange);
  const focusAfterResponse = useRef(false);
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
    if (isPending() || !canWrite) return;

    const nextErrors = validate(values);
    setErrors(nextErrors);
    if (fieldOrder.some((field) => nextErrors[field])) {
      focusFirstError(nextErrors);
      return;
    }

    const requestController = begin();
    if (!requestController) return;
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
      setFeedback({
        variant: "error",
        title: "Não foi possível cadastrar o profissional",
        message: "Tente novamente mais tarde.",
      });
    } catch {
      if (!requestController.signal.aborted) {
        setFeedback({
          variant: "error",
          title: "Não foi possível cadastrar o profissional",
          message: "Verifique sua conexão e tente novamente.",
        });
      }
    } finally {
      requestController.finish();
    }
  }

  return (
    <section aria-label="Cadastrar profissional" className={styles.panel}>
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
          hint={PROFESSIONAL_NAME_HINT}
          label="Nome"
          onChange={(event) => update("name", event.target.value)}
          disabled={submitting || !canWrite}
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
          disabled={submitting || !canWrite}
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
          disabled={submitting || !canWrite}
          ref={passwordRef}
          required
          value={values.temporary_password}
        />
        <ModalActions>
          <Button disabled={submitting || !canWrite} type="submit">
            {submitting ? "Cadastrando…" : "Cadastrar"}
          </Button>
        </ModalActions>
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
