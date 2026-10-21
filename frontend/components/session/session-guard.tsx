"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import type { ReactNode } from "react";

import { Button, Feedback } from "@/components/ui";

import { useAuthSession } from "./auth-provider";

export function SessionGuard({
  children,
  require,
}: {
  children: ReactNode;
  require: "authenticated" | "anonymous";
}) {
  const { state, reload } = useAuthSession();
  const router = useRouter();
  const allowed = state.status === require;

  useEffect(() => {
    if (state.status === "authenticated" && !allowed) router.replace("/testes");
    if (state.status === "anonymous" && !allowed) router.replace("/login");
  }, [allowed, router, state.status]);

  if (state.status === "unavailable") {
    return (
      <main className="page-shell">
        <Feedback
          action={
            <Button onClick={() => void reload()}>Tentar novamente</Button>
          }
          title="Não foi possível verificar sua sessão"
          variant="error"
        >
          Verifique sua conexão e tente novamente.
        </Feedback>
      </main>
    );
  }

  if (!allowed) {
    return (
      <main className="page-shell" role="status">
        {state.status === "loading" ? "Carregando sessão…" : "Redirecionando…"}
      </main>
    );
  }

  return children;
}
