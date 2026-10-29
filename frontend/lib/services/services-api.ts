import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type ServiceRecord = components["schemas"]["ServiceResponse"];
export type ServicePage = components["schemas"]["ServiceListResponse"];
export type CreateServiceRequest =
  components["schemas"]["CreateServiceRequest"];
export type UpdateServiceRequest =
  components["schemas"]["UpdateServiceRequest"];
type ListServicesOptions = {
  limit: number;
  offset: number;
  signal?: AbortSignal;
};

export function listServices({ limit, offset, signal }: ListServicesOptions) {
  const query = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  return apiFetch<ServicePage>(`/api/v1/services?${query}`, { signal });
}

export function createService(
  request: CreateServiceRequest,
  signal?: AbortSignal,
) {
  return apiFetch<ServiceRecord>("/api/v1/services", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    csrf: true,
    signal,
  });
}

export function getService(id: string, signal?: AbortSignal) {
  return apiFetch<ServiceRecord>(`/api/v1/services/${encodeURIComponent(id)}`, {
    signal,
  });
}
export function updateService(
  id: string,
  request: UpdateServiceRequest,
  signal?: AbortSignal,
) {
  return apiFetch<ServiceRecord>(`/api/v1/services/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    csrf: true,
    signal,
  });
}

export function setServiceStatus(
  id: string,
  active: boolean,
  signal?: AbortSignal,
) {
  return apiFetch<ServiceRecord>(
    `/api/v1/services/${encodeURIComponent(id)}/status`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ active }),
      csrf: true,
      signal,
    },
  );
}
