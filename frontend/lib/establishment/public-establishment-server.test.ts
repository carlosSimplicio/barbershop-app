import { afterEach, expect, test, vi } from "vitest";
import {
  getPublicEstablishment,
  getServerConfig,
} from "./public-establishment-server";

afterEach(() => {
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
});
function configure() {
  vi.stubEnv("API_INTERNAL_URL", "http://backend:8080");
  vi.stubEnv("PUBLIC_APP_URL", "https://example.test");
}
test("reads a public profile without cookies or persistent cache", async () => {
  configure();
  const profile = {
    name: "Estúdio",
    slug: "estudio",
    logo_url: "/api/v1/public/establishments/estudio/logo",
  };
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(Response.json(profile));
  expect(await getPublicEstablishment("estudio")).toEqual(profile);
  expect(fetch).toHaveBeenCalledWith(
    "http://backend:8080/api/v1/public/establishments/estudio",
    expect.objectContaining({
      cache: "no-store",
      credentials: "omit",
      redirect: "error",
    }),
  );
});
test("only a 404 becomes an absent establishment", async () => {
  configure();
  vi.spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(new Response(null, { status: 404 }))
    .mockResolvedValueOnce(new Response(null, { status: 503 }))
    .mockRejectedValueOnce(new Error("private backend detail"));
  expect(await getPublicEstablishment("estudio")).toBeNull();
  await expect(getPublicEstablishment("estudio")).rejects.toThrow(
    "Não foi possível consultar o estabelecimento.",
  );
  await expect(getPublicEstablishment("estudio")).rejects.toThrow(
    "Não foi possível consultar o estabelecimento.",
  );
});
test.each([
  "",
  "http://example.test",
  "https://example.test/path",
  "https://user:secret@example.test",
  "https://example.test?query=yes",
])("rejects invalid canonical origin %s", (origin) => {
  configure();
  vi.stubEnv("PUBLIC_APP_URL", origin);
  expect(() => getServerConfig()).toThrow("PUBLIC_APP_URL");
});
test("rejects an invalid internal origin without exposing its value", () => {
  configure();
  vi.stubEnv("API_INTERNAL_URL", "file:///private");
  expect(() => getServerConfig()).toThrow("API_INTERNAL_URL");
});

test("does not request API paths for malformed public slugs", async () => {
  configure();
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(Response.json({ name: "Estúdio" }));
  expect(await getPublicEstablishment("..")).toBeNull();
  expect(await getPublicEstablishment("other/path")).toBeNull();
  expect(fetch).not.toHaveBeenCalled();
});
