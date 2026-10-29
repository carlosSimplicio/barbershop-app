import type { ApiResult } from "@/lib/api/client";
import { serverServiceErrors, type FieldErrors } from "./service-form-model";
import type { ServiceRecord } from "./services-api";

type ServiceOperationResult =
  | { state: "confirmed"; record: ServiceRecord }
  | { state: "validation"; fields: FieldErrors }
  | {
      state:
        "unauthorized" | "missing" | "read_only" | "uncertain" | "rejected";
    };

export function classifyServiceOperation(
  result: ApiResult<ServiceRecord>,
  expectedStatus: 200 | 201,
): ServiceOperationResult {
  if (result.status === 401) return { state: "unauthorized" };
  if (result.status === 404) return { state: "missing" };
  if (result.state === "success" && result.status === expectedStatus)
    return { state: "confirmed", record: result.data };
  if (result.state === "error") {
    if (result.error.code === "read_only") return { state: "read_only" };
    const fields = serverServiceErrors(result.error.fields);
    if (Object.keys(fields).length) return { state: "validation", fields };
  }
  return {
    state:
      result.status >= 500 || result.status < 400 ? "uncertain" : "rejected",
  };
}
