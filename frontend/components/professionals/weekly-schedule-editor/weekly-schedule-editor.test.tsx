import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, expect, test, vi } from "vitest";
import type { ApiResult } from "@/lib/api/client";
import {
  getProfessionalWeeklySchedule,
  putProfessionalWeeklySchedule,
  type WeeklySchedule,
  type WorkPeriod,
} from "@/lib/professionals/weekly-schedule-api";
import { WeeklyScheduleEditor } from "./weekly-schedule-editor";

vi.mock("@/lib/professionals/weekly-schedule-api", () => ({
  getProfessionalWeeklySchedule: vi.fn(),
  putProfessionalWeeklySchedule: vi.fn(),
}));

const professionalId = "8dd05a16-1e31-4236-93fd-34cd56fc1139";
const secondId = "599b5ad4-8613-486c-b850-443ffaf701ba";
const morning = { weekday: 1, start_time: "08:00", end_time: "12:00" };
const afternoon = { weekday: 1, start_time: "13:00", end_time: "18:00" };

function week(
  periods: WorkPeriod[] = [],
  overrides: Partial<WeeklySchedule> = {},
): ApiResult<WeeklySchedule> {
  return {
    state: "success",
    status: 200,
    data: {
      professional_id: professionalId,
      time_zone: "America/Sao_Paulo",
      can_write: true,
      periods,
      ...overrides,
    },
  };
}

function rejection(status: number, code: string): ApiResult<WeeklySchedule> {
  return {
    state: "error",
    status,
    error: { code, message: "raw message", request_id: "test" },
  };
}

function deferred() {
  let resolve!: (result: ApiResult<WeeklySchedule>) => void;
  const promise = new Promise<ApiResult<WeeklySchedule>>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

function day(name = "Segunda-feira") {
  return screen.getByRole("group", { name });
}

function save() {
  fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
}

beforeEach(() => {
  vi.mocked(getProfessionalWeeklySchedule)
    .mockReset()
    .mockResolvedValue(week());
  vi.mocked(putProfessionalWeeklySchedule).mockReset();
});

test("edita vários períodos, aceita meia-noite e confirma a semana enviada após ordenar", async () => {
  const monday = { weekday: 1, start_time: "22:00", end_time: "24:00" };
  const tuesday = { weekday: 2, start_time: "00:00", end_time: "02:00" };
  vi.mocked(putProfessionalWeeklySchedule).mockResolvedValue(
    week([tuesday, monday]),
  );
  render(<WeeklyScheduleEditor professionalId={professionalId} />);
  expect(await screen.findByText("America/Sao_Paulo")).toBeVisible();
  expect(screen.getAllByRole("group")).toHaveLength(7);
  const mondayControls = within(day());
  const tuesdayControls = within(day("Terça-feira"));
  fireEvent.click(
    mondayControls.getByRole("button", { name: /Adicionar período/ }),
  );
  fireEvent.click(
    mondayControls.getByRole("button", { name: /Adicionar período/ }),
  );
  expect(mondayControls.getAllByRole("textbox")).toHaveLength(4);
  fireEvent.click(
    mondayControls.getAllByRole("button", { name: /Remover período/ })[1],
  );
  expect(mondayControls.getAllByRole("textbox")).toHaveLength(2);
  fireEvent.change(mondayControls.getByRole("textbox", { name: /Início/ }), {
    target: { value: "22:00" },
  });
  fireEvent.change(mondayControls.getByRole("textbox", { name: /Fim/ }), {
    target: { value: "24:00" },
  });
  fireEvent.click(
    tuesdayControls.getByRole("button", { name: /Adicionar período/ }),
  );
  fireEvent.change(tuesdayControls.getByRole("textbox", { name: /Início/ }), {
    target: { value: "00:00" },
  });
  fireEvent.change(tuesdayControls.getByRole("textbox", { name: /Fim/ }), {
    target: { value: "02:00" },
  });
  save();

  expect(await screen.findByText("Jornada salva")).toBeVisible();
  expect(putProfessionalWeeklySchedule).toHaveBeenCalledWith(
    professionalId,
    { periods: [monday, tuesday] },
    expect.any(AbortSignal),
  );
  expect(mondayControls.getByRole("textbox", { name: /Fim/ })).toHaveValue(
    "24:00",
  );
});

test("exibe o fuso da API e confirma o salvamento mesmo sem suporte no Intl", async () => {
  const timeZone = { time_zone: "Factory" };
  const edited = { ...morning, end_time: "11:00" };
  vi.mocked(getProfessionalWeeklySchedule).mockResolvedValue(
    week([morning], timeZone),
  );
  vi.mocked(putProfessionalWeeklySchedule).mockResolvedValue(
    week([edited], timeZone),
  );
  render(<WeeklyScheduleEditor professionalId={professionalId} />);
  expect(await screen.findByText("Factory")).toBeVisible();
  const end = within(day()).getByRole("textbox", { name: /Fim/ });
  fireEvent.change(end, { target: { value: "11:00" } });
  save();

  expect(await screen.findByText("Jornada salva")).toBeVisible();
  expect(end).toHaveValue("11:00");
});

test("envia uma semana vazia e mantém um único envio pendente", async () => {
  vi.mocked(getProfessionalWeeklySchedule).mockResolvedValue(week([morning]));
  const pending = deferred();
  vi.mocked(putProfessionalWeeklySchedule).mockReturnValue(pending.promise);
  const onBusyChange = vi.fn();
  render(
    <WeeklyScheduleEditor
      professionalId={professionalId}
      onBusyChange={onBusyChange}
    />,
  );
  fireEvent.click(
    await screen.findByRole("button", { name: /Remover período/ }),
  );
  save();
  fireEvent.submit(screen.getByRole("form", { name: "Jornada semanal" }));
  expect(screen.getByRole("button", { name: "Salvando…" })).toBeDisabled();
  expect(putProfessionalWeeklySchedule).toHaveBeenCalledExactlyOnceWith(
    professionalId,
    { periods: [] },
    expect.any(AbortSignal),
  );
  await act(async () => pending.resolve(week()));
  expect(await screen.findByText("Jornada salva")).toBeVisible();
  expect(onBusyChange.mock.calls).toEqual([[true], [false]]);
});

test("aponta intervalos invertidos e sobreposição localmente", async () => {
  vi.mocked(getProfessionalWeeklySchedule).mockResolvedValue(
    week([morning, afternoon]),
  );
  render(<WeeklyScheduleEditor professionalId={professionalId} />);
  await screen.findByText("America/Sao_Paulo");
  const endings = within(day()).getAllByRole("textbox", { name: /Fim/ });
  fireEvent.change(endings[0], { target: { value: "07:00" } });
  save();
  expect(screen.getByRole("alert")).toHaveTextContent(/início.*anterior.*fim/i);
  fireEvent.change(endings[0], { target: { value: "14:00" } });
  save();
  expect(screen.getByRole("alert")).toHaveTextContent(/sobreposição/i);
  expect(putProfessionalWeeklySchedule).not.toHaveBeenCalled();
});

test("consulta permanece disponível em modo somente leitura", async () => {
  vi.mocked(getProfessionalWeeklySchedule).mockResolvedValue(
    week([morning], { can_write: false }),
  );
  render(<WeeklyScheduleEditor professionalId={professionalId} />);
  expect(await screen.findByText("Modo somente leitura")).toBeVisible();
  expect(
    within(day()).getByRole("textbox", { name: /Início/ }),
  ).toHaveAttribute("readonly");
  expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  expect(putProfessionalWeeklySchedule).not.toHaveBeenCalled();
});

test("recusa read_only bloqueia outros salvamentos e comunica o painel", async () => {
  vi.mocked(getProfessionalWeeklySchedule).mockResolvedValue(week([morning]));
  vi.mocked(putProfessionalWeeklySchedule).mockResolvedValue(
    rejection(403, "read_only"),
  );
  const onReadOnly = vi.fn();
  render(
    <WeeklyScheduleEditor
      professionalId={professionalId}
      onReadOnly={onReadOnly}
    />,
  );
  await screen.findByText("America/Sao_Paulo");
  const end = within(day()).getByRole("textbox", { name: /Fim/ });
  fireEvent.change(end, { target: { value: "11:00" } });
  save();
  expect(await screen.findByText("Modo somente leitura")).toBeVisible();
  expect(end).toHaveValue("11:00");
  expect(screen.getByRole("button", { name: "Salvar" })).toBeDisabled();
  expect(onReadOnly).toHaveBeenCalledOnce();
});

test("preserva o rascunho após confirmações inválidas e falha temporária até nova tentativa explícita", async () => {
  vi.mocked(getProfessionalWeeklySchedule).mockResolvedValue(week([morning]));
  const edited = { ...morning, end_time: "11:00" };
  vi.mocked(putProfessionalWeeklySchedule)
    .mockResolvedValueOnce(week([afternoon]))
    .mockResolvedValueOnce(week([edited], { time_zone: " " }))
    .mockRejectedValueOnce(new Error("network"))
    .mockResolvedValueOnce(week([edited]));
  render(<WeeklyScheduleEditor professionalId={professionalId} />);
  await screen.findByText("America/Sao_Paulo");
  const end = within(day()).getByRole("textbox", { name: /Fim/ });
  fireEvent.change(end, { target: { value: "11:00" } });
  for (let attempt = 1; attempt <= 3; attempt++) {
    save();
    expect(
      await screen.findByText("Não foi possível salvar a jornada"),
    ).toBeVisible();
    expect(screen.queryByText("Jornada salva")).not.toBeInTheDocument();
    expect(end).toHaveValue("11:00");
    expect(putProfessionalWeeklySchedule).toHaveBeenCalledTimes(attempt);
    expect(getProfessionalWeeklySchedule).toHaveBeenCalledOnce();
  }
  save();
  expect(await screen.findByText("Jornada salva")).toBeVisible();
  expect(putProfessionalWeeklySchedule).toHaveBeenLastCalledWith(
    professionalId,
    { periods: [edited] },
    expect.any(AbortSignal),
  );
});

test("permite tentar novamente uma consulta temporariamente indisponível", async () => {
  vi.mocked(getProfessionalWeeklySchedule).mockRejectedValueOnce(
    new Error("network"),
  );
  render(<WeeklyScheduleEditor professionalId={professionalId} />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Tentar novamente" }),
  );
  expect(await screen.findByText("America/Sao_Paulo")).toBeVisible();
  expect(getProfessionalWeeklySchedule).toHaveBeenCalledTimes(2);
});

test("401 no salvamento retira a semana protegida", async () => {
  vi.mocked(getProfessionalWeeklySchedule).mockResolvedValue(week([morning]));
  vi.mocked(putProfessionalWeeklySchedule).mockResolvedValue(
    rejection(401, "authentication_required"),
  );
  const onUnauthorized = vi.fn();
  render(
    <WeeklyScheduleEditor
      professionalId={professionalId}
      onUnauthorized={onUnauthorized}
    />,
  );
  await screen.findByText("America/Sao_Paulo");
  save();
  await act(async () => {});
  expect(screen.queryByRole("form")).not.toBeInTheDocument();
  expect(screen.queryByDisplayValue("08:00")).not.toBeInTheDocument();
  expect(onUnauthorized).toHaveBeenCalledOnce();
});

test("descarta a consulta anterior ao trocar de profissional", async () => {
  const pending = deferred();
  vi.mocked(getProfessionalWeeklySchedule)
    .mockReturnValueOnce(pending.promise)
    .mockResolvedValueOnce(week([afternoon], { professional_id: secondId }));
  const { rerender } = render(
    <WeeklyScheduleEditor professionalId={professionalId} />,
  );
  const signal = vi.mocked(getProfessionalWeeklySchedule).mock.calls[0][1];
  rerender(<WeeklyScheduleEditor professionalId={secondId} />);
  expect(await screen.findByDisplayValue("13:00")).toBeVisible();
  await act(async () => pending.resolve(week([morning])));
  expect(signal?.aborted).toBe(true);
  expect(screen.queryByDisplayValue("08:00")).not.toBeInTheDocument();
});

test("descarta resultado de gravação obsoleta ao trocar de sessão", async () => {
  vi.mocked(getProfessionalWeeklySchedule)
    .mockResolvedValueOnce(week([morning]))
    .mockResolvedValueOnce(week([afternoon]));
  const pending = deferred();
  vi.mocked(putProfessionalWeeklySchedule).mockReturnValue(pending.promise);
  const onUnauthorized = vi.fn();
  const { rerender } = render(
    <WeeklyScheduleEditor
      professionalId={professionalId}
      sessionKey="first"
      onUnauthorized={onUnauthorized}
    />,
  );
  await screen.findByDisplayValue("08:00");
  save();
  const signal = vi.mocked(putProfessionalWeeklySchedule).mock.calls[0][2];
  rerender(
    <WeeklyScheduleEditor
      professionalId={professionalId}
      sessionKey="second"
      onUnauthorized={onUnauthorized}
    />,
  );
  expect(await screen.findByDisplayValue("13:00")).toBeVisible();
  await act(async () =>
    pending.resolve(rejection(401, "authentication_required")),
  );
  expect(signal?.aborted).toBe(true);
  expect(onUnauthorized).not.toHaveBeenCalled();
  expect(screen.getByDisplayValue("13:00")).toBeVisible();
});

test("trocar o callback de sessão conserva a consulta e o rascunho da mesma identidade", async () => {
  vi.mocked(getProfessionalWeeklySchedule).mockResolvedValue(week([morning]));
  const { rerender } = render(
    <WeeklyScheduleEditor
      professionalId={professionalId}
      onUnauthorized={vi.fn()}
    />,
  );
  const end = await screen.findByDisplayValue("12:00");
  fireEvent.change(end, { target: { value: "11:00" } });
  rerender(
    <WeeklyScheduleEditor
      professionalId={professionalId}
      onUnauthorized={vi.fn()}
    />,
  );
  await act(async () => {});
  expect(getProfessionalWeeklySchedule).toHaveBeenCalledOnce();
  expect(end).toHaveValue("11:00");
});
