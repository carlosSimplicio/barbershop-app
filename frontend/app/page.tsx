import type { Metadata } from "next";

import { ActionLink } from "@/components/ui";
import { TRIAL_DURATION_DAYS } from "@/lib/establishment/trial-policy";

import styles from "./page.module.css";

export const metadata: Metadata = {
  title: "Barbershop | Mais tempo para cuidar",
  description:
    "Agendamentos, equipe e finanças em uma plataforma para o seu estabelecimento.",
};

export default function Home() {
  return (
    <main className={styles.page}>
      <header className={styles.header}>
        <div className={styles.brand}>
          <span aria-hidden="true" className={styles.brandMark}>
            B
          </span>
          <span>Barbershop</span>
        </div>
        <span className={styles.headerNote}>Seu espaço para crescer</span>
      </header>

      <section aria-labelledby="home-title" className={styles.hero}>
        <div>
          <span className={styles.eyebrow}>
            Uma rotina mais leve começa aqui
          </span>
          <h1 id="home-title">
            Mais tempo para cuidar. <span>Menos tempo organizando.</span>
          </h1>
          <p className={styles.intro}>
            Reúna agenda e finanças em um só lugar. Seus clientes agendam pela
            página do estabelecimento, enquanto você acompanha o dia a dia da
            equipe com clareza.
          </p>
          <div className={styles.actions}>
            <ActionLink href="/cadastro">Criar minha conta</ActionLink>
            <ActionLink href="/login" variant="secondary">
              Entrar
            </ActionLink>
          </div>
          <p className={styles.trialNote}>
            Confirme seu e-mail para iniciar {TRIAL_DURATION_DAYS} dias de
            avaliação.
          </p>
        </div>

        <div aria-hidden="true" className={styles.preview}>
          <div className={styles.previewCard}>
            <div className={styles.previewHeader}>
              <span className={styles.previewMark}>B</span>
              <div>
                <strong>Seu dia em ordem</strong>
                <span>Visão simples da rotina</span>
              </div>
              <span className={styles.previewDots}>•••</span>
            </div>
            <div className={styles.previewBody}>
              <div className={styles.previewLabel}>
                <span>Agenda</span>
                <span>Hoje</span>
              </div>
              <div className={styles.previewRow}>
                <span>09:00</span>
                <i />
                <span>Atendimento</span>
              </div>
              <div className={styles.previewRow}>
                <span>10:30</span>
                <i />
                <span>Próximo horário</span>
              </div>
              <div className={styles.previewSummary}>
                <span>Seu trabalho, sem perder o ritmo.</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section aria-label="Como a plataforma ajuda" className={styles.features}>
        <article>
          <span className={styles.featureNumber}>01</span>
          <h2>Agendamentos mais simples</h2>
          <p>Um caminho direto para seus clientes encontrarem um horário.</p>
        </article>
        <article>
          <span className={styles.featureNumber}>02</span>
          <h2>Equipe no mesmo ritmo</h2>
          <p>Uma visão clara da rotina para proprietários e profissionais.</p>
        </article>
        <article>
          <span className={styles.featureNumber}>03</span>
          <h2>Números à vista</h2>
          <p>Agenda e finanças próximas das decisões do dia a dia.</p>
        </article>
      </section>
    </main>
  );
}
