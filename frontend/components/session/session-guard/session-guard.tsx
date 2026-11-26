"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";
import type { ReactNode } from "react";

import { Button, Feedback } from "@/components/ui";
import type { AuthPrincipal } from "@/lib/auth/session-api/session-api";

import { useAuthSession } from "../auth-provider/auth-provider";

export function canAccessProfessionalPanel(principal: AuthPrincipal) {
  return (
    principal.role === "professional" &&
    principal.password_change_required === false
  );
}

export function SessionGuard({
  children,
  require,
}: {
  children: ReactNode;
  require: "authenticated" | "anonymous" | "owner" | "professional";
}) {
  const { state, reload } = useAuthSession();
  const router = useRouter();
  const allowed =
    require === "anonymous"
      ? state.status === "anonymous"
      : state.status === "authenticated" &&
        (require === "authenticated" ||
          (require === "owner" && state.principal.role === "owner") ||
          (require === "professional" &&
            canAccessProfessionalPanel(state.principal)));

  useEffect(() => {
    if (state.status === "authenticated" && !allowed) {
      router.replace(state.principal.role === "owner" ? "/painel" : "/testes");
    }
    if (state.status === "anonymous" && !allowed) router.replace("/login");
  }, [allowed, router, state]);

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
