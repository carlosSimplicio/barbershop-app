import type { Metadata } from "next";

import { PasswordRecoveryBoundary } from "@/components/auth/password-recovery-boundary";

import styles from "../(session)/login/page.module.css";

export const metadata: Metadata = {
  title: "Redefinir senha | Barbershop",
  description: "Defina uma nova senha para sua conta.",
};

export default function PasswordRecoveryPage() {
  return (
    <main className={styles.page}>
      <section aria-labelledby="reset-title" className={styles.card}>
        <span className={styles.eyebrow}>Acesso à sua conta</span>
        <h1 id="reset-title">Redefinir senha</h1>
        <p className={styles.lead}>
          Defina uma nova senha para voltar a acessar sua conta.
        </p>
        <PasswordRecoveryBoundary />
      </section>
    </main>
  );
}
