import type { Metadata } from "next";

import { SessionGuard } from "@/components/session/session-guard";
import { EstablishmentPanel } from "@/components/testing/establishment-panel";
import { LogoutPanel } from "@/components/testing/logout-panel";
import { PasswordChangePanel } from "@/components/testing/password-change-panel";
import { SessionPanel } from "@/components/testing/session-panel";

import styles from "./page.module.css";

export const metadata: Metadata = {
  title: "Área de testes | Barbershop",
  description: "Área temporária para verificar as funções da sessão.",
};

export default function TestingPage() {
  return (
    <SessionGuard require="authenticated">
      <main className={styles.page}>
        <header className={styles.intro}>
          <span className={styles.eyebrow}>Ambiente temporário</span>
          <h1>Área de testes</h1>
          <p>Confira sua sessão atual e experimente as funções disponíveis.</p>
        </header>
        <div className={styles.sections}>
          <SessionPanel />
          <PasswordChangePanel />
          <EstablishmentPanel />
          <LogoutPanel />
        </div>
      </main>
    </SessionGuard>
  );
}
