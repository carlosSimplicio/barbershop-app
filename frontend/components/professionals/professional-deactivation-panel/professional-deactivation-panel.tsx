"use client";

import { useMutationRequest } from "@/lib/api/use-mutation-request/use-mutation-request";
import { useState, type RefObject } from "react";
import { Button, Feedback, Modal, ModalActions } from "@/components/ui";
import {
  deactivateProfessional,
  type Professional,
} from "@/lib/professionals/professionals-api";
import { classifyProfessionalOperation } from "@/lib/professionals/professional-operation-result";
import formStyles from "../create-professional-form/create-professional-form.module.css";

type Props = {
  returnFocusRef?: RefObject<HTMLElement | null>;
  fallbackFocusRef?: RefObject<HTMLElement | null>;
  record: Professional;
  canWrite: boolean;
  onClose: () => void;
  onRecordConfirmed: (record: Professional) => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange: (busy: boolean) => void;
};
type Notice = {
  variant: "success" | "error" | "warning";
  title: string;
  message: string;
};

export function ProfessionalDeactivationPanel({
  record,
  returnFocusRef,
  fallbackFocusRef,
  canWrite,
  onClose,
  onRecordConfirmed,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
}: Props) {
  const { submitting, begin, isPending } = useMutationRequest(onBusyChange);
  const [finished, setFinished] = useState(false);
  const [missing, setMissing] = useState(false);
  const [notice, setNotice] = useState<Notice | null>(null);
  function showError() {
    setNotice({
      variant: "error",
      title: "Não foi possível desativar o profissional",
      message: "Tente novamente mais tarde.",
    });
  }
  async function run() {
    if (isPending() || !canWrite || finished || missing) return;
    const request = begin();
    if (!request) return;
    try {
      const result = await deactivateProfessional(record.id, request.signal);
      if (request.signal.aborted) return;
      const outcome = classifyProfessionalOperation(result, 200);
      if (outcome.state === "unauthorized") {
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setMissing(true);
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
        !outcome.record.active
      ) {
        onRecordConfirmed(outcome.record);
        setFinished(!outcome.record.active);
        setNotice({
          variant: "success",
          title: "Profissional desativado",
          message:
            "As sessões foram encerradas e novos logins estão bloqueados. O cadastro foi preservado.",
        });
        return;
      }
      showError();
    } catch {
      if (!request.signal.aborted) showError();
    } finally {
      request.finish();
    }
  }
  return (
    <Modal
      open
      title="Desativar profissional"
      onClose={onClose}
      returnFocusRef={returnFocusRef}
      fallbackFocusRef={fallbackFocusRef}
      closeDisabled={submitting}
    >
      <section
        aria-label="Desativar profissional"
        className={formStyles.embedded}
      >
        <p>Desativar “{record.name}”?</p>
        <p>
          As sessões serão encerradas e o profissional não poderá entrar. O
          cadastro será preservado.
        </p>
        {notice ? (
          <Feedback variant={notice.variant} title={notice.title}>
            {notice.message}
          </Feedback>
        ) : null}
        <ModalActions>
          {!finished && !missing ? (
            <Button
              disabled={!canWrite || submitting}
              onClick={() => void run()}
            >
              {submitting ? "Aguarde…" : "Confirmar"}
            </Button>
          ) : null}
          <Button variant="secondary" disabled={submitting} onClick={onClose}>
            {finished || missing ? "Fechar" : "Cancelar"}
          </Button>
        </ModalActions>
      </section>
    </Modal>
  );
}
