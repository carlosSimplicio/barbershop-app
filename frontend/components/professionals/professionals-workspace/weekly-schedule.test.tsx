import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import type { ApiResult } from "@/lib/api/client";
import { useAuthSession } from "@/components/session/auth-provider/auth-provider";
import {
  getProfessional,
  listProfessionals,
  updateProfessional,
} from "@/lib/professionals/professionals-api";
import { listProfessionalServices } from "@/lib/professionals/professional-services-api";
import {
  getProfessionalWeeklySchedule,
  putProfessionalWeeklySchedule,
  type WeeklySchedule,
} from "@/lib/professionals/weekly-schedule-api";
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
vi.mock("@/lib/professionals/weekly-schedule-api", () => ({
  getProfessionalWeeklySchedule: vi.fn(),
  putProfessionalWeeklySchedule: vi.fn(),
}));

const ana = {
  id: "f8d20d6e-2d32-459e-8e81-dcd8d9ac88ed",
  name: "Ana",
  email: "ana@example.test",
  active: true,
};
const bia = {
  id: "c72e6257-b4ea-4985-9500-5c876f211801",
  name: "Bia",
  email: "bia@example.test",
  active: false,
};
const period = { weekday: 1, start_time: "08:00", end_time: "12:00" };

function list(canWrite: boolean) {
  vi.mocked(listProfessionals).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      items: [ana, bia],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: canWrite,
    },
  });
}

function pressEscape(dialog: HTMLElement) {
  fireEvent(dialog, new Event("cancel", { bubbles: true, cancelable: true }));
}

async function openEdit(name: string) {
  fireEvent.click(
    await screen.findByRole("button", { name: `Editar ${name}` }),
  );
  const edit = screen.getByRole("dialog", { name: "Editar profissional" });
  await within(edit).findByLabelText("Nome do profissional");
  return edit;
}

beforeEach(() => {
  vi.resetAllMocks();
  vi.mocked(useAuthSession).mockReturnValue({
    state: {
      status: "authenticated",
      principal: { user_id: "owner", establishment_id: "shop", role: "owner" },
    },
  } as ReturnType<typeof useAuthSession>);
  list(true);
  vi.mocked(getProfessional).mockImplementation(async (id) => ({
    state: "success",
    status: 200,
    data: id === ana.id ? ana : bia,
  }));
  vi.mocked(getProfessionalWeeklySchedule).mockImplementation(async (id) => ({
    state: "success",
    status: 200,
    data: {
      professional_id: id,
      time_zone: "America/Sao_Paulo",
      can_write: true,
      periods: [period],
    },
  }));
  vi.mocked(listProfessionalServices).mockResolvedValue({
    state: "success",
    status: 200,
    data: {
      professional: ana,
      items: [],
      limit: 20,
      offset: 0,
      has_more: false,
      can_write: true,
    },
  });
});

test("abre a jornada do alvo escolhido e devolve o foco sem perder a edição cadastral", async () => {
  render(<ProfessionalsWorkspace />);
  const edit = await openEdit("Bia");
  const name = within(edit).getByLabelText("Nome do profissional");
  fireEvent.change(name, { target: { value: "Nome em edição" } });
  const trigger = within(edit).getByRole("button", { name: "Jornada" });
  fireEvent.click(trigger);
  const dialog = screen.getByRole("dialog", { name: "Jornada de Bia" });
  expect(await within(dialog).findByDisplayValue("08:00")).toBeVisible();
  expect(getProfessionalWeeklySchedule).toHaveBeenCalledExactlyOnceWith(
    bia.id,
    expect.any(AbortSignal),
  );
  expect(name).toHaveValue("Nome em edição");
  expect(name).toBeDisabled();
  const cancel = within(dialog).getByRole("button", { name: "Cancelar" });
  expect(cancel).toHaveAttribute("data-variant", "secondary");
  fireEvent.click(cancel);
  await waitFor(() => expect(trigger).toHaveFocus());
  expect(name).toBeEnabled();
  expect(name).toHaveValue("Nome em edição");
  expect(getProfessional).toHaveBeenCalledOnce();
  expect(updateProfessional).not.toHaveBeenCalled();
});

test("coordena serviços e estado com a modal de jornada e bloqueia fechamento durante envio", async () => {
  let finish!: (result: ApiResult<WeeklySchedule>) => void;
  vi.mocked(putProfessionalWeeklySchedule).mockReturnValue(
    new Promise((resolve) => {
      finish = resolve;
    }),
  );
  render(<ProfessionalsWorkspace />);
  const edit = await openEdit("Ana");
  const editControls = within(edit);
  const schedule = editControls.getByRole("button", { name: "Jornada" });
  const services = editControls.getByRole("button", { name: "Serviços" });
  const deactivate = editControls.getByRole("button", {
    name: "Desativar Ana",
  });
  fireEvent.click(services);
  expect(schedule).toBeDisabled();
  pressEscape(await screen.findByRole("dialog", { name: "Serviços de Ana" }));
  await waitFor(() => expect(services).toHaveFocus());
  fireEvent.click(deactivate);
  expect(schedule).toBeDisabled();
  const confirmation = screen.getAllByRole("dialog").at(-1)!;
  pressEscape(confirmation);
  await waitFor(() => expect(deactivate).toHaveFocus());

  fireEvent.click(schedule);
  const dialog = screen.getByRole("dialog", { name: "Jornada de Ana" });
  const controls = within(dialog);
  expect(services).toBeDisabled();
  expect(deactivate).toBeDisabled();
  fireEvent.click(await controls.findByRole("button", { name: "Salvar" }));
  expect(controls.getByRole("button", { name: "Cancelar" })).toBeDisabled();
  expect(controls.getByRole("button", { name: "Fechar modal" })).toBeDisabled();
  pressEscape(dialog);
  expect(dialog).toBeVisible();
  expect(edit).toBeVisible();

  await act(async () =>
    finish({
      state: "error",
      status: 403,
      error: { code: "read_only", message: "raw", request_id: "test" },
    }),
  );
  expect(controls.getByText("Modo somente leitura")).toBeVisible();
  expect(controls.getByRole("button", { name: "Salvar" })).toBeDisabled();
  pressEscape(dialog);
  await waitFor(() => expect(schedule).toHaveFocus());
  expect(
    screen.getByRole("button", { name: "Adicionar profissional" }),
  ).toBeDisabled();
  expect(deactivate).toBeDisabled();
  expect(editControls.getByLabelText("Nome do profissional")).toBeDisabled();
  expect(schedule).toBeEnabled();
});

test("lista somente leitura permite consultar a jornada de profissional inativo", async () => {
  list(false);
  render(<ProfessionalsWorkspace />);
  const edit = await openEdit("Bia");
  const schedule = within(edit).getByRole("button", { name: "Jornada" });
  expect(schedule).toBeEnabled();
  fireEvent.click(schedule);
  const controls = within(
    screen.getByRole("dialog", { name: "Jornada de Bia" }),
  );
  expect(await controls.findByDisplayValue("08:00")).toHaveAttribute(
    "readonly",
  );
  expect(controls.getByRole("button", { name: "Salvar" })).toBeDisabled();
  expect(controls.getByText("Modo somente leitura")).toBeVisible();
  expect(putProfessionalWeeklySchedule).not.toHaveBeenCalled();
});
