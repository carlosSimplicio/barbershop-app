import type { Metadata } from "next";
import Link from "next/link";

import { RegistrationForm } from "@/components/auth/registration-form";
import { SessionGuard } from "@/components/session/session-guard";

import styles from "./page.module.css";

export const metadata: Metadata = {
  title: "Crie seu espaço | Barbershop",
  description:
    "Cadastre seu estabelecimento e confirme seu e-mail para começar.",
};

export default function RegistrationPage() {
  return (
    <SessionGuard require="anonymous">
      <main className={styles.page}>
        <aside className={styles.brandPanel}>
          <Link className={styles.brand} href="/">
            <span aria-hidden="true" className={styles.brandMark}>
              B
            </span>
            <span>Barbershop</span>
          </Link>
          <div className={styles.brandCopy}>
            <span className={styles.eyebrow}>Seu negócio, do seu jeito</span>
            <p className={styles.brandHeading}>
              Mais tempo para cuidar de quem chega.
            </p>
            <p>
              Organize agenda, equipe e financeiro em um só lugar — com uma
              experiência simples para você e seus clientes.
            </p>
            <p className={styles.brandNote}>
              Feito para barbearias, salões e estúdios de beleza.
            </p>
          </div>
        </aside>
        <section
          aria-labelledby="registration-title"
          className={styles.formPanel}
        >
          <div className={styles.formWrap}>
            <Link className={styles.mobileBrand} href="/">
              Barbershop
            </Link>
            <span className={styles.eyebrow}>Comece por aqui</span>
            <h1 id="registration-title">Crie seu espaço</h1>
            <p className={styles.lead}>
              Preencha os dados para criar sua conta. Sem cartão.
            </p>
            <RegistrationForm />
            <p className={styles.signIn}>
              Já tem uma conta? <Link href="/login">Entrar</Link>
            </p>
          </div>
        </section>
      </main>
    </SessionGuard>
  );
}
