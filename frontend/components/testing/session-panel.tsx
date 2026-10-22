"use client";

import { useAuthSession } from "@/components/session/auth-provider";
import { Button } from "@/components/ui";

import styles from "./testing-panel.module.css";

export function SessionPanel() {
  const { state, refreshing, reload } = useAuthSession();
  if (state.status !== "authenticated") return null;

  const { principal } = state;
  return (
    <section aria-labelledby="session-panel-title" className={styles.panel}>
      <h2 id="session-panel-title">Sessão atual</h2>
      <dl className={styles.details}>
        <div>
          <dt>Papel</dt>
          <dd>
            {principal.role === "owner" ? "Proprietário" : "Profissional"}
          </dd>
        </div>
        <div>
          <dt>Usuário</dt>
          <dd>{principal.user_id}</dd>
        </div>
        <div>
          <dt>Estabelecimento</dt>
          <dd>{principal.establishment_id}</dd>
        </div>
        <div>
          <dt>Troca de senha obrigatória</dt>
          <dd>{principal.password_change_required ? "Sim" : "Não"}</dd>
        </div>
      </dl>
      <Button disabled={refreshing} onClick={() => void reload()}>
        Consultar sessão novamente
      </Button>
    </section>
  );
}
