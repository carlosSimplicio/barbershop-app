"use client";

import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { Button, Feedback, ModalActions, TextField } from "@/components/ui";
import {
  getProfessional,
  updateProfessional,
  type Professional,
} from "@/lib/professionals/professionals-api";
import {
  classifyProfessionalOperation,
  isProfessional,
} from "@/lib/professionals/professional-operation-result";
import {
  validateProfessionalName,
  PROFESSIONAL_NAME_HINT,
} from "@/lib/professionals/professional-form-model";
import formStyles from "../create-professional-form/create-professional-form.module.css";

type Props = {
  professionalId: string;
  canWrite: boolean;
  blocked?: boolean;
  onRecordConfirmed: (record: Professional) => void;
  onClose: () => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange: (busy: boolean) => void;
  embedded?: boolean;
  renderActions?: (
    record: Professional,
    disabled: boolean,
    onChanged: (record: Professional) => void,
  ) => ReactNode;
};
type Load = "loading" | "ready" | "error" | "missing";
type Notice = {
  variant: "error" | "warning" | "success";
  title: string;
  message: string;
};

export function EditProfessionalPanel({
  professionalId,
  canWrite,
  blocked = false,
  onRecordConfirmed,
  onClose,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
  embedded = false,
  renderActions,
}: Props) {
  const [record, setRecord] = useState<Professional | null>(null);
  const [name, setName] = useState("");
  const [load, setLoad] = useState<Load>("loading");
  const [version, setVersion] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<Notice | null>(null);
  const pending = useRef(false);
  const mutation = useRef<AbortController | null>(null);
  const field = useRef<HTMLInputElement>(null);
  const panel = useRef<HTMLElement>(null);
  const focusError = useRef(false);
  const focusInitial = useRef(true);
  function applyRecord(confirmed: Professional, preserveDraft = true) {
    setName((current) =>
      preserveDraft && record && current.trim() !== record.name
        ? current
        : confirmed.name,
    );
    setRecord(confirmed);
    onRecordConfirmed(confirmed);
  }
  useEffect(() => () => mutation.current?.abort(), []);
  useEffect(() => {
    if (load === "ready" && focusInitial.current) {
      focusInitial.current = false;
      if (!embedded) panel.current?.focus();
    }
    if (!submitting && focusError.current) {
      focusError.current = false;
      field.current?.focus();
    }
  }, [load, submitting, error, embedded]);
  useEffect(() => {
    const controller = new AbortController();
    let counted = true;
    const release = () => {
      if (counted) {
        counted = false;
        onBusyChange(false);
      }
    };
    onBusyChange(true);
    void getProfessional(professionalId, controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return;
        if (result.status === 401) {
          onUnauthorized();
          return;
        }
        if (result.status === 404) {
          setLoad("missing");
          return;
        }
        if (
          result.state !== "success" ||
          result.status !== 200 ||
          !isProfessional(result.data) ||
          result.data.id !== professionalId
        ) {
          setLoad("error");
          return;
        }
        setName(result.data.name);
        setRecord(result.data);
        onRecordConfirmed(result.data);
        setLoad("ready");
      })
      .catch(() => {
        if (!controller.signal.aborted) setLoad("error");
      })
      .finally(release);
    return () => {
      controller.abort();
      release();
    };
  }, [
    professionalId,
    version,
    onBusyChange,
    onRecordConfirmed,
    onUnauthorized,
  ]);
  function reload() {
    if (pending.current || blocked) return;
    setLoad("loading");
    setVersion((value) => value + 1);
  }
  function showError() {
    setNotice({
      variant: "error",
      title: "Não foi possível salvar o profissional",
      message: "Tente novamente mais tarde.",
    });
  }
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!record || !canWrite || pending.current || load !== "ready") return;
    const validation = validateProfessionalName(name);
    setError(validation);
    if (validation) {
      field.current?.focus();
      return;
    }
    if (name.trim() === record.name) {
      setNotice({
        variant: "warning",
        title: "Nenhuma alteração para salvar",
        message: "Altere o nome para salvar o cadastro.",
      });
      return;
    }
    pending.current = true;
    setSubmitting(true);
    onBusyChange(true);
    setNotice(null);
    const controller = new AbortController();
    mutation.current = controller;
    try {
      const result = await updateProfessional(
        professionalId,
        { name: name.trim() },
        controller.signal,
      );
      if (controller.signal.aborted) return;
      const outcome = classifyProfessionalOperation(result, 200);
      if (outcome.state === "unauthorized") {
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setLoad("missing");
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
      if (
        outcome.state === "confirmed" &&
        outcome.record.id === professionalId
      ) {
        applyRecord(outcome.record, false);
        setNotice({
          variant: "success",
          title: "Profissional atualizado",
          message: "O nome foi salvo no cadastro.",
        });
        return;
      }
      if (outcome.state === "validation" && outcome.fields.name) {
        setError(outcome.fields.name);
        focusError.current = true;
        return;
      }
      showError();
    } catch {
      if (!controller.signal.aborted) showError();
    } finally {
      onBusyChange(false);
      if (!controller.signal.aborted) {
        pending.current = false;
        setSubmitting(false);
      }
    }
  }
  const disabled = !canWrite || submitting || load !== "ready";
  const editActions = (
    <ModalActions>
      {record && load !== "missing" ? (
        <Button type="submit" disabled={disabled}>
          {submitting ? "Salvando…" : "Salvar"}
        </Button>
      ) : null}
      <Button onClick={onClose} disabled={submitting || blocked}>
        Cancelar
      </Button>
    </ModalActions>
  );
  return (
    <section
      ref={panel}
      tabIndex={-1}
      aria-label="Editar profissional"
      className={embedded ? formStyles.embedded : formStyles.panel}
    >
      {!embedded ? <h2>Editar profissional</h2> : null}
      {record ? (
        renderActions ? (
          renderActions(record, disabled, applyRecord)
        ) : (
          <p>Estado: {record.active ? "Ativo" : "Inativo"}</p>
        )
      ) : null}
      {load === "loading" ? (
        <p role="status">Carregando profissional…</p>
      ) : null}
      {load === "error" ? (
        <Feedback
          variant="error"
          title="Não foi possível consultar o profissional"
          action={
            <ModalActions>
              <Button onClick={reload} disabled={submitting || blocked}>
                Tentar novamente
              </Button>
            </ModalActions>
          }
        >
          Verifique sua conexão e tente novamente.
        </Feedback>
      ) : null}
      {load === "missing" ? (
        <Feedback variant="warning" title="Profissional não encontrado">
          O cadastro selecionado não está disponível.
        </Feedback>
      ) : null}
      {notice ? (
        <Feedback variant={notice.variant} title={notice.title}>
          {notice.message}
        </Feedback>
      ) : null}
      {record && load !== "missing" ? (
        <form noValidate onSubmit={submit} className={formStyles.form}>
          <TextField
            label="Nome do profissional"
            hint={PROFESSIONAL_NAME_HINT}
            value={name}
            ref={field}
            error={error}
            disabled={disabled}
            required
            onChange={(event) => {
              setName(event.target.value);
              setError(undefined);
              setNotice(null);
            }}
          />
          <TextField
            label="E-mail do profissional"
            value={record.email}
            readOnly
            hint="O e-mail de acesso não pode ser alterado."
          />
          {editActions}
        </form>
      ) : (
        editActions
      )}
    </section>
  );
}
