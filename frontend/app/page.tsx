import { HealthStatus } from "@/components/health-status";

export default function Home() {
  return (
    <main className="page-shell">
      <section className="hero" aria-labelledby="page-title">
        <span className="eyebrow">Fundação do produto</span>
        <h1 id="page-title">Barbershop</h1>
        <p>
          A base da plataforma está pronta para receber agenda, clientes e
          gestão financeira.
        </p>
        <HealthStatus />
      </section>
    </main>
  );
}
