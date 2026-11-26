import type { Metadata } from "next";

import { SessionGuard } from "@/components/session/session-guard/session-guard";
import { ActionLink } from "@/components/ui";

import { OwnSchedule } from "./own-schedule";
import styles from "./page.module.css";

export const metadata: Metadata = {
  title: "Minha jornada | Barbershop",
  description: "Consulte e configure seus horários semanais.",
};

export default function MyWeeklySchedulePage() {
  return (
    <SessionGuard require="professional">
      <main className={styles.page}>
        <header className={styles.header}>
          <h1>Minha jornada</h1>
          <ActionLink
            href="/testes"
            variant="secondary"
            aria-label="Voltar à área de testes"
          >
            Voltar
          </ActionLink>
        </header>
        <OwnSchedule />
      </main>
    </SessionGuard>
  );
}
