"use client";

import { useEffect, useRef, useState } from "react";

type TokenCapture =
  | { status: "capturing" }
  | { status: "error" }
  | { status: "ready"; token: string | null };

// Consumers mount only after the address has been cleared successfully.
export function useTransactionalLinkToken(): TokenCapture {
  const [capture, setCapture] = useState<TokenCapture>({ status: "capturing" });
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

  return capture;
}
