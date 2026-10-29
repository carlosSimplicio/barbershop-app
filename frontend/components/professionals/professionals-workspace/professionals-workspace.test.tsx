import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";

import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  createProfessional,
  listProfessionals,
} from "@/lib/professionals/professionals-api";

import { ProfessionalsWorkspace } from "./professionals-workspace";

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/professionals/professionals-api", () => ({
  createProfessional: vi.fn(),
  listProfessionals: vi.fn(),
}));

beforeEach(() => {
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: {
        user_id: "owner-id",
        establishment_id: "shop-id",
        email: "owner@example.test",
        role: "owner",
        password_change_required: false,
      },
    },
    transitioning: false,
    reload: vi.fn(),
    runAction: vi.fn(),
  } as unknown as ReturnType<typeof useAuthSession>);
  vi.mocked(createProfessional).mockReset();
  vi.mocked(listProfessionals).mockReset();
});

test("atualiza a lista após criar o profissional", async () => {
  vi.mocked(listProfessionals)
    .mockResolvedValueOnce(page([]))
    .mockResolvedValueOnce(
      page([
        {
          id: "professional-id",
          name: "Ana Lima",
          email: "ana@example.test",
          active: true,
        },
      ]),
    );
  vi.mocked(createProfessional).mockResolvedValue({
    state: "success",
    status: 201,
    data: {
      id: "professional-id",
      name: "Ana Lima",
      email: "ana@example.test",
      active: true,
    },
  });

  render(<ProfessionalsWorkspace />);
  await screen.findByText("Nenhum profissional cadastrado.");
  fireEvent.change(screen.getByLabelText("Nome"), {
    target: { value: "Ana Lima" },
  });
  fireEvent.change(screen.getByLabelText("E-mail"), {
    target: { value: "ana@example.test" },
  });
  fireEvent.change(screen.getByLabelText("Senha temporária"), {
    target: { value: "senha-segura" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  );

  expect(await screen.findByText("Ana Lima")).toBeInTheDocument();
  await waitFor(() => expect(listProfessionals).toHaveBeenCalledTimes(2));
});

function page(
  items: Array<{
    id: string;
    name: string;
    email: string;
    active: boolean;
  }>,
) {
  return {
    state: "success" as const,
    status: 200,
    data: { items, limit: 20, offset: 0, has_more: false, can_write: true },
  };
}
