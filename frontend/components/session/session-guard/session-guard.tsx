"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import type { ReactNode } from "react";

import { Button, Feedback } from "@/components/ui";

import { useAuthSession } from "../auth-provider/auth-provider";

export function SessionGuard({
  children,
  require,
}: {
  children: ReactNode;
  require: "authenticated" | "anonymous" | "owner";
}) {
  const { state, reload } = useAuthSession();
  const router = useRouter();
  const allowed =
    require === "anonymous"
      ? state.status === "anonymous"
      : state.status === "authenticated" &&
        (require === "authenticated" || state.principal.role === "owner");

  useEffect(() => {
    if (state.status === "authenticated" && !allowed) {
      router.replace(
        require === "anonymous" && state.principal.role === "owner"
          ? "/painel/profissionais"
          : "/testes",
      );
    }
    if (state.status === "anonymous" && !allowed) router.replace("/login");
  }, [allowed, require, router, state]);

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
