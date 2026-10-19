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

test("preserva headers fornecidos como uma instancia de Headers", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(new Response(null, { status: 204 }));
  const headers = new Headers({
    "Content-Type": "application/json",
    "X-CSRF-Token": "csrf-token",
  });

  await apiFetch<void>("/api/v1/auth/refresh", {
    method: "POST",
    headers,
  });

  const requestHeaders = new Headers(fetchMock.mock.calls[0]?.[1]?.headers);
  expect(requestHeaders.get("Accept")).toBe("application/json");
  expect(requestHeaders.get("Content-Type")).toBe("application/json");
  expect(requestHeaders.get("X-CSRF-Token")).toBe("csrf-token");
});
