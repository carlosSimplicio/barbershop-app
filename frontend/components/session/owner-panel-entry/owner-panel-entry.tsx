"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { Button, Feedback } from "@/components/ui";
import { getEstablishmentSettings } from "@/lib/establishment/establishment-api";

type EntryState = "loading" | "error";

export function OwnerPanelEntry() {
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<EntryState>("loading");
  const router = useRouter();

  useEffect(() => {
    const controller = new AbortController();

    void getEstablishmentSettings(controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return;
        if (result.status !== 200 || result.state !== "success") {
          setState("error");
          return;
        }

        router.replace(
          result.data.logo ? "/painel/profissionais" : "/painel/configuracoes",
        );
      })
      .catch(() => {
        if (!controller.signal.aborted) setState("error");
      });

    return () => controller.abort();
  }, [attempt, router]);

  if (state === "loading") {
    return (
      <p aria-live="polite" role="status">
        Consultando configurações…
      </p>
    );
  }

  return (
    <Feedback
      action={
        <Button
          onClick={() => {
            setState("loading");
            setAttempt((current) => current + 1);
          }}
        >
          Tentar novamente
        </Button>
      }
      title="Não foi possível consultar as configurações"
      variant="error"
    >
      Verifique sua conexão e tente novamente.
    </Feedback>
  );
}
