"use client";

import { Button, Feedback } from "@/components/ui";

export default function EstablishmentError({ reset }: { reset: () => void }) {
  return (
    <main className="page-shell">
      <Feedback
        title="Não foi possível abrir esta página"
        variant="error"
        action={<Button onClick={reset}>Tentar novamente</Button>}
      >
        Verifique sua conexão e tente novamente.
      </Feedback>
    </main>
  );
}
