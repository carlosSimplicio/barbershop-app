import type { ApiResult } from "@/lib/api/client";
import { serverOfferingErrors, type FieldErrors } from "./offering-form-model";
import type { Offering } from "./catalog-api";

type OfferingOperationResult =
  | { state: "confirmed"; record: Offering }
  | { state: "validation"; fields: FieldErrors }
  | {
      state:
        "unauthorized" | "missing" | "read_only" | "uncertain" | "rejected";
    };

export function classifyOfferingOperation(
  result: ApiResult<Offering>,
  expectedStatus: 200 | 201,
): OfferingOperationResult {
  if (result.status === 401) return { state: "unauthorized" };
  if (result.status === 404) return { state: "missing" };
  if (result.state === "success" && result.status === expectedStatus)
    return { state: "confirmed", record: result.data };
  if (result.state === "error") {
    if (result.error.code === "read_only") return { state: "read_only" };
    const fields = serverOfferingErrors(result.error.fields);
    if (Object.keys(fields).length) return { state: "validation", fields };
  }
  return {
    state:
      result.status >= 500 || result.status < 400 ? "uncertain" : "rejected",
  };
}
