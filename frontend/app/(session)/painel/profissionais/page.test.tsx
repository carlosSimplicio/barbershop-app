import { render, screen, waitFor } from "@testing-library/react";
import { expect, test, vi } from "vitest";

import { AuthProvider } from "@/components/session/auth-provider";

import ProfessionalsPage from "./page";

const replace = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));
vi.mock("@/components/professionals/professionals-list-panel", () => ({
  ProfessionalsListPanel: () => <div>Lista de profissionais</div>,
}));

test("mostra a casca do painel somente para proprietário", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(sessionResponse("owner"));

  render(
    <AuthProvider>
      <ProfessionalsPage />
    </AuthProvider>,
  );

  expect(
    await screen.findByRole("heading", { name: "Profissionais" }),
  ).toBeVisible();
  expect(
    screen.getByRole("navigation", { name: "Navegação do painel" }),
  ).toBeVisible();
  expect(replace).not.toHaveBeenCalled();
});

test("não mostra o painel ao profissional e o redireciona para /testes", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    sessionResponse("professional"),
  );

  render(
    <AuthProvider>
      <ProfessionalsPage />
    </AuthProvider>,
  );

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/testes"));
  expect(
    screen.queryByRole("heading", { name: "Profissionais" }),
  ).not.toBeInTheDocument();
});

function sessionResponse(role: "owner" | "professional") {
  return new Response(
    JSON.stringify({
      user_id: "user-id",
      establishment_id: "establishment-id",
      email: "person@example.test",
      role,
      password_change_required: role === "professional",
    }),
    { headers: { "Content-Type": "application/json" }, status: 200 },
  );
}
