import { afterEach, expect, test, vi } from "vitest";

import { listProfessionals } from "./professionals-api";

afterEach(() => vi.restoreAllMocks());

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
