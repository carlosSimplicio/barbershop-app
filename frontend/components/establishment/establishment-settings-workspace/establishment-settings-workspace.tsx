"use client";

import { useEffect, useState } from "react";

import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import { Button, Feedback } from "@/components/ui";
import { getEstablishmentSettings } from "@/lib/establishment/establishment-api";
import type { EstablishmentSettings } from "@/lib/establishment/establishment-api";

import { SettingsForm } from "./establishment-settings-form";
import styles from "./establishment-settings-workspace.module.css";

type SettingsView =
  | { status: "loading" }
  | { status: "error" }
  | { status: "unauthorized" }
  | { status: "success"; settings: EstablishmentSettings };
export function EstablishmentSettingsWorkspace() {
  const { state } = useAuthSession();
  if (state.status !== "authenticated" || state.principal.role !== "owner") {
    return null;
  }

  const identity = `${state.principal.user_id}:${state.principal.establishment_id}`;
  return <SettingsRequest key={identity} />;
}

function SettingsRequest() {
  const [retryKey, setRetryKey] = useState(0);
  const [view, setView] = useState<SettingsView>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();

    void getEstablishmentSettings(controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return;
        if (result.status === 401) {
          setView({ status: "unauthorized" });
        } else if (result.status === 200 && result.state === "success") {
          setView({ status: "success", settings: result.data });
        } else {
          setView({ status: "error" });
        }
      })
      .catch(() => {
        if (!controller.signal.aborted) setView({ status: "error" });
      });

    return () => controller.abort();
  }, [retryKey]);

  if (view.status === "unauthorized") return null;

  if (view.status === "error") {
    return (
      <section
        aria-label="Configurações do estabelecimento"
        className={styles.panel}
      >
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
          title="Não foi possível consultar as configurações"
          variant="error"
        >
          Tente novamente mais tarde.
        </Feedback>
      </section>
    );
  }

  if (view.status === "loading") {
    return (
      <section
        aria-busy="true"
        aria-label="Configurações do estabelecimento"
        className={styles.panel}
      >
        <p aria-live="polite" className={styles.loading} role="status">
          Carregando configurações…
        </p>
      </section>
    );
  }

  return <SettingsForm settings={view.settings} />;
}
