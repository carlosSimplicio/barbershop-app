"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button, Feedback } from "@/components/ui";
import { createService } from "@/lib/services/services-api";
import {
  emptyServiceValues,
  focusServiceError,
  serverServiceErrors,
  validateServiceValues,
  type FieldErrors,
  type ServiceValues,
} from "@/lib/services/service-form-model";
import { ServiceFields } from "../service-fields/service-fields";
import styles from "./create-service-form.module.css";

type FeedbackState = {
  title: string;
  message: string;
  variant: "error" | "success" | "warning";
};
export function CreateServiceForm({
  canWrite,
  onCreated,
  onReadOnly,
  onUnauthorized,
}: {
  canWrite: boolean;
  onCreated: () => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
}) {
  const [values, setValues] = useState<ServiceValues>(emptyServiceValues);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const form = useRef<HTMLFormElement>(null);
  const pending = useRef(false);
  const pendingErrorFocus = useRef(false);
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => controller.current?.abort(), []);
  useEffect(() => {
    if (!submitting && pendingErrorFocus.current) {
      pendingErrorFocus.current = false;
      focusServiceError(form.current, errors);
    }
  }, [errors, submitting]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending.current || !canWrite) return;
    const checked = validateServiceValues(values);
    setErrors(checked.errors);
    if (!checked.request) {
      focusServiceError(form.current, checked.errors);
      return;
    }
    pending.current = true;
    setSubmitting(true);
    setFeedback(null);
    const request = new AbortController();
    controller.current = request;
    const uncertain = () =>
      setFeedback({
        variant: "warning",
        title: "Não foi possível confirmar o cadastro",
        message:
          "Atualize a lista para conferir se o serviço foi criado antes de tentar novamente.",
      });
    try {
      const result = await createService(checked.request, request.signal);
      if (request.signal.aborted) return;
      if (result.status === 401) {
        onUnauthorized();
        return;
      }
      if (result.state === "success" && result.status === 201) {
        setValues(emptyServiceValues);
        setErrors({});
        setFeedback({
          variant: "success",
          title: "Serviço cadastrado",
          message: "O serviço foi salvo no catálogo.",
        });
        onCreated();
        return;
      }
      if (result.state === "error" && result.error.code === "read_only") {
        onReadOnly();
        setFeedback({
          variant: "warning",
          title: "Cadastro indisponível",
          message: "O estabelecimento está em modo somente leitura.",
        });
        return;
      }
      if (result.state === "error") {
        const fieldErrors = serverServiceErrors(result.error.fields);
        if (Object.keys(fieldErrors).length) {
          pendingErrorFocus.current = true;
          setErrors(fieldErrors);
          return;
        }
      }
      if (result.status >= 500 || result.status < 400) uncertain();
      else
        setFeedback({
          variant: "error",
          title: "Não foi possível cadastrar o serviço",
          message: "Verifique os dados e tente novamente.",
        });
    } catch {
      if (!request.signal.aborted) uncertain();
    } finally {
      if (!request.signal.aborted) {
        pending.current = false;
        setSubmitting(false);
      }
    }
  }
  function change(field: keyof ServiceValues, value: string) {
    setValues((current) => ({ ...current, [field]: value }));
    setErrors({});
    setFeedback(null);
  }
  return (
    <section aria-label="Cadastrar serviço" className={styles.panel}>
      <h2>Cadastrar serviço</h2>
      {feedback ? (
        <Feedback variant={feedback.variant} title={feedback.title}>
          {feedback.message}
        </Feedback>
      ) : null}
      <form ref={form} onSubmit={submit} noValidate className={styles.form}>
        <ServiceFields
          values={values}
          errors={errors}
          disabled={!canWrite || submitting}
          onChange={change}
        />
        <Button type="submit" disabled={!canWrite || submitting}>
          {submitting ? "Cadastrando…" : "Cadastrar serviço"}
        </Button>
      </form>
    </section>
  );
}
