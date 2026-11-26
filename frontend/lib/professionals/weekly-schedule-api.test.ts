import { afterEach, expect, test, vi } from "vitest";
import {
  getProfessionalWeeklySchedule,
  putProfessionalWeeklySchedule,
} from "./weekly-schedule-api";

afterEach(() => {
  vi.restoreAllMocks();
  document.cookie = "__Secure-barbershop_csrf=; Max-Age=0; Path=/";
});

test("consulta e salva no mesmo recurso e envia CSRF com a semana completa", async () => {
  document.cookie =
    "__Secure-barbershop_csrf=fixture-csrf; Secure; SameSite=Strict; Path=/";
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(
      async () =>
        new Response(JSON.stringify({ periods: [] }), { status: 200 }),
    );
  const { signal } = new AbortController();
  const expectedPath =
    "/api/v1/professionals/professional%2Fid/weekly-schedule";

  await getProfessionalWeeklySchedule("professional/id", signal);
  await putProfessionalWeeklySchedule(
    "professional/id",
    { periods: [] },
    signal,
  );

  const [[getPath, get], [putPath, put]] = fetch.mock.calls;
  expect(getPath).toBe(expectedPath);
  expect(get?.credentials).toBe("include");
  expect(get?.signal).toBe(signal);
  expect(get?.body).toBeUndefined();
  expect(new Headers(get?.headers).has("X-CSRF-Token")).toBe(false);
  expect(putPath).toBe(expectedPath);
  expect(put?.method).toBe("PUT");
  expect(put?.credentials).toBe("include");
  expect(put?.signal).toBe(signal);
  expect(JSON.parse(String(put?.body))).toEqual({ periods: [] });
  expect(new Headers(put?.headers).get("X-CSRF-Token")).toBe("fixture-csrf");
});
