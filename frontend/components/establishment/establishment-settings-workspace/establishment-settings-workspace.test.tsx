import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, test, vi } from "vitest";

import type { EstablishmentSettings } from "@/lib/establishment/establishment-api";
import { BRAZILIAN_TIME_ZONES } from "@/lib/establishment/brazilian-time-zones";

import { EstablishmentSettingsWorkspace } from "./establishment-settings-workspace";

const session = vi.hoisted(() => ({
  establishmentId: "establishment-id",
  userId: "owner-id",
}));
const getSettingsMock = vi.hoisted(() => vi.fn());
const patchSettingsMock = vi.hoisted(() => vi.fn());

vi.mock("@/components/session/auth-provider/auth-provider", () => ({
  useAuthSession: () => ({
    state: {
      status: "authenticated",
      principal: {
        establishment_id: session.establishmentId,
        email: "owner@example.test",
        password_change_required: false,
        role: "owner",
        user_id: session.userId,
      },
    },
  }),
}));

vi.mock("@/lib/establishment/establishment-api", () => ({
  getEstablishmentSettings: getSettingsMock,
  patchEstablishmentSettings: patchSettingsMock,
}));

beforeEach(() => {
  session.establishmentId = "establishment-id";
  session.userId = "owner-id";
  getSettingsMock.mockReset();
  patchSettingsMock.mockReset();
});

afterEach(() => vi.restoreAllMocks());

test("carrega dados do perfil e apresenta o slug somente para leitura", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());

  render(<EstablishmentSettingsWorkspace />);

  expect(await screen.findByLabelText("Nome do estabelecimento")).toBeVisible();
  expect(screen.getByLabelText("Nome do estabelecimento")).toHaveValue(
    "Estúdio Aurora",
  );
  expect(screen.getByLabelText("Slug")).toHaveValue("estudio-aurora");
  expect(screen.getByLabelText("Slug")).toBeDisabled();
  expect(screen.getByRole("combobox", { name: "Rede social 1" })).toHaveValue(
    "instagram",
  );
  expect(
    Array.from(
      screen
        .getByRole("combobox", { name: "Rede social 1" })
        .querySelectorAll("option"),
    ).map((option) => option.textContent?.trim()),
  ).toEqual([
    "Selecione uma rede",
    "Instagram",
    "Facebook",
    "X",
    "YouTube",
    "TikTok",
    "LinkedIn",
  ]);
  expect(screen.getByLabelText("URL do Instagram")).toHaveValue("aurora");
  expect(screen.getByText("https://instagram.com/")).toBeVisible();
  expect(
    screen.getByRole("button", { name: "Adicionar rede social" }),
  ).toHaveTextContent("+");
  expect(
    screen.getByRole("button", { name: "Remover rede social Instagram" }),
  ).toHaveTextContent("×");
  expect(
    Array.from(
      document.querySelectorAll(
        "#name, #slug, #time_zone, #contact_phone, #description, #address",
      ),
    ).map((field) => field.id),
  ).toEqual([
    "name",
    "slug",
    "time_zone",
    "contact_phone",
    "description",
    "address",
  ]);
  const timeZoneSelect = screen.getByRole("combobox", { name: "Fuso horário" });
  expect(timeZoneSelect).toHaveValue("America/Sao_Paulo");
  expect(
    Array.from(timeZoneSelect.querySelectorAll("option")).map(
      (option) => (option as HTMLOptionElement).value,
    ),
  ).toEqual(BRAZILIAN_TIME_ZONES.map(({ value }) => value));
  expect(
    Array.from(timeZoneSelect.querySelectorAll("option")).map((option) =>
      option.textContent?.trim(),
    ),
  ).toEqual(BRAZILIAN_TIME_ZONES.map(({ label }) => label));
  expect(screen.queryByLabelText("Identificador IANA")).not.toBeInTheDocument();
  expect(screen.getByLabelText("Descrição")).toHaveValue("Cortes e barba");
  expect(screen.getByLabelText("Prazo de antecedência mínima")).toHaveValue(30);
});

test("preserva como legado um link cujo rótulo não corresponde ao domínio", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(
      successfulSettings({
        social_links: [
          { label: "Instagram", url: "https://example.test/profile" },
        ],
      }),
    ),
  );
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  expect(await screen.findByLabelText("Descrição")).toBeVisible();
  expect(screen.getByRole("combobox", { name: "Rede social 1" })).toHaveValue(
    "",
  );
  expect(screen.getByText(/Este link existente não corresponde/)).toBeVisible();

  fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
    target: { value: "Novo nome" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    { name: "Novo nome" },
    expect.any(AbortSignal),
  );
});

test("permite limpar dados opcionais, links e limites", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Descrição"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Endereço"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Telefone de contato"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Prazo de antecedência mínima"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Horizonte máximo de agendamento"), {
    target: { value: "" },
  });
  fireEvent.change(screen.getByLabelText("Prazo para alterar agendamento"), {
    target: { value: "" },
  });
  fireEvent.click(
    screen.getByRole("button", { name: "Remover rede social Instagram" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    {
      description: null,
      address: null,
      contact_phone: null,
      social_links: [],
      minimum_booking_notice_minutes: null,
      booking_horizon_days: null,
      customer_change_deadline_minutes: null,
    },
    expect.any(AbortSignal),
  );
});

test("valida limite inteiro e URL HTTPS antes de enviar", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Horizonte máximo de agendamento"), {
    target: { value: "0" },
  });
  fireEvent.change(screen.getByLabelText("URL do Instagram"), {
    target: { value: "" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  expect(
    await screen.findByText("Informe um número inteiro maior que zero."),
  ).toBeVisible();
  expect(screen.getByText("Informe o endereço do perfil.")).toBeVisible();
  expect(patchSettingsMock).not.toHaveBeenCalled();
});

test("seleciona uma rede social e mantém o prefixo da URL fora do campo editável", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(successfulSettings({ social_links: [] })),
  );
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);
  await screen.findByLabelText("Descrição");

  fireEvent.click(
    screen.getByRole("button", { name: "Adicionar rede social" }),
  );
  fireEvent.change(screen.getByRole("combobox", { name: "Rede social 1" }), {
    target: { value: "linkedin" },
  });

  expect(screen.getByText("https://linkedin.com/in/")).toBeVisible();
  expect(screen.getByLabelText("URL do LinkedIn")).toHaveValue("");
  expect(screen.getByLabelText("URL do LinkedIn")).toHaveFocus();
  fireEvent.change(screen.getByLabelText("URL do LinkedIn"), {
    target: { value: "estudio-aurora" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    {
      social_links: [
        {
          label: "LinkedIn",
          url: "https://linkedin.com/in/estudio-aurora",
        },
      ],
    },
    expect.any(AbortSignal),
  );
});

test("permite consultar novamente após uma falha temporária sem mostrar erro bruto", async () => {
  getSettingsMock
    .mockRejectedValueOnce(new Error("internal database details"))
    .mockResolvedValueOnce(successfulResult());

  render(<EstablishmentSettingsWorkspace />);

  expect(await screen.findByText("Tente novamente mais tarde.")).toBeVisible();
  expect(
    screen.queryByText("internal database details"),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));

  expect(await screen.findByLabelText("Descrição")).toHaveValue(
    "Cortes e barba",
  );
});

test("mapeia erro do telefone sem renderizar mensagem bruta do backend", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  patchSettingsMock.mockResolvedValue({
    error: {
      code: "validation_failed",
      fields: [
        {
          code: "invalid_phone",
          field: "contact_phone",
          message: "backend internal message",
        },
      ],
      message: "backend internal message",
      request_id: "request-id",
    },
    state: "error",
    status: 400,
  });

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Telefone de contato"), {
    target: { value: "+5511988887777" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  expect(
    await screen.findByText("Informe um telefone brasileiro com DDD."),
  ).toBeVisible();
  expect(
    screen.queryByText("backend internal message"),
  ).not.toBeInTheDocument();
});

test("bloqueia escrita e alteração de fuso conforme as permissões do backend", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(
      successfulSettings({ can_write: false, time_zone_editable: false }),
    ),
  );

  render(<EstablishmentSettingsWorkspace />);

  expect(
    await screen.findByText(
      "O trial expirou. Você pode consultar as configurações, mas não salvá-las.",
    ),
  ).toBeVisible();
  expect(screen.getByLabelText("Fuso horário")).toBeDisabled();
  expect(screen.getByLabelText("Descrição")).toBeDisabled();
  expect(
    screen.getByRole("button", { name: "Salvar configurações" }),
  ).toBeDisabled();
  expect(patchSettingsMock).not.toHaveBeenCalled();
});

test("mantém o fuso bloqueado sem impedir a edição dos demais dados", async () => {
  const currentSettings = successfulSettings({ time_zone_editable: false });
  getSettingsMock
    .mockResolvedValueOnce(successfulResult(currentSettings))
    .mockResolvedValueOnce(
      successfulResult({ ...currentSettings, description: "Nova descrição" }),
    );
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  expect(screen.getByLabelText("Fuso horário")).toBeDisabled();
  fireEvent.change(screen.getByLabelText("Descrição"), {
    target: { value: "Nova descrição" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    { description: "Nova descrição" },
    expect.any(AbortSignal),
  );
});

test("permite escolher qualquer fuso listado e salva a opção selecionada", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  patchSettingsMock.mockResolvedValue({ state: "empty", status: 204 });

  render(<EstablishmentSettingsWorkspace />);

  const timeZoneSelect = await screen.findByRole("combobox", {
    name: "Fuso horário",
  });
  fireEvent.change(timeZoneSelect, {
    target: { value: "America/Manaus" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  expect(patchSettingsMock).toHaveBeenCalledWith(
    { time_zone: "America/Manaus" },
    expect.any(AbortSignal),
  );
});

test("não representa fuso fora da lista como se fosse uma opção brasileira", async () => {
  getSettingsMock.mockResolvedValue(
    successfulResult(successfulSettings({ time_zone: "America/New_York" })),
  );

  render(<EstablishmentSettingsWorkspace />);

  const timeZoneSelect = await screen.findByRole("combobox", {
    name: "Fuso horário",
  });
  expect(timeZoneSelect).toHaveValue("");
  expect(
    Array.from(timeZoneSelect.querySelectorAll("option"))
      .map((option) => (option as HTMLOptionElement).value)
      .filter(Boolean),
  ).toEqual(BRAZILIAN_TIME_ZONES.map(({ value }) => value));
  expect(
    screen.getByRole("option", { name: "Selecione um fuso horário" }),
  ).toBeDisabled();

  fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
    target: { value: "Novo nome" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  expect(
    await screen.findByText(
      "Escolha um dos quatro fusos horários brasileiros.",
    ),
  ).toBeVisible();
  expect(patchSettingsMock).not.toHaveBeenCalled();
});

test("não exibe detalhes brutos quando o salvamento falha temporariamente", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  patchSettingsMock.mockRejectedValue(new Error("database connection details"));

  render(<EstablishmentSettingsWorkspace />);

  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
    target: { value: "Novo nome" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Salvar configurações" }));

  expect(
    await screen.findByText("Verifique sua conexão e tente novamente."),
  ).toBeVisible();
  expect(
    screen.queryByText("database connection details"),
  ).not.toBeInTheDocument();
});

test("ignora a resposta de um tenant anterior após troca de sessão", async () => {
  let resolveOldRequest:
    ((value: ReturnType<typeof successfulResult>) => void) | undefined;
  getSettingsMock.mockImplementation((signal: AbortSignal) => {
    if (session.establishmentId === "old-establishment") {
      return new Promise((resolve) => {
        resolveOldRequest = resolve;
        signal.addEventListener("abort", () =>
          resolve(successfulResult(successfulSettings({ name: "Antigo" }))),
        );
      });
    }
    return Promise.resolve(
      successfulResult(successfulSettings({ name: "Atual" })),
    );
  });
  session.establishmentId = "old-establishment";

  const { rerender } = render(<EstablishmentSettingsWorkspace />);
  session.establishmentId = "new-establishment";
  session.userId = "new-owner";
  rerender(<EstablishmentSettingsWorkspace />);

  expect(await screen.findByLabelText("Nome do estabelecimento")).toHaveValue(
    "Atual",
  );
  resolveOldRequest?.(successfulResult(successfulSettings({ name: "Antigo" })));
  expect(screen.getByLabelText("Nome do estabelecimento")).toHaveValue("Atual");
});

test("impede submissões simultâneas", async () => {
  getSettingsMock.mockResolvedValue(successfulResult());
  let resolvePatch:
    ((value: { state: "empty"; status: number }) => void) | undefined;
  patchSettingsMock.mockImplementation(
    () =>
      new Promise((resolve) => {
        resolvePatch = resolve;
      }),
  );

  render(<EstablishmentSettingsWorkspace />);
  await screen.findByLabelText("Descrição");
  fireEvent.change(screen.getByLabelText("Nome do estabelecimento"), {
    target: { value: "Novo nome" },
  });
  const form = screen
    .getByRole("button", { name: "Salvar configurações" })
    .closest("form");
  if (!form) throw new Error("Settings form was not rendered");
  fireEvent.submit(form);
  fireEvent.submit(form);

  await waitFor(() => expect(patchSettingsMock).toHaveBeenCalledOnce());
  resolvePatch?.({ state: "empty", status: 204 });
});

function successfulSettings(
  overrides: Partial<EstablishmentSettings> = {},
): EstablishmentSettings {
  return {
    id: "establishment-id",
    name: "Estúdio Aurora",
    slug: "estudio-aurora",
    time_zone: "America/Sao_Paulo",
    description: "Cortes e barba",
    address: "Rua Central, 123",
    contact_phone: "+5511999990000",
    social_links: [{ label: "Instagram", url: "https://instagram.com/aurora" }],
    minimum_booking_notice_minutes: 30,
    booking_horizon_days: 60,
    customer_change_deadline_minutes: 120,
    logo: null,
    time_zone_editable: true,
    can_write: true,
    slug_conflict: false,
    is_complete: false,
    is_publishable: false,
    ...overrides,
  };
}

function successfulResult(settings = successfulSettings()) {
  return { data: settings, state: "success", status: 200 } as const;
}
