import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type ServiceRecord = components["schemas"]["ServiceResponse"];
export type ServicePage = components["schemas"]["ServiceListResponse"];
export function listServices({
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
  return apiFetch<ServicePage>(`/api/v1/services?${query}`, { signal });
}
