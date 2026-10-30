"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import { Button, Feedback } from "@/components/ui";
import { DEFAULT_PAGE_SIZE, FIRST_PAGE_OFFSET } from "@/lib/api/pagination";
import {
  listProfessionals,
  type Professional,
} from "@/lib/professionals/professionals-api";
import { isProfessionalList } from "@/lib/professionals/professional-operation-result";
import { ProfessionalReactivationPanel } from "../professional-reactivation-panel/professional-reactivation-panel";
import { ProfessionalDeactivationPanel } from "../professional-deactivation-panel/professional-deactivation-panel";
import { CreateProfessionalForm } from "../create-professional-form/create-professional-form";
import { EditProfessionalPanel } from "../edit-professional-panel/edit-professional-panel";
import {
  ProfessionalsListPanel,
  type ProfessionalsView,
} from "../professionals-list-panel/professionals-list-panel";
import styles from "./professionals-workspace.module.css";

type ProfessionalStateAction = {
  kind: "deactivation" | "reactivation";
  record: Professional;
};

export function ProfessionalsWorkspace() {
  const { state } = useAuthSession();
  if (state.status !== "authenticated" || state.principal.role !== "owner")
    return null;
  return (
    <ProfessionalsRequest
      key={`${state.principal.user_id}:${state.principal.establishment_id}`}
    />
  );
}
function ProfessionalsRequest() {
  const unauthorized = useRef(false);
  const [view, setView] = useState<ProfessionalsView>({ status: "loading" });
  const [offset, setOffset] = useState(FIRST_PAGE_OFFSET);
  const [version, setVersion] = useState(0);
  const [pendingOperations, setPendingOperations] = useState(0);
  const [reconciling, setReconciling] = useState(false);
  const [readOnly, setReadOnly] = useState(false);
  const [stateAction, setStateAction] =
    useState<ProfessionalStateAction | null>(null);
  const [editorVersion, setEditorVersion] = useState(0);
  const statusOrigin = useRef<HTMLButtonElement | null>(null);
  const restoreStatusFocus = useRef(false);
  const [selected, setSelected] = useState<string | null>(null);
  const origin = useRef<HTMLButtonElement | null>(null);
  const restoreEditorFocus = useRef(false);
  const confirmationsDuringList = useRef<Map<string, Professional> | null>(
    null,
  );
  const list = useRef<HTMLElement>(null);
  const busy = pendingOperations > 0 || reconciling;
  const trackOperation = useCallback(
    (pending: boolean) =>
      setPendingOperations((count) => count + (pending ? 1 : -1)),
    [],
  );
  const onUnauthorized = useCallback(() => {
    unauthorized.current = true;
    setView({ status: "unauthorized" });
  }, []);
  const onReadOnly = useCallback(() => setReadOnly(true), []);
  const acceptRecord = useCallback((record: Professional) => {
    confirmationsDuringList.current?.set(record.id, record);
    setView((current) =>
      current.status === "success"
        ? {
            ...current,
            page: {
              ...current.page,
              items: current.page.items.map((item) =>
                item.id === record.id ? record : item,
              ),
            },
          }
        : current,
    );
  }, []);
  useEffect(() => {
    if (!stateAction && restoreStatusFocus.current) {
      restoreStatusFocus.current = false;
      if (statusOrigin.current?.isConnected && !statusOrigin.current.disabled)
        statusOrigin.current.focus();
      else list.current?.focus();
    }
  }, [stateAction, reconciling]);
  useEffect(() => {
    if (!selected && restoreEditorFocus.current) {
      restoreEditorFocus.current = false;
      if (origin.current?.isConnected && !origin.current.disabled)
        origin.current.focus();
      else list.current?.focus();
    }
  }, [selected, reconciling]);
  useEffect(() => {
    const controller = new AbortController();
    // Keep only confirmations received during this list request. A delayed list
    // must not overwrite a newer individual read; subsequent reloads start fresh.
    const confirmations = new Map<string, Professional>();
    confirmationsDuringList.current = confirmations;
    const release = () => {
      if (confirmationsDuringList.current === confirmations)
        confirmationsDuringList.current = null;
    };
    void listProfessionals({
      limit: DEFAULT_PAGE_SIZE,
      offset,
      signal: controller.signal,
    })
      .then((result) => {
        if (controller.signal.aborted || unauthorized.current) return;
        if (result.status === 401) onUnauthorized();
        else if (
          result.state === "success" &&
          result.status === 200 &&
          isProfessionalList(result.data)
        )
          setView({
            status: "success",
            page: {
              ...result.data,
              items: result.data.items.map(
                (record) => confirmations.get(record.id) ?? record,
              ),
            },
          });
        else setView({ status: "error" });
      })
      .catch(() => {
        if (!controller.signal.aborted && !unauthorized.current)
          setView({ status: "error" });
      })
      .finally(release);
    return () => {
      controller.abort();
      release();
    };
  }, [offset, version, onUnauthorized]);
  function refresh() {
    setView({ status: "loading" });
    setVersion((value) => value + 1);
  }
  function closeEditor() {
    restoreEditorFocus.current = true;
    setSelected(null);
    setReconciling(false);
  }
  function confirmStateRecord(record: Professional) {
    acceptRecord(record);
    setStateAction((current) =>
      current && current.record.id === record.id
        ? { ...current, record }
        : current,
    );
    if (record.id === selected) setEditorVersion((value) => value + 1);
  }
  function closeStateAction() {
    restoreStatusFocus.current = true;
    setStateAction(null);
    setReconciling(false);
  }
  if (view.status === "unauthorized") return null;
  const canWrite =
    !busy && !readOnly && view.status === "success" && view.page.can_write;
  return (
    <div className={styles.workspace}>
      <CreateProfessionalForm
        canWrite={canWrite && !stateAction}
        onCreated={() => {
          setOffset(FIRST_PAGE_OFFSET);
          refresh();
        }}
        onBusyChange={trackOperation}
        onReadOnly={onReadOnly}
        onUnauthorized={onUnauthorized}
      />
      {readOnly || (view.status === "success" && !view.page.can_write) ? (
        <Feedback variant="warning" title="Modo somente leitura">
          Você pode consultar os profissionais. Alterações ficam disponíveis com
          acesso ativo.
        </Feedback>
      ) : null}
      {selected ? (
        <EditProfessionalPanel
          key={selected}
          professionalId={selected}
          canWrite={canWrite && !stateAction}
          blocked={busy || !!stateAction}
          reloadVersion={editorVersion}
          onRecordConfirmed={acceptRecord}
          onClose={closeEditor}
          onReadOnly={onReadOnly}
          onUnauthorized={onUnauthorized}
          onBusyChange={trackOperation}
          onReconciliationChange={setReconciling}
        />
      ) : null}
      {stateAction?.kind === "deactivation" ? (
        <ProfessionalDeactivationPanel
          key={`deactivation:${stateAction.record.id}`}
          record={stateAction.record}
          canWrite={canWrite}
          onBusyChange={trackOperation}
          onReconciliationChange={setReconciling}
          onReadOnly={onReadOnly}
          onUnauthorized={onUnauthorized}
          onRecordConfirmed={confirmStateRecord}
          onClose={closeStateAction}
        />
      ) : null}
      {stateAction?.kind === "reactivation" ? (
        <ProfessionalReactivationPanel
          key={`reactivation:${stateAction.record.id}`}
          record={stateAction.record}
          canWrite={canWrite}
          onBusyChange={trackOperation}
          onReconciliationChange={setReconciling}
          onReadOnly={onReadOnly}
          onUnauthorized={onUnauthorized}
          onRecordConfirmed={confirmStateRecord}
          onClose={closeStateAction}
        />
      ) : null}
      <ProfessionalsListPanel
        view={view}
        busy={busy || !!stateAction}
        renderActions={(record) =>
          record.active ? (
            <Button
              key="deactivation"
              disabled={!canWrite || !!stateAction}
              aria-label={`Desativar ${record.name}`}
              onClick={(event) => {
                statusOrigin.current = event.currentTarget;
                setStateAction({ kind: "deactivation", record });
              }}
            >
              Desativar
            </Button>
          ) : (
            <Button
              key="reactivation"
              disabled={!canWrite || !!stateAction}
              aria-label={`Reativar ${record.name}`}
              onClick={(event) => {
                statusOrigin.current = event.currentTarget;
                setStateAction({ kind: "reactivation", record });
              }}
            >
              Reativar
            </Button>
          )
        }
        listRef={list}
        onRefresh={refresh}
        onPageChange={(next) => {
          setView({ status: "loading" });
          setOffset(next);
        }}
        onEdit={(record, button) => {
          origin.current = button;
          setSelected(record.id);
        }}
      />
    </div>
  );
}
