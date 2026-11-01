"use client";

import { useMutationRequest } from "@/lib/api/use-mutation-request/use-mutation-request";
import { useEffect, useRef, useState } from "react";
import { Button, Feedback, Modal, ModalActions } from "@/components/ui";
import { classifyOfferingOperation } from "@/lib/catalog/offering-operation-result";
import { setOfferingStatus, type Offering } from "@/lib/catalog/catalog-api";
import styles from "./offering-status-action.module.css";

export type OfferingStatusProps = {
  record: Offering;
  canWrite: boolean;
  onChanged: (record: Offering) => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onBusyChange: (busy: boolean) => void;
};
type Notice = {
  title: string;
  message: string;
  variant: "success" | "warning" | "error";
};
export function OfferingStatusAction({
  record,
  canWrite,
  onChanged,
  onReadOnly,
  onUnauthorized,
  onBusyChange,
}: OfferingStatusProps) {
  const [desiredActive, setDesiredActive] = useState<boolean | null>(null);
  const { submitting, begin, isPending } = useMutationRequest(onBusyChange);
  const [missing, setMissing] = useState(false);
  const [notice, setNotice] = useState<Notice | null>(null);
  const focusAfter = useRef(false);
  const action = useRef<HTMLButtonElement>(null);
  const container = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!submitting && focusAfter.current) {
      focusAfter.current = false;
      if (canWrite) action.current?.focus();
      else container.current?.focus();
    }
  }, [submitting, canWrite, notice, desiredActive]);

  async function run() {
    if (isPending() || !canWrite || desiredActive === null || missing) return;
    const request = begin();
    if (!request) return;
    try {
      const result = await setOfferingStatus(
        record.id,
        desiredActive,
        request.signal,
      );
      if (request.signal.aborted) return;
      const outcome = classifyOfferingOperation(result, 200);
      if (outcome.state === "unauthorized") {
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setMissing(true);
        setDesiredActive(null);
        setNotice({
          variant: "warning",
          title: "Serviço não encontrado",
          message: "Atualize a lista para consultar o catálogo disponível.",
        });
        return;
      }
      if (outcome.state === "confirmed") {
        setMissing(false);
        setDesiredActive(null);
        focusAfter.current = true;
        setNotice({
          variant: "success",
          title: outcome.record.active
            ? "Serviço ativado"
            : "Serviço desativado",
          message: outcome.record.active
            ? "O serviço está ativo no catálogo."
            : "O serviço está inativo e permanece no catálogo.",
        });
        onChanged(outcome.record);
        return;
      }
      if (outcome.state === "read_only") {
        onReadOnly();
        setNotice({
          variant: "warning",
          title: "Alteração indisponível",
          message: "O estabelecimento está em modo somente leitura.",
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
  function closeConfirmation() {
    if (isPending()) return;
    focusAfter.current = true;
    setDesiredActive(null);
  }
  function showError() {
    setNotice({
      variant: "error",
      title: "Não foi possível alterar o serviço",
      message: "Tente novamente mais tarde.",
    });
  }
  return (
    <div ref={container} tabIndex={-1} className={styles.action}>
      <div className={styles.header}>
        <p>Estado: {record.active ? "Ativo" : "Inativo"}</p>
        <ModalActions>
          <Button
            ref={action}
            disabled={!canWrite || submitting || missing}
            aria-label={`${record.active ? "Desativar" : "Ativar"} ${record.name}`}
            onClick={() => {
              setDesiredActive(!record.active);
              setNotice(null);
            }}
          >
            {record.active ? "Desativar" : "Ativar"}
          </Button>
        </ModalActions>
      </div>
      {desiredActive !== null ? (
        <Modal
          open
          title={desiredActive ? "Ativar serviço" : "Desativar serviço"}
          closeDisabled={submitting}
          fallbackFocusRef={container}
          onClose={closeConfirmation}
        >
          <div
            role="group"
            aria-label={`Confirmar alteração de ${record.name}`}
            className={styles.confirmation}
          >
            <p>
              {desiredActive ? "Ativar" : "Desativar"} “{record.name}”? O
              serviço permanecerá no catálogo.
            </p>
            {notice ? (
              <Feedback variant={notice.variant} title={notice.title}>
                {notice.message}
              </Feedback>
            ) : null}
            <ModalActions>
              <Button
                disabled={!canWrite || submitting}
                onClick={() => void run()}
              >
                Confirmar
              </Button>
              <Button
                variant="secondary"
                disabled={submitting}
                onClick={closeConfirmation}
              >
                Cancelar
              </Button>
            </ModalActions>
          </div>
        </Modal>
      ) : notice ? (
        <Feedback variant={notice.variant} title={notice.title}>
          {notice.message}
        </Feedback>
      ) : null}
    </div>
  );
}
