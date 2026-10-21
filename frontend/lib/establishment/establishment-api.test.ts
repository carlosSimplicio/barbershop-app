import { afterEach, expect, test, vi } from "vitest";

import { getEstablishment } from "./establishment-api";

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
