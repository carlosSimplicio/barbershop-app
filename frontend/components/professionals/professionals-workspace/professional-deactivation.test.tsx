import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  listProfessionals,
  getProfessional,
  deactivateProfessional,
} from "@/lib/professionals/professionals-api";
import { ProfessionalsWorkspace } from "./professionals-workspace";

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/professionals/professionals-api", () => ({
  listProfessionals: vi.fn(),
  createProfessional: vi.fn(),
  getProfessional: vi.fn(),
  updateProfessional: vi.fn(),
  deactivateProfessional: vi.fn(),
}));
const record = {
  id: "ana-id",
  name: "Ana",
  email: "ana@example.test",
  active: true,
};
beforeEach(() => {
  vi.resetAllMocks();
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: {
        user_id: "owner",
        establishment_id: "shop",
        role: "owner",
        email: "owner@example.test",
        password_change_required: false,
      },
    },
    transitioning: false,
    reload: vi.fn(),
    runAction: vi.fn(),
  } as unknown as ReturnType<typeof useAuthSession>);
  vi.mocked(listProfessionals).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      items: [record],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: true,
    },
  });
});
test("exige confirmação identificada e permite cancelar sem escrita", async () => {
  render(<ProfessionalsWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Desativar Ana" }));
  expect(screen.getByText(/sessões serão encerradas/i)).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Cancelar desativação" }));
  expect(screen.getByRole("button", { name: "Desativar Ana" })).toHaveFocus();
  expect(deactivateProfessional).not.toHaveBeenCalled();
});

function result(active: boolean) {
  return {
    state: "success" as const,
    status: 200,
    data: { ...record, active },
  };
}
async function confirmPanel() {
  render(<ProfessionalsWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Desativar Ana" }));
}

test("envia uma única desativação e atualiza a lista sem excluir o cadastro", async () => {
  let finish!: (value: ReturnType<typeof result>) => void;
  vi.mocked(deactivateProfessional).mockImplementation(
    () =>
      new Promise((resolve) => {
        finish = resolve;
      }),
  );
  await confirmPanel();
  const button = screen.getByRole("button", { name: "Confirmar desativação" });
  fireEvent.click(button);
  fireEvent.click(button);
  expect(deactivateProfessional).toHaveBeenCalledExactlyOnceWith(
    "ana-id",
    expect.any(AbortSignal),
  );
  expect(
    screen.getByRole("button", { name: "Atualizar lista" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  ).toBeDisabled();
  await act(async () => finish(result(false)));
  expect(screen.getByRole("list")).toHaveTextContent("Ana");
  expect(screen.getByRole("list")).toHaveTextContent("Inativo");
  expect(screen.getByText("Profissional desativado")).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Fechar confirmação" }));
  expect(
    screen.getByRole("region", { name: "Lista de profissionais" }),
  ).toHaveFocus();
});

test("consulta antes de outra ação após resultado incerto sem repetir POST", async () => {
  vi.mocked(deactivateProfessional).mockRejectedValue(new Error("network"));
  vi.mocked(getProfessional).mockResolvedValue(result(false));
  await confirmPanel();
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  await screen.findByText("Não foi possível confirmar a desativação");
  expect(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Cancelar desativação" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Atualizar lista" }),
  ).toBeDisabled();
  fireEvent.click(
    screen.getByRole("button", { name: "Consultar estado do profissional" }),
  );
  await screen.findByText("Estado consultado");
  expect(screen.getByRole("list")).toHaveTextContent("Inativo");
  expect(deactivateProfessional).toHaveBeenCalledOnce();
  expect(getProfessional).toHaveBeenCalledExactlyOnceWith(
    "ana-id",
    expect.any(AbortSignal),
  );
});

test("sincroniza estado no editor preservando seu rascunho", async () => {
  vi.mocked(getProfessional)
    .mockResolvedValueOnce(result(true))
    .mockResolvedValue(result(false));
  vi.mocked(deactivateProfessional).mockResolvedValue(result(false));
  render(<ProfessionalsWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Editar Ana" }));

  await waitFor(() =>
    expect(screen.getByLabelText("Nome do profissional")).toBeEnabled(),
  );

  fireEvent.change(screen.getByLabelText("Nome do profissional"), {
    target: { value: "Rascunho" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Desativar Ana" }));
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );

  await screen.findByText("Estado: Inativo");
  expect(screen.getByLabelText("Nome do profissional")).toHaveValue("Rascunho");
  expect(screen.getByRole("list")).not.toHaveTextContent("Rascunho");
});

test("read_only bloqueia todas as escritas sem mostrar mensagem bruta", async () => {
  vi.mocked(deactivateProfessional).mockResolvedValue({
    state: "error",
    status: 403,
    error: { code: "read_only", message: "privado", request_id: "id" },
  });
  await confirmPanel();
  fireEvent.click(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  );
  await screen.findByText("Modo somente leitura");
  expect(
    screen.getByRole("button", { name: "Cadastrar profissional" }),
  ).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Confirmar desativação" }),
  ).toBeDisabled();
  expect(screen.queryByText("privado")).not.toBeInTheDocument();
});
