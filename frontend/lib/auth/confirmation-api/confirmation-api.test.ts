import { afterEach, expect, test, vi } from "vitest";

import { requestEmailConfirmation } from "./confirmation-api";

afterEach(() => vi.restoreAllMocks());

test("solicita confirmação pelo endpoint público apenas com o e-mail", async () => {
  const fetchMock = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(new Response(null, { status: 202 }));

  await expect(
    requestEmailConfirmation({ email: "owner@example.com" }),
  ).resolves.toEqual({ state: "empty", status: 202 });

  expect(fetchMock).toHaveBeenCalledOnce();
  const [url, init] = fetchMock.mock.calls[0];
  expect(url).toBe("/api/v1/auth/email-confirmation");
  expect(init?.method).toBe("POST");
  expect(init?.credentials).toBe("include");
  expect(JSON.parse(String(init?.body))).toEqual({
    email: "owner@example.com",
  });
  expect(new Headers(init?.headers).has("X-CSRF-Token")).toBe(false);
});
