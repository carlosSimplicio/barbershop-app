import { afterEach, expect, test, vi } from "vitest";

import { apiFetch } from "./client";

afterEach(() => {
  vi.restoreAllMocks();
});

test("representa resposta 202 sem corpo como vazia", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(null, { status: 202 }),
  );

  await expect(
    apiFetch<void>("/api/v1/auth/register", { method: "POST" }),
  ).resolves.toEqual({ state: "empty" });
});
