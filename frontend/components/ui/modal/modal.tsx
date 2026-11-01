"use client";

import {
  useEffect,
  useId,
  useRef,
  type ReactNode,
  type RefObject,
} from "react";

import { IconButton } from "../icon-button/icon-button";
import styles from "./modal.module.css";

export type ModalProps = {
  children: ReactNode;
  open: boolean;
  title: string;
  description?: string;
  onClose: () => void;
  closeDisabled?: boolean;
  fallbackFocusRef?: RefObject<HTMLElement | null>;
};

export function Modal({
  children,
  open,
  title,
  description,
  onClose,
  closeDisabled = false,
  fallbackFocusRef,
}: ModalProps) {
  const id = useId();
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    if (!open) return;
    const dialog = dialogRef.current!;
    const origin = document.activeElement;
    const fallback = fallbackFocusRef?.current;
    dialog.showModal();
    titleRef.current?.focus();
    return () => {
      dialog.close();
      if (
        origin instanceof HTMLElement &&
        origin.isConnected &&
        !origin.matches(":disabled")
      ) {
        origin.focus();
      } else {
        if (fallback?.isConnected) fallback.focus();
      }
    };
  }, [open, fallbackFocusRef]);

  useEffect(() => {
    if (!open) return;
    if (!dialogRef.current?.contains(document.activeElement))
      titleRef.current?.focus();
    if (!closeDisabled) return;
    function protectPendingDialog(event: KeyboardEvent) {
      // Disabling the focused submit button can move focus to the body.
      // Capture keys there as well, before native dialog dismissal or tabbing.
      if (event.key === "Escape") {
        event.preventDefault();
        event.stopPropagation();
      } else if (
        event.key === "Tab" &&
        !dialogRef.current?.contains(document.activeElement)
      ) {
        event.preventDefault();
        titleRef.current?.focus();
      }
    }
    document.addEventListener("keydown", protectPendingDialog, true);
    return () =>
      document.removeEventListener("keydown", protectPendingDialog, true);
  }, [open, closeDisabled]);

  return (
    <dialog
      ref={dialogRef}
      className={styles.dialog}
      aria-labelledby={`${id}-title`}
      aria-describedby={description ? `${id}-description` : undefined}
      onCancel={(event) => {
        // Keep Escape controlled so an in-flight mutation cannot be dismissed.
        event.preventDefault();
        if (!closeDisabled) onClose();
      }}
      onKeyDown={(event) => {
        if (event.key !== "Tab") return;
        const controls = Array.from(
          event.currentTarget.querySelectorAll<HTMLElement>(
            'a[href], button, input, select, textarea, [contenteditable="true"], [tabindex]',
          ),
        ).filter(
          (element) =>
            element.tabIndex >= 0 &&
            !element.matches(":disabled") &&
            !element.closest("[hidden], [inert]") &&
            getComputedStyle(element).visibility === "visible" &&
            element.getClientRects().length > 0,
        );
        const first = controls[0];
        const last = controls.at(-1);
        const active = document.activeElement;
        // Cycle explicitly at the boundaries; browsers may otherwise move
        // focus into browser chrome after the final control of a dialog.
        if (!first || !last) {
          event.preventDefault();
          titleRef.current?.focus();
        } else if (!controls.some((element) => element === active)) {
          event.preventDefault();
          (event.shiftKey ? last : first).focus();
        } else if (event.shiftKey && active === first) {
          event.preventDefault();
          last.focus();
        } else if (!event.shiftKey && active === last) {
          event.preventDefault();
          first.focus();
        }
      }}
    >
      <div className={styles.header}>
        <h2 ref={titleRef} id={`${id}-title`} tabIndex={-1}>
          {title}
        </h2>
        <IconButton
          aria-label="Fechar modal"
          disabled={closeDisabled}
          onClick={onClose}
        >
          <svg aria-hidden="true" viewBox="0 0 24 24">
            <path d="m6 6 12 12M18 6 6 18" />
          </svg>
        </IconButton>
      </div>
      {description ? (
        <p className={styles.description} id={`${id}-description`}>
          {description}
        </p>
      ) : null}
      <div className={styles.content}>{children}</div>
    </dialog>
  );
}
