import { afterEach, expect, test, vi } from "vitest";

import { createProfessional, listProfessionals } from "./professionals-api";

afterEach(() => vi.restoreAllMocks());

test("cadastra profissional por POST tipado com CSRF", async () => {
  document.cookie =
    "__Secure-barbershop_csrf=csrf-test-token; Secure; SameSite=Strict; Path=/";
  const response = {
    id: "professional-id",
    name: "Ana Lima",
    email: "ana@example.test",
    active: true,
  };
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(response), {
      status: 201,
      headers: { "Content-Type": "application/json" },
    }),
  );

  await expect(
    createProfessional({
      name: "Ana Lima",
      email: "ana@example.test",
      temporary_password: "senha-segura",
    }),
  ).resolves.toEqual({ state: "success", status: 201, data: response });

  const [path, init] = fetchMock.mock.calls[0];
  expect(path).toBe("/api/v1/professionals");
  expect(init?.method).toBe("POST");
  expect(init?.credentials).toBe("include");
  expect(new Headers(init?.headers).get("Content-Type")).toBe(
    "application/json",
  );
  expect(new Headers(init?.headers).get("X-CSRF-Token")).toBe(
    "csrf-test-token",
  );
  expect(JSON.parse(String(init?.body))).toEqual({
    name: "Ana Lima",
    email: "ana@example.test",
    temporary_password: "senha-segura",
  });
  document.cookie = "__Secure-barbershop_csrf=; Max-Age=0";
});

test("consulta uma página tipada sem enviar identificador de estabelecimento", async () => {
  const response = {
    items: [],
    limit: 20,
    offset: 40,
    has_more: false,
  };
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(response), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  const controller = new AbortController();

  await expect(
    listProfessionals({ limit: 20, offset: 40, signal: controller.signal }),
  ).resolves.toEqual({ state: "success", status: 200, data: response });

  const [path, init] = fetchMock.mock.calls[0];
  expect(path).toBe("/api/v1/professionals?limit=20&offset=40");
  expect(init?.signal).toBe(controller.signal);
  expect(init?.credentials).toBe("include");
});
