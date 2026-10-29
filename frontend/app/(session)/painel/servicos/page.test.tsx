import { render, screen, waitFor } from "@testing-library/react";
import { expect, test, vi } from "vitest";

import { AuthProvider } from "@/components/session/auth-provider/auth-provider";

import ServicesPage from "./page";

const replace = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));
vi.mock("@/components/catalog/catalog-workspace/catalog-workspace", () => ({
  CatalogWorkspace: () => <div>Lista de serviços</div>,
}));

test("mostra a casca do painel somente para proprietário", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(sessionResponse("owner"));

  render(
    <AuthProvider>
      <ServicesPage />
    </AuthProvider>,
  );

  expect(
    await screen.findByRole("heading", { name: "Serviços" }),
  ).toBeVisible();
  expect(
    screen.getByRole("navigation", { name: "Navegação do painel" }),
  ).toBeVisible();
  expect(screen.getByRole("link", { name: "Serviços" })).toHaveAttribute(
    "aria-current",
    "page",
  );
  expect(replace).not.toHaveBeenCalled();
});

test("não mostra o painel ao profissional e o redireciona para /testes", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    sessionResponse("professional"),
  );

  render(
    <AuthProvider>
      <ServicesPage />
    </AuthProvider>,
  );

  await waitFor(() => expect(replace).toHaveBeenCalledWith("/testes"));
  expect(
    screen.queryByRole("heading", { name: "Serviços" }),
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
