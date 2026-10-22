"use client";

import { useRef, useState } from "react";

import { useAuthSession } from "@/components/session/auth-provider";
import { Button, Feedback } from "@/components/ui";
import { logoutCurrentSession } from "@/lib/auth/logout-api";

import styles from "./testing-panel.module.css";

export function LogoutPanel() {
  const { state, transitioning, runAction } = useAuthSession();
  const pending = useRef(false);
  const [status, setStatus] = useState<"idle" | "loading" | "error">("idle");
  if (state.status !== "authenticated") return null;

  async function leave() {
    if (pending.current || transitioning) return;
    pending.current = true;
    setStatus("loading");
    try {
      const result = await runAction(logoutCurrentSession, "anonymous");
      if (result.state === "error") {
        setStatus("error");
      }
    } catch {
      setStatus("error");
    } finally {
      pending.current = false;
    }
  }

  return (
    <section aria-labelledby="logout-panel-title" className={styles.panel}>
      <h2 id="logout-panel-title">Encerrar sessão</h2>
      {status === "error" ? (
        <Feedback title="Não foi possível sair" variant="error">
          Não foi possível confirmar o encerramento. Tente novamente.
        </Feedback>
      ) : null}
      <Button
        disabled={transitioning || status === "loading"}
        onClick={() => void leave()}
      >
        {status === "loading" ? "Saindo…" : "Sair da sessão"}
      </Button>
    </section>
  );
}
