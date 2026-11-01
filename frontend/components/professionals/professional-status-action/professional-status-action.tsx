"use client";

import { useRef, useState } from "react";
import { Button, ModalActions } from "@/components/ui";
import type { Professional } from "@/lib/professionals/professionals-api";
import { ProfessionalDeactivationPanel } from "../professional-deactivation-panel/professional-deactivation-panel";
import { ProfessionalReactivationPanel } from "../professional-reactivation-panel/professional-reactivation-panel";
import styles from "./professional-status-action.module.css";

type Props = {
  record: Professional;
  canWrite: boolean;
  disabled: boolean;
  onOpenChange: (open: boolean) => void;
  onBusyChange: (pending: boolean) => void;
  onReadOnly: () => void;
  onUnauthorized: () => void;
  onChanged: (record: Professional) => void;
};
type Confirmation = {
  kind: "deactivation" | "reactivation";
  record: Professional;
};

export function ProfessionalStatusAction({
  record,
  canWrite,
  disabled,
  onOpenChange,
  onBusyChange,
  onReadOnly,
  onUnauthorized,
  onChanged,
}: Props) {
  const [confirmation, setConfirmation] = useState<Confirmation | null>(null);
  const action = useRef<HTMLButtonElement>(null);
  const container = useRef<HTMLDivElement>(null);
  function close() {
    setConfirmation(null);
    onOpenChange(false);
  }
  function acceptRecord(confirmed: Professional) {
    onChanged(confirmed);
    setConfirmation((current) =>
      current ? { ...current, record: confirmed } : current,
    );
  }
  const confirmationProps = {
    record: confirmation?.record ?? record,
    canWrite,
    onBusyChange,
    onReadOnly,
    onUnauthorized,
    onRecordConfirmed: acceptRecord,
    onClose: close,
    returnFocusRef: action,
    fallbackFocusRef: container,
  };
  return (
    <div ref={container} tabIndex={-1}>
      <div className={styles.header}>
        <p>Estado: {record.active ? "Ativo" : "Inativo"}</p>
        <ModalActions>
          <Button
            ref={action}
            disabled={disabled || !!confirmation}
            aria-label={`${record.active ? "Desativar" : "Reativar"} ${record.name}`}
            onClick={() => {
              setConfirmation({
                kind: record.active ? "deactivation" : "reactivation",
                record,
              });
              onOpenChange(true);
            }}
          >
            {record.active ? "Desativar" : "Reativar"}
          </Button>
        </ModalActions>
      </div>
      {confirmation?.kind === "deactivation" ? (
        <ProfessionalDeactivationPanel
          key={`deactivation:${confirmation.record.id}`}
          {...confirmationProps}
        />
      ) : null}
      {confirmation?.kind === "reactivation" ? (
        <ProfessionalReactivationPanel
          key={`reactivation:${confirmation.record.id}`}
          {...confirmationProps}
        />
      ) : null}
    </div>
  );
}
