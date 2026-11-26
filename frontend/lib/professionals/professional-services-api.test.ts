import { afterEach, expect, test, vi } from "vitest";
import {
  listProfessionalServices,
  setProfessionalService,
} from "./professional-services-api";

afterEach(() => vi.restoreAllMocks());
test("consulta uma página do profissional com cancelamento e sem tenant informado", async () => {
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), { status: 200 }),
    );
  const controller = new AbortController();
  await listProfessionalServices("professional/id", {
    limit: 20,
    offset: 40,
    signal: controller.signal,
  });
  expect(fetch).toHaveBeenCalledOnce();
  const [path, init] = fetch.mock.calls[0];
  expect(path).toBe(
    "/api/v1/professionals/professional%2Fid/services?limit=20&offset=40",
  );
  expect(init?.signal).toBe(controller.signal);
  expect(init?.credentials).toBe("include");
  expect(init?.body).toBeUndefined();
});

test("define apenas o vínculo solicitado por PUT com CSRF e estado explícito", async () => {
  document.cookie =
    "__Secure-barbershop_csrf=fixture-csrf; Secure; SameSite=Strict; Path=/";
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(
      new Response(JSON.stringify({ associated: false }), { status: 200 }),
    );
  const controller = new AbortController();
  await setProfessionalService(
    "professional/id",
    "service/id",
    { associated: false },
    controller.signal,
  );
  const [path, init] = fetch.mock.calls[0];
  expect(path).toBe(
    "/api/v1/professionals/professional%2Fid/services/service%2Fid",
  );
  expect(init?.method).toBe("PUT");
  expect(init?.signal).toBe(controller.signal);
  expect(JSON.parse(String(init?.body))).toEqual({ associated: false });
  expect(new Headers(init?.headers).get("X-CSRF-Token")).toBe("fixture-csrf");
  expect(new Headers(init?.headers).get("Content-Type")).toBe(
    "application/json",
  );
  document.cookie = "__Secure-barbershop_csrf=; Max-Age=0";
});
