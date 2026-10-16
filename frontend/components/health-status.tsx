"use client";

import { useEffect, useState } from "react";

import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

type Health = components["schemas"]["HealthResponse"];
type ViewState = "loading" | "success" | "error";

const labels: Record<ViewState, string> = {
  loading: "Verificando a API…",
  success: "Frontend e API conectados",
  error: "API indisponível no momento",
};

export function HealthStatus() {
  const [state, setState] = useState<ViewState>("loading");

  useEffect(() => {
    const controller = new AbortController();

    void apiFetch<Health>("/api/v1/health", { signal: controller.signal })
      .then((result) =>
        setState(result.state === "success" ? "success" : "error"),
      )
      .catch((error: unknown) => {
        if (!(error instanceof DOMException && error.name === "AbortError")) {
          setState("error");
        }
      });

    return () => controller.abort();
  }, []);

  return (
    <div
      className="health-status"
      data-state={state}
      role="status"
      aria-live="polite"
    >
      {labels[state]}
    </div>
  );
}
