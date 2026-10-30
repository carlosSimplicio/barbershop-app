"use client";

import { useEffect, useRef, useState } from "react";
import { Button, Feedback } from "@/components/ui";
import {
  deactivateProfessional,
  getProfessional,
  type Professional,
} from "@/lib/professionals/professionals-api";
import { classifyProfessionalOperation } from "@/lib/professionals/professional-operation-result";
import formStyles from "../create-professional-form/create-professional-form.module.css";
import styles from "./professional-deactivation-panel.module.css";

type Props = {
  record: Professional;
  canWrite: boolean;
  onClose: () => void;
  onRecordConfirmed: (record: Professional) => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange: (busy: boolean) => void;
  onReconciliationChange: (needed: boolean) => void;
};
type Notice = {
  variant: "success" | "error" | "warning";
  title: string;
  message: string;
};

export function ProfessionalDeactivationPanel({
  record,
  canWrite,
  onClose,
  onRecordConfirmed,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
  onReconciliationChange,
}: Props) {
  const [submitting, setSubmitting] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const [finished, setFinished] = useState(false);
  const [missing, setMissing] = useState(false);
  const [notice, setNotice] = useState<Notice | null>(null);
  const pending = useRef(false);
  const controller = useRef<AbortController | null>(null);
  const panel = useRef<HTMLElement>(null);
  useEffect(() => {
    panel.current?.focus();
    return () => controller.current?.abort();
  }, []);
  function markUncertain() {
    setUncertain(true);
    onReconciliationChange(true);
    setNotice({
      variant: "warning",
      title: "Não foi possível confirmar a desativação",
      message:
        "Consulte o estado do profissional antes de outra ação. A desativação não será repetida automaticamente.",
    });
  }
  async function run(read: boolean) {
    if (
      pending.current ||
      (!read && (!canWrite || uncertain || finished || missing))
    )
      return;
    pending.current = true;
    setSubmitting(true);
    onBusyChange(true);
    const request = new AbortController();
    controller.current = request;
    try {
      const result = read
        ? await getProfessional(record.id, request.signal)
        : await deactivateProfessional(record.id, request.signal);
      if (request.signal.aborted) return;
      const outcome = classifyProfessionalOperation(result, 200);
      if (outcome.state === "unauthorized") {
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setMissing(true);
        setUncertain(false);
        onReconciliationChange(false);
        setNotice({
          variant: "warning",
          title: "Profissional não encontrado",
          message: "O cadastro não está disponível. Atualize a lista.",
        });
        return;
      }
      if (outcome.state === "read_only") {
        onReadOnly();
        setNotice({
          variant: "warning",
          title: "Desativação indisponível",
          message: "O estabelecimento está em modo somente leitura.",
        });
        return;
      }
      if (
        outcome.state === "confirmed" &&
        outcome.record.id === record.id &&
        (read || !outcome.record.active)
      ) {
        onRecordConfirmed(outcome.record);
        setUncertain(false);
        onReconciliationChange(false);
        setFinished(!outcome.record.active);
        setNotice(
          read
            ? {
                variant: "warning",
                title: "Estado consultado",
                message: outcome.record.active
                  ? "O profissional está ativo. Confira o cadastro antes de confirmar outra desativação."
                  : "O profissional está inativo. O cadastro foi preservado.",
              }
            : {
                variant: "success",
                title: "Profissional desativado",
                message:
                  "As sessões foram encerradas e novos logins estão bloqueados. O cadastro foi preservado.",
              },
        );
        return;
      }
      if (read) {
        setNotice({
          variant: "error",
          title: "Não foi possível consultar o estado",
          message: "Tente consultar novamente antes de outra ação.",
        });
        return;
      }
      if (outcome.state === "uncertain" || outcome.state === "confirmed")
        markUncertain();
      else
        setNotice({
          variant: "error",
          title: "Não foi possível desativar o profissional",
          message: "Confira o cadastro e tente novamente.",
        });
    } catch {
      if (!request.signal.aborted) {
        if (read)
          setNotice({
            variant: "error",
            title: "Não foi possível consultar o estado",
            message: "Tente consultar novamente antes de outra ação.",
          });
        else markUncertain();
      }
    } finally {
      onBusyChange(false);
      if (!request.signal.aborted) {
        pending.current = false;
        setSubmitting(false);
      }
    }
  }
  return (
    <section
      ref={panel}
      tabIndex={-1}
      aria-label="Desativar profissional"
      className={formStyles.panel}
    >
      <h2>Desativar {record.name}?</h2>
      <p>
        As sessões serão encerradas e o profissional não poderá entrar. O
        cadastro será preservado.
      </p>
      {notice ? (
        <Feedback variant={notice.variant} title={notice.title}>
          {notice.message}
        </Feedback>
      ) : null}
      <div className={styles.actions}>
        {!finished && !missing ? (
          <Button
            disabled={!canWrite || submitting || uncertain}
            onClick={() => void run(false)}
          >
            {submitting ? "Aguarde…" : "Confirmar desativação"}
          </Button>
        ) : null}
        {uncertain ? (
          <Button disabled={submitting} onClick={() => void run(true)}>
            Consultar estado do profissional
          </Button>
        ) : null}
        <Button disabled={submitting || uncertain} onClick={onClose}>
          {finished || missing ? "Fechar confirmação" : "Cancelar desativação"}
        </Button>
      </div>
    </section>
  );
}
