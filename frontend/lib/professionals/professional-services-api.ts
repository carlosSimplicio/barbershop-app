import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type ProfessionalService =
  components["schemas"]["ProfessionalServiceResponse"];
export type ProfessionalServicesPage =
  components["schemas"]["ProfessionalServicesResponse"];
type ListOptions = { limit: number; offset: number; signal?: AbortSignal };

export function listProfessionalServices(
  professionalId: string,
  { limit, offset, signal }: ListOptions,
) {
  const query = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  return apiFetch<ProfessionalServicesPage>(
    `/api/v1/professionals/${encodeURIComponent(professionalId)}/services?${query}`,
    { signal },
  );
}

export function setProfessionalService(
  professionalId: string,
  serviceId: string,
  request: components["schemas"]["ProfessionalServiceRequest"],
  signal?: AbortSignal,
) {
  return apiFetch<ProfessionalService>(
    `/api/v1/professionals/${encodeURIComponent(professionalId)}/services/${encodeURIComponent(serviceId)}`,
    {
      method: "PUT",
      csrf: true,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
      signal,
    },
  );
}
