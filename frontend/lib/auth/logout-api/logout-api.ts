import { apiFetch } from "@/lib/api/client";

export async function logoutCurrentSession() {
  const result = await apiFetch<void>("/api/v1/auth/logout", {
    method: "POST",
    csrf: true,
  });
  if (result.state !== "error" && result.status !== 204) {
    throw new Error("Unexpected logout response");
  }
  return result;
}
