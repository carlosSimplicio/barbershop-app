import type { Metadata } from "next";

import { SessionGuard } from "@/components/session/session-guard/session-guard";
import { OwnerPanelShell } from "@/components/session/owner-panel-shell/owner-panel-shell";
import { ProfessionalsWorkspace } from "@/components/professionals/professionals-workspace/professionals-workspace";

import styles from "./page.module.css";

export const metadata: Metadata = {
  title: "Profissionais | Barbershop",
  description: "Área de profissionais do estabelecimento.",
};

export default function ProfessionalsPage() {
  return (
    <SessionGuard require="owner">
      <OwnerPanelShell>
        <section className={styles.intro}>
          <span className={styles.eyebrow}>Painel do estabelecimento</span>
          <h1>Profissionais</h1>
        </section>
        <ProfessionalsWorkspace />
      </OwnerPanelShell>
    </SessionGuard>
  );
}
