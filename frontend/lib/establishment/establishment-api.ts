import type { components } from "@/types/generated/api";
import { apiFetch } from "@/lib/api/client";

export type Establishment = components["schemas"]["EstablishmentResponse"];

export function getEstablishment(signal?: AbortSignal) {
  return apiFetch<Establishment>("/api/v1/establishment", {
    signal,
  });
}
