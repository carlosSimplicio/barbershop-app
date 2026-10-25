"use client";

import { useEffect, useRef, useState } from "react";

import { AuthProvider } from "@/components/session/auth-provider/auth-provider";
import { Feedback } from "@/components/ui";

import { EmailVerification } from "../email-verification/email-verification";

type Capture =
  | { status: "capturing" }
  | { status: "error" }
  | { status: "ready"; token: string | null };

export function EmailConfirmationBoundary() {
  const [capture, setCapture] = useState<Capture>({ status: "capturing" });
  const token = useRef<string | null>(null);

  useEffect(() => {
    let active = true;
    if (token.current === null) {
      const fragment = window.location.hash;
      try {
        window.history.replaceState(
          window.history.state,
          "",
          window.location.pathname,
        );
      } catch {
        void Promise.resolve().then(() => {
          if (active) setCapture({ status: "error" });
        });
        return () => {
          active = false;
        };
      }
      token.current = new URLSearchParams(fragment.slice(1)).get("token") || "";
    }

    void Promise.resolve().then(() => {
      if (active) {
        setCapture({ status: "ready", token: token.current || null });
      }
    });
    return () => {
      active = false;
    };
  }, []);

  if (capture.status === "error") {
    return (
      <Feedback title="Não foi possível preparar a confirmação" variant="error">
        Não conseguimos abrir este link com segurança. Tente novamente mais
        tarde.
      </Feedback>
    );
  }

  if (capture.status === "capturing") {
    return <div role="status">Preparando confirmação…</div>;
  }

  return (
    <AuthProvider>
      <EmailVerification token={capture.token} />
    </AuthProvider>
  );
}
