import Link from "next/link";

import styles from "./not-found.module.css";

export default function EstablishmentNotFound() {
  return (
    <main className="page-shell">
      <div className="hero">
        <h1 className={styles.title}>Estabelecimento não encontrado</h1>
        <p>Esta página não está disponível.</p>
        <Link href="/">Voltar ao início</Link>
      </div>
    </main>
  );
}
