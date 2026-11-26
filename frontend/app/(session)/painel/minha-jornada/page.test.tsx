import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, expect, test, vi } from "vitest";

import { AuthProvider } from "@/components/session/auth-provider/auth-provider";
import { configureApiSessionObserver } from "@/lib/api/client";
import type { AuthPrincipal } from "@/lib/auth/session-api/session-api";

import MyWeeklySchedulePage from "./page";

const replace = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));

const professionalId = "4bb49346-9b95-419c-a7bd-4a2e2a8728ef";
const path = `/api/v1/professionals/${professionalId}/weekly-schedule`;
const period = { weekday: 1, start_time: "08:00", end_time: "12:00" };

function session(role: AuthPrincipal["role"], pending = false) {
  return Response.json({
    user_id: professionalId,
    establishment_id: "076bbdc4-ab97-4e38-8fd6-5af7b0d52daf",
    email: "professional@example.test",
    role,
    password_change_required: pending,
  } satisfies AuthPrincipal);
}

function schedule(canWrite: boolean, endTime = period.end_time) {
  return Response.json({
    professional_id: professionalId,
    time_zone: "America/Sao_Paulo",
    can_write: canWrite,
    periods: [{ ...period, end_time: endTime }],
  });
}

function renderPage() {
  render(
    <AuthProvider>
      <MyWeeklySchedulePage />
    </AuthProvider>,
  );
}

afterEach(() => {
  configureApiSessionObserver(null);
  replace.mockReset();
  vi.restoreAllMocks();
  document.cookie = "__Secure-barbershop_csrf=; Max-Age=0; Path=/";
});

test("consulta e grava a própria jornada usando a identidade da sessão", async () => {
  document.cookie =
    "__Secure-barbershop_csrf=fixture-csrf; Secure; SameSite=Strict; Path=/";
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(session("professional"))
    .mockResolvedValueOnce(schedule(true))
    .mockResolvedValueOnce(schedule(true, "11:00"));
  renderPage();
  const end = await screen.findByRole("textbox", { name: /Fim do período/ });
  expect(screen.getByRole("heading", { name: "Minha jornada" })).toBeVisible();
  expect(end).toHaveValue("12:00");
  fireEvent.change(end, { target: { value: "11:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));

  expect(await screen.findByText("Jornada salva")).toBeVisible();
  expect(fetch.mock.calls.map(([url]) => url)).toEqual([
    "/api/v1/auth/me",
    path,
    path,
  ]);
  const put = fetch.mock.calls[2][1];
  expect(put?.method).toBe("PUT");
  expect(new Headers(put?.headers).get("X-CSRF-Token")).toBe("fixture-csrf");
  expect(JSON.parse(String(put?.body))).toEqual({
    periods: [{ ...period, end_time: "11:00" }],
  });
  expect(replace).not.toHaveBeenCalled();
});

test("consulta a própria jornada em modo somente leitura", async () => {
  const fetch = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValueOnce(session("professional"))
    .mockResolvedValueOnce(schedule(false));
  renderPage();

  expect(await screen.findByText("Modo somente leitura")).toBeVisible();
  expect(screen.getByDisplayValue("08:00")).toHaveAttribute("readonly");
  expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  expect(fetch.mock.calls.map(([url]) => url)).toEqual([
    "/api/v1/auth/me",
    path,
  ]);
});

test.each([
  ["profissional com troca pendente", session("professional", true), "/testes"],
  ["proprietário", session("owner"), "/painel"],
  ["visitante", new Response(null, { status: 401 }), "/login"],
] as const)(
  "impede o acesso direto de %s antes de montar o editor",
  async (_state, response, destination) => {
    const fetch = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(response);
    renderPage();

    await waitFor(() => expect(replace).toHaveBeenCalledWith(destination));
    expect(
      screen.queryByRole("heading", { name: "Minha jornada" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("form", { name: "Jornada semanal" }),
    ).not.toBeInTheDocument();
    expect(fetch.mock.calls.map(([url]) => url)).toEqual(["/api/v1/auth/me"]);
  },
);
