import { apiFetch } from "@/lib/api/client";
import type { components } from "@/types/generated/api";

export type WeeklySchedule =
  components["schemas"]["ProfessionalWeeklyScheduleResponse"];
export type WorkPeriod = components["schemas"]["ProfessionalWorkPeriod"];
type WeeklyScheduleInput =
  components["schemas"]["ProfessionalWeeklyScheduleRequest"];

function schedulePath(professionalId: string): `/api/v1/${string}` {
  return `/api/v1/professionals/${encodeURIComponent(professionalId)}/weekly-schedule`;
}

export function getProfessionalWeeklySchedule(
  professionalId: string,
  signal?: AbortSignal,
) {
  return apiFetch<WeeklySchedule>(schedulePath(professionalId), { signal });
}

export function putProfessionalWeeklySchedule(
  professionalId: string,
  input: WeeklyScheduleInput,
  signal?: AbortSignal,
) {
  return apiFetch<WeeklySchedule>(schedulePath(professionalId), {
    method: "PUT",
    csrf: true,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
    signal,
  });
}
