"use client";

import { useRef, useState } from "react";

import { useAuthSession } from "@/components/session/auth-provider";
import { Button, Feedback } from "@/components/ui";

import styles from "./testing-panel.module.css";

type Status = "idle" | "loading" | "error" | "unavailable";

export function ManualRefreshPanel() {
  const { state, refresh } = useAuthSession();
  const pending = useRef(false);
  const [status, setStatus] = useState<Status>("idle");
  if (state.status !== "authenticated") return null;

  async function renew() {
    if (pending.current || status === "unavailable") return;
    pending.current = true;
    setStatus("loading");
    try {
      const result = await refresh();
      if (result.state === "error") {
        setStatus(
          result.error.code === "session_recovery_unavailable"
            ? "unavailable"
            : "error",
        );
      } else {
        setStatus("idle");
      }
    } catch {
      setStatus("error");
    } finally {
      pending.current = false;
    }
  }

  return (
    <section aria-labelledby="refresh-panel-title" className={styles.panel}>
      <h2 id="refresh-panel-title">Renovação da sessão</h2>
      {status === "error" ? (
        <Feedback title="Não foi possível renovar a sessão" variant="error">
          Verifique sua conexão e tente novamente.
        </Feedback>
      ) : null}
      {status === "unavailable" ? (
        <Feedback title="Renovação incerta" variant="warning">
          Não é seguro renovar novamente agora. Se a sessão expirar, entre
          novamente.
        </Feedback>
      ) : null}
      <Button
        disabled={status === "loading" || status === "unavailable"}
        onClick={() => void renew()}
      >
        {status === "loading" ? "Renovando…" : "Renovar sessão"}
      </Button>
    </section>
  );
}
