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
    can_write: true,
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

test("consulta e edita nome com cancelamento e CSRF somente na escrita", async () => {
  const { getProfessional, updateProfessional } =
    await import("./professionals-api");
  document.cookie =
    "__Secure-barbershop_csrf=csrf-test-token; Secure; SameSite=Strict; Path=/";
  const fetchMock = vi.spyOn(globalThis, "fetch").mockImplementation(
    async () =>
      new Response(
        JSON.stringify({
          id: "id",
          name: "Ana",
          email: "ana@example.test",
          active: false,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
  );
  const controller = new AbortController();
  await getProfessional("id", controller.signal);
  await updateProfessional("id", { name: "Ana" }, controller.signal);
  const [getPath, getInit] = fetchMock.mock.calls[0];
  const [patchPath, patchInit] = fetchMock.mock.calls[1];
  expect(getPath).toBe("/api/v1/professionals/id");
  expect(patchPath).toBe(getPath);
  expect(getInit?.signal).toBe(controller.signal);
  expect(patchInit?.signal).toBe(controller.signal);
  expect(new Headers(getInit?.headers).get("X-CSRF-Token")).toBeNull();
  expect(patchInit?.method).toBe("PATCH");
  expect(new Headers(patchInit?.headers).get("X-CSRF-Token")).toBe(
    "csrf-test-token",
  );
  expect(JSON.parse(String(patchInit?.body))).toEqual({ name: "Ana" });
  document.cookie = "__Secure-barbershop_csrf=; Max-Age=0";
});
