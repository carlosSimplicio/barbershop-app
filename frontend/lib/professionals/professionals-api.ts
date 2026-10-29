import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type Professional = components["schemas"]["ProfessionalResponse"];
export type ProfessionalList =
  components["schemas"]["ProfessionalListResponse"];
export type CreateProfessionalRequest =
  components["schemas"]["CreateProfessionalRequest"];

export type UpdateProfessionalRequest =
  components["schemas"]["UpdateProfessionalRequest"];

export function createProfessional(
  request: CreateProfessionalRequest,
  signal?: AbortSignal,
) {
  return apiFetch<Professional>("/api/v1/professionals", {
    method: "POST",
    signal,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    csrf: true,
  });
}

export function listProfessionals({
  limit,
  offset,
  signal,
}: {
  limit: number;
  offset: number;
  signal?: AbortSignal;
}) {
  const query = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });

  return apiFetch<ProfessionalList>(`/api/v1/professionals?${query}`, {
    signal,
  });
}

export function getProfessional(id: string, signal?: AbortSignal) {
  return apiFetch<Professional>(
    `/api/v1/professionals/${encodeURIComponent(id)}`,
    { signal },
  );
}
export function updateProfessional(
  id: string,
  request: UpdateProfessionalRequest,
  signal?: AbortSignal,
) {
  return apiFetch<Professional>(
    `/api/v1/professionals/${encodeURIComponent(id)}`,
    {
      method: "PATCH",
      signal,
      csrf: true,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
    },
  );
}
