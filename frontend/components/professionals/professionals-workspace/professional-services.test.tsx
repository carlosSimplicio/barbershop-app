import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  getProfessional,
  listProfessionals,
  updateProfessional,
} from "@/lib/professionals/professionals-api";
import {
  listProfessionalServices,
  setProfessionalService,
} from "@/lib/professionals/professional-services-api";
import { ProfessionalsWorkspace } from "./professionals-workspace";

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: vi.fn(),
}));
vi.mock("@/lib/professionals/professionals-api", () => ({
  getProfessional: vi.fn(),
  listProfessionals: vi.fn(),
  updateProfessional: vi.fn(),
  createProfessional: vi.fn(),
  deactivateProfessional: vi.fn(),
  reactivateProfessional: vi.fn(),
}));
vi.mock("@/lib/professionals/professional-services-api", () => ({
  listProfessionalServices: vi.fn(),
  setProfessionalService: vi.fn(),
}));
const record = {
  id: "professional-id",
  name: "Ana",
  email: "ana@example.test",
  active: true,
};
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: { user_id: "owner", establishment_id: "shop", role: "owner" },
    },
  } as ReturnType<typeof useAuthSession>);
  vi.mocked(getProfessional).mockResolvedValue({
    state: "success",
    status: 200,
    data: record,
  });
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
  vi.mocked(listProfessionalServices).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      professional: record,
      items: [],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: true,
    },
  });
});

test("serviços abre sobre edição, preserva rascunho e devolve foco sem salvar", async () => {
  render(<ProfessionalsWorkspace />);
  const origin = await screen.findByRole("button", { name: "Editar Ana" });
  origin.focus();
  fireEvent.click(origin);
  const name = await screen.findByLabelText("Nome do profissional");
  fireEvent.change(name, { target: { value: "Nome em edição" } });
  const services = screen.getByRole("button", { name: "Serviços" });
  fireEvent.click(services);
  const upper = await screen.findByRole("dialog", { name: "Serviços de Ana" });
  await within(upper).findByText("Nenhum serviço cadastrado.");
  expect(name).toHaveValue("Nome em edição");
  expect(name).toBeDisabled();
  expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "Desativar Ana" })).toBeDisabled();
  fireEvent(upper, new Event("cancel", { bubbles: true, cancelable: true }));
  await waitFor(() => expect(services).toHaveFocus());
  expect(
    screen.getByRole("dialog", { name: "Editar profissional" }),
  ).toBeVisible();
  expect(name).toHaveValue("Nome em edição");
  expect(name).toBeEnabled();
  fireEvent.click(screen.getByRole("button", { name: "Cancelar" }));
  await waitFor(() => expect(origin).toHaveFocus());
  expect(updateProfessional).not.toHaveBeenCalled();
  expect(getProfessional).toHaveBeenCalledOnce();
});

test("consulta serviços de inativo em modo somente leitura", async () => {
  vi.mocked(getProfessional).mockResolvedValue({
    state: "success",
    status: 200,
    data: { ...record, active: false },
  });
  vi.mocked(listProfessionals).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      items: [{ ...record, active: false }],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: false,
    },
  });
  render(<ProfessionalsWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Editar Ana" }));
  const services = await screen.findByRole("button", { name: "Serviços" });
  expect(services).toBeEnabled();
  fireEvent.click(services);
  expect(
    await screen.findByRole("dialog", { name: "Serviços de Ana" }),
  ).toBeVisible();
  expect(screen.getByRole("button", { name: "Reativar Ana" })).toBeDisabled();
});

test("salva vínculo sem publicar o nome editado e propaga read_only ao workspace", async () => {
  const offering = {
    id: "offering",
    name: "Corte",
    price_cents: 0,
    duration_minutes: 30,
    active: false,
    associated: false,
  };
  vi.mocked(listProfessionalServices).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      professional: record,
      items: [offering],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: true,
    },
  });
  vi.mocked(setProfessionalService)
    .mockResolvedValueOnce({
      state: "success",
      status: 200,
      data: { ...offering, associated: true },
    })
    .mockResolvedValueOnce({
      state: "error",
      status: 403,
      error: { code: "read_only", message: "raw", request_id: "id" },
    });
  render(<ProfessionalsWorkspace />);
  fireEvent.click(await screen.findByRole("button", { name: "Editar Ana" }));
  const name = await screen.findByLabelText("Nome do profissional");
  fireEvent.change(name, { target: { value: "Nome em edição" } });
  fireEvent.click(screen.getByRole("button", { name: "Serviços" }));
  fireEvent.click(
    await screen.findByRole("button", { name: "Associar Corte" }),
  );
  expect(await screen.findByText("Serviço associado")).toBeVisible();
  expect(name).toHaveValue("Nome em edição");
  expect(screen.getByRole("button", { name: "Editar Ana" })).toBeDisabled();
  fireEvent.click(screen.getByRole("button", { name: "Remover Corte" }));
  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "Remover Corte" }),
    ).toBeDisabled(),
  );
  fireEvent.click(screen.getByRole("button", { name: "Fechar" }));
  expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  expect(name).toHaveValue("Nome em edição");
  expect(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  ).toBeDisabled();
  expect(updateProfessional).not.toHaveBeenCalled();
  expect(listProfessionals).toHaveBeenCalledOnce();
  expect(listProfessionalServices).toHaveBeenCalledOnce();
});
