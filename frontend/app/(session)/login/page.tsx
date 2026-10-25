import type { Metadata } from "next";
import Link from "next/link";

import { LoginForm } from "@/components/auth/login-form/login-form";
import { SessionGuard } from "@/components/session/session-guard/session-guard";

import styles from "./page.module.css";

export const metadata: Metadata = {
  title: "Entrar | Barbershop",
  description: "Acesse sua conta para gerenciar seu estabelecimento.",
};

export default function LoginPage() {
  return (
    <SessionGuard require="anonymous">
      <main className={styles.page}>
        <section aria-labelledby="login-title" className={styles.card}>
          <Link className={styles.brand} href="/">
            Barbershop
          </Link>
          <span className={styles.eyebrow}>Seu espaço de trabalho</span>
          <h1 id="login-title">Entre na sua conta</h1>
          <p className={styles.lead}>
            Acesse sua agenda e acompanhe seu estabelecimento em um só lugar.
          </p>
          <LoginForm />
          <p className={styles.signUp}>
            Ainda não tem uma conta? <Link href="/cadastro">Cadastre-se</Link>
          </p>
        </section>
      </main>
    </SessionGuard>
  );
}
