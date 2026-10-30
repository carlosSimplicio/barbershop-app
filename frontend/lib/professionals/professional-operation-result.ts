import type { ApiResult } from "@/lib/api/client";
import {
  professionalFieldErrors,
  type ProfessionalFieldErrors,
} from "./professional-form-model";
import type { Professional, ProfessionalList } from "./professionals-api";

type ProfessionalOperationResult =
  | { state: "confirmed"; record: Professional }
  | { state: "validation"; fields: ProfessionalFieldErrors }
  | {
      state:
        | "unauthorized"
        | "missing"
        | "read_only"
        | "uncertain"
        | "rejected"
        | "already_active";
    };

export function isProfessional(value: unknown): value is Professional {
  if (!value || typeof value !== "object") return false;
  const record = value as Partial<Professional>;
  return (
    typeof record.id === "string" &&
    typeof record.name === "string" &&
    typeof record.email === "string" &&
    typeof record.active === "boolean"
  );
}
export function isProfessionalList(value: unknown): value is ProfessionalList {
  if (!value || typeof value !== "object") return false;
  const page = value as Partial<ProfessionalList>;
  return (
    Array.isArray(page.items) &&
    page.items.every(isProfessional) &&
    typeof page.can_write === "boolean" &&
    typeof page.has_more === "boolean" &&
    Number.isInteger(page.limit) &&
    Number.isInteger(page.offset) &&
    page.limit! > 0 &&
    page.offset! >= 0
  );
}
export function classifyProfessionalOperation(
  result: ApiResult<Professional>,
  expected: 200 | 201,
): ProfessionalOperationResult {
  if (result.status === 401) return { state: "unauthorized" };
  if (result.status === 404) return { state: "missing" };
  if (
    result.state === "success" &&
    result.status === expected &&
    isProfessional(result.data)
  )
    return { state: "confirmed", record: result.data };
  if (result.state === "error" && result.status < 500) {
    if (result.error.code === "read_only") return { state: "read_only" };
    if (
      result.status === 409 &&
      result.error.code === "professional_already_active"
    )
      return { state: "already_active" };
    const fields = professionalFieldErrors(result.error.fields);
    if (Object.keys(fields).length) return { state: "validation", fields };
  }
  return {
    state:
      result.status >= 500 || result.status < 400 ? "uncertain" : "rejected",
  };
}
