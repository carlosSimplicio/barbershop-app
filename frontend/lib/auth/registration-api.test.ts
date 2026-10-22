import { afterEach, expect, test, vi } from "vitest";

import { registerOwner } from "./registration-api";

afterEach(() => vi.restoreAllMocks());

test("envia o cadastro ao endpoint público com os cinco campos e sem CSRF", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(new Response(null, { status: 202 }));

  const result = await registerOwner({
    establishment_name: "Estúdio Aurora",
    slug: "estudio-aurora",
    time_zone: "America/Sao_Paulo",
    email: "owner@example.com",
    password: "senha-segura",
  });

  expect(result).toEqual({ state: "empty", status: 202 });
  expect(fetchMock).toHaveBeenCalledOnce();
  const [url, init] = fetchMock.mock.calls[0];
  expect(url).toBe("/api/v1/auth/register");
  expect(init?.method).toBe("POST");
  expect(init?.credentials).toBe("include");
  expect(JSON.parse(String(init?.body))).toEqual({
    establishment_name: "Estúdio Aurora",
    slug: "estudio-aurora",
    time_zone: "America/Sao_Paulo",
    email: "owner@example.com",
    password: "senha-segura",
  });
  const headers = new Headers(init?.headers);
  expect(headers.get("Content-Type")).toBe("application/json");
  expect(headers.has("X-CSRF-Token")).toBe(false);
});
