"use client";

import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { Button, Feedback } from "@/components/ui";
import { classifyOfferingOperation } from "@/lib/catalog/offering-operation-result";
import {
  getOffering,
  updateOffering,
  type Offering,
} from "@/lib/catalog/catalog-api";
import {
  buildOfferingPatch,
  emptyOfferingValues,
  focusOfferingError,
  offeringToFormValues,
  mergeOfferingDraft,
  type FieldErrors,
} from "@/lib/catalog/offering-form-model";
import { OfferingFields } from "../offering-fields/offering-fields";
import formStyles from "../offering-form.module.css";
import styles from "./edit-offering-panel.module.css";

export type EditOfferingProps = {
  offeringId: string;
  canWrite: boolean;
  onRecordConfirmed: (record: Offering) => void;
  onClose: () => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange?: (busy: boolean) => void;
  embedded?: boolean;
  blocked?: boolean;
  renderActions?: (
    record: Offering,
    disabled: boolean,
    onChanged: (record: Offering) => void,
  ) => ReactNode;
};
type Load = "loading" | "ready" | "error" | "missing" | "unauthorized";
type Notice = {
  title: string;
  message: string;
  variant: "success" | "error" | "warning";
};

export function EditOfferingPanel({
  offeringId,
  canWrite,
  onRecordConfirmed,
  onClose,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
  embedded = false,
  blocked = false,
  renderActions,
}: EditOfferingProps) {
  const [record, setRecord] = useState<Offering | null>(null);
  const [values, setValues] = useState(emptyOfferingValues);
  const [load, setLoad] = useState<Load>("loading");
  const [version, setVersion] = useState(0);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [notice, setNotice] = useState<Notice | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const pending = useRef(false);
  const pendingErrorFocus = useRef(false);
  const focusOnLoad = useRef(true);
  const form = useRef<HTMLFormElement>(null);
  const panel = useRef<HTMLElement>(null);
  const mutation = useRef<AbortController | null>(null);
  function applyRecord(confirmed: Offering, preserveDraft = true) {
    setValues((current) =>
      preserveDraft && record
        ? mergeOfferingDraft(current, record, confirmed)
        : offeringToFormValues(confirmed),
    );
    setRecord(confirmed);
    onRecordConfirmed(confirmed);
  }
  useEffect(() => () => mutation.current?.abort(), []);
  useEffect(() => {
    if (load === "ready" && focusOnLoad.current) {
      focusOnLoad.current = false;
      if (!embedded) panel.current?.focus();
    }
    if (!submitting && load === "ready" && pendingErrorFocus.current) {
      pendingErrorFocus.current = false;
      focusOfferingError(form.current, errors);
    }
  }, [load, errors, submitting, embedded]);

  useEffect(() => {
    const controller = new AbortController();
    void getOffering(offeringId, controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return;
        if (result.status === 401) {
          setLoad("unauthorized");
          onUnauthorized();
          return;
        }
        if (result.status === 404) {
          setLoad("missing");
          return;
        }
        if (result.state !== "success" || result.status !== 200) {
          setLoad("error");
          return;
        }
        setValues(offeringToFormValues(result.data));
        setRecord(result.data);
        onRecordConfirmed(result.data);
        setLoad("ready");
      })
      .catch(() => {
        if (!controller.signal.aborted) setLoad("error");
      });
    return () => controller.abort();
  }, [offeringId, version, onUnauthorized, onRecordConfirmed]);

  function reload() {
    if (pending.current) return;
    setLoad("loading");
    setVersion((value) => value + 1);
  }
  function showError() {
    setNotice({
      variant: "error",
      title: "Não foi possível salvar o serviço",
      message: "Tente novamente mais tarde.",
    });
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!record || !canWrite || pending.current || load !== "ready") return;
    const checked = buildOfferingPatch(values, record);
    setErrors(checked.errors);
    if (!checked.request) {
      focusOfferingError(form.current, checked.errors);
      return;
    }
    if (!Object.keys(checked.request).length) {
      setNotice({
        variant: "warning",
        title: "Nenhuma alteração para salvar",
        message: "Edite um dos campos para atualizar o serviço.",
      });
      return;
    }
    pending.current = true;
    onBusyChange?.(true);
    setSubmitting(true);
    setNotice(null);
    const controller = new AbortController();
    mutation.current = controller;
    try {
      const result = await updateOffering(
        offeringId,
        checked.request,
        controller.signal,
      );
      if (controller.signal.aborted) return;
      const outcome = classifyOfferingOperation(result, 200);
      if (outcome.state === "unauthorized") {
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setLoad("missing");
        return;
      }
      if (outcome.state === "confirmed") {
        applyRecord(outcome.record, false);
        setNotice({
          variant: "success",
          title: "Serviço atualizado",
          message: "As alterações foram salvas no catálogo.",
        });
        return;
      }
      if (outcome.state === "read_only") {
        onReadOnly();
        setNotice({
          variant: "warning",
          title: "Edição indisponível",
          message: "O estabelecimento está em modo somente leitura.",
        });
        return;
      }
      if (outcome.state === "validation") {
        pendingErrorFocus.current = true;
        setErrors(outcome.fields);
        return;
      }
      showError();
    } catch {
      if (!controller.signal.aborted) showError();
    } finally {
      onBusyChange?.(false);
      if (!controller.signal.aborted) {
        pending.current = false;
        setSubmitting(false);
      }
    }
  }
  if (load === "unauthorized") return null;
  const disabled = !canWrite || submitting || load !== "ready";
  return (
    <section
      ref={panel}
      tabIndex={-1}
      aria-label="Editar serviço"
      className={embedded ? formStyles.embedded : formStyles.panel}
    >
      {!embedded ? <h2>Editar serviço</h2> : null}
      {load === "loading" ? <p role="status">Carregando serviço…</p> : null}
      {load === "error" ? (
        <Feedback
          variant="error"
          title="Não foi possível consultar o serviço"
          action={
            <Button onClick={reload} disabled={submitting || blocked}>
              Tentar novamente
            </Button>
          }
        >
          Verifique sua conexão e tente novamente.
        </Feedback>
      ) : null}
      {load === "missing" ? (
        <Feedback variant="warning" title="Serviço não encontrado">
          O serviço selecionado não está disponível.
        </Feedback>
      ) : null}
      {notice ? (
        <Feedback variant={notice.variant} title={notice.title}>
          {notice.message}
        </Feedback>
      ) : null}
      {record && load !== "missing" ? (
        <form
          ref={form}
          onSubmit={submit}
          noValidate
          className={`${formStyles.form} ${styles.form}`}
        >
          <p>Estado: {record.active ? "Ativo" : "Inativo"}</p>
          <OfferingFields
            values={values}
            errors={errors}
            disabled={disabled}
            onChange={(field, value) => {
              setValues((current) => ({ ...current, [field]: value }));
              setErrors({});
              setNotice(null);
            }}
          />
          <Button type="submit" disabled={disabled}>
            {submitting ? "Salvando…" : "Salvar alterações"}
          </Button>
        </form>
      ) : null}
      {/* Keep state confirmations dismissible when the edited resource is missing. */}
      {record ? renderActions?.(record, disabled, applyRecord) : null}
      <div className={styles.actions}>
        <Button onClick={onClose} disabled={submitting || blocked}>
          Cancelar edição
        </Button>
      </div>
    </section>
  );
}
