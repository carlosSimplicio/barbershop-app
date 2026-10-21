"use client";

import { useEffect, useRef, useState } from "react";

import { useAuthSession } from "@/components/session/auth-provider";
import { Button, Feedback } from "@/components/ui";
import {
  getEstablishment,
  type Establishment,
} from "@/lib/establishment/establishment-api";

import styles from "./testing-panel.module.css";

type View =
  | { status: "idle" | "loading" | "error" }
  | { status: "success"; establishment: Establishment };

export function EstablishmentPanel() {
  const { state } = useAuthSession();
  if (state.status !== "authenticated") return null;

  const { principal } = state;
  const identity = `${principal.user_id}:${principal.establishment_id}:${principal.password_change_required}`;
  return (
    <EstablishmentRequest
      key={identity}
      passwordChangeRequired={principal.password_change_required}
    />
  );
}

function EstablishmentRequest({
  passwordChangeRequired,
}: {
  passwordChangeRequired: boolean;
}) {
  const [view, setView] = useState<View>({ status: "idle" });
  const request = useRef<AbortController | null>(null);

  useEffect(() => () => request.current?.abort(), []);

  async function consult() {
    if (passwordChangeRequired || request.current) return;
    const controller = new AbortController();
    request.current = controller;
    setView({ status: "loading" });
    try {
      const result = await getEstablishment(controller.signal);
      if (controller.signal.aborted) return;
      if (result.status === 200 && result.state === "success") {
        setView({ status: "success", establishment: result.data });
      } else {
        setView({ status: "error" });
      }
    } catch {
      if (!controller.signal.aborted) setView({ status: "error" });
    } finally {
      if (request.current === controller) request.current = null;
    }
  }

  return (
    <section
      aria-labelledby="establishment-panel-title"
      className={styles.panel}
    >
      <h2 id="establishment-panel-title">Estabelecimento</h2>
      {passwordChangeRequired ? (
        <Feedback title="Consulta indisponível" variant="warning">
          Troque sua senha para consultar o estabelecimento.
        </Feedback>
      ) : null}
      {view.status === "error" ? (
        <Feedback
          title="Não foi possível consultar o estabelecimento"
          variant="error"
        >
          Tente novamente mais tarde.
        </Feedback>
      ) : null}
      {view.status === "success" ? (
        <EstablishmentDetails establishment={view.establishment} />
      ) : null}
      <Button
        disabled={passwordChangeRequired || view.status === "loading"}
        onClick={() => void consult()}
      >
        {view.status === "loading"
          ? "Consultando…"
          : "Consultar estabelecimento"}
      </Button>
    </section>
  );
}

function EstablishmentDetails({
  establishment,
}: {
  establishment: Establishment;
}) {
  return (
    <dl className={styles.details}>
      <div>
        <dt>Nome</dt>
        <dd>{establishment.name}</dd>
      </div>
      <div>
        <dt>Endereço público</dt>
        <dd>{establishment.slug}</dd>
      </div>
      <div>
        <dt>Fuso horário</dt>
        <dd>{establishment.time_zone}</dd>
      </div>
      <div>
        <dt>Avaliação</dt>
        <dd>
          {establishment.trial === null
            ? "Sem período de avaliação"
            : establishment.trial.status === "active"
              ? "Ativo"
              : "Expirado"}
        </dd>
      </div>
    </dl>
  );
}
