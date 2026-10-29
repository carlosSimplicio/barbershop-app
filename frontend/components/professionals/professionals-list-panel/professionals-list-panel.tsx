"use client";

import { useEffect, useState } from "react";

import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import { Button, Feedback, RecordList } from "@/components/ui";
import { DEFAULT_PAGE_SIZE, FIRST_PAGE_OFFSET } from "@/lib/api/pagination";
import {
  listProfessionals,
  type Professional,
  type ProfessionalList,
} from "@/lib/professionals/professionals-api";

import styles from "./professionals-list-panel.module.css";

type View =
  | { status: "loading" | "error" | "unauthorized" }
  | { status: "success"; page: ProfessionalList };

export function ProfessionalsListPanel() {
  const { state } = useAuthSession();
  if (state.status !== "authenticated" || state.principal.role !== "owner") {
    return null;
  }

  const identity = `${state.principal.user_id}:${state.principal.establishment_id}`;
  return <ProfessionalsRequest key={identity} />;
}

function ProfessionalsRequest() {
  const [offset, setOffset] = useState(FIRST_PAGE_OFFSET);
  const [retryKey, setRetryKey] = useState(0);
  const [view, setView] = useState<View>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();

    void listProfessionals({
      limit: DEFAULT_PAGE_SIZE,
      offset,
      signal: controller.signal,
    })
      .then((result) => {
        if (controller.signal.aborted) return;
        if (result.status === 401) {
          setView({ status: "unauthorized" });
        } else if (result.status === 200 && result.state === "success") {
          setView({ status: "success", page: result.data });
        } else {
          setView({ status: "error" });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setView({ status: "error" });
      });

    return () => controller.abort();
  }, [offset, retryKey]);

  if (view.status === "unauthorized") return null;

  return (
    <section
      aria-busy={view.status === "loading"}
      aria-label="Lista de profissionais"
      className={styles.panel}
    >
      {view.status === "loading" ? (
        <p aria-live="polite" className={styles.loading} role="status">
          Carregando profissionais…
        </p>
      ) : null}
      {view.status === "error" ? (
        <Feedback
          action={
            <Button
              onClick={() => {
                setView({ status: "loading" });
                setRetryKey((current) => current + 1);
              }}
            >
              Tentar novamente
            </Button>
          }
          title="Não foi possível consultar os profissionais"
          variant="error"
        >
          Tente novamente mais tarde.
        </Feedback>
      ) : null}
      {view.status === "success" ? (
        <>
          <RecordList<Professional>
            emptyMessage="Nenhum profissional cadastrado."
            getKey={(professional) => professional.id}
            items={view.page.items}
            label="Profissionais cadastrados"
            renderItem={(professional) => (
              <ProfessionalRecord professional={professional} />
            )}
          />
          {view.page.offset > FIRST_PAGE_OFFSET || view.page.has_more ? (
            <nav
              aria-label="Paginação de profissionais"
              className={styles.pagination}
            >
              <Button
                disabled={view.page.offset === FIRST_PAGE_OFFSET}
                onClick={() => {
                  setView({ status: "loading" });
                  setOffset(
                    Math.max(
                      FIRST_PAGE_OFFSET,
                      view.page.offset - DEFAULT_PAGE_SIZE,
                    ),
                  );
                }}
              >
                Anterior
              </Button>
              <span aria-live="polite">
                Página {Math.floor(view.page.offset / view.page.limit) + 1}
              </span>
              <Button
                disabled={!view.page.has_more}
                onClick={() => {
                  setView({ status: "loading" });
                  setOffset(view.page.offset + view.page.limit);
                }}
              >
                Próxima
              </Button>
            </nav>
          ) : null}
        </>
      ) : null}
    </section>
  );
}

function ProfessionalRecord({ professional }: { professional: Professional }) {
  return (
    <div className={styles.record}>
      <div className={styles.identity}>
        <strong>{professional.name}</strong>
        <span>{professional.email}</span>
      </div>
      <span className={styles.status} data-active={professional.active}>
        {professional.active ? "Ativo" : "Inativo"}
      </span>
    </div>
  );
}
