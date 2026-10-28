import { afterEach, expect, test, vi } from "vitest";

import {
  getEstablishment,
  getEstablishmentSettings,
  patchEstablishmentSettings,
} from "./establishment-api";

afterEach(() => vi.restoreAllMocks());

test("consulta o estabelecimento da sessão sem enviar ID de tenant", async () => {
  const establishment = {
    id: "establishment-id",
    name: "Estúdio Aurora",
    slug: "estudio-aurora",
    time_zone: "America/Sao_Paulo",
    trial: null,
  };
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(establishment), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );

  await expect(getEstablishment()).resolves.toEqual({
    state: "success",
    status: 200,
    data: establishment,
  });
  const [path, init] = fetchMock.mock.calls[0];
  expect(path).toBe("/api/v1/establishment");
  expect(init?.method).toBeUndefined();
  expect(init?.body).toBeUndefined();
  expect(init?.credentials).toBe("include");
  expect(new Headers(init?.headers).has("X-CSRF-Token")).toBe(false);
});

test("expõe operações de consulta e atualização das configurações", async () => {
  const settings = { id: "establishment-id", name: "Estúdio Aurora" };
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify(settings), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  const controller = new AbortController();

  await expect(
    getEstablishmentSettings(controller.signal),
  ).resolves.toMatchObject({
    data: settings,
    state: "success",
  });
  const [path, getInit] = fetchMock.mock.calls[0];
  expect(path).toBe("/api/v1/establishment/settings");
  expect(getInit?.method).toBeUndefined();
  expect(getInit?.signal).toBe(controller.signal);
  expect(getInit?.body).toBeUndefined();

  vi.spyOn(document, "cookie", "get").mockReturnValue(
    "__Secure-barbershop_csrf=csrf-current",
  );
  fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }));
  const patch = { description: null, social_links: [] };

  await expect(patchEstablishmentSettings(patch)).resolves.toEqual({
    state: "empty",
    status: 204,
  });
  const [patchPath, patchInit] = fetchMock.mock.calls[1];
  expect(patchPath).toBe("/api/v1/establishment/settings");
  expect(patchInit?.method).toBe("PATCH");
  expect(patchInit?.body).toBe(JSON.stringify(patch));
  expect(new Headers(patchInit?.headers).get("X-CSRF-Token")).toBe(
    "csrf-current",
  );
});
