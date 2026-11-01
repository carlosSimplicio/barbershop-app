"use client";

import type { Ref } from "react";
import { IconButton, RecordListPanel, RecordSummary } from "@/components/ui";
import type {
  Professional,
  ProfessionalList,
} from "@/lib/professionals/professionals-api";

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
};

export function ProfessionalsListPanel({
  view,
  busy,
  listRef,
  onRefresh,
  onPageChange,
  onEdit,
}: Props) {
  return (
    <RecordListPanel<Professional>
      ref={listRef}
      title="Profissionais cadastrados"
      label="Lista de profissionais"
      paginationLabel="Paginação de profissionais"
      view={view}
      busy={busy}
      onRefresh={onRefresh}
      onPageChange={onPageChange}
      messages={{
        loading: "Carregando profissionais…",
        empty: "Nenhum profissional cadastrado.",
        errorTitle: "Não foi possível consultar os profissionais",
        errorDescription: "Tente novamente mais tarde.",
      }}
      getKey={(record) => record.id}
      renderItem={(record) => (
        <RecordSummary
          title={record.name}
          status={{
            label: record.active ? "Ativo" : "Inativo",
            tone: record.active ? "success" : "neutral",
          }}
          actions={
            <IconButton
              disabled={busy}
              aria-label={`Editar ${record.name}`}
              title={`Editar ${record.name}`}
              onClick={(event) => onEdit(record, event.currentTarget)}
            >
              <svg aria-hidden="true" viewBox="0 0 24 24">
                <path d="m16 3 5 5-13 13-5 1 1-5L16 3ZM14 5l5 5" />
              </svg>
            </IconButton>
          }
        >
          <span>{record.email}</span>
        </RecordSummary>
      )}
    />
  );
}
