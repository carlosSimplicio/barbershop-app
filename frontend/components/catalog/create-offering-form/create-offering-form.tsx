"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button, Feedback } from "@/components/ui";
import { classifyOfferingOperation } from "@/lib/catalog/offering-operation-result";
import { createOffering } from "@/lib/catalog/catalog-api";
import {
  emptyOfferingValues,
  focusOfferingError,
  validateOfferingValues,
  type FieldErrors,
  type OfferingValues,
} from "@/lib/catalog/offering-form-model";
import { OfferingFields } from "../offering-fields/offering-fields";
import styles from "../offering-form.module.css";

type FeedbackState = {
  title: string;
  message: string;
  variant: "error" | "success" | "warning";
};
export function CreateOfferingForm({
  canWrite,
  onCreated,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
  embedded = false,
  onReviewList,
}: {
  canWrite: boolean;
  onCreated: () => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange?: (busy: boolean) => void;
  embedded?: boolean;
  onReviewList?: () => void;
}) {
  const [values, setValues] = useState<OfferingValues>(emptyOfferingValues);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const [verification, setVerification] = useState(false);
  const form = useRef<HTMLFormElement>(null);
  const pending = useRef(false);
  const pendingErrorFocus = useRef(false);
  const controller = useRef<AbortController | null>(null);
  useEffect(() => () => controller.current?.abort(), []);
  useEffect(() => {
    if (!submitting && pendingErrorFocus.current) {
      pendingErrorFocus.current = false;
      focusOfferingError(form.current, errors);
    }
  }, [errors, submitting]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending.current || !canWrite || uncertain) return;
    const checked = validateOfferingValues(values);
    setErrors(checked.errors);
    if (!checked.request) {
      focusOfferingError(form.current, checked.errors);
      return;
    }
    pending.current = true;
    onBusyChange?.(true);
    setSubmitting(true);
    setFeedback(null);
    const request = new AbortController();
    controller.current = request;
    const markUncertain = () => {
      setUncertain(true);
      setFeedback({
        variant: "warning",
        title: "Não foi possível confirmar o cadastro",
        message:
          "Atualize a lista para conferir se o serviço foi criado antes de tentar novamente.",
      });
    };
    try {
      const result = await createOffering(checked.request, request.signal);
      if (request.signal.aborted) return;
      const outcome = classifyOfferingOperation(result, 201);
      if (outcome.state === "unauthorized") {
        onUnauthorized();
        return;
      }
      if (outcome.state === "confirmed") {
        setValues(emptyOfferingValues);
        setErrors({});
        setFeedback({
          variant: "success",
          title: "Serviço cadastrado",
          message: "O serviço foi salvo no catálogo.",
        });
        onCreated();
        return;
      }
      if (outcome.state === "read_only") {
        onReadOnly();
        setFeedback({
          variant: "warning",
          title: "Cadastro indisponível",
          message: "O estabelecimento está em modo somente leitura.",
        });
        return;
      }
      if (outcome.state === "validation") {
        pendingErrorFocus.current = true;
        setErrors(outcome.fields);
        return;
      }
      if (outcome.state === "uncertain") markUncertain();
      else
        setFeedback({
          variant: "error",
          title: "Não foi possível cadastrar o serviço",
          message: "Verifique os dados e tente novamente.",
        });
    } catch {
      if (!request.signal.aborted) markUncertain();
    } finally {
      onBusyChange?.(false);
      if (!request.signal.aborted) {
        pending.current = false;
        setSubmitting(false);
      }
    }
  }
  function change(field: keyof OfferingValues, value: string) {
    setValues((current) => ({ ...current, [field]: value }));
    setErrors({});
    setFeedback(null);
  }
  return (
    <section
      aria-label="Cadastrar serviço"
      className={embedded ? styles.embedded : styles.panel}
    >
      {!embedded ? <h2>Cadastrar serviço</h2> : null}
      {feedback ? (
        <Feedback variant={feedback.variant} title={feedback.title}>
          {feedback.message}
        </Feedback>
      ) : null}
      {uncertain ? (
        <Button
          disabled={!canWrite}
          onClick={() => {
            onCreated();
            setVerification(true);
            onReviewList?.();
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
      <form ref={form} onSubmit={submit} noValidate className={styles.form}>
        <OfferingFields
          values={values}
          errors={errors}
          disabled={!canWrite || submitting || uncertain}
          onChange={change}
        />
        <Button type="submit" disabled={!canWrite || submitting || uncertain}>
          {submitting ? "Cadastrando…" : "Cadastrar serviço"}
        </Button>
      </form>
    </section>
  );
}
