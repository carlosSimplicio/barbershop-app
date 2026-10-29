"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import { Button, Feedback, RecordList } from "@/components/ui";
import {
  DEFAULT_PAGE_SIZE,
  FIRST_PAGE_OFFSET,
  MAX_PAGE_OFFSET,
} from "@/lib/api/pagination";
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
  const [pendingOperations, setPendingOperations] = useState(0);
  const writing = pendingOperations > 0;
  const trackOperation = useCallback((busy: boolean) => {
    setPendingOperations((count) => count + (busy ? 1 : -1));
  }, []);
  const [editorVersion, setEditorVersion] = useState(0);
  const editOrigin = useRef<HTMLButtonElement | null>(null);
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
  return (
    <div className={styles.workspace}>
      <CreateOfferingForm
        canWrite={
          !writing &&
          !readOnly &&
          view.status === "success" &&
          view.page.can_write
        }
        onBusyChange={trackOperation}
        onCreated={() => {
          setOffset(FIRST_PAGE_OFFSET);
          refresh();
        }}
        onReadOnly={() => setReadOnly(true)}
        onUnauthorized={onUnauthorized}
      />
      {readOnly || (view.status === "success" && !view.page.can_write) ? (
        <Feedback variant="warning" title="Modo somente leitura">
          Você pode consultar os serviços. Alterações ficam disponíveis com
          acesso ativo.
        </Feedback>
      ) : null}
      {selected ? (
        <EditOfferingPanel
          key={selected}
          offeringId={selected}
          canWrite={
            !writing &&
            !readOnly &&
            view.status === "success" &&
            view.page.can_write
          }
          reloadVersion={editorVersion}
          onBusyChange={trackOperation}
          onRecordConfirmed={acceptRecord}
          onClose={() => {
            setSelected(null);
            if (editOrigin.current?.isConnected) editOrigin.current.focus();
            else catalog.current?.focus();
          }}
          onReadOnly={() => setReadOnly(true)}
          onUnauthorized={onUnauthorized}
        />
      ) : null}
      <section
        ref={catalog}
        tabIndex={-1}
        aria-label="Catálogo de serviços"
        aria-busy={view.status === "loading"}
        className={styles.panel}
      >
        <h2>Serviços cadastrados</h2>
        <Button
          onClick={refresh}
          disabled={writing || view.status === "loading"}
        >
          Atualizar lista
        </Button>
        {view.status === "loading" ? (
          <p role="status" aria-live="polite">
            Carregando serviços…
          </p>
        ) : null}
        {view.status === "error" ? (
          <>
            <Feedback
              variant="error"
              title="Não foi possível consultar os serviços"
            >
              Verifique sua conexão e tente novamente.
            </Feedback>
            <Button onClick={refresh}>Tentar novamente</Button>
          </>
        ) : null}
        {view.status === "success" ? (
          <>
            <RecordList
              items={view.page.items}
              getKey={(item) => item.id}
              label="Serviços cadastrados"
              emptyMessage="Nenhum serviço cadastrado."
              renderItem={(item) => (
                <div className={styles.record}>
                  <div className={styles.details}>
                    <strong>{item.name}</strong>
                    <span>{formatOfferingPrice(item.price_cents)}</span>
                    <span>{item.duration_minutes} minutos</span>
                  </div>
                  <div className={styles.recordActions}>
                    <span>{item.active ? "Ativo" : "Inativo"}</span>
                    <Button
                      disabled={writing || readOnly || !view.page.can_write}
                      onClick={(event) => {
                        editOrigin.current = event.currentTarget;
                        setSelected(item.id);
                      }}
                      aria-label={`Editar ${item.name}`}
                    >
                      Editar
                    </Button>
                    <OfferingStatusAction
                      record={item}
                      canWrite={!writing && !readOnly && view.page.can_write}
                      onBusyChange={trackOperation}
                      onChanged={(record) => {
                        acceptRecord(record);
                        if (selected === record.id)
                          setEditorVersion((value) => value + 1);
                      }}
                      onReadOnly={() => setReadOnly(true)}
                      onUnauthorized={onUnauthorized}
                    />
                  </div>
                </div>
              )}
            />
            <nav
              aria-label="Paginação de serviços"
              className={styles.pagination}
            >
              <Button
                disabled={writing || offset === FIRST_PAGE_OFFSET}
                onClick={() =>
                  changePage(
                    Math.max(FIRST_PAGE_OFFSET, offset - DEFAULT_PAGE_SIZE),
                  )
                }
              >
                Anterior
              </Button>
              <span>Página {Math.floor(offset / DEFAULT_PAGE_SIZE) + 1}</span>
              <Button
                disabled={
                  writing ||
                  !view.page.has_more ||
                  offset > MAX_PAGE_OFFSET - DEFAULT_PAGE_SIZE
                }
                onClick={() => changePage(offset + DEFAULT_PAGE_SIZE)}
              >
                Próxima
              </Button>
            </nav>
          </>
        ) : null}
      </section>
    </div>
  );
}
