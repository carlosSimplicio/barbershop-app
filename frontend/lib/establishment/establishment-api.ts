import type { components } from "@/types/generated/api";
import { apiFetch } from "@/lib/api/client";

export type Establishment = components["schemas"]["EstablishmentResponse"];
export type EstablishmentSettings =
  components["schemas"]["EstablishmentSettingsResponse"];
export type EstablishmentSettingsPatch =
  components["schemas"]["EstablishmentSettingsPatchRequest"];

export function getEstablishment(signal?: AbortSignal) {
  return apiFetch<Establishment>("/api/v1/establishment", {
    signal,
  });
}

export function getEstablishmentSettings(signal?: AbortSignal) {
  return apiFetch<EstablishmentSettings>("/api/v1/establishment/settings", {
    signal,
  });
}

export function patchEstablishmentSettings(
  request: EstablishmentSettingsPatch,
  signal?: AbortSignal,
) {
  return apiFetch<void>("/api/v1/establishment/settings", {
    body: JSON.stringify(request),
    csrf: true,
    headers: { "Content-Type": "application/json" },
    method: "PATCH",
    signal,
  });
}
