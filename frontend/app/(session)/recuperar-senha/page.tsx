import type { Metadata } from "next";
import Link from "next/link";

import { PasswordRecoveryRequestForm } from "@/components/auth/password-recovery-request-form";
import { SessionGuard } from "@/components/session/session-guard";

import styles from "../login/page.module.css";

export const metadata: Metadata = {
  title: "Recuperar senha | Barbershop",
  description: "Solicite um link para redefinir sua senha.",
};

export default function PasswordRecoveryRequestPage() {
  return (
    <SessionGuard require="anonymous">
      <main className={styles.page}>
        <section aria-labelledby="recovery-title" className={styles.card}>
          <Link className={styles.brand} href="/">
            Barbershop
          </Link>
          <span className={styles.eyebrow}>Acesso à sua conta</span>
          <h1 id="recovery-title">Recuperar senha</h1>
          <p className={styles.lead}>
            Informe seu e-mail para solicitar um link de redefinição de senha.
          </p>
          <PasswordRecoveryRequestForm />
          <p className={styles.signUp}>
            Lembrou sua senha? <Link href="/login">Voltar ao login</Link>
          </p>
        </section>
      </main>
    </SessionGuard>
  );
}
