"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button, Feedback } from "@/components/ui";
import { classifyServiceOperation } from "@/lib/services/service-operation-result";
import {
  getService,
  updateService,
  type ServiceRecord,
} from "@/lib/services/services-api";
import {
  buildServicePatch,
  emptyServiceValues,
  focusServiceError,
  recordToServiceValues,
  reconcileServiceValues,
  type FieldErrors,
} from "@/lib/services/service-form-model";
import { ServiceFields } from "../service-fields/service-fields";
import formStyles from "../service-form.module.css";
import styles from "./edit-service-panel.module.css";

export type EditServiceProps = {
  serviceId: string;
  canWrite: boolean;
  onRecordConfirmed: (record: ServiceRecord) => void;
  onClose: () => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange?: (busy: boolean) => void;
  reloadVersion?: number;
};
type Load = "loading" | "ready" | "error" | "missing" | "unauthorized";
type Notice = {
  title: string;
  message: string;
  variant: "success" | "error" | "warning";
};

export function EditServicePanel({
  serviceId,
  canWrite,
  onRecordConfirmed,
  onClose,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
  reloadVersion = 0,
}: EditServiceProps) {
  const [record, setRecord] = useState<ServiceRecord | null>(null);
  const [values, setValues] = useState(emptyServiceValues);
  const [load, setLoad] = useState<Load>("loading");
  const [loadedVersion, setLoadedVersion] = useState(reloadVersion);
  const [version, setVersion] = useState(0);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [notice, setNotice] = useState<Notice | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const baseline = useRef<ServiceRecord | null>(null);
  const pending = useRef(false);
  const pendingErrorFocus = useRef(false);
  const focusOnLoad = useRef(true);
  const form = useRef<HTMLFormElement>(null);
  const panel = useRef<HTMLElement>(null);
  const mutation = useRef<AbortController | null>(null);
  useEffect(() => () => mutation.current?.abort(), []);
  useEffect(() => {
    if (load === "ready" && focusOnLoad.current) {
      focusOnLoad.current = false;
      panel.current?.focus();
    }
    if (!submitting && load === "ready" && pendingErrorFocus.current) {
      pendingErrorFocus.current = false;
      focusServiceError(form.current, errors);
    }
  }, [load, errors, submitting]);

  useEffect(() => {
    const controller = new AbortController();
    void getService(serviceId, controller.signal)
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
        setLoadedVersion(reloadVersion);
        setRecord(result.data);
        onRecordConfirmed(result.data);
        const previous = baseline.current;
        setValues((current) =>
          previous
            ? reconcileServiceValues(current, previous, result.data)
            : recordToServiceValues(result.data),
        );
        baseline.current = result.data;
        if (previous)
          setNotice({
            variant: "warning",
            title: "Estado atualizado",
            message:
              "Campos editados preservados; os demais dados foram atualizados. Revise antes de salvar.",
          });
        setUncertain(false);
        setLoad("ready");
      })
      .catch(() => {
        if (!controller.signal.aborted) setLoad("error");
      });
    return () => controller.abort();
  }, [serviceId, version, reloadVersion, onUnauthorized, onRecordConfirmed]);

  function reload() {
    if (pending.current) return;
    setLoad("loading");
    setVersion((value) => value + 1);
  }
  function markUncertain() {
    setUncertain(true);
    setNotice({
      variant: "warning",
      title: "Não foi possível confirmar a alteração",
      message:
        "Recarregue o serviço para conferir o estado salvo antes de enviar novamente. Seu rascunho será preservado.",
    });
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (
      !record ||
      !canWrite ||
      pending.current ||
      uncertain ||
      load !== "ready" ||
      loadedVersion !== reloadVersion
    )
      return;
    const checked = buildServicePatch(values, record);
    setErrors(checked.errors);
    if (!checked.request) {
      focusServiceError(form.current, checked.errors);
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
      const result = await updateService(
        serviceId,
        checked.request,
        controller.signal,
      );
      if (controller.signal.aborted) return;
      const outcome = classifyServiceOperation(result, 200);
      if (outcome.state === "unauthorized") {
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setLoad("missing");
        return;
      }
      if (outcome.state === "confirmed") {
        setRecord(outcome.record);
        baseline.current = outcome.record;
        setValues(recordToServiceValues(outcome.record));
        setNotice({
          variant: "success",
          title: "Serviço atualizado",
          message: "As alterações foram salvas no catálogo.",
        });
        onRecordConfirmed(outcome.record);
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
      if (outcome.state === "uncertain") markUncertain();
      else
        setNotice({
          variant: "error",
          title: "Não foi possível salvar o serviço",
          message: "Verifique os dados e tente novamente.",
        });
    } catch {
      if (!controller.signal.aborted) markUncertain();
    } finally {
      onBusyChange?.(false);
      if (!controller.signal.aborted) {
        pending.current = false;
        setSubmitting(false);
      }
    }
  }
  if (load === "unauthorized") return null;
  const disabled =
    !canWrite ||
    submitting ||
    uncertain ||
    load !== "ready" ||
    loadedVersion !== reloadVersion;
  return (
    <section
      ref={panel}
      tabIndex={-1}
      aria-label="Editar serviço"
      className={formStyles.panel}
    >
      <h2>Editar serviço</h2>
      {load === "loading" ? <p role="status">Carregando serviço…</p> : null}
      {load === "error" ? (
        <Feedback variant="error" title="Não foi possível consultar o serviço">
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
          <ServiceFields
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
      <div className={styles.actions}>
        <Button onClick={reload} disabled={submitting || load === "loading"}>
          Recarregar serviço
        </Button>
        <Button onClick={onClose} disabled={submitting}>
          Cancelar edição
        </Button>
      </div>
    </section>
  );
}
