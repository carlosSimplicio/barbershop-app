"use client";

import { useEffect, useRef, useState } from "react";
import { Button, Feedback } from "@/components/ui";
import { classifyOfferingOperation } from "@/lib/catalog/offering-operation-result";
import {
  getOffering,
  setOfferingStatus,
  type Offering,
} from "@/lib/catalog/catalog-api";
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
  const [target, setTarget] = useState<boolean | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [uncertain, setUncertain] = useState(false);
  const [missing, setMissing] = useState(false);
  const [notice, setNotice] = useState<Notice | null>(null);
  const controller = useRef<AbortController | null>(null);
  const pending = useRef(false);
  const focusAfter = useRef(false);
  const action = useRef<HTMLButtonElement>(null);
  const container = useRef<HTMLDivElement>(null);
  useEffect(() => () => controller.current?.abort(), []);
  useEffect(() => {
    if (!submitting && focusAfter.current) {
      focusAfter.current = false;
      if (canWrite) action.current?.focus();
      else container.current?.focus();
    }
  }, [submitting, canWrite, notice]);

  async function run(check: boolean) {
    if (
      pending.current ||
      (!check && (!canWrite || target === null || uncertain || missing))
    )
      return;
    pending.current = true;
    setSubmitting(true);
    onBusyChange(true);
    const request = new AbortController();
    controller.current = request;
    try {
      let result;
      if (check) result = await getOffering(record.id, request.signal);
      else {
        if (target === null) return;
        result = await setOfferingStatus(record.id, target, request.signal);
      }
      if (request.signal.aborted) return;
      const outcome = classifyOfferingOperation(result, 200);
      if (outcome.state === "unauthorized") {
        onUnauthorized();
        return;
      }
      if (outcome.state === "missing") {
        setMissing(true);
        setTarget(null);
        setNotice({
          variant: "warning",
          title: "Serviço não encontrado",
          message: "Atualize a lista para consultar o catálogo disponível.",
        });
        return;
      }
      if (outcome.state === "confirmed") {
        setUncertain(false);
        setMissing(false);
        setTarget(null);
        focusAfter.current = true;
        setNotice({
          variant: "success",
          title: check
            ? "Estado conferido"
            : outcome.record.active
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
      if (check)
        setNotice({
          variant: "error",
          title: "Não foi possível consultar o estado",
          message:
            "Verifique sua conexão e consulte novamente antes de alterar.",
        });
      else if (outcome.state === "uncertain") markUncertain();
      else
        setNotice({
          variant: "error",
          title: "Não foi possível alterar o serviço",
          message: "Atualize a lista e tente novamente.",
        });
    } catch {
      if (!request.signal.aborted) {
        if (check)
          setNotice({
            variant: "error",
            title: "Não foi possível consultar o estado",
            message:
              "Verifique sua conexão e consulte novamente antes de alterar.",
          });
        else markUncertain();
      }
    } finally {
      pending.current = false;
      if (!request.signal.aborted) setSubmitting(false);
      onBusyChange(false);
    }
  }
  function markUncertain() {
    setUncertain(true);
    setNotice({
      variant: "warning",
      title: "Não foi possível confirmar a alteração de estado",
      message:
        "Consulte o serviço para conferir o estado salvo antes de repetir a ação.",
    });
  }
  return (
    <div ref={container} tabIndex={-1} className={styles.action}>
      <Button
        ref={action}
        disabled={!canWrite || submitting || uncertain || missing}
        aria-label={`${record.active ? "Desativar" : "Ativar"} ${record.name}`}
        onClick={() => {
          setTarget(!record.active);
          setNotice(null);
        }}
      >
        {record.active ? "Desativar" : "Ativar"}
      </Button>
      {target !== null && !uncertain ? (
        <div
          role="group"
          aria-label={`Confirmar alteração de ${record.name}`}
          className={styles.confirmation}
        >
          <p>
            {target ? "Ativar" : "Desativar"} “{record.name}”? O serviço
            permanecerá no catálogo.
          </p>
          <div className={styles.buttons}>
            <Button
              disabled={!canWrite || submitting}
              onClick={() => void run(false)}
            >
              {target ? "Confirmar ativação" : "Confirmar desativação"}
            </Button>
            <Button
              disabled={submitting}
              onClick={() => {
                setTarget(null);
                if (canWrite) action.current?.focus();
                else container.current?.focus();
              }}
            >
              Cancelar alteração
            </Button>
          </div>
        </div>
      ) : null}
      {notice ? (
        <Feedback variant={notice.variant} title={notice.title}>
          {notice.message}
        </Feedback>
      ) : null}
      {uncertain ? (
        <Button disabled={submitting} onClick={() => void run(true)}>
          Consultar estado do serviço
        </Button>
      ) : null}
    </div>
  );
}
