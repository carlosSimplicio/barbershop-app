"use client";

import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { Button, Feedback, PasswordField, TextField } from "@/components/ui";
import {
  registerOwner,
  type RegisterRequest,
} from "@/lib/auth/registration-api";

import styles from "./registration-form.module.css";

type FieldName = keyof RegisterRequest;
type FieldErrors = Partial<Record<FieldName, string>>;

const fieldOrder: FieldName[] = [
  "establishment_name",
  "slug",
  "time_zone",
  "email",
  "password",
];

const serverMessages: Record<string, string> = {
  invalid_name: "Informe um nome com até 120 caracteres.",
  invalid_slug: "Use de 3 a 63 letras minúsculas, números ou hífens.",
  reserved_slug: "Este endereço está reservado. Escolha outro.",
  invalid_time_zone: "Informe um fuso horário IANA válido.",
  invalid_email: "Digite um e-mail válido.",
  invalid_password: "Use de 8 a 32 caracteres, sem espaços.",
  email_unavailable: "Este e-mail já está cadastrado.",
  slug_unavailable: "Este endereço não está disponível.",
};

export function RegistrationForm() {
  const [values, setValues] = useState<RegisterRequest>(() => ({
    establishment_name: "",
    slug: "",
    time_zone: "America/Sao_Paulo",
    email: "",
    password: "",
  }));
  const [errors, setErrors] = useState<FieldErrors>({});
  const [feedback, setFeedback] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [pending, setPending] = useState(false);
  const slugEdited = useRef(false);
  const timeZoneEdited = useRef(false);
  const submittingRef = useRef(false);
  const nameRef = useRef<HTMLInputElement>(null);
  const slugRef = useRef<HTMLInputElement>(null);
  const timeZoneRef = useRef<HTMLInputElement>(null);
  const emailRef = useRef<HTMLInputElement>(null);
  const passwordRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!timeZoneEdited.current) {
      setValues((current) => ({ ...current, time_zone: detectTimeZone() }));
    }
  }, []);

  function focusFirstError(fieldErrors: FieldErrors) {
    const refs = {
      establishment_name: nameRef,
      slug: slugRef,
      time_zone: timeZoneRef,
      email: emailRef,
      password: passwordRef,
    };
    const first = fieldOrder.find((field) => fieldErrors[field]);
    if (first) refs[first].current?.focus();
  }

  function update(field: FieldName, value: string) {
    setValues((current) => ({ ...current, [field]: value }));
    setErrors((current) => ({ ...current, [field]: undefined }));
    setFeedback(null);
  }

  function updateName(value: string) {
    setValues((current) => ({
      ...current,
      establishment_name: value,
      slug: slugEdited.current ? current.slug : slugify(value),
    }));
    setErrors((current) => ({
      ...current,
      establishment_name: undefined,
      slug: slugEdited.current ? current.slug : undefined,
    }));
    setFeedback(null);
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
      const result = await registerOwner({
        ...values,
        establishment_name: values.establishment_name.trim(),
        email: values.email.trim(),
      });
      if (result.status === 202 && result.state !== "error") {
        setPending(true);
        return;
      }
      if (result.state === "error") {
        if (result.error.code === "registration_rate_limited") {
          setFeedback("Muitas tentativas. Tente novamente mais tarde.");
          return;
        }
        const serverErrors: FieldErrors = {};
        for (const issue of result.error.fields ?? []) {
          if (fieldOrder.includes(issue.field as FieldName)) {
            const message = serverMessages[issue.code];
            if (message) serverErrors[issue.field as FieldName] = message;
          }
        }
        if (fieldOrder.some((field) => serverErrors[field])) {
          setErrors(serverErrors);
          focusFirstError(serverErrors);
          return;
        }
      }
      setFeedback("Não foi possível concluir o cadastro. Tente novamente.");
    } catch {
      setFeedback(
        "Não foi possível concluir o cadastro. Verifique sua conexão e tente novamente.",
      );
    } finally {
      submittingRef.current = false;
      setSubmitting(false);
    }
  }

  if (pending) {
    return (
      <section
        aria-labelledby="registration-pending-title"
        className={styles.pending}
      >
        <h2 id="registration-pending-title">Confira seu e-mail</h2>
        <Feedback title="Cadastro recebido" variant="success">
          Confirme seu e-mail para ativar a conta. Sua avaliação começa somente
          após a confirmação.
        </Feedback>
      </section>
    );
  }

  return (
    <form
      className={styles.form}
      noValidate
      onSubmit={(event) => void submit(event)}
    >
      {feedback ? (
        <Feedback title="Não foi possível criar sua conta" variant="error">
          {feedback}
        </Feedback>
      ) : null}
      <TextField
        autoComplete="organization"
        readOnly={submitting}
        error={errors.establishment_name}
        label="Nome do estabelecimento"
        onChange={(event) => updateName(event.target.value)}
        placeholder="Ex.: Estúdio Aurora"
        ref={nameRef}
        required
        value={values.establishment_name}
      />
      <TextField
        autoCapitalize="none"
        autoComplete="off"
        readOnly={submitting}
        error={errors.slug}
        hint="Seus clientes encontrarão você por este endereço. Ele não poderá ser alterado."
        label="Endereço da sua página"
        maxLength={63}
        onChange={(event) => {
          slugEdited.current = true;
          update("slug", event.target.value);
        }}
        placeholder="estudio-aurora"
        ref={slugRef}
        required
        spellCheck={false}
        value={values.slug}
      />
      <div className={styles.row}>
        <TextField
          autoComplete="email"
          readOnly={submitting}
          error={errors.email}
          label="Seu e-mail"
          maxLength={254}
          onChange={(event) => update("email", event.target.value)}
          placeholder="voce@exemplo.com"
          ref={emailRef}
          required
          type="email"
          value={values.email}
        />
        <div>
          <TextField
            autoComplete="off"
            readOnly={submitting}
            error={errors.time_zone}
            hint="Confirme ou altere para o fuso IANA do seu estabelecimento."
            label="Fuso horário"
            list="registration-time-zones"
            onChange={(event) => {
              timeZoneEdited.current = true;
              update("time_zone", event.target.value);
            }}
            ref={timeZoneRef}
            required
            value={values.time_zone}
          />
          <datalist id="registration-time-zones">
            <option value="America/Sao_Paulo" />
            <option value="America/Manaus" />
            <option value="America/Rio_Branco" />
            <option value="America/Noronha" />
          </datalist>
        </div>
      </div>
      <PasswordField
        autoComplete="new-password"
        readOnly={submitting}
        error={errors.password}
        hint="Use de 8 a 32 caracteres, sem espaços."
        label="Crie uma senha"
        onChange={(event) => update("password", event.target.value)}
        ref={passwordRef}
        required
        value={values.password}
      />
      <Button className={styles.submit} disabled={submitting} type="submit">
        {submitting ? "Criando…" : "Criar meu espaço"}
      </Button>
      <p className={styles.note}>
        Enviaremos um link para confirmar seu e-mail. Os 14 dias de avaliação
        começam somente depois da confirmação.
      </p>
    </form>
  );
}

function detectTimeZone() {
  try {
    const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    if (timeZone && timeZone !== "Local") return timeZone;
  } catch {
    // Browser did not expose a usable time zone.
  }
  return "America/Sao_Paulo";
}

function slugify(value: string) {
  return value
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 63)
    .replace(/-$/, "");
}

function validate(values: RegisterRequest): FieldErrors {
  const errors: FieldErrors = {};
  if (
    !values.establishment_name.trim() ||
    [...values.establishment_name].length > 120
  ) {
    errors.establishment_name = "Informe o nome do estabelecimento.";
  }
  if (!values.slug) {
    errors.slug = "Escolha o endereço da sua página.";
  } else if (
    values.slug.length < 3 ||
    values.slug.length > 63 ||
    !/^[a-z0-9](?:[a-z0-9-]*[a-z0-9])$/.test(values.slug)
  ) {
    errors.slug = "Use de 3 a 63 letras minúsculas, números ou hífens.";
  } else if (["admin", "api", "login", "assets"].includes(values.slug)) {
    errors.slug = "Este endereço está reservado. Escolha outro.";
  }
  if (!values.time_zone || values.time_zone === "Local") {
    errors.time_zone = "Informe um fuso horário IANA válido.";
  }
  if (
    !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email.trim()) ||
    values.email.length > 254
  ) {
    errors.email = values.email.trim()
      ? "Digite um e-mail válido."
      : "Informe seu e-mail.";
  }
  if (!values.password) {
    errors.password = "Crie uma senha.";
  } else if (
    [...values.password].length < 8 ||
    [...values.password].length > 32 ||
    /\s/.test(values.password)
  ) {
    errors.password = "Use de 8 a 32 caracteres, sem espaços.";
  }
  return errors;
}
