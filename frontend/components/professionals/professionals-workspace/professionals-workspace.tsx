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
import { ProfessionalServicesPanel } from "../professional-services-panel/professional-services-panel";
import { ProfessionalStatusAction } from "../professional-status-action/professional-status-action";
import { CreateProfessionalForm } from "../create-professional-form/create-professional-form";
import { EditProfessionalPanel } from "../edit-professional-panel/edit-professional-panel";
import {
  ProfessionalsListPanel,
  type ProfessionalsView,
} from "../professionals-list-panel/professionals-list-panel";
import styles from "./professionals-workspace.module.css";

type SecondaryPanel = "status" | "services" | null;

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
  const [secondaryPanel, setSecondaryPanel] = useState<SecondaryPanel>(null);
  const servicesAction = useRef<HTMLButtonElement>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const origin = useRef<HTMLButtonElement | null>(null);
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
    if (pendingOperations > 0 || secondaryPanel) return;
    setSelected(null);
    setSecondaryPanel(null);
  }
  if (view.status === "unauthorized") return null;
  const writeAllowed =
    !readOnly && view.status === "success" && view.page.can_write;
  const canWrite = !busy && writeAllowed;
  return (
    <OwnerPanelPage
      title="Profissionais"
      description="Cadastre e acompanhe os profissionais que atendem no estabelecimento."
      actions={
        <IconButton
          aria-label="Adicionar profissional"
          aria-haspopup="dialog"
          disabled={!canWrite || !!secondaryPanel}
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
            canWrite={canWrite && !secondaryPanel}
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
            returnFocusRef={origin}
            title="Editar profissional"
            onClose={closeEditor}
            closeDisabled={busy || !!secondaryPanel}
            fallbackFocusRef={list}
          >
            <EditProfessionalPanel
              professionalId={selected}
              canWrite={canWrite && !secondaryPanel}
              blocked={busy || !!secondaryPanel}
              onRecordConfirmed={acceptRecord}
              onClose={closeEditor}
              onReadOnly={onReadOnly}
              onUnauthorized={onUnauthorized}
              onBusyChange={trackOperation}
              renderActions={(record, disabled, onChanged) => (
                <>
                  <ProfessionalStatusAction
                    record={record}
                    canWrite={canWrite}
                    disabled={disabled || !!secondaryPanel}
                    onOpenChange={(open) =>
                      setSecondaryPanel(open ? "status" : null)
                    }
                    onBusyChange={trackOperation}
                    onReadOnly={onReadOnly}
                    onUnauthorized={onUnauthorized}
                    onChanged={onChanged}
                  />
                  <ModalActions>
                    <Button
                      ref={servicesAction}
                      disabled={busy || !!secondaryPanel}
                      onClick={() => setSecondaryPanel("services")}
                    >
                      Serviços
                    </Button>
                  </ModalActions>
                  {secondaryPanel === "services" ? (
                    <ProfessionalServicesPanel
                      professional={record}
                      canWrite={writeAllowed}
                      onReadOnly={onReadOnly}
                      onBusyChange={trackOperation}
                      onClose={() => setSecondaryPanel(null)}
                      onUnauthorized={onUnauthorized}
                      returnFocusRef={servicesAction}
                    />
                  ) : null}
                </>
              )}
            />
          </Modal>
        ) : null}
        <ProfessionalsListPanel
          view={view}
          busy={busy || !!secondaryPanel}
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
