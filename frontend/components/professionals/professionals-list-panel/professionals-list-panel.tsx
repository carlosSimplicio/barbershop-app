"use client";

import type { Ref, ReactNode } from "react";
import { Button, Feedback, RecordList } from "@/components/ui";
import {
  DEFAULT_PAGE_SIZE,
  FIRST_PAGE_OFFSET,
  MAX_PAGE_OFFSET,
} from "@/lib/api/pagination";
import type {
  Professional,
  ProfessionalList,
} from "@/lib/professionals/professionals-api";
import styles from "./professionals-list-panel.module.css";

export type ProfessionalsView =
  | { status: "loading" | "error" | "unauthorized" }
  | { status: "success"; page: ProfessionalList };
type Props = {
  view: ProfessionalsView;
  busy: boolean;
  listRef: Ref<HTMLElement>;
  onRefresh: () => void;
  onPageChange: (offset: number) => void;
  onEdit: (record: Professional, origin: HTMLButtonElement) => void;
  renderActions?: (record: Professional) => ReactNode;
};

export function ProfessionalsListPanel({
  view,
  busy,
  listRef,
  onRefresh,
  onPageChange,
  onEdit,
  renderActions,
}: Props) {
  if (view.status === "unauthorized") return null;
  return (
    <section
      ref={listRef}
      tabIndex={-1}
      aria-busy={view.status === "loading"}
      aria-label="Lista de profissionais"
      className={styles.panel}
    >
      <h2>Profissionais cadastrados</h2>
      <Button onClick={onRefresh} disabled={busy || view.status === "loading"}>
        Atualizar lista
      </Button>
      {view.status === "loading" ? (
        <p aria-live="polite" className={styles.loading} role="status">
          Carregando profissionais…
        </p>
      ) : null}
      {view.status === "error" ? (
        <Feedback
          action={
            <Button disabled={busy} onClick={onRefresh}>
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
            getKey={(record) => record.id}
            items={view.page.items}
            label="Profissionais cadastrados"
            renderItem={(record) => (
              <div className={styles.record}>
                <div className={styles.identity}>
                  <strong>{record.name}</strong>
                  <span>{record.email}</span>
                </div>
                <div className={styles.actions}>
                  <span className={styles.status} data-active={record.active}>
                    {record.active ? "Ativo" : "Inativo"}
                  </span>
                  <Button
                    disabled={busy}
                    aria-label={`Editar ${record.name}`}
                    onClick={(event) => onEdit(record, event.currentTarget)}
                  >
                    Editar
                  </Button>
                  {renderActions?.(record)}
                </div>
              </div>
            )}
          />
          {view.page.offset > FIRST_PAGE_OFFSET || view.page.has_more ? (
            <nav
              aria-label="Paginação de profissionais"
              className={styles.pagination}
            >
              <Button
                disabled={busy || view.page.offset === FIRST_PAGE_OFFSET}
                onClick={() =>
                  onPageChange(
                    Math.max(
                      FIRST_PAGE_OFFSET,
                      view.page.offset - DEFAULT_PAGE_SIZE,
                    ),
                  )
                }
              >
                Anterior
              </Button>
              <span aria-live="polite">
                Página {Math.floor(view.page.offset / view.page.limit) + 1}
              </span>
              <Button
                disabled={
                  busy ||
                  !view.page.has_more ||
                  view.page.offset > MAX_PAGE_OFFSET - DEFAULT_PAGE_SIZE
                }
                onClick={() =>
                  onPageChange(view.page.offset + DEFAULT_PAGE_SIZE)
                }
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
