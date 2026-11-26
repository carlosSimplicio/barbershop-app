import type { ApiResult } from "@/lib/api/client";
import { MAX_PAGE_OFFSET } from "@/lib/api/pagination";
import { OFFERING_LIMITS } from "@/lib/catalog/offering-limits";
import type {
  ProfessionalService,
  ProfessionalServicesPage,
} from "./professional-services-api";

type AssociationResult =
  | { state: "confirmed"; record: ProfessionalService }
  | {
      state:
        "unauthorized" | "missing" | "read_only" | "forbidden" | "rejected";
    };

export function isProfessionalService(
  value: unknown,
): value is ProfessionalService {
  if (!value || typeof value !== "object") return false;
  const item = value as Partial<ProfessionalService>;
  return (
    typeof item.id === "string" &&
    typeof item.name === "string" &&
    typeof item.active === "boolean" &&
    typeof item.associated === "boolean" &&
    Number.isSafeInteger(item.price_cents) &&
    item.price_cents! >= OFFERING_LIMITS.priceCents.min &&
    item.price_cents! <= OFFERING_LIMITS.priceCents.max &&
    Number.isSafeInteger(item.duration_minutes) &&
    item.duration_minutes! >= OFFERING_LIMITS.durationMinutes.min &&
    item.duration_minutes! <= OFFERING_LIMITS.durationMinutes.max
  );
}
export function isProfessionalServicesPage(
  value: unknown,
): value is ProfessionalServicesPage {
  if (!value || typeof value !== "object") return false;
  const page = value as Partial<ProfessionalServicesPage>;
  return (
    !!page.professional &&
    typeof page.professional.id === "string" &&
    typeof page.professional.name === "string" &&
    typeof page.professional.active === "boolean" &&
    Array.isArray(page.items) &&
    page.items.every(isProfessionalService) &&
    typeof page.can_write === "boolean" &&
    typeof page.has_more === "boolean" &&
    Number.isSafeInteger(page.limit) &&
    page.limit! > 0 &&
    page.items.length <= page.limit! &&
    Number.isSafeInteger(page.offset) &&
    page.offset! >= 0 &&
    page.offset! <= MAX_PAGE_OFFSET
  );
}

export function classifyProfessionalServiceOperation(
  result: ApiResult<ProfessionalService>,
  serviceId: string,
  associated: boolean,
): AssociationResult {
  if (result.status === 401) return { state: "unauthorized" };
  if (result.status === 404) return { state: "missing" };
  if (
    result.state === "success" &&
    result.status === 200 &&
    isProfessionalService(result.data) &&
    result.data.id === serviceId &&
    result.data.associated === associated
  )
    return { state: "confirmed", record: result.data };
  if (result.state === "error" && result.status === 403)
    return {
      state: result.error.code === "read_only" ? "read_only" : "forbidden",
    };
  return { state: "rejected" };
}
