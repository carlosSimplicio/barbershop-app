"use client";

import type { Key, ReactNode, Ref } from "react";

import { FIRST_PAGE_OFFSET, MAX_PAGE_OFFSET } from "@/lib/api/pagination";
import { Button } from "../button/button";
import { Feedback } from "../feedback/feedback";
import { IconButton } from "../icon-button/icon-button";
import { RecordList } from "../record-list/record-list";
import styles from "./record-list-panel.module.css";

export type RecordListView<T> =
  | { status: "loading" | "error" | "unauthorized" }
  | {
      status: "success";
      page: {
        items: readonly T[];
        limit: number;
        offset: number;
        has_more: boolean;
      };
    };

export type RecordListPanelProps<T> = {
  ref?: Ref<HTMLElement>;
  title: string;
  label: string;
  paginationLabel: string;
  view: RecordListView<T>;
  busy: boolean;
  messages: {
    loading: string;
    empty: string;
    errorTitle: string;
    errorDescription: string;
  };
  getKey: (record: T) => Key;
  renderItem: (record: T) => ReactNode;
  onRefresh: () => void;
  onPageChange: (offset: number) => void;
};

export function RecordListPanel<T>({
  ref,
  title,
  label,
  paginationLabel,
  view,
  busy,
  messages,
  getKey,
  renderItem,
  onRefresh,
  onPageChange,
}: RecordListPanelProps<T>) {
  if (view.status === "unauthorized") return null;

  return (
    <section
      ref={ref}
      tabIndex={-1}
      aria-label={label}
      aria-busy={view.status === "loading"}
      className={styles.panel}
    >
      <div className={styles.heading}>
        <h2>{title}</h2>
        <IconButton
          aria-label="Atualizar lista"
          title="Atualizar lista"
          onClick={onRefresh}
          disabled={busy || view.status === "loading"}
        >
          <svg aria-hidden="true" viewBox="0 0 24 24">
            <path d="M3 12a9 9 0 0 1 15.36-6.36L21 8M21 3v5h-5M21 12a9 9 0 0 1-15.36 6.36L3 16M3 21v-5h5" />
          </svg>
        </IconButton>
      </div>
      {view.status === "loading" ? (
        <p role="status" aria-live="polite" className={styles.loading}>
          {messages.loading}
        </p>
      ) : null}
      {view.status === "error" ? (
        <Feedback
          variant="error"
          title={messages.errorTitle}
          action={
            <Button disabled={busy} onClick={onRefresh}>
              Tentar novamente
            </Button>
          }
        >
          {messages.errorDescription}
        </Feedback>
      ) : null}
      {view.status === "success" ? (
        <>
          <RecordList
            className={styles.scrollArea}
            role="region"
            aria-label={title}
            tabIndex={0}
            items={view.page.items}
            getKey={getKey}
            label={title}
            emptyMessage={messages.empty}
            renderItem={renderItem}
          />
          <nav aria-label={paginationLabel} className={styles.pagination}>
            <Button
              disabled={busy || view.page.offset === FIRST_PAGE_OFFSET}
              onClick={() =>
                onPageChange(
                  Math.max(
                    FIRST_PAGE_OFFSET,
                    view.page.offset - view.page.limit,
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
                view.page.offset > MAX_PAGE_OFFSET - view.page.limit
              }
              onClick={() => onPageChange(view.page.offset + view.page.limit)}
            >
              Próxima
            </Button>
          </nav>
        </>
      ) : null}
    </section>
  );
}
