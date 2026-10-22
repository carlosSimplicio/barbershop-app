import { afterEach, expect, test, vi } from "vitest";

import { apiFetch } from "./client";

afterEach(() => {
  vi.restoreAllMocks();
});

test("representa resposta 202 sem corpo como vazia e preserva o status", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(null, { status: 202 }),
  );

  await expect(
    apiFetch<void>("/api/v1/auth/register", { method: "POST" }),
  ).resolves.toEqual({ state: "empty", status: 202 });
});

test("preserva headers fornecidos e envia cookies", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(new Response(null, { status: 204 }));

  await apiFetch<void>("/api/v1/auth/refresh", {
    method: "POST",
    headers: new Headers({ "Content-Type": "application/json" }),
  });

  const options = fetchMock.mock.calls[0]?.[1];
  const headers = new Headers(options?.headers);
  expect(options?.credentials).toBe("include");
  expect(headers.get("Accept")).toBe("application/json");
  expect(headers.get("Content-Type")).toBe("application/json");
});

test("lê o CSRF atual apenas quando solicitado", async () => {
  vi.spyOn(document, "cookie", "get").mockReturnValue(
    "theme=dark; __Secure-barbershop_csrf=csrf-atual",
  );
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(new Response(null, { status: 204 }));

  await apiFetch<void>("/api/v1/auth/logout", {
    csrf: true,
    headers: { "X-CSRF-Token": "csrf-antigo" },
    method: "POST",
  });
  await apiFetch<void>("/api/v1/auth/login", { method: "POST" });

  expect(
    new Headers(fetchMock.mock.calls[0]?.[1]?.headers).get("X-CSRF-Token"),
  ).toBe("csrf-atual");
  expect(
    new Headers(fetchMock.mock.calls[1]?.[1]?.headers).has("X-CSRF-Token"),
  ).toBe(false);
});

test("conserva detalhes válidos de erro e rejeita formato inválido", async () => {
  const error = {
    code: "validation_failed",
    fields: [
      { code: "invalid_email", field: "email", message: "E-mail inválido." },
    ],
    message: "Verifique os campos.",
    request_id: "request-id",
  };
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(jsonResponse(error, 400))
    .mockResolvedValueOnce(jsonResponse({ ...error, fields: "invalid" }, 400));

  await expect(apiFetch("/api/v1/auth/login")).resolves.toEqual({
    error,
    state: "error",
    status: 400,
  });
  await expect(apiFetch("/api/v1/auth/login")).resolves.toMatchObject({
    error: { code: "unexpected_response" },
    state: "error",
    status: 400,
  });
});

function jsonResponse(body: unknown, status: number) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    status,
  });
}
