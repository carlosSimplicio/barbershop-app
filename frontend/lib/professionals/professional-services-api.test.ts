import { afterEach, expect, test, vi } from "vitest";
import { listProfessionalServices } from "./professional-services-api";

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
