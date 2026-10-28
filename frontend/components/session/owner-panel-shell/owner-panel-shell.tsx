import Link from "next/link";
import type { ReactNode } from "react";

import styles from "./owner-panel-shell.module.css";

export function OwnerPanelShell({
  activePage = "professionals",
  children,
}: {
  activePage?: "professionals" | "settings" | "none";
  children: ReactNode;
}) {
  return (
    <div className={styles.page}>
      <a className={styles.skipLink} href="#panel-main">
        Pular para o conteúdo principal
      </a>
      <header className={styles.header}>
        <Link className={styles.brand} href="/painel/profissionais">
          Barbershop
        </Link>
        <nav aria-label="Navegação do painel" className={styles.navigation}>
          <Link
            aria-current={activePage === "professionals" ? "page" : undefined}
            className={styles.link}
            href="/painel/profissionais"
          >
            Profissionais
          </Link>
          <Link
            aria-current={activePage === "settings" ? "page" : undefined}
            className={styles.link}
            href="/painel/configuracoes"
          >
            Configurações
          </Link>
          <Link className={styles.link} href="/testes">
            Área de testes
          </Link>
        </nav>
      </header>
      <main className={styles.content} id="panel-main">
        {children}
      </main>
    </div>
  );
}
