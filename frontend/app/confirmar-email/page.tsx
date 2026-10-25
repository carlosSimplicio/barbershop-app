import type { Metadata } from "next";

import { EmailConfirmationBoundary } from "@/components/auth/email-confirmation-boundary/email-confirmation-boundary";

import styles from "./page.module.css";

export const metadata: Metadata = {
  title: "Confirme seu e-mail | Barbershop",
  description: "Confirme o e-mail para ativar sua conta.",
};

export default function EmailConfirmationPage() {
  return (
    <main className="page-shell">
      <section aria-labelledby="confirmation-title" className={styles.panel}>
        <span className={styles.brand}>Barbershop</span>
        <h1 id="confirmation-title">Confirme seu e-mail</h1>
        <EmailConfirmationBoundary />
      </section>
    </main>
  );
}
