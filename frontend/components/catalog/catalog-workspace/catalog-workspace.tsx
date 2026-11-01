"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  Feedback,
  IconButton,
  Modal,
  RecordListPanel,
  RecordSummary,
} from "@/components/ui";
import { OwnerPanelPage } from "@/components/session/owner-panel-shell/owner-panel-page";
import { DEFAULT_PAGE_SIZE, FIRST_PAGE_OFFSET } from "@/lib/api/pagination";
import {
  listOfferings,
  type Offering,
  type CatalogPage,
} from "@/lib/catalog/catalog-api";
import { formatOfferingPrice } from "@/lib/catalog/money";
import { CreateOfferingForm } from "../create-offering-form/create-offering-form";
import { EditOfferingPanel } from "../edit-offering-panel/edit-offering-panel";
import { OfferingStatusAction } from "../offering-status-action/offering-status-action";
import styles from "./catalog-workspace.module.css";

type View =
  | { status: "loading" | "error" | "unauthorized" }
  | { status: "success"; page: CatalogPage };

export function CatalogWorkspace() {
  const { state } = useAuthSession();
  if (state.status !== "authenticated" || state.principal.role !== "owner")
    return null;
  return (
    <CatalogRequest
      key={`${state.principal.user_id}:${state.principal.establishment_id}`}
    />
  );
}

function CatalogRequest() {
  const [creating, setCreating] = useState(false);
  const [statusReconciling, setStatusReconciling] = useState(false);
  const [pendingOperations, setPendingOperations] = useState(0);
  const writing = pendingOperations > 0;
  const trackOperation = useCallback((busy: boolean) => {
    setPendingOperations((count) => count + (busy ? 1 : -1));
  }, []);
  const [editorVersion, setEditorVersion] = useState(0);
  const editOrigin = useRef<HTMLButtonElement | null>(null);
  const restoreEditorFocus = useRef(false);
  const catalog = useRef<HTMLElement>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [readOnly, setReadOnly] = useState(false);
  const [offset, setOffset] = useState(FIRST_PAGE_OFFSET);
  const [version, setVersion] = useState(0);
  const [view, setView] = useState<View>({ status: "loading" });
  const onUnauthorized = useCallback(
    () => setView({ status: "unauthorized" }),
    [],
  );

  useEffect(() => {
    if (!selected && restoreEditorFocus.current) {
      restoreEditorFocus.current = false;
      if (editOrigin.current?.isConnected && !editOrigin.current.disabled)
        editOrigin.current.focus();
      else catalog.current?.focus();
    }
  }, [selected]);

  function closeEditor() {
    if (writing || statusReconciling) return;
    restoreEditorFocus.current = true;
    setSelected(null);
  }

  useEffect(() => {
    const controller = new AbortController();
    void listOfferings({
      limit: DEFAULT_PAGE_SIZE,
      offset,
      signal: controller.signal,
    })
      .then((result) => {
        if (controller.signal.aborted) return;
        if (result.status === 401) setView({ status: "unauthorized" });
        else if (result.state === "success" && result.status === 200)
          setView({ status: "success", page: result.data });
        else setView({ status: "error" });
      })
      .catch(() => {
        if (!controller.signal.aborted) setView({ status: "error" });
      });
    return () => controller.abort();
  }, [offset, version]);

  function changePage(next: number) {
    setView({ status: "loading" });
    setOffset(next);
  }
  function refresh() {
    setView({ status: "loading" });
    setVersion((value) => value + 1);
  }

  const acceptRecord = useCallback((record: Offering) => {
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
  if (view.status === "unauthorized") return null;
  const canWrite =
    !writing &&
    !statusReconciling &&
    !readOnly &&
    view.status === "success" &&
    view.page.can_write;
  return (
    <OwnerPanelPage
      title="Serviços"
      description="Organize os serviços, preços e durações do estabelecimento."
      actions={
        <IconButton
          aria-label="Adicionar serviço"
          aria-haspopup="dialog"
          disabled={!canWrite}
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
          title="Cadastrar serviço"
          onClose={() => setCreating(false)}
          closeDisabled={writing}
          fallbackFocusRef={catalog}
        >
          <CreateOfferingForm
            embedded
            canWrite={canWrite}
            onReviewList={() => setCreating(false)}
            onBusyChange={trackOperation}
            onCreated={() => {
              setOffset(FIRST_PAGE_OFFSET);
              refresh();
            }}
            onReadOnly={() => setReadOnly(true)}
            onUnauthorized={onUnauthorized}
          />
        </Modal>
        {readOnly || (view.status === "success" && !view.page.can_write) ? (
          <Feedback variant="warning" title="Modo somente leitura">
            Você pode consultar os serviços. Alterações ficam disponíveis com
            acesso ativo.
          </Feedback>
        ) : null}
        {selected ? (
          <Modal
            key={selected}
            open
            title="Editar serviço"
            onClose={closeEditor}
            closeDisabled={writing || statusReconciling}
            fallbackFocusRef={catalog}
          >
            <EditOfferingPanel
              embedded
              offeringId={selected}
              canWrite={canWrite}
              blocked={writing || statusReconciling}
              reloadVersion={editorVersion}
              onBusyChange={trackOperation}
              onRecordConfirmed={acceptRecord}
              onClose={closeEditor}
              onReadOnly={() => setReadOnly(true)}
              onUnauthorized={onUnauthorized}
              renderActions={(record, disabled) => (
                <OfferingStatusAction
                  record={record}
                  canWrite={canWrite && !disabled}
                  onBusyChange={trackOperation}
                  onReconciliationChange={setStatusReconciling}
                  onChanged={(confirmed) => {
                    acceptRecord(confirmed);
                    setEditorVersion((value) => value + 1);
                  }}
                  onReadOnly={() => setReadOnly(true)}
                  onUnauthorized={onUnauthorized}
                />
              )}
            />
          </Modal>
        ) : null}
        <RecordListPanel<Offering>
          ref={catalog}
          title="Serviços cadastrados"
          label="Catálogo de serviços"
          paginationLabel="Paginação de serviços"
          view={view}
          busy={writing || statusReconciling}
          messages={{
            loading: "Carregando serviços…",
            empty: "Nenhum serviço cadastrado.",
            errorTitle: "Não foi possível consultar os serviços",
            errorDescription: "Verifique sua conexão e tente novamente.",
          }}
          onRefresh={refresh}
          onPageChange={changePage}
          getKey={(item) => item.id}
          renderItem={(item) => (
            <RecordSummary
              title={item.name}
              status={{
                label: item.active ? "Ativo" : "Inativo",
                tone: item.active ? "success" : "neutral",
              }}
              actions={
                <IconButton
                  disabled={!canWrite}
                  onClick={(event) => {
                    editOrigin.current = event.currentTarget;
                    setSelected(item.id);
                  }}
                  aria-label={`Editar ${item.name}`}
                  title={`Editar ${item.name}`}
                >
                  <svg aria-hidden="true" viewBox="0 0 24 24">
                    <path d="m16 3 5 5-13 13-5 1 1-5L16 3ZM14 5l5 5" />
                  </svg>
                </IconButton>
              }
            >
              <span>{formatOfferingPrice(item.price_cents)}</span>
              <span>{item.duration_minutes} minutos</span>
            </RecordSummary>
          )}
        />
      </div>
    </OwnerPanelPage>
  );
}
