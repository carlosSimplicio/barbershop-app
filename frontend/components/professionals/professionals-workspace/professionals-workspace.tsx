"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  Button,
  Feedback,
  IconButton,
  Modal,
  ModalActions,
} from "@/components/ui";
import { OwnerPanelPage } from "@/components/session/owner-panel-shell/owner-panel-page";
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
  const [creating, setCreating] = useState(false);
  const unauthorized = useRef(false);
  const [view, setView] = useState<ProfessionalsView>({ status: "loading" });
  const [offset, setOffset] = useState(FIRST_PAGE_OFFSET);
  const [version, setVersion] = useState(0);
  const [pendingOperations, setPendingOperations] = useState(0);
  const [readOnly, setReadOnly] = useState(false);
  const [stateAction, setStateAction] =
    useState<ProfessionalStateAction | null>(null);
  const statusOrigin = useRef<HTMLButtonElement | null>(null);
  const editorContent = useRef<HTMLDivElement>(null);
  const restoreStatusFocus = useRef(false);
  const [selected, setSelected] = useState<string | null>(null);
  const origin = useRef<HTMLButtonElement | null>(null);
  const restoreEditorFocus = useRef(false);
  const list = useRef<HTMLElement>(null);
  const busy = pendingOperations > 0;
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
    if (!stateAction && restoreStatusFocus.current && pendingOperations === 0) {
      restoreStatusFocus.current = false;
      if (statusOrigin.current?.isConnected && !statusOrigin.current.disabled)
        statusOrigin.current.focus();
      else editorContent.current?.focus();
    }
  }, [stateAction, pendingOperations]);
  useEffect(() => {
    if (!selected && restoreEditorFocus.current) {
      restoreEditorFocus.current = false;
      if (origin.current?.isConnected && !origin.current.disabled)
        origin.current.focus();
      else list.current?.focus();
    }
  }, [selected]);
  useEffect(() => {
    const controller = new AbortController();
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
            page: result.data,
          });
        else setView({ status: "error" });
      })
      .catch(() => {
        if (!controller.signal.aborted && !unauthorized.current)
          setView({ status: "error" });
      });
    return () => {
      controller.abort();
    };
  }, [offset, version, onUnauthorized]);
  function refresh() {
    setView({ status: "loading" });
    setVersion((value) => value + 1);
  }
  function closeEditor() {
    if (pendingOperations > 0) return;
    restoreEditorFocus.current = true;
    setSelected(null);
    setStateAction(null);
    restoreStatusFocus.current = false;
  }
  function closeStateAction() {
    restoreStatusFocus.current = true;
    setStateAction(null);
  }
  if (view.status === "unauthorized") return null;
  const canWrite =
    !busy && !readOnly && view.status === "success" && view.page.can_write;
  return (
    <OwnerPanelPage
      title="Profissionais"
      description="Cadastre e acompanhe os profissionais que atendem no estabelecimento."
      actions={
        <IconButton
          aria-label="Adicionar profissional"
          aria-haspopup="dialog"
          disabled={!canWrite || !!stateAction}
          onClick={() => setCreating(true)}
        >
          <svg aria-hidden="true" viewBox="0 0 24 24">
            <path d="M12 5v14M5 12h14" />
          </svg>
        </IconButton>
      }
    >
      <div className={styles.workspace}>
        <Modal
          open={creating}
          title="Cadastrar profissional"
          description="Crie o acesso inicial para quem trabalha no estabelecimento."
          onClose={() => setCreating(false)}
          closeDisabled={busy}
          fallbackFocusRef={list}
        >
          <CreateProfessionalForm
            embedded
            canWrite={canWrite && !stateAction}
            onCreated={() => {
              setOffset(FIRST_PAGE_OFFSET);
              refresh();
            }}
            onBusyChange={trackOperation}
            onReadOnly={onReadOnly}
            onUnauthorized={onUnauthorized}
          />
        </Modal>
        {readOnly || (view.status === "success" && !view.page.can_write) ? (
          <Feedback variant="warning" title="Modo somente leitura">
            Você pode consultar os profissionais. Alterações ficam disponíveis
            com acesso ativo.
          </Feedback>
        ) : null}
        {selected ? (
          <Modal
            key={selected}
            open
            title="Editar profissional"
            onClose={closeEditor}
            closeDisabled={pendingOperations > 0}
            fallbackFocusRef={list}
          >
            <div ref={editorContent} tabIndex={-1}>
              <EditProfessionalPanel
                embedded
                professionalId={selected}
                canWrite={canWrite && !stateAction}
                blocked={busy || !!stateAction}
                onRecordConfirmed={acceptRecord}
                onClose={closeEditor}
                onReadOnly={onReadOnly}
                onUnauthorized={onUnauthorized}
                onBusyChange={trackOperation}
                renderActions={(record, disabled, onChanged) => (
                  <>
                    <div className={styles.statusHeader}>
                      <p>Estado: {record.active ? "Ativo" : "Inativo"}</p>
                      <ModalActions>
                        <Button
                          ref={statusOrigin}
                          disabled={disabled || !!stateAction}
                          aria-label={`${record.active ? "Desativar" : "Reativar"} ${record.name}`}
                          onClick={() =>
                            setStateAction({
                              kind: record.active
                                ? "deactivation"
                                : "reactivation",
                              record,
                            })
                          }
                        >
                          {record.active ? "Desativar" : "Reativar"}
                        </Button>
                      </ModalActions>
                    </div>
                    {stateAction?.kind === "deactivation" ? (
                      <ProfessionalDeactivationPanel
                        key={`deactivation:${stateAction.record.id}`}
                        record={stateAction.record}
                        canWrite={canWrite}
                        onBusyChange={trackOperation}
                        onReadOnly={onReadOnly}
                        onUnauthorized={onUnauthorized}
                        onRecordConfirmed={(confirmed) => {
                          onChanged(confirmed);
                          setStateAction((current) =>
                            current
                              ? { ...current, record: confirmed }
                              : current,
                          );
                        }}
                        onClose={closeStateAction}
                      />
                    ) : null}
                    {stateAction?.kind === "reactivation" ? (
                      <ProfessionalReactivationPanel
                        key={`reactivation:${stateAction.record.id}`}
                        record={stateAction.record}
                        canWrite={canWrite}
                        onBusyChange={trackOperation}
                        onReadOnly={onReadOnly}
                        onUnauthorized={onUnauthorized}
                        onRecordConfirmed={(confirmed) => {
                          onChanged(confirmed);
                          setStateAction((current) =>
                            current
                              ? { ...current, record: confirmed }
                              : current,
                          );
                        }}
                        onClose={closeStateAction}
                      />
                    ) : null}
                  </>
                )}
              />
            </div>
          </Modal>
        ) : null}
        <ProfessionalsListPanel
          view={view}
          busy={busy || !!stateAction}
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
    </OwnerPanelPage>
  );
}
