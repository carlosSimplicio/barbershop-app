"use client";

import { useMutationRequest } from "@/lib/api/use-mutation-request/use-mutation-request";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button, Feedback, ModalActions } from "@/components/ui";
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
}: {
  canWrite: boolean;
  onCreated: () => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange?: (busy: boolean) => void;
  embedded?: boolean;
}) {
  const [values, setValues] = useState<OfferingValues>(emptyOfferingValues);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [feedback, setFeedback] = useState<FeedbackState | null>(null);
  const { submitting, begin, isPending } = useMutationRequest(onBusyChange);
  const form = useRef<HTMLFormElement>(null);
  const pendingErrorFocus = useRef(false);
  useEffect(() => {
    if (!submitting && pendingErrorFocus.current) {
      pendingErrorFocus.current = false;
      focusOfferingError(form.current, errors);
    }
  }, [errors, submitting]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (isPending() || !canWrite) return;
    const checked = validateOfferingValues(values);
    setErrors(checked.errors);
    if (!checked.request) {
      focusOfferingError(form.current, checked.errors);
      return;
    }
    const request = begin();
    if (!request) return;
    setFeedback(null);
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
      setFeedback({
        variant: "error",
        title: "Não foi possível cadastrar o serviço",
        message: "Verifique os dados e tente novamente.",
      });
    } catch {
      if (!request.signal.aborted)
        setFeedback({
          variant: "error",
          title: "Não foi possível cadastrar o serviço",
          message: "Verifique sua conexão e tente novamente.",
        });
    } finally {
      request.finish();
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
      <form ref={form} onSubmit={submit} noValidate className={styles.form}>
        <OfferingFields
          values={values}
          errors={errors}
          disabled={!canWrite || submitting}
          onChange={change}
        />
        <ModalActions>
          <Button type="submit" disabled={!canWrite || submitting}>
            {submitting ? "Cadastrando…" : "Cadastrar"}
          </Button>
        </ModalActions>
      </form>
    </section>
  );
}
