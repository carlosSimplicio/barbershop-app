import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type Offering = components["schemas"]["ServiceResponse"];
export type CatalogPage = components["schemas"]["ServiceListResponse"];
export type CreateOfferingRequest =
  components["schemas"]["CreateServiceRequest"];
export type UpdateOfferingRequest =
  components["schemas"]["UpdateServiceRequest"];
type ListOfferingsOptions = {
  limit: number;
  offset: number;
  signal?: AbortSignal;
};

export function listOfferings({ limit, offset, signal }: ListOfferingsOptions) {
  const query = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  return apiFetch<CatalogPage>(`/api/v1/services?${query}`, { signal });
}

export function createOffering(
  request: CreateOfferingRequest,
  signal?: AbortSignal,
) {
  return apiFetch<Offering>("/api/v1/services", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
    csrf: true,
    signal,
  });
}

export function getOffering(offeringId: string, signal?: AbortSignal) {
  return apiFetch<Offering>(
    `/api/v1/services/${encodeURIComponent(offeringId)}`,
    {
      signal,
    },
  );
}
export function updateOffering(
  offeringId: string,
  request: UpdateOfferingRequest,
  signal?: AbortSignal,
) {
  return apiFetch<Offering>(
    `/api/v1/services/${encodeURIComponent(offeringId)}`,
    {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
      csrf: true,
      signal,
    },
  );
}

export function setOfferingStatus(
  offeringId: string,
  active: boolean,
  signal?: AbortSignal,
) {
  return apiFetch<Offering>(
    `/api/v1/services/${encodeURIComponent(offeringId)}/status`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ active }),
      csrf: true,
      signal,
    },
  );
}
